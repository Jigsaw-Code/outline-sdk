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

package dnsovertcp

import (
	"container/list"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/internal/ddltimer"
	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
)

var (
	errClosed          = net.ErrClosed
	errInvalidResolver = errors.New("invalid DNS resolver address")
	errInvalidDNSMsg   = errors.New("invalid DNS message")
	errShortBuffer     = io.ErrShortBuffer
	errTimedOut        = os.ErrDeadlineExceeded
)

// From [RFC 1035], the DNS message header contains the following fields:
//
//  	                              1  1  1  1  1  1
//  	0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
//
//   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//   |                      ID                       |
//   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//   |QR|   Opcode  |AA|TC|RD|RA|   Z    |   RCODE   |
//   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//   |                    QDCOUNT                    |
//   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//   |                    ANCOUNT                    |
//   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//   |                    NSCOUNT                    |
//   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//   |                    ARCOUNT                    |
//   +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
//
// Also from [RFC 1035 TCP usage]:
//
//   Messages sent over TCP connections use server port 53 (decimal). The message is prefixed
//   with a two byte length field which gives the message length, excluding the two byte
//   length field. This length field allows the low-level processing to assemble a complete
//   message before beginning to parse it.
//
// [RFC 1035]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.1.1
// [RFC 1035 TCP usage]: https://datatracker.ietf.org/doc/html/rfc1035#section-4.2.2

const (
	dnsServerPort       = 53
	dnsTcpSizeBufferLen = 2           // Buffer length (2 bytes) of the prefixed size in TCP
	dnsUdpMinMsgLen     = 12          // A DNS message must at least containing the header
	dnsUdpMaxMsgLen     = 512         // https://datatracker.ietf.org/doc/html/rfc1035#section-2.3.4
	dnsUdpTruncatedByte = 2           // The byte in the header containing TC bit
	dnsUdpTruncatedBit  = uint8(0x02) // The TC bit within dnsTruncatedByte
)

type dnsOverTcpHandler struct {
	sd transport.StreamDialer
}

type dnsResult struct {
	p    []byte
	n    int
	addr net.Addr
	err  error
	wg   *sync.WaitGroup
}

type dnsOverTcpConn struct {
	mu   sync.Mutex
	done chan struct{}

	sd    transport.StreamDialer
	conns *list.List
	wrDdl atomic.Pointer[time.Time]

	dnsRecv chan *dnsResult
	recvDdl *ddltimer.DeadlineTimer
}

// Compilation guard against interface implementation
var _ transport.PacketListener = (*dnsOverTcpHandler)(nil)
var _ net.PacketConn = (*dnsOverTcpConn)(nil)

func NewPacketListener(sd transport.StreamDialer) (transport.PacketListener, error) {
	return &dnsOverTcpHandler{
		sd: sd,
	}, nil
}

func (h *dnsOverTcpHandler) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	conn := &dnsOverTcpConn{
		done:    make(chan struct{}),
		sd:      h.sd,
		conns:   list.New(),
		dnsRecv: make(chan *dnsResult),
		recvDdl: ddltimer.New(),
	}
	conn.wrDdl.Store(&time.Time{})
	return conn, nil
}

func (c *dnsOverTcpConn) Close() (err error) {
	// Lock() for safely reading the `c.conns` list
	c.mu.Lock()
	defer c.mu.Unlock()

	err = c.forEachConnIgnoreErrClosed(func(conn net.Conn) error {
		return conn.Close()
	})

	c.recvDdl.Stop()

	select {
	case <-c.done:
		// prevent panic of closing a closed channel
	default:
		close(c.done)
	}
	return
}

func (c *dnsOverTcpConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if c.alreadyClosed() {
		return 0, nil, io.EOF
	}
	if len(p) == 0 {
		return 0, nil, errShortBuffer
	}

	recv := &dnsResult{
		p:  p,
		wg: &sync.WaitGroup{},
	}
	recv.wg.Add(1)

	// Wait until either one of the following:
	//   1. one of the `listenDNSResponseOverTCP` goroutine responded through recv.wg.Done()
	//   2. c is closed
	//   3. read timed out (`listenDNSResponseOverTCP` won't respond if conn.Read() timed out)
	select {
	case c.dnsRecv <- recv:
		recv.wg.Wait()
	case <-c.done:
		return 0, nil, io.EOF
	case <-c.recvDdl.Timeout():
		return 0, nil, errTimedOut
	}

	n, addr, err = recv.n, recv.addr, recv.err
	if errors.Is(err, net.ErrClosed) {
		err = io.EOF
	}
	return
}

func (c *dnsOverTcpConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if c.alreadyClosed() {
		return 0, errClosed
	}
	if udpAddr, ok := addr.(*net.UDPAddr); !ok || udpAddr.Port != dnsServerPort {
		return 0, errInvalidResolver
	}
	if len(p) < dnsUdpMinMsgLen {
		return 0, errInvalidDNSMsg
	}

	conn, n, err := c.sendDNSRequestOverTCP(p, addr)
	if err != nil {
		return
	}

	go c.listenDNSResponseOverTCP(conn, addr)
	return
}

func (c *dnsOverTcpConn) LocalAddr() net.Addr {
	// Lock() for safely reading the `c.conns` list
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conns.Len() > 0 {
		return c.conns.Front().Value.(net.Conn).LocalAddr()
	}
	return nil
}

// SetDeadline is equivalent to calling both SetReadDeadline and SetWriteDeadline.
// A zero value for t means I/O operations will not time out.
func (c *dnsOverTcpConn) SetDeadline(t time.Time) error {
	return errors.Join(c.SetReadDeadline(t), c.SetWriteDeadline(t))
}

// SetReadDeadline sets the deadline for future ReadFrom calls and any currently-blocked ReadFrom call.
// A zero value for t means ReadFrom will not time out.
func (c *dnsOverTcpConn) SetReadDeadline(t time.Time) error {
	// Lock() for reading the `c.conns` list and updating recvDdl
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.alreadyClosed() {
		return errClosed
	}

	c.recvDdl.SetDeadline(t)
	return c.forEachConnIgnoreErrClosed(func(conn net.Conn) error {
		return conn.SetReadDeadline(t)
	})
}

// SetWriteDeadline sets the deadline for future WriteTo calls and any currently-blocked WriteTo call.
// A zero value for t means WriteTo will not time out.
func (c *dnsOverTcpConn) SetWriteDeadline(t time.Time) error {
	// Lock() for reading the `c.conns` list and updating wrDdl
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.alreadyClosed() {
		return errClosed
	}

	c.wrDdl.Store(&t)
	return c.forEachConnIgnoreErrClosed(func(conn net.Conn) error {
		return conn.SetWriteDeadline(t)
	})
}

func (c *dnsOverTcpConn) alreadyClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

func (c *dnsOverTcpConn) forEachConnIgnoreErrClosed(w func(net.Conn) error) error {
	errs := make([]error, 0, c.conns.Len())
	for e := c.conns.Front(); e != nil; e = e.Next() {
		err := w(e.Value.(net.Conn))
		if errors.Is(err, net.ErrClosed) {
			err = nil
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (c *dnsOverTcpConn) sendDNSRequestOverTCP(req []byte, resolver net.Addr) (e *list.Element, n int, err error) {
	ctx, cancel := context.WithDeadline(context.Background(), *c.wrDdl.Load())
	defer cancel()

	// Create TCP conn and add it to c.conns
	conn, err := c.sd.Dial(ctx, resolver.String())
	if err != nil {
		return
	}
	c.mu.Lock()
	e = c.conns.PushFront(conn)
	c.mu.Unlock()

	// If there are errors, remove conn from c.conns
	defer func() {
		if e != nil && err != nil {
			conn.Close()
			c.mu.Lock()
			c.conns.Remove(e)
			c.mu.Unlock()
			e = nil
		}
	}()

	if err = conn.SetReadDeadline(c.recvDdl.Deadline()); err != nil {
		return
	}
	if err = conn.SetWriteDeadline(*c.wrDdl.Load()); err != nil {
		return
	}

	// DoT: message length prefix
	var lenpfx [2]byte // make sure buffer is stack allocated
	binary.BigEndian.PutUint16(lenpfx[:], (uint16)(len(req)))
	if _, err = conn.Write(lenpfx[:]); err != nil {
		return
	}

	// DoT: actual DNS request message
	n, err = conn.Write(req)
	return
}

func (c *dnsOverTcpConn) listenDNSResponseOverTCP(e *list.Element, resolver net.Addr) {
	conn := e.Value.(net.Conn)

	defer func() {
		conn.Close()
		c.mu.Lock()
		c.conns.Remove(e)
		c.mu.Unlock()
	}()

	// stack allocated buffer for reading both the DoT length prefix and the DNS response
	var buf [dnsUdpMaxMsgLen + dnsTcpSizeBufferLen]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		return
	}

	// DoT: message length prefix
	lenpfx := binary.BigEndian.Uint16(buf[:dnsTcpSizeBufferLen])

	// wait for DNS result request or Close
	select {
	case dns := <-c.dnsRecv:
		dns.addr = resolver
		dns.n = copy(dns.p, buf[dnsTcpSizeBufferLen:n])
		dns.err = err

		// Try to set the DNS TC (Truncated) bit
		if dns.n < int(lenpfx) && dns.n > dnsUdpTruncatedByte {
			dns.p[dnsUdpTruncatedByte] |= dnsUdpTruncatedBit
		}

		dns.wg.Done()
	case <-c.done:
	}
}
