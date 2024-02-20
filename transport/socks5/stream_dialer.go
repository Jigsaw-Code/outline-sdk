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
// --> THIS IS CHANGED: It will send the method and the connect requests in one packet, to avoid an unnecessary roundtrip.
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

	// Buffer large enough for method and connect requests with a domain name address.
	// header := [3 + 4 + 256 + 2]byte{}

	var methodRequest []byte
	if c.credentials == (Credentials{}) {
		// Method request:
		// VER = 5, NMETHODS = 1, METHODS = 0 (no auth)
		methodRequest = []byte{5, 1, 0}
		fmt.Println("No auth option selected")
	} else {
		// Method request:
		// VER = 5, NMETHODS = 1, METHODS = 2 (username/password)
		methodRequest = []byte{5, 1, 2}
		//b := append(header[:0], 5, 1, 2)
	}

	_, err = proxyConn.Write(methodRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to write method request: %w", err)
	}

	// Read method response (VER, METHOD).
	var methodResponse [2]byte
	if _, err = io.ReadFull(proxyConn, methodResponse[:]); err != nil {
		return nil, fmt.Errorf("failed to read method server response")
	}
	if methodResponse[0] != 5 {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", methodResponse[0])
	}

	// Check if the server supports the authentication method we sent.
	if methodResponse[1] != 2 && methodResponse[1] != 0 {
		return nil, fmt.Errorf("unsupported SOCKS authentication method %v. Expected 2", methodResponse[1])
	}

	// Handle username/password authentication
	if methodResponse[1] == 2 {
		if err := c.performUserPassAuth(proxyConn, c.credentials); err != nil {
			return nil, err
		}
	}

	// Connect request:
	// VER = 5, CMD = 1 (connect), RSV = 0
	//b = append(b, 5, 1, 0)

	// Destination address Address (ATYP, DST.ADDR, DST.PORT)
	connectRequest, err := appendSOCKS5Address([]byte{5, 1, 0}, remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 address: %w", err)
	}

	// We merge the method and connect requests because we send a single authentication
	// method, so there's no point in waiting for the response. This eliminates a roundtrip.
	_, err = proxyConn.Write(connectRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to write SOCKS5 request: %w", err)
	}

	// Read connect response (VER, REP, RSV, ATYP, BND.ADDR, BND.PORT).
	// See https://datatracker.ietf.org/doc/html/rfc1928#section-6.
	var connectResponse [4]byte
	if _, err = io.ReadFull(proxyConn, connectResponse[:]); err != nil {
		return nil, fmt.Errorf("failed to read connect server response: %w", err)
	}
	if connectResponse[0] != 5 {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", connectResponse[0])
	}
	if connectResponse[1] != 0 {
		return nil, ReplyCode(connectResponse[1])
	}

	// Read and ignore the BND.ADDR and BND.PORT
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
	// Reads the bound address and port, but we currently ignore them.
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

// https://datatracker.ietf.org/doc/html/rfc1929
func (c *streamDialer) performUserPassAuth(conn transport.StreamConn, cred Credentials) error {
	// Username/Password authentication request
	// VER = 1, ULEN, UNAME, PLEN, PASSWD
	var authRequest []byte
	authRequest = append(authRequest, byte(1)) // VER
	authRequest = append(authRequest, byte(len(cred.Username)))
	authRequest = append(authRequest, cred.Username...)
	authRequest = append(authRequest, byte(len(cred.Password)))
	authRequest = append(authRequest, cred.Password...)

	if _, err := conn.Write(authRequest); err != nil {
		return fmt.Errorf("failed to write auth request: %w", err)
	}

	var authResponse [2]byte
	if _, err := io.ReadFull(conn, authResponse[:]); err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	if authResponse[0] != 1 {
		return fmt.Errorf("invalid auth response version: %v", authResponse[0])
	}
	if authResponse[1] != 0 {
		return fmt.Errorf("authentication failed: %v", authResponse[1])
	}

	return nil
}
