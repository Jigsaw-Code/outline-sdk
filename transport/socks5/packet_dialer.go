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
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

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
	buffer := make([]byte, 65536) // Maximum size for UDP packet
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

	atyp := pkt[3]
	addrLen := 0
	switch atyp {
	case addrTypeIPv4:
		addrLen = 4
	case addrTypeIPv6:
		addrLen = 16
	case addrTypeDomainName:
		// Domain name's first byte is the length of the name
		addrLen = int(pkt[4]) + 1 // +1 for the length byte itself
	default:
		return 0, fmt.Errorf("unknown address type %#x", atyp)
	}

	pkt = pkt[4:] // Skip the header
	addr := pkt[:addrLen]

	pkt = pkt[addrLen:] // Skip the address
	port := binary.BigEndian.Uint16(pkt[:2])
	fmt.Printf("Received packet from %d:%d\n", addr, port)

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
	// Encapsulate the payload in a SOCKS5 UDP packet
	// this is the minimum preallocated header size (10 bytes)
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
	//time.Sleep(1 * time.Millisecond)

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

// func readAddress(conn *net.Conn) (string, error) {
// 	// Read the address type
// 	fmt.Println("Reading address type")
// 	addrType := make([]byte, 1)
// 	addrLen := 0
// 	n, err := io.ReadFull(*conn, addrType)
// 	//n, err := *conn.Read(addrType)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read address type: %w", err)
// 	}
// 	fmt.Printf("Read address type %d bytes\n", n)
// 	// Read the address type
// 	switch addrType[0] {
// 	case addrTypeIPv4:
// 		addrLen = 4
// 	case addrTypeIPv6:
// 		addrLen = 16
// 	case addrTypeDomainName:
// 		// Domain name's first byte is the length of the name
// 		domainAddrLen := make([]byte, 1)
// 		_, err := io.ReadFull(*conn, domainAddrLen)
// 		//_, err := reader.Read(domainAddrLen)
// 		if err != nil {
// 			return "", fmt.Errorf("failed to read domain address length: %w", err)
// 		}
// 		addrLen = int(domainAddrLen[0])
// 	default:
// 		return "", fmt.Errorf("unknown address type %#x", addrType[0])
// 	}
// 	fmt.Printf("Address length is: %d\n", addrLen)
// 	addr := make([]byte, addrLen)
// 	_, err = io.ReadFull(*conn, addr)
// 	//_, err = reader.Read(addr)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read address: %w", err)
// 	}
// 	return string(addr), nil
// }
