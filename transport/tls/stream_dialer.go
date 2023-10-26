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
	"crypto/x509"
	"errors"
	"fmt"
	"net"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// StreamDialer is a [transport.StreamDialer] that uses TLS to wrap the inner StreamDialer.
type StreamDialer struct {
	// dialer provides the underlying connection to be wrapped.
	dialer transport.StreamDialer
	// options to configure the tls.Config.
	options []ClientOption
}

var _ transport.StreamDialer = (*StreamDialer)(nil)

// NewStreamDialer creates a [StreamDialer] that wraps the connections from the baseDialer with TLS
// configured with the given options.
func NewStreamDialer(baseDialer transport.StreamDialer, options ...ClientOption) (*StreamDialer, error) {
	if baseDialer == nil {
		return nil, errors.New("base dialer must not be nil")
	}
	return &StreamDialer{baseDialer, options}, nil
}

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
	innerConn, err := d.dialer.Dial(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}
	conn, err := WrapConn(ctx, innerConn, remoteAddr, d.options...)
	if err != nil {
		innerConn.Close()
		return nil, err
	}
	return conn, nil
}

// clientConfig encodes the parameters for a TLS client connection.
type clientConfig struct {
	ServerName      string
	CertificateName string
	NextProtos      []string
	SessionCache    tls.ClientSessionCache
}

// ToStdConfig creates a [tls.Config] based on the configured parameters.
func (cfg *clientConfig) ToStdConfig() *tls.Config {
	certificateName := cfg.CertificateName
	if certificateName == "" {
		certificateName = cfg.ServerName
	}
	return &tls.Config{
		ServerName:         cfg.ServerName,
		NextProtos:         cfg.NextProtos,
		ClientSessionCache: cfg.SessionCache,
		// Set InsecureSkipVerify to skip the default validation we are
		// replacing. This will not disable VerifyConnection.
		InsecureSkipVerify: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			// This replicates the logic in the standard library verification:
			// https://cs.opensource.google/go/go/+/master:src/crypto/tls/handshake_client.go;l=982;drc=b5f87b5407916c4049a3158cc944cebfd7a883a9
			// And the documentation example:
			// https://pkg.go.dev/crypto/tls#example-Config-VerifyConnection
			opts := x509.VerifyOptions{
				DNSName:       certificateName,
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			return err
		},
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

// WithCertificateName sets the hostname to be used for the certificate validation.
// If absent, defaults to SNI.
func WithCertificateName(hostname string) ClientOption {
	return func(_ string, _ int, config *clientConfig) {
		config.CertificateName = hostname
	}
}
