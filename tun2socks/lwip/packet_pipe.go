// Copyright 2023 Jigsaw Operations LLC
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

package lwip

import (
	"errors"
	"io"
	"net"
	"sync"
)

// A thread safe in-memory IP packet buffer for LwIPTun2SocksDevice's read and write.
// Keep in mind there might be multiple reads and writes happending at the same time.
type packetPipe struct {
	mu   sync.Mutex
	once sync.Once
	eof  bool

	// read buffer (when no WriteTo has been called)
	mtu int
	buf net.Buffers
	rd  chan int

	// WriteTo writer (when an active WriteTo is in-progress)
	wr      io.Writer
	wrDone  chan struct{}
	wrBytes int64
	wrErr   error
}

func newPacketPipe(mtu int) *packetPipe {
	return &packetPipe{
		mtu: mtu,
		rd:  make(chan int),
	}
}

func (b *packetPipe) close() {
	b.once.Do(func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.eof = true
		close(b.rd)
	})
}

// read tries to peel one packet from the internal `buf`. If len(buf) == 0, read will block until next packet arrives.
func (b *packetPipe) read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.eof {
		return 0, io.EOF
	}

	// wait until we have at least one valid data in the buffer (blocking)
	for len(b.buf) == 0 {
		b.mu.Unlock()
		<-b.rd
		b.mu.Lock()

		if b.eof {
			return 0, io.EOF
		}
	}

	if len(p) < len(b.buf[0]) {
		return 0, io.ErrShortBuffer
	}
	n := copy(p, b.buf[0])
	b.buf = b.buf[1:]
	return n, nil
}

// write appends a copy of `p` to the internal `buf`. A duplication of `p` is required because lwIP will reuse the
// content of `p` after this function returns. If there are reads waiting, write will send a signal to it.
func (b *packetPipe) write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.eof {
		return 0, net.ErrClosed
	}
	if len(p) == 0 {
		return 0, nil
	}
	if len(p) > b.mtu {
		return 0, io.ErrShortWrite
	}

	if b.wr != nil {
		return b.redirectToWriter(p)
	} else {
		// append to read buffer
		t := make([]byte, len(p))
		n := copy(t, p)
		b.buf = append(b.buf, t)

		// send signal to an awaiting read (unblocking)
		select {
		case b.rd <- n:
		default:
		}
		return n, nil
	}
}

// writeTo redirect all writes to a specific Writer `w`. The redirection logic happens in write function.
// It will block until either EOF or an error occured. All reads will also be blocked until writeTo finishes.
func (b *packetPipe) writeTo(w io.Writer) (int64, error) {
	if w == nil {
		return 0, errors.New("invalid writer w")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.eof {
		// eof should not be treated as error
		return 0, nil
	}
	if b.wr != nil {
		return 0, net.ErrWriteToConnected
	}

	b.wr = w
	b.wrBytes = 0
	b.wrErr = nil
	b.wrDone = make(chan struct{})

	// flush remaining buffer data to w
	for len(b.buf) > 0 {
		_, err := b.redirectToWriter(b.buf[0])
		b.buf = b.buf[1:]
		if err != nil {
			return b.wrBytes, err
		}
	}

	b.mu.Unlock()
	<-b.wrDone
	b.mu.Lock()

	return b.wrBytes, b.wrErr
}

// redirectToWriter write all data in `p` to b.wr (caller needs to make sure b.wr is not nil)
func (b *packetPipe) redirectToWriter(p []byte) (int, error) {
	n, err := b.wr.Write(p)
	b.wrBytes += int64(n)
	if err != nil {
		// terminate the recent WriteTo immediately
		b.wr = nil
		b.wrErr = err
		close(b.wrDone)
	}
	return n, err
}
