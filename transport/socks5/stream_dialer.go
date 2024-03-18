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
	"errors"
	"fmt"
	"io"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

const (
	maxCredentialLength = 255
	minCredentialLength = 0x01
	socksProtocolVer    = 0x05
	noAuthMethod        = 0
	userPassAuthMethod  = 2
	numberOfAuthMethods = 1
	authVersion         = 1
	connectCommand      = 1
	rsv                 = 0
	authSuccess         = 0
	addrLengthIPv4      = 4
	addrLengthIPv6      = 16
	bufferSize          = 3 + 1 + 1 + 255 + 1 + 255
)

// https://datatracker.ietf.org/doc/html/rfc1929
// Credentials can be nil, and that means no authentication.
type Credentials struct {
	username []byte
	password []byte
}

func NewCredentials(username, password string) (Credentials, error) {
	var c Credentials
	if err := c.setUsername(username); err != nil {
		return c, err
	}
	if err := c.setPassword(password); err != nil {
		return c, err
	}
	return c, nil
}

// SetUsername sets the username field, ensuring it doesn't exceed 255 bytes in length and is at least 1 byte.
func (c *Credentials) setUsername(username string) error {
	usernameBytes := []byte(username)
	if len(usernameBytes) > maxCredentialLength {
		return errors.New("username exceeds 255 bytes")
	}
	if len(usernameBytes) < minCredentialLength {
		return errors.New("username must be at least 1 byte")
	}
	c.username = usernameBytes
	return nil
}

// SetPassword sets the password field, ensuring it doesn't exceed 255 bytes in length and is at least 1 byte.
func (c *Credentials) setPassword(password string) error {
	passwordBytes := []byte(password)
	if len(passwordBytes) > maxCredentialLength {
		return errors.New("password exceeds 255 bytes")
	}
	if len(passwordBytes) < minCredentialLength {
		return errors.New("password must be at least 1 byte")
	}
	c.password = passwordBytes
	return nil
}

// NewStreamDialer creates a [transport.StreamDialer] that routes connections to a SOCKS5
// proxy listening at the given [transport.StreamEndpoint].
func NewStreamDialer(endpoint transport.StreamEndpoint, cred *Credentials) (transport.StreamDialer, error) {
	if endpoint == nil {
		return nil, errors.New("argument endpoint must not be nil")
	}
	return &streamDialer{proxyEndpoint: endpoint, credentials: cred}, nil
}

type streamDialer struct {
	proxyEndpoint transport.StreamEndpoint
	credentials   *Credentials
}

var _ transport.StreamDialer = (*streamDialer)(nil)

// DialStream implements [transport.StreamDialer].DialStream using SOCKS5.
// It will send the auth method, auth credentials (if auth is chosen), and
// the connect requests in one packet, to avoid an additional roundtrip.
// The returned [error] will be of type [ReplyCode] if the server sends a SOCKS error reply code, which
// you can check against the error constants in this package using [errors.Is].
func (c *streamDialer) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
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
	// The maximum size of the buffer is
	// 3 (1 socks version + 1 method selection + 1 methods)
	// + 1 (auth version) + 1 (username length) + 255 (username) + 1 (password length) + 255 (password)

	var buffer [bufferSize]byte

	if c.credentials == nil {
		// Method selection part: VER = 5, NMETHODS = 1, METHODS = 0 (no auth)
		// +----+----------+----------+
		// |VER | NMETHODS | METHODS  |
		// +----+----------+----------+
		// | 1  |    1     | 1 to 255 |
		// +----+----------+----------+
		buffer[0] = socksProtocolVer
		buffer[1] = numberOfAuthMethods
		buffer[2] = noAuthMethod
		if _, err := proxyConn.Write(buffer[:3]); err != nil {
			return nil, fmt.Errorf("failed to write method selection: %w", err)
		}
	} else {
		// https://datatracker.ietf.org/doc/html/rfc1929
		// Method selection part: VER = 5, NMETHODS = 1, METHODS = 2 (username/password)
		buffer[0] = socksProtocolVer
		buffer[1] = numberOfAuthMethods
		buffer[2] = userPassAuthMethod
		offset := 3

		// Authentication part: VER = 1, ULEN, UNAME, PLEN, PASSWD
		// +----+------+----------+------+----------+
		// |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
		// +----+------+----------+------+----------+
		// | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
		// +----+------+----------+------+----------+
		// Authentication part: VER = 1, ULEN, UNAME, PLEN, PASSWD
		buffer[offset] = authVersion
		offset++
		buffer[offset] = byte(len(c.credentials.username))
		offset++
		copy(buffer[offset:], c.credentials.username)
		offset += len(c.credentials.username)
		buffer[offset] = byte(len(c.credentials.password))
		offset++
		copy(buffer[offset:], c.credentials.password)
		offset += len(c.credentials.password)

		if _, err := proxyConn.Write(buffer[:offset]); err != nil {
			return nil, fmt.Errorf("failed to write method selection and authentication: %w", err)
		}
	}

	// Connect request:
	// VER = 5, CMD = 1 (connect), RSV = 0, DST.ADDR, DST.PORT
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	buffer[0] = socksProtocolVer
	buffer[1] = connectCommand
	buffer[2] = rsv
	connectRequest, err := appendSOCKS5Address(buffer[:3], remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 address: %w", err)
	}

	// We merge the method and connect requests and only perform one write
	// because we send a single authentication method, so there's no point
	// in waiting for the response. This eliminates a roundtrip.
	_, err = proxyConn.Write(connectRequest)
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
	if _, err = io.ReadFull(proxyConn, buffer[:2]); err != nil {
		return nil, fmt.Errorf("failed to read method server response")
	}
	if buffer[0] != socksProtocolVer {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", buffer[0])
	}
	if buffer[1] == userPassAuthMethod {
		// 2. Read authentication version and status
		// VER = 1, STATUS = 0
		// +----+--------+
		// |VER | STATUS |
		// +----+--------+
		// | 1  |   1    |
		// +----+--------+
		// VER = 1 means the server should be expecting username/password authentication.
		// var subNegotiation [2]byte
		if _, err = io.ReadFull(proxyConn, buffer[2:4]); err != nil {
			return nil, fmt.Errorf("failed to read sub-negotiation version and status: %w", err)
		}
		if buffer[2] != authVersion {
			return nil, authVersionError(buffer[2])
		}
		if buffer[3] != authSuccess {
			return nil, fmt.Errorf("authentication failed: %v", buffer[3])
		}
	}
	// Check if the server supports the authentication method we sent.
	// 0 is no auth, 2 is username/password
	// if buffer[1] != 0 && buffer[1] != 2 {
	// 	return nil, fmt.Errorf("unsupported SOCKS authentication method %v. Expected 2", buffer[1])
	// }
	// 3. Read connect response (VER, REP, RSV, ATYP, BND.ADDR, BND.PORT).
	// See https://datatracker.ietf.org/doc/html/rfc1928#section-6.
	// +----+-----+-------+------+----------+----------+
	// |VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	//var connectResponse [4]byte
	if _, err = io.ReadFull(proxyConn, buffer[4:8]); err != nil {
		fmt.Printf("failed to read connect server response: %v", err)
		return nil, fmt.Errorf("failed to read connect server response: %w", err)
	}

	if buffer[4] != socksProtocolVer {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", buffer[4])
	}

	if buffer[5] != 0 {
		return nil, ReplyCode(buffer[5])
	}

	// 4. Read and ignore the BND.ADDR and BND.PORT
	var bndAddrLen int
	switch buffer[7] {
	case addrTypeIPv4:
		bndAddrLen = addrLengthIPv4
	case addrTypeIPv6:
		bndAddrLen = addrLengthIPv6
	case addrTypeDomainName:
		//var lengthByte [1]byte
		_, err := io.ReadFull(proxyConn, buffer[9:10])
		if err != nil {
			return nil, fmt.Errorf("failed to read address length in connect response: %w", err)
		}
		bndAddrLen = int(buffer[9])
	default:
		return nil, fmt.Errorf("invalid address type %v", buffer[7])
	}
	// 5. Reads the bound address and port, but we currently ignore them.
	// TODO(fortuna): Should we expose the remote bound address as the net.Conn.LocalAddr()?
	//bndAddr := make([]byte, bndAddrLen)
	if _, err = io.ReadFull(proxyConn, buffer[8:8+bndAddrLen]); err != nil {
		return nil, fmt.Errorf("failed to read bound address: %w", err)
	}
	// We also ignore the remote bound port number.
	// Read the port (2 bytes)
	//var bndPort [2]byte
	if _, err = io.ReadFull(proxyConn, buffer[9+bndAddrLen:11+bndAddrLen]); err != nil {
		return nil, fmt.Errorf("failed to read bound port: %w", err)
	}

	dialSuccess = true
	return proxyConn, nil
}
