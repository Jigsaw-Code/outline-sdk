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

package dnstruncate

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/internal/ddltimer"
	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
)

var (
	errClosed          = net.ErrClosed
	errInvalidResolver = errors.New("invalid DNS resolver address")
	errInvalidDNSMsg   = errors.New("invalid DNS message")
	errTimedOut        = os.ErrDeadlineExceeded
)

// From [RFC 1035], the DNS message header contains the following fields:
//
//		                              1  1  1  1  1  1
//		0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
//
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                      ID                       |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                    QDCOUNT                    |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                    ANCOUNT                    |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                    NSCOUNT                    |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//	 |                    ARCOUNT                    |
//	 +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//
// [RFC 1035]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.1.1
const (
	dnsServerPort   = 53  // https://datatracker.ietf.org/doc/html/rfc1035#section-4.2
	dnsUdpMinMsgLen = 12  // A DNS message must at least containing the header
	dnsUdpMaxMsgLen = 512 // https://datatracker.ietf.org/doc/html/rfc1035#section-2.3.4

	dnsUdpAnswerByte   = 2           // The byte in the header containing QR and TC bit
	dnsUdpResponseBit  = uint8(0x10) // The QR bit within dnsUdpAnswerByte
	dnsUdpTruncatedBit = uint8(0x02) // The TC bit within dnsUdpAnswerByte
	dnsUdpRCodeByte    = 3           // The byte in the header containing RCODE
	dnsUdpRCodeMask    = uint8(0x0f) // The RCODE bits within dnsUdpRCodeByte
	dnsQDCntStartByte  = 4           // The starting byte of QDCOUNT
	dnsQDCntEndByte    = 5           // The ending byte (inclusive) of QDCOUNT
	dnsARCntStartByte  = 6           // The starting byte of ANCOUNT
	dnsARCntEndByte    = 7           // The ending byte (inclusive) of ANCOUNT
)

// dnsTruncateListener is a [transport.PacketListener] that can create dnsTruncateConn to handle DNS requests.
type dnsTruncateListener struct {
}

type dnsReq struct {
	p    []byte          // A buffer from WriteTo containing a DNS request
	addr net.Addr        // DNS resolver's network address
	n    int             // Indicates how many bytes have been processed by ReadFrom
	wg   *sync.WaitGroup // To mark that ReadFrom has finished processing this request
}

// dnsTruncateConn is an implementation of [net.PacketConn] that handles DNS requests in UDP protocol, without sending
// the requests to the actual DNS resolver. It sets the TC (truncated) bit in the DNS response header to tell the
// caller to resend the DNS request over TCP.
//
// dnsTruncateConn uses a single channel architecture to bridge DNS requests from WriteTo and DNS handling logic in
// ReadFrom. There is no need to allocate any additional buffers; and all methods are thread-safe.
type dnsTruncateConn struct {
	done chan struct{}
	req  chan *dnsReq

	rdDdl *ddltimer.DeadlineTimer
	wrDdl *ddltimer.DeadlineTimer
}

// Compilation guard against interface implementation
var _ transport.PacketListener = (*dnsTruncateListener)(nil)
var _ net.PacketConn = (*dnsTruncateConn)(nil)

// NewPacketListener creates a new [transport.PacketListener] that can be used to handle DNS requests if the remote
// proxy doesn't support UDP traffic. It sets the TC (truncated) bit in the DNS response header to tell the caller to
// resend the DNS request over TCP.
//
// This [transport.PacketListener] should only be used if the remote proxy server doesn't support UDP traffic at all.
// Note that all other non-DNS UDP packets will be dropped by this [transport.PacketListener].
func NewPacketListener() (transport.PacketListener, error) {
	return &dnsTruncateListener{}, nil
}

// ListenPacket creates a new [net.PacketConn] to handle UDP traffic.
func (h *dnsTruncateListener) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	conn := &dnsTruncateConn{
		done:  make(chan struct{}),
		req:   make(chan *dnsReq),
		rdDdl: ddltimer.New(),
		wrDdl: ddltimer.New(),
	}
	return conn, nil
}

// Close closes the connection. Any blocked ReadFrom or WriteTo operations will be unblocked.
func (c *dnsTruncateConn) Close() error {
	select {
	case <-c.done:
		// prevent panic of closing a closed channel
	default:
		close(c.done)
	}
	c.rdDdl.Stop()
	c.wrDdl.Stop()
	return nil
}

// ReadFrom reads a DNS request packet from a pending WriteTo, setting the TC (truncated) bit and copying the response
// into p. It returns the number of bytes copied into p and the DNS resolver's address (passed to WriteTo).
//
// ReadFrom returns [io.EOF] if the connection has been closed. It returns [os.ErrDeadlineExceeded] if the read
// deadline is exceeded; see SetDeadline and SetReadDeadline.
//
// ReadFrom will block until one WriteTo sends a DNS request. So you need to use goroutine to send and receive the
// response:
//
//	go c.WriteTo(request, resolverAddr)
//	c.ReadFrom(response)
func (c *dnsTruncateConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if c.alreadyClosed() {
		return 0, nil, io.EOF
	}

	select {
	case req := <-c.req:
		defer req.wg.Done()
		req.n = copy(p, req.p)

		// Set "Response", "Truncated" and "NoError"
		p[dnsUdpAnswerByte] |= (dnsUdpResponseBit | dnsUdpTruncatedBit)
		p[dnsUdpRCodeByte] &= ^dnsUdpRCodeMask

		// Copy QDCOUNT to ANCOUNT. This is an incorrect workaround for some DNS clients (such as Windows 7);
		// because without these clients won't retry over TCP.
		//
		// For reference: https://github.com/eycorsican/go-tun2socks/blob/master/proxy/dnsfallback/udp.go#L59-L63
		copy(p[dnsARCntStartByte:dnsARCntEndByte+1], p[dnsQDCntStartByte:dnsQDCntEndByte+1])

		return req.n, req.addr, nil
	case <-c.rdDdl.Timeout():
		return 0, nil, errTimedOut
	case <-c.done:
		return 0, nil, io.EOF
	}
}

// WriteTo parses a packet from p, and determines whether it is a valid DNS request. If so, it will return the DNS
// response with TC (truncated) bit set when the caller calls ReadFrom. If it is not a valid DNS request, the packet
// will be discarded and returns [ErrNotDNSRequest].
//
// WriteTo returns [os.ErrDeadlineExceeded] if the write deadline is exceeded; see SetDeadline and SetWriteDeadline.
//
// WriteTo will block until one ReadFrom consumes the DNS request. So you need to use goroutine to send and receive
// the response:
//
//	go c.WriteTo(request, resolverAddr)
//	c.ReadFrom(response)
func (c *dnsTruncateConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if c.alreadyClosed() {
		return 0, errClosed
	}
	if udpAddr, ok := addr.(*net.UDPAddr); !ok || udpAddr.Port != dnsServerPort {
		return 0, errInvalidResolver
	}
	if len(p) < dnsUdpMinMsgLen {
		return 0, errInvalidDNSMsg
	}

	req := &dnsReq{
		p:    p,
		addr: addr,
		wg:   &sync.WaitGroup{},
	}
	req.wg.Add(1)
	select {
	case c.req <- req:
		req.wg.Wait()
		return req.n, nil
	case <-c.wrDdl.Timeout():
		return 0, errTimedOut
	case <-c.done:
		return 0, errClosed
	}
}

// LocalAddr always returns nil because we never connect to any servers.
func (c *dnsTruncateConn) LocalAddr() net.Addr {
	return nil
}

// SetDeadline is equivalent to calling both SetReadDeadline and SetWriteDeadline.
// A zero value for t means I/O operations will not time out.
func (c *dnsTruncateConn) SetDeadline(t time.Time) error {
	return errors.Join(c.SetReadDeadline(t), c.SetWriteDeadline(t))
}

// SetReadDeadline sets the deadline for future ReadFrom calls and any currently-blocked ReadFrom call.
// A zero value for t means ReadFrom will not time out.
func (c *dnsTruncateConn) SetReadDeadline(t time.Time) error {
	if c.alreadyClosed() {
		return errClosed
	}
	c.rdDdl.SetDeadline(t)
	return nil
}

// SetWriteDeadline sets the deadline for future WriteTo calls and any currently-blocked WriteTo call.
// A zero value for t means WriteTo will not time out.
func (c *dnsTruncateConn) SetWriteDeadline(t time.Time) error {
	if c.alreadyClosed() {
		return errClosed
	}
	c.wrDdl.SetDeadline(t)
	return nil
}

func (c *dnsTruncateConn) alreadyClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}
