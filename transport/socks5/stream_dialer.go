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

package socks5

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// https://datatracker.ietf.org/doc/html/rfc1929
// Credentials can be nil, and that means no authentication.
type credentials struct {
	username []byte
	password []byte
}

// NewStreamDialer creates a [transport.StreamDialer] that routes connections to a SOCKS5
// proxy listening at the given [transport.StreamEndpoint].
func NewStreamDialer(endpoint transport.StreamEndpoint) (*StreamDialer, error) {
	if endpoint == nil {
		return nil, errors.New("argument endpoint must not be nil")
	}
	return &StreamDialer{proxyEndpoint: endpoint, cred: nil}, nil
}

type StreamDialer struct {
	proxyEndpoint transport.StreamEndpoint
	cred          *credentials
	// TODO: check flag is dialer is meant for TCP transport or UDP associatation only
	// udpAssociateEndpoint transport.UDPEndpoint
	udpAssociate bool
}

var _ transport.StreamDialer = (*StreamDialer)(nil)

func (c *StreamDialer) SetCredentials(username, password []byte) error {
	if len(username) > 255 {
		return errors.New("username exceeds 255 bytes")
	}
	if len(username) == 0 {
		return errors.New("username must be at least 1 byte")
	}

	if len(password) > 255 {
		return errors.New("password exceeds 255 bytes")
	}
	if len(password) == 0 {
		return errors.New("password must be at least 1 byte")
	}

	c.cred = &credentials{username: username, password: password}
	return nil
}

// DialStream implements [transport.StreamDialer].DialStream using SOCKS5.
// It will send the auth method, auth credentials (if auth is chosen), and
// the connect requests in one packet, to avoid an additional roundtrip.
// The returned [error] will be of type [ReplyCode] if the server sends a SOCKS error reply code, which
// you can check against the error constants in this package using [errors.Is].
func (c *StreamDialer) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	proxyConn, err := c.proxyEndpoint.ConnectStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not connect to SOCKS5 proxy: %w", err)
	}
	dialSuccess := false
	defer func() {
		if !dialSuccess {
			proxyConn.Close()
		}
	}()
	// For protocol details, see https://datatracker.ietf.org/doc/html/rfc1928#section-3
	// Creating a single buffer for method selection, authentication, and connection request
	// Buffer large enough for method, auth, and connect requests with a domain name address.
	// The maximum buffer size is:
	// 3 (1 socks version + 1 method selection + 1 methods)
	// + 1 (auth version) + 1 (username length) + 255 (username) + 1 (password length) + 255 (password)
	// + 256 (max domain name length)
	var buffer [(1 + 1 + 1) + (1 + 1 + 255 + 1 + 255) + 256]byte

	// Method selection and authentication request.
	b := makeMethodSelectionAndAuthRequest(c.cred, buffer[:0])

	if c.udpAssociate {
		// Append UDP associate request
		err = appendUDPAssociateRequest(&b, remoteAddr)
		if err != nil {
			return nil, err
		}
		fmt.Print("UDP ASSOCIATE\n")
	} else {
		// Append Connect request
		fmt.Print("CONNECT\n")
		err = appendConnectRequest(&b, remoteAddr)
		if err != nil {
			return nil, err
		}
	}

	// We merge the method and connect requests and only perform one write
	// because we send a single authentication method, so there's no point
	// in waiting for the response. This eliminates a roundtrip.
	fmt.Printf("Combined SOCKS5 request: %v\n", b)
	_, err = proxyConn.Write(b)
	if err != nil {
		return nil, fmt.Errorf("failed to write combined SOCKS5 request: %w", err)
	}

	// Reading the response:
	// 1. Read method response (VER, METHOD).
	// +----+--------+
	// |VER | METHOD |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	// buffer[0]: VER, buffer[1]: METHOD
	// Reuse buffer for better performance.
	if _, err = io.ReadFull(proxyConn, buffer[:2]); err != nil {
		return nil, fmt.Errorf("failed to read method server response: %w", err)
	}
	if buffer[0] != 5 {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", buffer[0])
	}

	switch buffer[1] {
	case authMethodNoAuth:
		// No authentication required.
	case authMethodUserPass:
		// 2. Read authentication version and status
		// VER = 1, STATUS = 0
		// +----+--------+
		// |VER | STATUS |
		// +----+--------+
		// | 1  |   1    |
		// +----+--------+
		// VER = 1 means the server should be expecting username/password authentication.
		// buffer[2]: VER, buffer[3]: STATUS
		if _, err = io.ReadFull(proxyConn, buffer[2:4]); err != nil {
			return nil, fmt.Errorf("failed to read authentication version and status: %w", err)
		}
		if buffer[2] != 1 {
			return nil, fmt.Errorf("invalid authentication version %v. Expected 1", buffer[2])
		}
		if buffer[3] != 0 {
			return nil, fmt.Errorf("authentication failed: %v", buffer[3])
		}
	default:
		return nil, fmt.Errorf("unsupported SOCKS authentication method %v. Expected 2", buffer[1])
	}

	// 3. Read connect or associate response (VER, REP, RSV, ATYP, BND.ADDR, BND.PORT).
	// See https://datatracker.ietf.org/doc/html/rfc1928#section-6.
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	// buffer[0]: VER
	// buffer[1]: REP
	// buffer[2]: RSV
	// buffer[3]: ATYP
	if _, err = io.ReadFull(proxyConn, buffer[:4]); err != nil {
		return nil, fmt.Errorf("failed to read connect server response: %w", err)
	}

	if buffer[0] != 5 {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", buffer[0])
	}

	// if REP is not 0, it means the server returned an error.
	if buffer[1] != 0 {
		return nil, ReplyCode(buffer[1])
	}

	// 4. Read address and length
	var bndAddrLen int
	switch buffer[3] {
	case addrTypeIPv4:
		bndAddrLen = 4
	case addrTypeIPv6:
		bndAddrLen = 16
	case addrTypeDomainName:
		// buffer[8]: length of the domain name
		_, err := io.ReadFull(proxyConn, buffer[:1])
		if err != nil {
			return nil, fmt.Errorf("failed to read address length in connect response: %w", err)
		}
		bndAddrLen = int(buffer[0])
	default:
		return nil, fmt.Errorf("invalid address type %v", buffer[3])
	}
	fmt.Printf("Address name length: %v\n", bndAddrLen)
	// 5. Reads the bound address and port, but we currently ignore them.
	// TODO(fortuna): Should we expose the remote bound address as the net.Conn.LocalAddr()?
	if _, err := io.ReadFull(proxyConn, buffer[:bndAddrLen]); err != nil {
		return nil, fmt.Errorf("failed to read bound address: %w", err)
	}
	// print hex address
	fmt.Printf("Bound HEX Address: %x\n", buffer[:bndAddrLen])
	ipAddress := net.IP(buffer[:bndAddrLen]).String()
	fmt.Printf("Bound IP address: %v\n", ipAddress)
	// We read but ignore the remote bound port number: BND.PORT
	if _, err = io.ReadFull(proxyConn, buffer[:2]); err != nil {
		return nil, fmt.Errorf("failed to read bound port: %w", err)
	}
	// Convert the two bytes to an integer using BigEndian
	port := binary.BigEndian.Uint16(buffer[:2])
	// Convert the port number to a string
	portStr := strconv.Itoa(int(port))
	// Print the result
	fmt.Println("Port number is:", portStr)
	dialSuccess = true
	if c.udpAssociate {
		//proxyEndpoint := transport.UDPEndpoint{Address: net.JoinHostPort(ipAddress.String(), port.String())}
		// return proxyEndpoint.Address, nil
		fmt.Printf("Bound Address: %v:%v\n", ipAddress, portStr)
		return nil, nil
	} else {
		return proxyConn, nil
	}
}

// type packetListener struct {
// 	endpoint transport.PacketEndpoint
// }

// var _ transport.PacketListener = (*packetListener)(nil)

// type packetConn struct {
// 	net.Conn
// }

// func (c *packetListener) ListenPacket(ctx context.Context) (net.PacketConn, error) {
// 	proxyConn, err := c.endpoint.ConnectPacket(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("could not connect to endpoint: %w", err)
// 	}
// 	conn := packetConn{Conn: proxyConn}
// 	return &conn, nil
// }

// func (c *StreamDialer) UDPAssociate(ctx context.Context, remoteAddr string) (string, error) {
// 	proxyConn, err := c.proxyEndpoint.ConnectStream(ctx)
// 	if err != nil {
// 		return "", fmt.Errorf("could not connect to SOCKS5 proxy: %w", err)
// 	}
// 	dialSuccess := false
// 	defer func() {
// 		if !dialSuccess {
// 			proxyConn.Close()
// 		}
// 	}()
// 	// For protocol details, see https://datatracker.ietf.org/doc/html/rfc1928#section-3
// 	// Creating a single buffer for method selection, authentication, and connection request
// 	// Buffer large enough for method, auth, and connect requests with a domain name address.
// 	// The maximum buffer size is:
// 	// 3 (1 socks version + 1 method selection + 1 methods)
// 	// + 1 (auth version) + 1 (username length) + 255 (username) + 1 (password length) + 255 (password)
// 	// + 256 (max domain name length)
// 	var buffer [(1 + 1 + 1) + (1 + 1 + 255 + 1 + 255) + 256]byte

// 	// Method selection and authentication request.
// 	b := makeMethodSelectionAndAuthRequest(c.cred, buffer[:0])

// 	// Append Connect request
// 	err = appendUDPAssociateRequest(&b, remoteAddr)
// 	if err != nil {
// 		return "", err
// 	}

// 	// We merge the method and connect requests and only perform one write
// 	// because we send a single authentication method, so there's no point
// 	// in waiting for the response. This eliminates a roundtrip.
// 	_, err = proxyConn.Write(b)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to write combined SOCKS5 request: %w", err)
// 	}

// 	// Reading the response:
// 	// 1. Read method response (VER, METHOD).
// 	// +----+--------+
// 	// |VER | METHOD |
// 	// +----+--------+
// 	// | 1  |   1    |
// 	// +----+--------+
// 	// buffer[0]: VER, buffer[1]: METHOD
// 	// Reuse buffer for better performance.
// 	if _, err = io.ReadFull(proxyConn, buffer[:2]); err != nil {
// 		return "", fmt.Errorf("failed to read method server response: %w", err)
// 	}
// 	if buffer[0] != 5 {
// 		return "", fmt.Errorf("invalid protocol version %v. Expected 5", buffer[0])
// 	}

// 	switch buffer[1] {
// 	case authMethodNoAuth:
// 		// No authentication required.
// 	case authMethodUserPass:
// 		// 2. Read authentication version and status
// 		// VER = 1, STATUS = 0
// 		// +----+--------+
// 		// |VER | STATUS |
// 		// +----+--------+
// 		// | 1  |   1    |
// 		// +----+--------+
// 		// VER = 1 means the server should be expecting username/password authentication.
// 		// buffer[2]: VER, buffer[3]: STATUS
// 		if _, err = io.ReadFull(proxyConn, buffer[2:4]); err != nil {
// 			return "", fmt.Errorf("failed to read authentication version and status: %w", err)
// 		}
// 		if buffer[2] != 1 {
// 			return "", fmt.Errorf("invalid authentication version %v. Expected 1", buffer[2])
// 		}
// 		if buffer[3] != 0 {
// 			return "", fmt.Errorf("authentication failed: %v", buffer[3])
// 		}
// 	default:
// 		return "", fmt.Errorf("unsupported SOCKS authentication method %v. Expected 2", buffer[1])
// 	}

// 	// 3. Read associate response (VER, REP, RSV, ATYP, BND.ADDR, BND.PORT).
// 	// See https://datatracker.ietf.org/doc/html/rfc1928#section-6.
// 	// +----+-----+-------+------+----------+----------+
// 	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
// 	// +----+-----+-------+------+----------+----------+
// 	// | 1  |  1  | X'00' |  1   | Variable |    2     |
// 	// +----+-----+-------+------+----------+----------+
// 	// buffer[0]: VER
// 	// buffer[1]: REP - reply code
// 	// buffer[2]: RSV - reserved
// 	// buffer[3]: ATYP
// 	if _, err = io.ReadFull(proxyConn, buffer[:4]); err != nil {
// 		return "", fmt.Errorf("failed to read connect server response: %w", err)
// 	}

// 	if buffer[0] != 5 {
// 		return "", fmt.Errorf("invalid protocol version %v. Expected 5", buffer[0])
// 	}

// 	// if REP is not 0, it means the server returned an error.
// 	if buffer[1] != 0 {
// 		return "", ReplyCode(buffer[1])
// 	}

// 	// 4. Read address and length
// 	var bndAddrLen int
// 	switch buffer[3] {
// 	case addrTypeIPv4:
// 		bndAddrLen = 4
// 	case addrTypeIPv6:
// 		bndAddrLen = 16
// 	case addrTypeDomainName:
// 		// buffer[8]: length of the domain name
// 		_, err := io.ReadFull(proxyConn, buffer[:1])
// 		if err != nil {
// 			return "", fmt.Errorf("failed to read address length in connect response: %w", err)
// 		}
// 		bndAddrLen = int(buffer[0])
// 	default:
// 		return "", fmt.Errorf("invalid address type %v", buffer[3])
// 	}
// 	// 5. Reads the bound address and port, but we currently ignore them.
// 	// TODO(fortuna): Should we expose the remote bound address as the net.Conn.LocalAddr()?
// 	if _, err := io.ReadFull(proxyConn, buffer[:bndAddrLen]); err != nil {
// 		return "", fmt.Errorf("failed to read bound address: %w", err)
// 	}
// 	ipAddress := net.IP(buffer[:bndAddrLen])
// 	// We read but ignore the remote bound port number: BND.PORT
// 	if _, err = io.ReadFull(proxyConn, buffer[:2]); err != nil {
// 		return "", fmt.Errorf("failed to read bound port: %w", err)
// 	}
// 	port := net.PortFromBytes(buffer[:2])

// 	proxyEndpoint := transport.UDPEndpoint{Address: net.JoinHostPort(ipAddress.String(), port.String())}
// 	return proxyEndpoint.Address, nil
// }

// // func (e packetListener) DialPacket(ctx context.Context, address string) (net.Conn, error) {
// // 	netAddr, err := transport.MakeNetAddr("udp", address)
// // 	if err != nil {
// // 		return nil, err
// // 	}
// // 	packetConn, err := e.Listener.ListenPacket(ctx)
// // 	if err != nil {
// // 		return nil, fmt.Errorf("could not create PacketConn: %w", err)
// // 	}
// // 	return &boundPacketConn{
// // 		PacketConn: packetConn,
// // 		remoteAddr: netAddr,
// // 	}, nil
// // }

func makeMethodSelectionAndAuthRequest(cred *credentials, buffer []byte) []byte {
	var b []byte

	if cred == nil {
		// Method selection part: VER = 5, NMETHODS = 1, METHODS = 0 (no auth)
		// +----+----------+----------+
		// |VER | NMETHODS | METHODS  |
		// +----+----------+----------+
		// | 1  |    1     | 1 to 255 |
		// +----+----------+----------+
		b = append(buffer[:0], 5, 1, 0)
	} else {
		// https://datatracker.ietf.org/doc/html/rfc1929
		// Method selection part: VER = 5, NMETHODS = 1, METHODS = 2 (username/password)
		b = append(buffer[:0], 5, 1, authMethodUserPass)

		// Authentication part: VER = 1, ULEN = 1, UNAME = 1~255, PLEN = 1, PASSWD = 1~255
		// +----+------+----------+------+----------+
		// |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
		// +----+------+----------+------+----------+
		// | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
		// +----+------+----------+------+----------+
		b = append(b, 1)
		b = append(b, byte(len(cred.username)))
		b = append(b, cred.username...)
		b = append(b, byte(len(cred.password)))
		b = append(b, cred.password...)
	}
	return b
}

func appendConnectRequest(b *[]byte, remoteAddr string) error {
	// Connect request:
	// VER = 5, CMD = 1 (connect), RSV = 0, DST.ADDR, DST.PORT
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	var err error
	*b = append(*b, 5, 1, 0)
	// TODO: Probably more memory efficient if remoteAddr is added to the buffer directly.
	*b, err = appendSOCKS5Address(*b, remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to create SOCKS5 address: %w", err)
	}
	return nil
}

func appendUDPAssociateRequest(b *[]byte, remoteAddr string) error {
	// UDP associate request:
	// VER = 5, CMD = 3 (associate), RSV = 0, DST.ADDR, DST.PORT
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	var err error
	*b = append(*b, 5, 3, 0)
	*b, err = appendSOCKS5Address(*b, remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to create SOCKS5 address: %w", err)
	}
	return nil
}
