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

type Credentials struct {
	Username string
	Password string
}

// NewStreamDialer creates a [transport.StreamDialer] that routes connections to a SOCKS5
// proxy listening at the given [transport.StreamEndpoint].
func NewStreamDialer(endpoint transport.StreamEndpoint, cred Credentials) (transport.StreamDialer, error) {
	if endpoint == nil {
		return nil, errors.New("argument endpoint must not be nil")
	}
	return &streamDialer{proxyEndpoint: endpoint, credentials: cred}, nil
}

type streamDialer struct {
	proxyEndpoint transport.StreamEndpoint
	credentials   Credentials
}

var _ transport.StreamDialer = (*streamDialer)(nil)

// DialStream implements [transport.StreamDialer].DialStream using SOCKS5.
// It will send the auth method, sub-negotiation, and the connect requests in one packet, to avoid an unnecessary roundtrip.
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
	var buffer []byte

	if c.credentials == (Credentials{}) {
		// Method selection part: VER = 5, NMETHODS = 1, METHODS = 0 (no auth)
		buffer = append(buffer, 5, 1, 0)
	} else {
		// https://datatracker.ietf.org/doc/html/rfc1929
		// Method selection part: VER = 5, NMETHODS = 1, METHODS = 2 (username/password)
		buffer = append(buffer, 5, 1, 2)

		// Authentication part: VER = 1, ULEN, UNAME, PLEN, PASSWD
		buffer = append(buffer, 1) // Auth version
		buffer = append(buffer, byte(len(c.credentials.Username)))
		buffer = append(buffer, c.credentials.Username...)
		buffer = append(buffer, byte(len(c.credentials.Password)))
		buffer = append(buffer, c.credentials.Password...)
	}

	// Connect request part: VER = 5, CMD = 1 (connect), RSV = 0, DST.ADDR, DST.PORT
	connectRequest, err := appendSOCKS5Address([]byte{5, 1, 0}, remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 address: %w", err)
	}
	buffer = append(buffer, connectRequest...)

	// Sending the combined request
	_, err = proxyConn.Write(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to write combined SOCKS5 request: %w", err)
	}

	// Read several response parts in one go, to avoid an unnecessary roundtrip.
	// 1. Read method response (VER, METHOD).
	var methodResponse [2]byte
	if _, err = io.ReadFull(proxyConn, methodResponse[:]); err != nil {
		return nil, fmt.Errorf("failed to read method server response")
	}
	if methodResponse[0] != 5 {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", methodResponse[0])
	}
	if methodResponse[1] == 2 {
		// 2. Read sub-negotiation version and status
		// VER = 1, STATUS = 0
		var subNegotiation [2]byte
		if _, err = io.ReadFull(proxyConn, subNegotiation[:]); err != nil {
			return nil, fmt.Errorf("failed to read sub-negotiation version and status: %w", err)
		}
		if subNegotiation[0] != 1 {
			return nil, fmt.Errorf("unkown sub-negotioation version")
		}
		if subNegotiation[1] != 0 {
			return nil, fmt.Errorf("authentication failed: %v", subNegotiation[1])
		}
	}
	// Check if the server supports the authentication method we sent.
	// 0 is no auth, 2 is username/password
	if methodResponse[1] != 0 && methodResponse[1] != 2 {
		return nil, fmt.Errorf("unsupported SOCKS authentication method %v. Expected 2", methodResponse[1])
	}
	// 3. Read connect response (VER, REP, RSV, ATYP, BND.ADDR, BND.PORT).
	// See https://datatracker.ietf.org/doc/html/rfc1928#section-6.
	var connectResponse [4]byte
	if _, err = io.ReadFull(proxyConn, connectResponse[:]); err != nil {
		fmt.Printf("failed to read connect server response: %v", err)
		return nil, fmt.Errorf("failed to read connect server response: %w", err)
	}

	fmt.Printf("Connect response: %v", connectResponse)

	if connectResponse[0] != 5 {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", connectResponse[0])
	}

	if connectResponse[1] != 0 {
		return nil, ReplyCode(connectResponse[1])
	}

	// 4. Read and ignore the BND.ADDR and BND.PORT
	var bndAddrLen int
	switch connectResponse[3] {
	case addrTypeIPv4:
		bndAddrLen = 4
	case addrTypeIPv6:
		bndAddrLen = 16
	case addrTypeDomainName:
		var lengthByte [1]byte
		_, err := io.ReadFull(proxyConn, lengthByte[:])
		if err != nil {
			return nil, fmt.Errorf("failed to read address length in connect response: %w", err)
		}
		bndAddrLen = int(lengthByte[0])
	default:
		return nil, fmt.Errorf("invalid address type %v", connectResponse[3])
	}
	// 5. Reads the bound address and port, but we currently ignore them.
	// TODO(fortuna): Should we expose the remote bound address as the net.Conn.LocalAddr()?
	bndAddr := make([]byte, bndAddrLen)
	if _, err = io.ReadFull(proxyConn, bndAddr); err != nil {
		return nil, fmt.Errorf("failed to read bound address: %w", err)
	}
	// We also ignore the remote bound port number.
	// Read the port (2 bytes)
	var bndPort [2]byte
	if _, err = io.ReadFull(proxyConn, bndPort[:]); err != nil {
		return nil, fmt.Errorf("failed to read bound port: %w", err)
	}

	dialSuccess = true
	return proxyConn, nil
}
