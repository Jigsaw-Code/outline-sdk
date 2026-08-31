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

	// async read call and its result
	rdBuf chan []byte
	rdN   chan int

	// pktPool recycles the packet copies handed to the asyncWriter used by WriteTo. It is per
	// device rather than package scoped, so buffers cannot leak between instances when
	// ConfigureDevice reconfigures the singleton.
	pktPool sync.Pool
}

// Singleton instance
var instMu sync.Mutex
var inst *lwIPDevice = nil

// newLwIPDevice creates a device with the given handlers and registers it with the lwIP stack.
// It must be called with instMu held, since lwIP's handler registration is global.
func newLwIPDevice(tcp *tcpHandler, udp lwip.UDPConnHandler) *lwIPDevice {
	d := &lwIPDevice{
		tcp:   tcp,
		udp:   udp,
		stack: lwip.NewLWIPStack(),
		done:  make(chan struct{}),
		rdBuf: make(chan []byte),
		rdN:   make(chan int),
	}
	d.pktPool.New = func() any { b := make([]byte, 0, packetBufCap); return &b }

	lwip.RegisterTCPConnHandler(d.tcp)
	lwip.RegisterUDPConnHandler(d.udp)
	lwip.RegisterOutputFn(d.forwardOutgoingIPPacket)
	return d
}

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
	inst = newLwIPDevice(newTCPHandler(sd), newUDPHandler(pp))

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
	inst = newLwIPDevice(newTCPHandler(sd), newUDPRelayHandler(pr))

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

// forwardOutgoingIPPacket writes an IP packet response `b` to this device. The packet can be read by calling the Read
// function, or it can be redirected to an [io.Writer] if the WriteTo function has been called. forwardOutgoingIPPacket
// blocks until the packet is successfully consumed by a Read or WriteTo.
//
// forwardOutgoingIPPacket can be used as an output function for lwIP.
//
// `b` aliases the C pbuf payload (see go-tun2socks core/output_export.go), which is freed as soon as this function
// returns. That is why the consumer must copy it out before signalling completion: Read copies into the caller's
// buffer, and WriteTo copies into a pooled buffer owned by its [asyncWriter]. Neither consumer performs a blocking
// device write while lwIP holds its global lock -- doing so would stall every TCP and UDP flow in the stack for the
// duration of the syscall.
//
// forwardOutgoingIPPacket might be called by multiple goroutines (for example, when multiple UDP packets arrive at the
// same time). We sequentialize the calls by using channels, if performance issues arise in the future, we can use
// other more performant but more error-prone methods (e.g. the [sync] package) to resolve them.
func (d *lwIPDevice) forwardOutgoingIPPacket(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	select {
	case d.rdBuf <- b:
		select {
		case n := <-d.rdN:
			return n, nil
		case <-d.done:
			return 0, network.ErrClosed
		}
	case <-d.done:
		return 0, network.ErrClosed
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
		d.rdN <- n
		return n, nil
	case <-d.done:
		return 0, io.EOF
	}
}

// WriteTo implements [io.WriterTo]. It writes all IP packets from TCP/UDP responses to `w` until all data is written
// or an error occurs.
//
// Packets are handed to `w` by a background goroutine, so a slow `w` (such as a TUN device) does not stall the lwIP
// stack. Two consequences of that:
//   - the returned count is the number of bytes queued for `w`, which may be slightly ahead of the bytes `w` has
//     actually accepted;
//   - a write error is reported at least one packet after the packet that caused it. WriteTo still returns as soon as
//     it observes the error, and packets already queued at that point are dropped.
//
// WriteTo returns the total number of bytes written and any error encountered during the write. If there are no more
// data available, WriteTo returns nil error instead of [io.EOF].
func (d *lwIPDevice) WriteTo(w io.Writer) (int64, error) {
	aw := d.newAsyncWriter(w)
	defer aw.Close()

	nw := int64(0)
	for {
		select {
		case s := <-d.rdBuf:
			// Write copies `s` before returning, which is what releases the lwIP pbuf below.
			n, err := aw.Write(s)
			nw += int64(n)
			select {
			case d.rdN <- n:
				if err != nil {
					return nw, err
				}
			case <-d.done:
				return nw, nil
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
