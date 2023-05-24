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

package lwip2transport

import (
	"errors"
	"io"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/network"
	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	lwipLib "github.com/eycorsican/go-tun2socks/core"
)

const packetMTU = 1500

// LwIPTransportDevice is a [network.IPDevice] that can translate IP packets to TCP/UDP traffic and vice versa. It uses
// the [lwIP library] to perform the translation.
//
// LwIPTransportDevice must be a singleton object due to limitations in [lwIP library].
//
// To use a LwIPTransportDevice:
//  1. Call [NewDevice] with two handlers for TCP and UDP traffic.
//  2. [Write] IP packets to the device. The device will translate the IP packets to TCP/UDP traffic and send them to
//     the appropriate handlers.
//  3. [Read] IP packets from the device to get the TCP/UDP responses.
//
// A LwIPTransportDevice is NOT thread-safe. However it is safe to use [Write], [Read]/[WriteTo] and [Close] in
// different goroutines. But keep in mind that only one goroutine can use [Write] at a time; and only one goroutine can
// use the [Read] or [WriteTo] methods at a time.
//
// [lwIP library]: https://savannah.nongnu.org/projects/lwip/
type LwIPTransportDevice interface {
	network.IPDevice
}

// Compilation Guard Against Interface Implementation
var _ network.IPDevice = (*lwIPDevice)(nil)
var _ LwIPTransportDevice = (*lwIPDevice)(nil)

type lwIPDevice struct {
	tcp   *tcpHandler
	udp   *udpHandler
	stack lwipLib.LWIPStack

	// whether the device has been closed
	done chan struct{}

	// async read call and its result
	rdBuf chan []byte
	rdN   chan int
}

// NewDevice creates a new LwIPTransportDevice. This device uses a [transport.StreamDialer] `sd` to handle TCP streams
// and a [transport.PacketListener] `pl` to handle UDP packets.
//
// You can only have one active LwIPTransportDevice per process. If you try to create more than one, the behavior is
// undefined. However, it's OK to [Close] an existing LwIPTransportDevice and then create a new one.
func NewDevice(sd transport.StreamDialer, pl transport.PacketListener) (LwIPTransportDevice, error) {
	if sd == nil || pl == nil {
		return nil, errors.New("both sd and pl are required")
	}

	d := &lwIPDevice{
		tcp:   newTCPHandler(sd),
		udp:   newUDPHandler(pl, 30*time.Second),
		stack: lwipLib.NewLWIPStack(),
		done:  make(chan struct{}),
		rdBuf: make(chan []byte),
		rdN:   make(chan int),
	}
	lwipLib.RegisterTCPConnHandler(d.tcp)
	lwipLib.RegisterUDPConnHandler(d.udp)
	lwipLib.RegisterOutputFn(d.writeResponse)

	return d, nil
}

// Close implements [io.Closer] and [network.IPDevice]. It closes the device, rendering it unusable for I/O.
//
// Close does not close other objects that are passed to this device, such as the [transport.StreamDialer],
// [transport.PacketListener] or [io.Writer]. You are responsible for closing these objects yourself.
func (d *lwIPDevice) Close() error {
	close(d.done)
	return d.stack.Close()
}

// MTU implements [network.IPDevice]. It returns the maximum buffer size of a single IP packet that can be processed by
// this device.
func (d *lwIPDevice) MTU() int {
	return packetMTU
}

// writeResponse writes an incoming IP packet response `b` to this device. The packet can be read by calling the [Read]
// function, or it can be redirected to an [io.Writer] if the [WriteTo] function has been called. writeResponse blocks
// until the packet is successfully consumed by a [Read] or [WriteTo].
//
// writeResponse can be used as an output function for lwIP.
//
// writeResponse might be called by multiple goroutines (for example, when multiple UDP packets arrive at the same
// time). We sequentialize the calls by using channels, if performance issues arise in the future, we can use other
// more performant but more error-prone methods (e.g. the [sync] package) to resolve them.
func (d *lwIPDevice) writeResponse(b []byte) (int, error) {
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
// or an error occurs. This function will not allocate any intermediate buffers.
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
// Write returns [network.ErrMsgSize] if len(b) > [MTU].
func (d *lwIPDevice) Write(b []byte) (int, error) {
	select {
	case <-d.done:
		return 0, network.ErrClosed
	default:
	}
	if len(b) > d.MTU() {
		return 0, network.ErrMsgSize
	}
	n, err := d.stack.Write(b)
	// workaround: lwip netstack did not use a standard error code
	if err != nil && err.Error() == "stack closed" {
		return n, network.ErrClosed
	}
	return n, err
}
