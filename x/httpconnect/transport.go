// Copyright 2025 The Outline Authors
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

package httpconnect

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
)

type TransportOption func(c *transportConfig)

// WithTLS configures the transport to use TLS.
func WithTLS(tlsConf *tls.Config) TransportOption {
	return func(c *transportConfig) {
		c.tlsConf = tlsConf
	}
}

type transportConfig struct {
	tlsConf *tls.Config
}

// NewHTTPProxyTransport creates a net/http Transport that establishes a connection to the proxy using the given [transport.StreamDialer].
// The proxy address must be in the form "host:port".
//
// For HTTP/1 and HTTP/2 over a stream connection.
func NewHTTPProxyTransport(dialer transport.StreamDialer, proxyAddr string, opts ...TransportOption) (*http.Transport, error) {
	if dialer == nil {
		return nil, errors.New("dialer must not be nil")
	}
	_, _, err := net.SplitHostPort(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy address %s: %w", proxyAddr, err)
	}

	cfg := &transportConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	tr := &http.Transport{
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return dialer.DialStream(ctx, proxyAddr)
		},
		TLSClientConfig: cfg.tlsConf,
	}

	err = http2.ConfigureTransport(tr)
	if err != nil {
		return nil, fmt.Errorf("failed to configure http2 transport: %w", err)
	}

	return tr, nil
}

// NewHTTP3ProxyTransport creates an HTTP/3 transport that establishes a QUIC connection to the proxy using the given [transport.PacketDialer].
// The proxy address must be in the form "host:port".
//
// For HTTP/3 over QUIC over a datagram connection.
func NewHTTP3ProxyTransport(dialer transport.PacketDialer, proxyAddr string, opts ...TransportOption) (*http3.Transport, error) {
	if dialer == nil {
		return nil, errors.New("dialer must not be nil")
	}
	_, _, err := net.SplitHostPort(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy address %s: %w", proxyAddr, err)
	}

	cfg := &transportConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	tr := &http3.Transport{
		Dial: func(ctx context.Context, _ string, tlsCfg *tls.Config, quicCfg *quic.Config) (quic.EarlyConnection, error) {
			parsedProxyAddr, err := transport.MakeNetAddr("udp", proxyAddr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse proxy address %s: %w", proxyAddr, err)
			}

			conn, err := dialer.DialPacket(ctx, proxyAddr)
			if err != nil {
				return nil, fmt.Errorf("failed to dial proxy %s: %w", proxyAddr, err)
			}

			// quic.DialEarly expects a net.PacketConn, but transport.PacketDialer returns a net.Conn connection
			// the connection by transport.PacketDialer is assumed to be "bound", so we wrap it with in a boundConn that implements net.PacketConn
			packetConn := newBoundConn(conn, parsedProxyAddr)

			return quic.DialEarly(ctx, packetConn, parsedProxyAddr, tlsCfg, quicCfg)
		},
		TLSClientConfig: cfg.tlsConf,
	}

	return tr, nil
}

var _ net.PacketConn = (*boundConn)(nil)

type boundConn struct {
	net.Conn
	remoteAddr net.Addr
}

// Used for [quic.DialEarly] to work with [transport.PacketDialer]'s [net.Conn].
func newBoundConn(conn net.Conn, remoteAddr net.Addr) *boundConn {
	return &boundConn{Conn: conn, remoteAddr: remoteAddr}
}

func (c *boundConn) ReadFrom(p []byte) (int, net.Addr, error) {
	n, err := c.Conn.Read(p)
	return n, c.remoteAddr, err
}

func (c *boundConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if addr != c.remoteAddr {
		return 0, fmt.Errorf("unexpected address: %v", addr)
	}
	return c.Conn.Write(p)
}
