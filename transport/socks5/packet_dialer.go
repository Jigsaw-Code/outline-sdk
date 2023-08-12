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

type packetDialerAdaptor struct {
	endpoint transport.PacketEndpoint
}

func (d *packetDialerAdaptor) Dial(network, addr string) (c net.Conn, err error) {
	return d.endpoint.Connect(context.Background())
}

func (d *packetDialerAdaptor) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.endpoint.Connect(ctx)
}

var _ proxy.Dialer = (*packetDialerAdaptor)(nil)
var _ proxy.ContextDialer = (*packetDialerAdaptor)(nil)

// NewPacketDialer creates a client that routes connections to a SOCKS5 proxy listening at
// the given [transport.PacketEndpoint].
func NewPacketDialer(endpoint transport.PacketEndpoint) (*PacketDialer, error) {
	// See https://pkg.go.dev/golang.org/x/net/proxy#SOCKS5
	if endpoint == nil {
		return nil, errors.New("argument endpoint must not be nil")
	}
	// TODO: use the endpoint
	proxyDialer := &packetDialerAdaptor{endpoint: endpoint}
	socks5Dialer, err := proxy.SOCKS5("udp", "unused", nil, proxyDialer)
	if err != nil {
		return nil, err
	}
	contextDialer, ok := socks5Dialer.(proxy.ContextDialer)
	if !ok {
		// This should never happen.
		return nil, errors.New("SOCKS5 dialer is not a proxy.ContextDialer")
	}
	d := PacketDialer{dialer: contextDialer}
	return &d, nil
}

type PacketDialer struct {
	dialer proxy.ContextDialer
}

func (c *PacketDialer) Dial(ctx context.Context, remoteAddr string) (net.Conn, error) {
	return c.dialer.DialContext(ctx, "udp", remoteAddr)
}

var _ transport.PacketDialer = (*PacketDialer)(nil)
