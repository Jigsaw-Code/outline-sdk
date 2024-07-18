// Copyright 2024 Jigsaw Operations LLC
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
	"time"

	"github.com/Jigsaw-Code/outline-sdk/internal/slicepool"
	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// clientUDPBufferSize is the maximum supported UDP packet size in bytes.
const clientUDPBufferSize = 16 * 1024

// udpPool stores the byte slices used for storing packets.
var udpPool = slicepool.MakePool(clientUDPBufferSize)

type packetConn struct {
	dstAddr net.Addr
	pc      net.Conn
	sc      transport.StreamConn
}

var _ net.PacketConn = (*packetConn)(nil)

func (p *packetConn) LocalAddr() net.Addr {
	return p.pc.LocalAddr()
}

func (p *packetConn) RemoteAddr() net.Addr {
	return p.dstAddr
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

func (p *packetConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	lazySlice := udpPool.LazySlice()
	buffer := lazySlice.Acquire()
	defer lazySlice.Release()
	n, err = p.pc.Read(buffer)
	if err != nil {
		return 0, nil, err
	}
	// Minimum size of header is 10 bytes
	// 2 bytes for reserved, 1 byte for fragment, 1 byte for address type, 4 byte for ipv4, 2 bytes for port
	if n < 10 {
		return 0, nil, fmt.Errorf("invalid SOCKS5 UDP packet: too short")
	}

	pkt := buffer[:n]

	// Start parsing the header
	rsv := pkt[:2]
	if rsv[0] != 0x00 || rsv[1] != 0x00 {
		return 0, nil, fmt.Errorf("invalid reserved bytes: expected 0x0000, got %#x%#x", rsv[0], rsv[1])
	}

	frag := pkt[2]
	if frag != 0 {
		return 0, nil, errors.New("fragmentation is not supported")
	}

	// Do something with address?
	address, addrLen, err := readAddress(bytes.NewReader(pkt[3:]))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read address: %w", err)
	}

	// Calculate the start position of the actual data
	headerLength := 4 + addrLen + 2 // RSV (2) + FRAG (1) + ATYP (1) + ADDR (variable) + PORT (2)
	if n < headerLength {
		return 0, nil, fmt.Errorf("invalid SOCKS5 UDP packet: header too short")
	}

	// Copy the payload into the provided buffer
	payloadLength := n - headerLength
	if payloadLength > len(b) {
		return 0, nil, io.ErrShortBuffer
	}
	copy(b, buffer[headerLength:n])

	addr, err = net.ResolveUDPAddr("udp", address)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to resolve address: %w", err)
	}
	return payloadLength, addr, nil
}

func (p *packetConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	// Encapsulate the payload in a SOCKS5 UDP packet as specified in
	// https://datatracker.ietf.org/doc/html/rfc1928#section-7
	// The minimum preallocated header size (10 bytes)
	header := make([]byte, 10)
	header = append(header[:0],
		0x00, 0x00, // Reserved
		0x00, // Fragment number
		// To be appended below:
		// ATYP, IPv4, IPv6, Domain Name, Port
	)
	header, err := appendSOCKS5Address(header, addr.String())
	if err != nil {
		return 0, fmt.Errorf("failed to append SOCKS5 address: %w", err)
	}
	// Combine the header and the payload
	fullPacket := append(header, b...)
	return p.pc.Write(fullPacket)
}

func (p *packetConn) Close() error {
	return errors.Join(p.sc.Close(), p.pc.Close())
}

// DialPacket creates a packet [net.Conn] via SOCKS5.
func (c *Client) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	// Connect to the SOCKS5 server and perform UDP association
	sc, bindAddr, err := c.request(ctx, CmdUDPAssociate, "0.0.0.0:0")
	if err != nil {
		return nil, err
	}

	host, port, err := net.SplitHostPort(bindAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bound address: %w", err)
	}

	if ipAddr := net.ParseIP(host); ipAddr != nil && ipAddr.IsUnspecified() {
		schost, _, err := net.SplitHostPort(sc.RemoteAddr().String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse tcp address: %w", err)
		}
		host = schost
	}

	packetEndpoint := &transport.PacketDialerEndpoint{Dialer: c.pd, Address: net.JoinHostPort(host, port)}
	err = c.EnablePacketListener(packetEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to enable packet listener: %w", err)
	}

	proxyConn, err := c.pe.ConnectPacket(ctx)
	if err != nil {
		sc.Close()
		return nil, fmt.Errorf("could not connect to packet endpoint: %w", err)
	}
	return &packetConn{pc: proxyConn, sc: sc, dstAddr: proxyConn.RemoteAddr()}, nil
}

func (c *Client) EnablePacketListener(endpoint transport.PacketEndpoint) error {
	if endpoint == nil {
		return errors.New("argument endpoint must not be nil")
	}
	c.pe = endpoint
	return nil
}
