// Copyright 2023 The Outline Authors
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
	"errors"
	"io"
	"sync"

	lwip "github.com/eycorsican/go-tun2socks/core"
	"golang.getoutline.org/sdk/network"
	"golang.getoutline.org/sdk/network/packetrelay"
	"golang.getoutline.org/sdk/transport"
)

const packetMTU = 1500

// Compilation guard against interface implementation
var _ network.IPDevice = (*lwIPDevice)(nil)

type lwIPDevice struct {
	tcp   *tcpHandler
	udp   lwip.UDPConnHandler
	stack lwip.LWIPStack

	// whether the device has been closed
	done chan struct{}

	// Outgoing IP packets queued for the TUN writer. Buffered so that lwIP's output
	// callback never blocks: it runs with the global lwIP lock held, so blocking here
	// stalls every TCP and UDP flow for the duration of a TUN write(2).
	rdBuf chan []byte
}

// outQueueDepth bounds the outgoing packet queue. At ~2 KiB per slot this caps
// queue memory at ~512 KiB. Once full, forwardOutgoingIPPacket blocks, which is
// genuine backpressure (the TUN writer is truly slower) rather than the
// unconditional per-packet stall it replaces.
const outQueueDepth = 256

// pktPool recycles the packet copies handed to the TUN writer.
var pktPool = sync.Pool{New: func() any { b := make([]byte, 0, 2048); return &b }}

// Singleton instance
var instMu sync.Mutex
var inst *lwIPDevice = nil

// ConfigureDevice configures the singleton LwIP device using the [transport.StreamDialer] to handle TCP streams and
// the [transport.PacketProxy] to handle UDP packets.
//
// LwIP device is a [network.IPDevice] that can translate IP packets to TCP/UDP traffic and vice versa. It uses the
// [lwIP library] to perform the translation.
//
// LwIP device must be a singleton object due to limitations in [lwIP library]. If you try to call ConfigureDevice more
// than once, we will Close the previous device and reconfigure it.
//
// To use a LwIP device:
//  1. Call [ConfigureDevice] with two handlers for TCP and UDP traffic.
//  2. Write IP packets to the device. The device will translate the IP packets to TCP/UDP traffic and send them to the
//     appropriate handlers.
//  3. Read IP packets from the device to get the TCP/UDP responses.
//
// A LwIP device is NOT thread-safe. However it is safe to use Write, Read/WriteTo and Close in different goroutines.
// Keep in mind that only one goroutine can call Write at a time, and only one goroutine can use either Read or
// WriteTo at a time.
//
// Deprecated: Use [ConfigureDeviceWithRelay] instead.
//
// [lwIP library]: https://savannah.nongnu.org/projects/lwip/
func ConfigureDevice(sd transport.StreamDialer, pp network.PacketProxy) (network.IPDevice, error) {
	if sd == nil || pp == nil {
		return nil, errors.New("both sd and pp are required")
	}

	instMu.Lock()
	defer instMu.Unlock()

	if inst != nil {
		inst.Close()
	}
	inst = &lwIPDevice{
		tcp:   newTCPHandler(sd),
		udp:   newUDPHandler(pp),
		stack: lwip.NewLWIPStack(),
		done:  make(chan struct{}),
		rdBuf: make(chan []byte, outQueueDepth),
	}
	lwip.RegisterTCPConnHandler(inst.tcp)
	lwip.RegisterUDPConnHandler(inst.udp)
	lwip.RegisterOutputFn(inst.forwardOutgoingIPPacket)

	return inst, nil
}

// ConfigureDeviceWithRelay configures the singleton LwIP device using the [transport.StreamDialer] to handle TCP streams and
// the [packetrelay.PacketRelay] to handle UDP packets.
//
// This is the modernized equivalent of [ConfigureDevice], allowing native adoption of the flow-based PacketRelay API.
func ConfigureDeviceWithRelay(sd transport.StreamDialer, pr packetrelay.PacketRelay) (network.IPDevice, error) {
	if sd == nil || pr == nil {
		return nil, errors.New("both sd and pr are required")
	}

	instMu.Lock()
	defer instMu.Unlock()

	if inst != nil {
		inst.Close()
	}
	inst = &lwIPDevice{
		tcp:   newTCPHandler(sd),
		udp:   newUDPRelayHandler(pr),
		stack: lwip.NewLWIPStack(),
		done:  make(chan struct{}),
		rdBuf: make(chan []byte, outQueueDepth),
	}
	lwip.RegisterTCPConnHandler(inst.tcp)
	lwip.RegisterUDPConnHandler(inst.udp)
	lwip.RegisterOutputFn(inst.forwardOutgoingIPPacket)

	return inst, nil
}

// Close implements [io.Closer] and [network.IPDevice]. It closes the device, rendering it unusable for I/O.
//
// Close does not close other objects that are passed to this device, such as the [transport.StreamDialer],
// [transport.PacketListener] or [io.Writer]. You are responsible for closing these objects yourself.
func (d *lwIPDevice) Close() error {
	// make sure we don't close the channel twice
	select {
	case <-d.done:
		return nil
	default:
		close(d.done)
		return d.stack.Close()
	}
}

// MTU implements [network.IPDevice]. It returns the maximum buffer size of a single IP packet that can be processed by
// this device.
func (d *lwIPDevice) MTU() int {
	return packetMTU
}

// forwardOutgoingIPPacket queues an IP packet response `b` for delivery to the TUN device. The
// packet can be read by calling the Read function, or redirected to an [io.Writer] via WriteTo.
//
// forwardOutgoingIPPacket can be used as an output function for lwIP.
//
// It does NOT wait for the packet to reach the TUN. lwIP invokes this callback while holding its
// global lock, so waiting here serializes the entire stack behind one blocking write(2) syscall --
// measured at ~20k packets/sec on Android with the CPU 86% idle. Instead we copy the packet into a
// pooled buffer, hand it to a buffered queue, and return immediately.
//
// The copy is mandatory, not an optimization: `b` aliases the C pbuf payload (see go-tun2socks
// core/output_export.go), which is freed as soon as this function returns.
func (d *lwIPDevice) forwardOutgoingIPPacket(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	bufp := pktPool.Get().(*[]byte)
	*bufp = append((*bufp)[:0], b...)
	select {
	case d.rdBuf <- *bufp:
		return len(b), nil
	case <-d.done:
		pktPool.Put(bufp)
		return 0, network.ErrClosed
	}
}

// recycle returns a packet buffer to the pool after it has been written to the TUN.
func recycle(s []byte) {
	if cap(s) >= 2048 {
		b := s[:0]
		pktPool.Put(&b)
	}
}

// Read implements [io.Reader] and [network.IPDevice]. It reads one IP packet from the TCP/UDP response, blocking until
// a packet arrives or this device is closed. If a packet is too long to fit in the supplied buffer `p`, the excess
// bytes are discarded.
//
// Read returns [io.EOF] error if this device is closed or no more data is available.
func (d *lwIPDevice) Read(p []byte) (int, error) {
	select {
	case s := <-d.rdBuf:
		n := copy(p, s)
		recycle(s)
		return n, nil
	case <-d.done:
		return 0, io.EOF
	}
}

// WriteTo implements [io.WriterTo]. It writes all IP packets from TCP/UDP responses to `w` until all data is written
// or an error occurs. Packets come from a pooled buffer filled by forwardOutgoingIPPacket.
//
// WriteTo returns the total number of bytes written and any error encountered during the write. If there are no more
// data available, WriteTo returns nil error instead of [io.EOF].
func (d *lwIPDevice) WriteTo(w io.Writer) (int64, error) {
	nw := int64(0)
	for {
		select {
		case s := <-d.rdBuf:
			n, err := w.Write(s)
			nw += int64(n)
			recycle(s)
			if err != nil {
				return nw, err
			}
		case <-d.done:
			return nw, nil
		}
	}
}

// Write implements [io.Writer] and [network.IPDevice]. It writes a single IP packet to this device. The device will
// then translate the IP packet into a TCP or UDP traffic.
//
// Write returns [network.ErrClosed] if this device is already closed.
func (d *lwIPDevice) Write(b []byte) (int, error) {
	select {
	case <-d.done:
		return 0, network.ErrClosed
	default:
	}
	n, err := d.stack.Write(b)
	// Workaround: lwip netstack did not use a typed error.
	if err != nil && err.Error() == "stack closed" {
		return n, network.ErrClosed
	}
	return n, err
}
