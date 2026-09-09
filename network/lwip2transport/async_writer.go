// Copyright 2026 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lwip2transport

import (
	"io"
	"sync"

	"golang.getoutline.org/sdk/network"
)

const (
	// writeQueueDepth bounds the number of outgoing packets queued for the destination writer.
	// At packetBufCap per slot this caps queue memory at ~512 KiB. Once full, Write blocks, which
	// is genuine backpressure: the destination is truly slower than lwIP is producing.
	writeQueueDepth = 256

	// packetBufCap is the capacity of a pooled packet buffer. It is larger than packetMTU so that
	// a full-MTU packet never forces the append in Write to reallocate.
	packetBufCap = 2048
)

// asyncWriter decouples the caller of Write from the latency of the underlying writer. Write copies
// the packet into a pooled buffer, queues it, and returns; a background goroutine performs the
// actual writes.
//
// This exists because lwIP invokes its output callback while holding the global stack lock. Any
// blocking work on that path -- notably a write(2) to a TUN device -- serializes every TCP and UDP
// flow in the stack behind one syscall.
//
// An asyncWriter is not safe for concurrent use by multiple goroutines; it is created and used by a
// single [lwIPDevice.WriteTo] call.
type asyncWriter struct {
	w    io.Writer
	pool *sync.Pool // borrowed from the lwIPDevice, not owned

	ch   chan []byte
	done chan struct{}
	exit chan struct{} // closed when drain returns

	closeOnce sync.Once

	mu  sync.Mutex
	err error // first error observed by drain
}

// newAsyncWriter returns an [asyncWriter] that writes to w from a background goroutine, recycling
// packet buffers through the device's pool. The caller must Close it to stop that goroutine.
func (d *lwIPDevice) newAsyncWriter(w io.Writer) *asyncWriter {
	aw := &asyncWriter{
		w:    w,
		pool: &d.pktPool,
		ch:   make(chan []byte, writeQueueDepth),
		done: make(chan struct{}),
		exit: make(chan struct{}),
	}
	go aw.drain()
	return aw
}

// Write copies p into a pooled buffer, queues it for the underlying writer, and returns len(p). It
// does NOT wait for the write to complete.
//
// The copy is what makes it safe for the caller to reuse or release p as soon as Write returns --
// on the lwIP path p aliases the C pbuf payload, which is freed once the output callback returns.
//
// Write reports the first error encountered by the background writer, so an error surfaces on a
// later call than the packet that caused it. It returns [network.ErrClosed] once the asyncWriter has
// been closed.
//
// A Write parked on a full queue must also wake when the background writer stops, which is why it
// waits on aw.exit as well as aw.done: if the destination fails, nothing will ever drain the queue
// again, and aw.done is closed by a Close that cannot run until this Write returns.
func (aw *asyncWriter) Write(p []byte) (int, error) {
	if err := aw.loadErr(); err != nil {
		return 0, err
	}
	// Checked before the send below, which would otherwise be free to pick the (still empty) queue
	// over the closed done channel.
	select {
	case <-aw.done:
		return 0, network.ErrClosed
	default:
	}
	bufp := aw.pool.Get().(*[]byte)
	*bufp = append((*bufp)[:0], p...)
	select {
	case aw.ch <- *bufp:
		return len(p), nil
	case <-aw.exit:
		// The background writer is gone and nothing will drain the queue again.
		aw.recycle(*bufp)
		if err := aw.loadErr(); err != nil {
			return 0, err
		}
		return 0, network.ErrClosed
	case <-aw.done:
		aw.recycle(*bufp)
		return 0, network.ErrClosed
	}
}

// writeOwned queues an already-pooled buffer and takes ownership of it. Unlike Write it does not
// copy, so the caller must not touch buf afterwards. It is used by [lwIPDevice.WriteTo], where the
// packet was already copied out of the lwIP pbuf by forwardOutgoingIPPacket -- copying again here
// would be pure waste.
func (aw *asyncWriter) writeOwned(buf []byte) (int, error) {
	if err := aw.loadErr(); err != nil {
		aw.recycle(buf)
		return 0, err
	}
	select {
	case <-aw.done:
		aw.recycle(buf)
		return 0, network.ErrClosed
	default:
	}
	n := len(buf)
	select {
	case aw.ch <- buf:
		return n, nil
	case <-aw.exit:
		aw.recycle(buf)
		if err := aw.loadErr(); err != nil {
			return 0, err
		}
		return 0, network.ErrClosed
	case <-aw.done:
		aw.recycle(buf)
		return 0, network.ErrClosed
	}
}

// drain writes queued packets until the asyncWriter is closed or a write fails. It stops on the
// first error: the underlying writer is a device, and a failed write to it is not transient.
func (aw *asyncWriter) drain() {
	defer close(aw.exit)
	for {
		select {
		case pkt := <-aw.ch:
			_, err := aw.w.Write(pkt)
			aw.recycle(pkt)
			if err != nil {
				aw.storeErr(err)
				return
			}
		case <-aw.done:
			return
		}
	}
}

// Close stops the background writer and returns the first error it observed, if any.
//
// Packets still queued are dropped rather than flushed. Close runs when the device is going away,
// and flushing would let a stuck write on the destination block shutdown indefinitely. Close does
// wait for the in-flight write to finish, so the underlying writer is idle when it returns.
//
// Close does not close the underlying writer.
func (aw *asyncWriter) Close() error {
	aw.closeOnce.Do(func() { close(aw.done) })
	<-aw.exit
	for {
		select {
		case pkt := <-aw.ch:
			aw.recycle(pkt)
		default:
			return aw.loadErr()
		}
	}
}

// recycle returns a packet buffer to the pool once it has been written.
func (aw *asyncWriter) recycle(s []byte) {
	if cap(s) >= packetBufCap {
		s = s[:0]
		aw.pool.Put(&s)
	}
}

func (aw *asyncWriter) loadErr() error {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	return aw.err
}

func (aw *asyncWriter) storeErr(err error) {
	aw.mu.Lock()
	defer aw.mu.Unlock()
	if aw.err == nil {
		aw.err = err
	}
}
