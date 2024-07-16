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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
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

var _ net.Conn = (*packetConn)(nil)

func (c *packetConn) LocalAddr() net.Addr {
	return c.pc.LocalAddr()
}

func (c *packetConn) RemoteAddr() net.Addr {
	return c.dstAddr
}

func (c *packetConn) SetDeadline(t time.Time) error {
	return c.pc.SetDeadline(t)
}

func (c *packetConn) SetReadDeadline(t time.Time) error {
	return c.pc.SetReadDeadline(t)
}

func (c *packetConn) SetWriteDeadline(t time.Time) error {
	return c.pc.SetWriteDeadline(t)
}

func (c *packetConn) Read(b []byte) (int, error) {
	lazySlice := udpPool.LazySlice()
	buffer := lazySlice.Acquire()
	defer lazySlice.Release()
	n, err := c.pc.Read(buffer)
	if err != nil {
		return 0, err
	}
	// Minimum size of header is 10 bytes
	// 2 bytes for reserved, 1 byte for fragment, 1 byte for address type, 4 byte for ipv4, 2 bytes for port
	if n < 10 {
		return 0, fmt.Errorf("invalid SOCKS5 UDP packet: too short")
	}

	pkt := buffer[:n]

	// Start parsing the header
	rsv := pkt[:2]
	if rsv[0] != 0x00 || rsv[1] != 0x00 {
		return 0, fmt.Errorf("invalid reserved bytes: expected 0x0000, got %#x%#x", rsv[0], rsv[1])
	}

	frag := pkt[2]
	if frag != 0 {
		return 0, errors.New("fragmentation is not supported")
	}

	// Do something with address?
	_, addrLen, err := readAddress(bytes.NewReader(pkt[3:]))
	if err != nil {
		return 0, fmt.Errorf("failed to read address: %w", err)
	}

	// Calculate the start position of the actual data
	headerLength := 4 + addrLen + 2 // RSV (2) + FRAG (1) + ATYP (1) + ADDR (variable) + PORT (2)
	if n < headerLength {
		return 0, fmt.Errorf("invalid SOCKS5 UDP packet: header too short")
	}

	// Copy the payload into the provided buffer
	payloadLength := n - headerLength
	if payloadLength > len(b) {
		return 0, io.ErrShortBuffer
	}
	copy(b, buffer[headerLength:n])

	return payloadLength, nil
}

func (c *packetConn) Write(b []byte) (int, error) {
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
	header, err := appendSOCKS5Address(header, c.dstAddr.String())
	if err != nil {
		return 0, fmt.Errorf("failed to append SOCKS5 address: %w", err)
	}
	// Combine the header and the payload
	fullPacket := append(header, b...)
	return c.pc.Write(fullPacket)
}

func (c *packetConn) Close() error {
	return errors.Join(c.sc.Close(), c.pc.Close())
}

// DialPacket creates a packet [net.Conn] via SOCKS5.
func (d *Dialer) DialPacket(ctx context.Context, dstAddr string) (net.Conn, error) {
	netDstAddr, err := transport.MakeNetAddr("udp", dstAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %w", err)
	}
	sc, bindAddr, err := d.request(ctx, CmdUDPAssociate, "0.0.0.0:0")
	//fmt.Println("Bound address is:", bindAddr)
	if err != nil {
		return nil, err
	}

	// Wait for the bind to be ready
	// time.Sleep(10 * time.Millisecond)

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

	pc, err := d.pd.DialPacket(ctx, net.JoinHostPort(host, port))
	if err != nil {
		sc.Close()
		return nil, fmt.Errorf("failed to connect to packet endpoint: %w", err)
	}

	return &packetConn{netDstAddr, pc, sc}, nil
}

func readAddress(reader io.Reader) (string, int, error) {
	// Read the address type
	// The maximum buffer size is:
	// 1 address type + 1 address length + 256 (max domain name length)
	var buffer [1 + 1 + 256]byte
	addrLen := 0
	_, err := io.ReadFull(reader, buffer[:1])
	if err != nil {
		return "", 0, fmt.Errorf("failed to read address type: %w", err)
	}
	// Read the address type
	switch buffer[0] {
	case addrTypeIPv4:
		addrLen = 4
	case addrTypeIPv6:
		addrLen = 16
	case addrTypeDomainName:
		// Domain name's first byte is the length of the name
		// Read domainAddrLen
		_, err := io.ReadFull(reader, buffer[1:])
		if err != nil {
			return "", 0, fmt.Errorf("failed to read domain address length: %w", err)
		}
		addrLen = int(buffer[1])
	default:
		return "", 0, fmt.Errorf("unknown address type %#x", buffer[0])
	}
	// Read host address
	_, err = io.ReadFull(reader, buffer[:addrLen])
	if err != nil {
		return "", 0, fmt.Errorf("failed to read address: %w", err)
	}
	host := net.IP(buffer[:addrLen]).String()
	// Read port number
	_, err = io.ReadFull(reader, buffer[:2])
	if err != nil {
		return "", 0, fmt.Errorf("failed to read port: %w", err)
	}
	p := binary.BigEndian.Uint16(buffer[:2])
	portStr := strconv.FormatUint(uint64(p), 10)
	addr := net.JoinHostPort(host, portStr)
	return addr, addrLen, nil
}
