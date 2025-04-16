// Copyright 2023 The Outline Authors
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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

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

func TestExpired(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{})
	require.NoError(t, err)
	_, err = sd.DialStream(context.Background(), "expired.badssl.com:443")
	var certErr x509.CertificateInvalidError
	require.ErrorAs(t, err, &certErr)
	require.Equal(t, x509.Expired, certErr.Reason)
}

func TestRevoked(t *testing.T) {
	t.Skip("Certificate revocation list is not working")

	// TODO(fortuna): implement proper revocation test.
	// See https://www.cossacklabs.com/blog/tls-validation-implementing-ocsp-and-crl-in-go/

	sd, err := NewStreamDialer(&transport.TCPDialer{})
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "revoked.badssl.com:443")

	// Revocation error is thrown by the system API and is not strongly typed
	require.ErrorContains(t, err, "certificate is revoked")
	require.Nil(t, conn)
}

func TestIP(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{})
	require.NoError(t, err)
	conn, err := sd.DialStream(context.Background(), "8.8.8.8:443")
	require.NoError(t, err)
	conn.Close()
}

func TestIPOverride(t *testing.T) {
	sd, err := NewStreamDialer(&transport.TCPDialer{}, WithCertVerifier(&StandardCertVerifier{CertificateName: "8.8.8.8"}))
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
	sd, err := NewStreamDialer(&transport.TCPDialer{}, WithSNI("decoy.android.com"), WithCertVerifier(&StandardCertVerifier{CertificateName: "www.youtube.com"}))
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

// Helper function to create a self-signed certificate (Root CA)
func createRootCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{"Test Root CA"}},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert, privKey
}

// Helper function to create a leaf certificate signed by a parent
func createLeafCert(t *testing.T, dnsNames []string, ipAddresses []net.IP, parentCert *x509.Certificate, parentKey *ecdsa.PrivateKey, notBefore, notAfter time.Time) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: dnsNames[0]}, // Use first DNS name as CN
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, // Server cert
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, parentCert, &privKey.PublicKey, parentKey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert, privKey
}

func TestGeneratedCert_Valid(t *testing.T) {
	// 1. Generate Certs
	rootCA, rootKey := createRootCA(t)
	leafCert, _ := createLeafCert(t, []string{"test.local"}, nil, rootCA, rootKey, time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour))

	// 2. Setup Root Pool for Client
	rootPool := x509.NewCertPool()
	rootPool.AddCert(rootCA)

	verificationContext := &CertVerificationContext{PeerCertificates: []*x509.Certificate{leafCert}}

	sysVerifier := &StandardCertVerifier{CertificateName: "test.local"}
	require.ErrorContains(t, sysVerifier.VerifyCertificate(verificationContext), "not trusted")

	customVerifier := &StandardCertVerifier{CertificateName: "test.local", Roots: rootPool}
	require.NoError(t, customVerifier.VerifyCertificate(verificationContext))

	wrongDomainVerifier := &StandardCertVerifier{CertificateName: "other.local", Roots: rootPool}
	var hostErr x509.HostnameError
	require.ErrorAs(t, wrongDomainVerifier.VerifyCertificate(verificationContext), &hostErr)
	require.Equal(t, "other.local", hostErr.Host)
	require.Equal(t, leafCert, hostErr.Certificate)
}
