package smart

import (
	//"bytes"
	"context"
	//"crypto/tls"
	//"errors"
	//"fmt"
	"io"
	"net"

	//"net/http"
	//"net/http/httptest"
	//"strings"
	//"sync"
	"testing"
	"time"

	//"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	//"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
	//"github.com/Jigsaw-Code/outline-sdk/x/configurl"
	"github.com/stretchr/testify/require"
	//"gopkg.in/yaml.v3"
)

// Test helpers

// mockStreamDialer is a mock implementation of transport.StreamDialer for testing.
type mockStreamDialer struct {
	dialFunc func(ctx context.Context, addr string) (transport.StreamConn, error)
}

func (m *mockStreamDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	return m.dialFunc(ctx, addr)
}

// mockPacketDialer is a mock implementation of transport.PacketDialer for testing.
type mockPacketDialer struct {
	dialFunc func(ctx context.Context, addr string) (transport.PacketConn, error)
}

func (m *mockPacketDialer) DialPacket(ctx context.Context, addr string) (transport.PacketConn, error) {
	return m.dialFunc(ctx, addr)
}

// mockStreamConn is a mock implementation of transport.StreamConn for testing.
type mockStreamConn struct {
	readFunc  func(b []byte) (n int, err error)
	writeFunc func(b []byte) (n int, err error)
	closeFunc func() error
}

func (m *mockStreamConn) Read(b []byte) (n int, err error) {
	return m.readFunc(b)
}

func (m *mockStreamConn) Write(b []byte) (n int, err error) {
	return m.writeFunc(b)
}

func (m *mockStreamConn) Close() error {
	return m.closeFunc()
}

func (m *mockStreamConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func (m *mockStreamConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 80}
}

func (m *mockStreamConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockStreamConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockStreamConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// mockPacketConn is a mock implementation of transport.PacketConn for testing.
type mockPacketConn struct {
	readFromFunc func(b []byte) (n int, addr net.Addr, err error)
	writeToFunc  func(b []byte, addr net.Addr) (n int, err error)
	closeFunc    func() error
}

func (m *mockPacketConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	return m.readFromFunc(b)
}

func (m *mockPacketConn) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	return m.writeToFunc(b, addr)
}

func (m *mockPacketConn) Close() error {
	return m.closeFunc()
}

func (m *mockPacketConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func (m *mockPacketConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockPacketConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockPacketConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// TestNewDialer tests the NewDialer function.
func TestNewDialer(t *testing.T) {
	testCases := []struct {
		name          string
		config        string
		testDomains   []string
		wantErr       bool
		wantDialError bool
	}{
		{
			name:        "valid config with proxyless",
			config:      `dns: [{system: {}}]`,
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:        "valid config with fallback",
			config:      `fallback: ["ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpxYXJNZEJ0VkYyamhaMnFFWGRtNWFY@159.203.76.146:9953/?outline=1"]`,
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:          "invalid config",
			config:        `invalid`,
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "empty config",
			config:        ``,
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "empty test domains",
			config:        `dns: [{system: {}}]`,
			testDomains:   []string{},
			wantErr:       false,
			wantDialError: false,
		},
		{
			name:          "invalid fallback",
			config:        `fallback: ["invalid"]`,
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid dns",
			config:        `dns: [{invalid: {}}]`,
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := &StrategyFinder{
				TestTimeout:  5 * time.Second,
				LogWriter:    io.Discard,
				StreamDialer: &transport.TCPDialer{},
				PacketDialer: &transport.UDPDialer{},
			}
			_, err := f.NewDialer(context.Background(), tc.testDomains, []byte(tc.config))
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewProxylessDialer(t *testing.T) {
	testCases := []struct {
		name          string
		config        configConfig
		testDomains   []string
		wantErr       bool
		wantDialError bool
	}{
		{
			name:        "valid config with system dns",
			config:      configConfig{DNS: []dnsEntryConfig{{System: &struct{}{}}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:        "valid config with https dns",
			config:      configConfig{DNS: []dnsEntryConfig{{HTTPS: &httpsEntryConfig{Name: "8.8.8.8"}}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:        "valid config with tls dns",
			config:      configConfig{DNS: []dnsEntryConfig{{TLS: &tlsEntryConfig{Name: "8.8.8.8"}}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:        "valid config with udp dns",
			config:      configConfig{DNS: []dnsEntryConfig{{UDP: &udpEntryConfig{Address: "8.8.8.8"}}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:        "valid config with tcp dns",
			config:      configConfig{DNS: []dnsEntryConfig{{TCP: &tcpEntryConfig{Address: "8.8.8.8"}}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:          "invalid config with empty https name",
			config:        configConfig{DNS: []dnsEntryConfig{{HTTPS: &httpsEntryConfig{Name: ""}}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with empty tls name",
			config:        configConfig{DNS: []dnsEntryConfig{{TLS: &tlsEntryConfig{Name: ""}}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with empty udp address",
			config:        configConfig{DNS: []dnsEntryConfig{{UDP: &udpEntryConfig{Address: ""}}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with empty tcp address",
			config:        configConfig{DNS: []dnsEntryConfig{{TCP: &tcpEntryConfig{Address: ""}}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with invalid dns entry",
			config:        configConfig{DNS: []dnsEntryConfig{{}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with no dns entry",
			config:        configConfig{DNS: []dnsEntryConfig{}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:        "valid config with tls",
			config:      configConfig{DNS: []dnsEntryConfig{{System: &struct{}{}}}, TLS: []string{""}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:          "invalid config with empty tls",
			config:        configConfig{DNS: []dnsEntryConfig{{System: &struct{}{}}}, TLS: []string{}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := &StrategyFinder{
				TestTimeout:  5 * time.Second,
				LogWriter:    io.Discard,
				StreamDialer: &transport.TCPDialer{},
				PacketDialer: &transport.UDPDialer{},
			}
			_, err := f.newProxylessDialer(context.Background(), tc.testDomains, tc.config)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFindFallback(t *testing.T) {
	testCases := []struct {
		name          string
		fallback      []string
		testDomains   []string
		wantErr       bool
		wantDialError bool
	}{
		{
			name:        "valid fallback",
			fallback:    []string{"ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTpxYXJNZEJ0VkYyamhaMnFFWGRtNWFY@159.203.76.146:9953/?outline=1"},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:          "invalid fallback",
			fallback:      []string{"invalid"},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "empty fallback",
			fallback:      []string{},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := &StrategyFinder{
				TestTimeout:  5 * time.Second,
				LogWriter:    io.Discard,
				StreamDialer: &transport.TCPDialer{},
				PacketDialer: &transport.UDPDialer{},
			}
			_, err := f.findFallback(context.Background(), tc.testDomains, tc.fallback)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFindTLS(t *testing.T) {
	testCases := []struct {
		name          string
		tls           []string
		testDomains   []string
		wantErr       bool
		wantDialError bool
	}{
		{
			name:        "valid tls",
			tls:         []string{""},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:          "invalid tls",
			tls:           []string{"invalid"},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "empty tls",
			tls:           []string{},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := &StrategyFinder{
				TestTimeout:  5 * time.Second,
				LogWriter:    io.Discard,
				StreamDialer: &transport.TCPDialer{},
				PacketDialer: &transport.UDPDialer{},
			}
			_, err := f.findTLS(context.Background(), tc.testDomains, &transport.TCPDialer{}, tc.tls)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFindDNS(t *testing.T) {
	testCases := []struct {
		name          string
		dns           []dnsEntryConfig
		testDomains   []string
		wantErr       bool
		wantDialError bool
	}{
		{
			name:        "valid dns",
			dns:         []dnsEntryConfig{{System: &struct{}{}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:          "invalid dns",
			dns:           []dnsEntryConfig{{}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "empty dns",
			dns:           []dnsEntryConfig{},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:        "valid config with https dns",
			dns:         []dnsEntryConfig{{HTTPS: &httpsEntryConfig{Name: "8.8.8.8"}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:        "valid config with tls dns",
			dns:         []dnsEntryConfig{{TLS: &tlsEntryConfig{Name: "8.8.8.8"}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:        "valid config with udp dns",
			dns:         []dnsEntryConfig{{UDP: &udpEntryConfig{Address: "8.8.8.8:53"}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:        "valid config with tcp dns",
			dns:         []dnsEntryConfig{{TCP: &tcpEntryConfig{Address: "8.8.8.8:53"}}},
			testDomains: []string{"www.google.com"},
			wantErr:     false,
		},
		{
			name:          "invalid config with empty https name",
			dns:           []dnsEntryConfig{{HTTPS: &httpsEntryConfig{Name: ""}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with empty tls name",
			dns:           []dnsEntryConfig{{TLS: &tlsEntryConfig{Name: ""}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with empty udp address",
			dns:           []dnsEntryConfig{{UDP: &udpEntryConfig{Address: ""}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with empty tcp address",
			dns:           []dnsEntryConfig{{TCP: &tcpEntryConfig{Address: ""}}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
		{
			name:          "invalid config with invalid dns entry",
			dns:           []dnsEntryConfig{{}},
			testDomains:   []string{"www.google.com"},
			wantErr:       true,
			wantDialError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := &StrategyFinder{
				TestTimeout:  5 * time.Second,
				LogWriter:    io.Discard,
				StreamDialer: &transport.TCPDialer{},
				PacketDialer: &transport.UDPDialer{},
			}
			_, err := f.findDNS(context.Background(), tc.testDomains, tc.dns)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
