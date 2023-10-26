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
	Options []ClientOption
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

// Dial implements [transport.StreamDialer].Dial.
func (d *StreamDialer) Dial(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.Dialer.Dial(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := WrapConn(ctx, innerConn, remoteAddr, d.Options...)
	if err != nil {
		innerConn.Close()
		return nil, err
	}
	return conn, nil
}

// clientConfig encodes the parameters for a TLS client connection.
type clientConfig struct {
	ServerName   string
	NextProtos   []string
	SessionCache tls.ClientSessionCache
}

// ToStdConfig creates a [tls.Config] based on the configured parameters.
func (cfg *clientConfig) ToStdConfig() *tls.Config {
	return &tls.Config{
		ServerName:         cfg.ServerName,
		NextProtos:         cfg.NextProtos,
		ClientSessionCache: cfg.SessionCache,
	}
}

// ClientOption allows configuring the parameters to be used for a client TLS connection.
type ClientOption func(host string, port int, config *clientConfig)

// WrapConn wraps a [transport.StreamConn] in a TLS connection.
func WrapConn(ctx context.Context, conn transport.StreamConn, remoteAdr string, options ...ClientOption) (transport.StreamConn, error) {
	host, portStr, err := net.SplitHostPort(remoteAdr)
	if err != nil {
		return nil, fmt.Errorf("could not parse remote address: %w", err)
	}
	port, err := net.DefaultResolver.LookupPort(ctx, "tcp", portStr)
	if err != nil {
		return nil, fmt.Errorf("could not resolve port: %w", err)
	}
	cfg := clientConfig{ServerName: host}
	for _, option := range options {
		option(host, port, &cfg)
	}
	tlsConn := tls.Client(conn, cfg.ToStdConfig())
	err = tlsConn.HandshakeContext(ctx)
	if err != nil {
		return nil, err
	}
	return streamConn{tlsConn, conn}, nil
}

// WithSNI sets the host name for [Server Name Indication](https://datatracker.ietf.org/doc/html/rfc6066#section-3) (SNI)
func WithSNI(hostName string) ClientOption {
	return func(_ string, _ int, config *clientConfig) {
		config.ServerName = hostName
	}
}

// WithALPN sets the protocol name list for [Application-Layer Protocol Negotiation](https://datatracker.ietf.org/doc/html/rfc7301) (ALPN).
// The list of protocol IDs can be found in [IANA's registry](https://www.iana.org/assignments/tls-extensiontype-values/tls-extensiontype-values.xhtml#alpn-protocol-ids).
func WithALPN(protocolNameList []string) ClientOption {
	return func(_ string, _ int, config *clientConfig) {
		config.NextProtos = protocolNameList
	}
}

// WithSessionCache sets the [tls.ClientSessionCache] to enable session resumption of TLS connections.
func WithSessionCache(sessionCache tls.ClientSessionCache) ClientOption {
	return func(_ string, _ int, config *clientConfig) {
		config.SessionCache = sessionCache
	}
}
