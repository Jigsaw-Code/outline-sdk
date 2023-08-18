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
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/proxy"
)

// NewStreamDialer creates a client that routes connections to a SOCKS5 proxy listening at
// the given [transport.StreamEndpoint].
func NewStreamDialer(endpoint transport.StreamEndpoint) (transport.StreamDialer, error) {
	// See https://pkg.go.dev/golang.org/x/net/proxy#SOCKS5
	if endpoint == nil {
		return nil, errors.New("argument endpoint must not be nil")
	}
	return &StreamDialer{proxyEndpoint: endpoint}, nil
}

type StreamDialer struct {
	proxyEndpoint transport.StreamEndpoint
}

var _ transport.StreamDialer = (*StreamDialer)(nil)

func (c *StreamDialer) Dial(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	proxyConn, err := c.proxyEndpoint.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not connect to SOCKS5 proxy: %w", err)
	}
	socks5Dialer, err := proxy.SOCKS5("tcp", "unused", nil, &fixedConnDialer{proxyConn})
	if err != nil {
		return nil, err
	}
	socks5Conn, err := socks5Dialer.(proxy.ContextDialer).DialContext(ctx, "tcp", remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("could not establish SOCKS5 tunnel: %w", err)
	}
	return transport.WrapConn(proxyConn, socks5Conn, socks5Conn), nil
}

type fixedConnDialer struct {
	conn net.Conn
}

func (d *fixedConnDialer) Dial(network, addr string) (c net.Conn, err error) {
	return d.conn, nil
}

func (d *fixedConnDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.conn, nil
}

var _ proxy.Dialer = (*fixedConnDialer)(nil)
var _ proxy.ContextDialer = (*fixedConnDialer)(nil)
