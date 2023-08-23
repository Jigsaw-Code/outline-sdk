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

// NewStreamDialer creates a [transport.StreamDialer] that routes connections to a SOCKS5
// proxy listening at the given [transport.StreamEndpoint].
func NewStreamDialer(endpoint transport.StreamEndpoint) (transport.StreamDialer, error) {
	if endpoint == nil {
		return nil, errors.New("argument endpoint must not be nil")
	}
	return &streamDialer{proxyEndpoint: endpoint}, nil
}

type streamDialer struct {
	proxyEndpoint transport.StreamEndpoint
}

var _ transport.StreamDialer = (*streamDialer)(nil)

// Dial implements [transport.StreamDialer].Dial
func (c *streamDialer) Dial(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	proxyConn, err := c.proxyEndpoint.Connect(ctx)
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

	// Buffer large enough for method and connect requests with a domain name address
	header := [3 + 4 + 256 + 2]byte{}

	// Method request:
	// VER = 5, NMETHODS = 1, METHODS = 0 (no auth)
	b := append(header[:0], 5, 1, 0)

	// Connect request:
	// VER = 5, CMD = 1 (connect), RSV = 0
	b = append(b, 5, 1, 0)
	// Destination address Address (ATYP, DST.ADDR, DST.PORT)
	b, err = appendSOCKS5Address(b, remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS5 address: %w", err)
	}

	// We merge the method and connect requests because we send a single authentication
	// method, so there's no point in waiting for the response. This eliminates a roundtrip.
	_, err = proxyConn.Write(b)
	if err != nil {
		return nil, fmt.Errorf("failed to write SOCKS5 request: %w", err)
	}

	// Read method response (VER, METHOD).
	if _, err = io.ReadFull(proxyConn, header[:2]); err != nil {
		return nil, fmt.Errorf("failed to read method server response")
	}
	if header[0] != 5 {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", header[0])
	}
	if header[1] != 0 {
		return nil, fmt.Errorf("unsupported SOCKS authentication method %v. Expected 0 (no auth)", header[1])
	}

	// Read connect response (VER, REP, RSV, ATYP, BND.ADDR, BND.PORT).
	// See https://datatracker.ietf.org/doc/html/rfc1928#section-6.
	if _, err = io.ReadFull(proxyConn, header[:4]); err != nil {
		return nil, fmt.Errorf("failed to read connect server response")
	}
	if header[0] != 5 {
		return nil, fmt.Errorf("invalid protocol version %v. Expected 5", header[0])
	}
	toRead := 0
	switch header[3] {
	case addrTypeIPv4:
		toRead = 4
	case addrTypeIPv6:
		toRead = 16
	case addrTypeDomainName:
		_, err := io.ReadFull(proxyConn, header[:1])
		if err != nil {
			return nil, fmt.Errorf("failed to read address length in connect response: %w", err)
		}
		toRead = int(header[0])
	}
	// Reads the bound address and port, but we currently ignore them.
	// TODO(fortuna): Should we expose the remote bound address as the net.Conn.LocalAddr()?
	_, err = io.ReadFull(proxyConn, header[:toRead])
	if err != nil {
		return nil, fmt.Errorf("failed to read address in connect response: %w", err)
	}
	// We also ignore the remote bound port number.
	_, err = io.ReadFull(proxyConn, header[:2])
	if err != nil {
		return nil, fmt.Errorf("failed to read port number in connect response: %w", err)
	}

	dialSuccess = true
	return proxyConn, nil
}
