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

package tls

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// StreamDialer is a [transport.StreamDialer] that uses TLS to wrap the inner Dialer.
type StreamDialer struct {
	// Dialer provides the underlying connection to be wrapped.
	Dialer transport.StreamDialer
	// Options to configure the tls.Config.
	Options []Option
}

var _ transport.StreamDialer = (*StreamDialer)(nil)

// streamConn wraps a [tls.Conn] to provide a [transport.StreamConn] interface.
type streamConn struct {
	*tls.Conn
	innerConn transport.StreamConn
}

var _ transport.StreamConn = (*streamConn)(nil)

func (c streamConn) CloseRead() error {
	return c.innerConn.CloseRead()
}

func (c streamConn) CloseWrite() error {
	return c.innerConn.CloseWrite()
}

// Dial implements [transport.StreamDialer].Dial.
func (d *StreamDialer) Dial(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.Dialer.Dial(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	return WrapConn(innerConn, remoteAddr, d.Options...)
}

// Option allows tweaking the [tls.Config] to be used for a connection.
type Option func(host string, port int, config *tls.Config)

// WrapConn wraps a [transport.StreamConn] in a TLS connection.
func WrapConn(conn transport.StreamConn, remoteAdr string, options ...Option) (transport.StreamConn, error) {
	host, portStr, err := net.SplitHostPort(remoteAdr)
	if err != nil {
		return nil, fmt.Errorf("could not parse remote address: %w", err)
	}
	port, err := net.DefaultResolver.LookupPort(context.Background(), "tcp", portStr)
	if err != nil {
		return nil, fmt.Errorf("could not resolve port: %w", err)
	}
	cfg := tls.Config{ServerName: host}
	for _, option := range options {
		option(host, port, &cfg)
	}
	return streamConn{tls.Client(conn, &cfg), conn}, nil
}

// WithSNI sets the host name for [Server Name Indication](https://datatracker.ietf.org/doc/html/rfc6066#section-3) (SNI)
func WithSNI(hostName string) Option {
	return func(_ string, _ int, config *tls.Config) {
		config.ServerName = hostName
	}
}

// WithALPN sets the protocol name list for [Application-Layer Protocol Negotiation](https://datatracker.ietf.org/doc/html/rfc7301) (ALPN).
// The list of protocol IDs can be found in [IANA's registry](https://www.iana.org/assignments/tls-extensiontype-values/tls-extensiontype-values.xhtml#alpn-protocol-ids).
func WithALPN(procolNameList []string) Option {
	return func(_ string, _ int, config *tls.Config) {
		config.NextProtos = procolNameList
	}
}
