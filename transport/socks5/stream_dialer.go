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
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/proxy"
)

type streamDialerAdaptor struct {
	endpoint transport.StreamEndpoint
}

func (d *streamDialerAdaptor) Dial(network, addr string) (c net.Conn, err error) {
	return d.endpoint.Connect(context.Background())
}

func (d *streamDialerAdaptor) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.endpoint.Connect(ctx)
}

var _ proxy.Dialer = (*streamDialerAdaptor)(nil)
var _ proxy.ContextDialer = (*streamDialerAdaptor)(nil)

// NewStreamDialer creates a client that routes connections to a SOCKS5 proxy listening at
// the given [transport.StreamEndpoint].
func NewStreamDialer(endpoint transport.StreamEndpoint) (*StreamDialer, error) {
	// See https://pkg.go.dev/golang.org/x/net/proxy#SOCKS5
	if endpoint == nil {
		return nil, errors.New("argument endpoint must not be nil")
	}
	proxyDialer := &streamDialerAdaptor{endpoint: endpoint}
	socks5Dialer, err := proxy.SOCKS5("tcp", "unused", nil, proxyDialer)
	if err != nil {
		return nil, err
	}
	contextDialer, ok := socks5Dialer.(proxy.ContextDialer)
	if !ok {
		// This should never happen.
		return nil, errors.New("SOCKS5 dialer is not a proxy.ContextDialer")
	}
	d := StreamDialer{dialer: contextDialer}
	return &d, nil
}

type StreamDialer struct {
	dialer proxy.ContextDialer
}

type streamConnAdaptor struct {
	net.Conn
}

func (a *streamConnAdaptor) CloseRead() error {
	return nil
}

func (a *streamConnAdaptor) CloseWrite() error {
	return a.Close()
}

var _ transport.StreamConn = (*streamConnAdaptor)(nil)

func (c *StreamDialer) Dial(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	netConn, err := c.dialer.DialContext(ctx, "tcp", remoteAddr)
	if err != nil {
		return nil, err
	}
	return &streamConnAdaptor{netConn}, err
}

var _ transport.StreamDialer = (*StreamDialer)(nil)
