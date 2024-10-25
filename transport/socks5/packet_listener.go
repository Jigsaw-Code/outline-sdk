// Copyright 2024 The Outline Authors
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

package socks5

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/internal/slicepool"
	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// clientUDPBufferSize is the maximum supported UDP packet size in bytes.
const clientUDPBufferSize = 16 * 1024

// udpPool stores the byte slices used for storing packets.
var udpPool = slicepool.MakePool(clientUDPBufferSize)

type packetConn struct {
	pc net.Conn
	sc io.Closer
}

var _ net.PacketConn = (*packetConn)(nil)

func (p *packetConn) LocalAddr() net.Addr {
	return p.pc.LocalAddr()
}

func (p *packetConn) SetDeadline(t time.Time) error {
	return p.pc.SetDeadline(t)
}

func (p *packetConn) SetReadDeadline(t time.Time) error {
	return p.pc.SetReadDeadline(t)
}

func (c *packetConn) SetWriteDeadline(t time.Time) error {
	return c.pc.SetWriteDeadline(t)
}

// ReadFrom reads the packet from the SOCKS5 server and extract the payload
// The packet format is specified in https://datatracker.ietf.org/doc/html/rfc1928#section-7
func (p *packetConn) ReadFrom(b []byte) (int, net.Addr, error) {
	lazySlice := udpPool.LazySlice()
	buffer := lazySlice.Acquire()
	defer lazySlice.Release()

	n, err := p.pc.Read(buffer)
	if err != nil {
		return 0, nil, err
	}
	// Minimum packet size
	if n < 10 {
		return 0, nil, errors.New("invalid SOCKS5 UDP packet: too short")
	}

	// Using bytes.Buffer to handle data
	buf := bytes.NewBuffer(buffer[:n])

	// Read and check reserved bytes
	rsv := make([]byte, 2)
	if _, err := buf.Read(rsv); err != nil {
		return 0, nil, err
	}
	if rsv[0] != 0x00 || rsv[1] != 0x00 {
		return 0, nil, fmt.Errorf("invalid reserved bytes: expected 0x0000, got %#x%#x", rsv[0], rsv[1])
	}

	// Read fragment byte
	frag, err := buf.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	if frag != 0 {
		return 0, nil, errors.New("fragmentation is not supported")
	}

	// Read address using socks.ReadAddr which must now accept a bytes.Buffer directly
	address, err := readAddr(buf)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read address: %w", err)
	}

	// Convert the address to a net.Addr
	addr, err := transport.MakeNetAddr("udp", addrToString(address))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to convert address: %w", err)
	}

	// Payload handling: remaining bytes in the buffer are the payload
	payload := buf.Bytes()
	payloadLength := len(payload)
	if payloadLength > len(b) {
		return 0, nil, io.ErrShortBuffer
	}
	copy(b, payload)

	return payloadLength, addr, nil
}

// WriteTo encapsulates the payload in a SOCKS5 UDP packet as specified in
// https://datatracker.ietf.org/doc/html/rfc1928#section-7
// and write it to the SOCKS5 server via the underlying connection.
func (p *packetConn) WriteTo(b []byte, addr net.Addr) (int, error) {

	// The minimum preallocated header size (10 bytes)
	lazySlice := udpPool.LazySlice()
	buffer := lazySlice.Acquire()
	defer lazySlice.Release()
	buffer = append(buffer[:0],
		0x00, 0x00, // Reserved
		0x00, // Fragment number
		// To be appended below:
		// ATYP, IPv4, IPv6, Domain Name, Port
	)
	buffer, err := appendSOCKS5Address(buffer, addr.String())
	if err != nil {
		return 0, fmt.Errorf("failed to append SOCKS5 address: %w", err)
	}
	// Combine the header and the payload
	return p.pc.Write(append(buffer, b...))
}

// Close closes both the underlying stream and packet connections.
func (p *packetConn) Close() error {
	return errors.Join(p.sc.Close(), p.pc.Close())
}

// ListenPacket creates a [net.PacketConn] for UDP communication via the SOCKS5 server.
func (c *Client) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	// Connect to the SOCKS5 server and perform UDP association
	// Since local address is not known in advance, we use unspecified address
	// which means the server is going to accept incoming packets from any address
	// on the bind port on the server. The bind address is determined and returned by
	// the server.
	// https://datatracker.ietf.org/doc/html/rfc1928#section-6
	// Whoile binding address to specific client address has its advantages, it also creates some
	// challenges such as NAT traveral if client is behind NAT.
	sc, bindAddr, err := c.connectAndRequest(ctx, CmdUDPAssociate, "0.0.0.0:0")
	if err != nil {
		return nil, err
	}

	// If the returned bind IP address is unspecified (i.e. "0.0.0.0" or "::"),
	// then use the IP address of the SOCKS5 server
	if ipAddr := bindAddr.IP; ipAddr.IsValid() && ipAddr.IsUnspecified() {
		schost, _, err := net.SplitHostPort(sc.RemoteAddr().String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse tcp address: %w", err)
		}

		bindAddr.IP, err = netip.ParseAddr(schost)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bind address: %w", err)
		}
	}

	proxyConn, err := c.pd.DialPacket(ctx, addrToString(bindAddr))
	if err != nil {
		sc.Close()
		return nil, fmt.Errorf("could not connect to packet endpoint: %w", err)
	}
	return &packetConn{pc: proxyConn, sc: sc}, nil
}
