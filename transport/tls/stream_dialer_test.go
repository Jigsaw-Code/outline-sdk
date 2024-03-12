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
	"crypto/x509"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
)

func TestDomain(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{})
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "dns.google:443")
	require.NoError(t, err)
	tlsConn, ok := conn.(streamConn)
	require.True(t, ok)
	require.True(t, tlsConn.ConnectionState().HandshakeComplete)
	require.NoError(t, conn.CloseWrite())
	require.NoError(t, conn.CloseRead())
	conn.Close()
}

func TestUntrustedRoot(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{})
	require.NoError(t, err)
	_, err = sd.DialStream(context.Background(), "untrusted-root.badssl.com:443")
	var certErr x509.UnknownAuthorityError
	require.ErrorAs(t, err, &certErr)
}

func TestRevoked(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{})
	require.NoError(t, err)
	_, err = sd.DialStream(context.Background(), "revoked.badssl.com:443")
	var certErr x509.CertificateInvalidError
	require.ErrorAs(t, err, &certErr)
	require.Equal(t, x509.Expired, certErr.Reason)
}

func TestIP(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{})
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "8.8.8.8:443")
	require.NoError(t, err)
	conn.Close()
}

func TestIPOverride(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{}, WithCertificateName("8.8.8.8"))
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "dns.google:443")
	require.NoError(t, err)
	conn.Close()
}

func TestFakeSNI(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{}, WithSNI("decoy.example.com"))
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "www.youtube.com:443")
	require.NoError(t, err)
	conn.Close()
}

func TestNoSNI(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{}, WithSNI(""))
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "dns.google:443")
	require.NoError(t, err)
	conn.Close()
}

func TestAllCustom(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{}, WithSNI("decoy.android.com"), WithCertificateName("www.youtube.com"))
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "www.google.com:443")
	require.NoError(t, err)
	conn.Close()
}

func TestHostSelector(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{},
		IfHost("dns.google", WithSNI("decoy.example.com")),
		IfHost("www.youtube.com", WithSNI("notyoutube.com")),
	)
	require.NoError(t, err)

	conn, err := sd.DialStream(context.Background(), "dns.google:443")
	require.NoError(t, err)
	tlsConn := conn.(streamConn)
	require.Equal(t, "decoy.example.com", tlsConn.ConnectionState().ServerName)
	conn.Close()

	conn, err = sd.DialStream(context.Background(), "www.youtube.com:443")
	require.NoError(t, err)
	tlsConn = conn.(streamConn)
	require.Equal(t, "notyoutube.com", tlsConn.ConnectionState().ServerName)
	conn.Close()
}

func TestWithSNI(t *testing.T) {
	var cfg ClientConfig
	WithSNI("example.com")("", &cfg)
	require.Equal(t, "example.com", cfg.ServerName)
}

func TestWithALPN(t *testing.T) {
	var cfg ClientConfig
	WithALPN([]string{"h2", "http/1.1"})("", &cfg)
	require.Equal(t, []string{"h2", "http/1.1"}, cfg.NextProtos)
}

// Make sure there are no connection leakage in DialStream
func TestDialStreamCloseInnerConnOnError(t *testing.T) {
	inner := &connCounterDialer{base: &transport.TCPDialer{}}
	sd, err := NewStreamDialer(inner)
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "invalid-address?987654321")
	require.Error(t, err)
	require.Nil(t, conn)
	require.Zero(t, inner.activeConns)
}

// Private test helpers

// connCounterDialer is a StreamDialer that counts the number of active StreamConns.
type connCounterDialer struct {
	base        transport.StreamDialer
	activeConns int
}

type countedStreamConn struct {
	transport.StreamConn
	counter *connCounterDialer
}

func (d *connCounterDialer) DialStream(ctx context.Context, raddr string) (transport.StreamConn, error) {
	conn, err := d.base.DialStream(ctx, raddr)
	if conn != nil {
		d.activeConns++
	}
	return countedStreamConn{conn, d}, err
}

func (c countedStreamConn) Close() error {
	c.counter.activeConns--
	return c.StreamConn.Close()
}
