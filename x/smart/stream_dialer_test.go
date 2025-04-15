package smart

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockResolverFactory is a mock implementation of ResolverFactory.
type MockResolverFactory struct {
	NewResolverFunc func(entry dnsEntryConfig) (dns.Resolver, bool, error)
}

func (m *MockResolverFactory) NewResolver(entry dnsEntryConfig) (dns.Resolver, bool, error) {
	return m.NewResolverFunc(entry)
}

// MockDialerFactory is a mock implementation of DialerFactory.
type MockDialerFactory struct {
	NewStreamDialerFunc func(ctx context.Context, baseDialer transport.StreamDialer, config string) (transport.StreamDialer, error)
}

func (m *MockDialerFactory) NewStreamDialer(ctx context.Context, baseDialer transport.StreamDialer, config string) (transport.StreamDialer, error) {
	return m.NewStreamDialerFunc(ctx, baseDialer, config)
}

// MockPsiphonDialerFactory is a mock implementation of PsiphonDialerFactory.
type MockPsiphonDialerFactory struct {
	NewPsiphonDialerFunc func(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error)
}

func (m *MockPsiphonDialerFactory) NewPsiphonDialer(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error) {
	return m.NewPsiphonDialerFunc(ctx, psiphonJSON, psiphonSignature)
}

func TestParseConfig_InvalidConfig(t *testing.T) {
	config := `
dns:
  - randomkey: {}
`
	configBytes := []byte(config)
	finder := NewStrategyFinder(nil, nil, nil, nil)
	_, err := finder.parseConfig(configBytes)
	require.Error(t, err)
}

func TestParseConfig_ValidConfig(t *testing.T) {
	config := `
dns:
  - system: {}
  - udp: { address: ns1.tic.ir }
  - tcp: { address: ns1.tic.ir }
  - udp: { address: tmcell.tm }
  - udp: { address: dns1.transtelecom.net. }
  - tls:
      name: captive-portal.badssl.com
      address: captive-portal.badssl.com:443
  - https: { name: mitm-software.badssl.com }

tls:
  - ""
  - split:1
  - split:2
  - split:5
  - tlsfrag:1

fallback:
  - ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTprSzdEdHQ0MkJLOE9hRjBKYjdpWGFK@1.2.3.4:9999/?outline=1
  - psiphon: {
      "PropagationChannelId":"FFFFFFFFFFFFFFFF",
      "SponsorId":"FFFFFFFFFFFFFFFF",
    }
  - socks5://192.168.1.10:1080
`
	configBytes := []byte(config)
	finder := NewStrategyFinder(nil, nil, nil, nil)
	parsedConfig, err := finder.parseConfig(configBytes)
	require.NoError(t, err)

	expectedConfig := configConfig{
		DNS: []dnsEntryConfig{
			{System: &struct{}{}},
			{UDP: &udpEntryConfig{Address: "ns1.tic.ir"}},
			{TCP: &tcpEntryConfig{Address: "ns1.tic.ir"}},
			{UDP: &udpEntryConfig{Address: "tmcell.tm"}},
			{UDP: &udpEntryConfig{Address: "dns1.transtelecom.net."}},
			{TLS: &tlsEntryConfig{Name: "captive-portal.badssl.com", Address: "captive-portal.badssl.com:443"}},
			{HTTPS: &httpsEntryConfig{Name: "mitm-software.badssl.com"}},
		},
		TLS: []string{"", "split:1", "split:2", "split:5", "tlsfrag:1"},
		Fallback: []fallbackEntryConfig{
			"ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTprSzdEdHQ0MkJLOE9hRjBKYjdpWGFK@1.2.3.4:9999/?outline=1",
			fallbackEntryStructConfig{
				Psiphon: map[string]any{
					"PropagationChannelId": "FFFFFFFFFFFFFFFF",
					"SponsorId":            "FFFFFFFFFFFFFFFF",
				},
			},
			"socks5://192.168.1.10:1080",
		},
	}

	require.Equal(t, expectedConfig, parsedConfig)
}

func TestParseConfig_YamlPsiphonConfig(t *testing.T) {
	config := `
fallback:
  - psiphon:
      PropagationChannelId: FFFFFFFFFFFFFFFF
      SponsorId: FFFFFFFFFFFFFFFF
`
	configBytes := []byte(config)
	finder := NewStrategyFinder(context.Background(), &transport.TCPDialer{}, &transport.UDPDialer{}, nil)
	parsedConfig, err := finder.parseConfig(configBytes)
	require.NoError(t, err)

	expectedConfig := configConfig{
		Fallback: []fallbackEntryConfig{
			fallbackEntryStructConfig{
				Psiphon: map[string]any{
					"PropagationChannelId": "FFFFFFFFFFFFFFFF",
					"SponsorId":            "FFFFFFFFFFFFFFFF",
				},
			},
		},
	}

	require.Equal(t, expectedConfig, parsedConfig)
}

func Test_getPsiphonConfigSignature_ValidFields(t *testing.T) {
	finder := NewStrategyFinder(context.Background(), &transport.TCPDialer{}, &transport.UDPDialer{}, nil)
	config := []byte(`{
        "PropagationChannelId": "FFFFFFFFFFFFFFFF",
        "SponsorId": "FFFFFFFFFFFFFFFF",
        "ClientPlatform": "outline",
        "ClientVersion": "1"
    }`)
	expected := "{PropagationChannelId: FFFFFFFFFFFFFFFF, SponsorId: FFFFFFFFFFFFFFFF}"
	actual := finder.getPsiphonConfigSignature(config)
	require.Equal(t, expected, actual)
}

func Test_getPsiphonConfigSignature_InvalidFields(t *testing.T) {
	// If we don't understand the psiphon config we received for any reason
	// then just output it as an opaque string

	finder := NewStrategyFinder(context.Background(), &transport.TCPDialer{}, &transport.UDPDialer{}, nil)
	config := []byte(`{"ClientPlatform": "outline", "ClientVersion": "1"}`)
	expected := `{"ClientPlatform": "outline", "ClientVersion": "1"}`
	actual := finder.getPsiphonConfigSignature(config)
	require.Equal(t, expected, actual)
}

func TestNewDialer_BrokenConfig(t *testing.T) {
	configBytes := []byte(`
dns:
    - udp: { address: dns.does-not-exist.rthsdfvsdfg.com }
    - tcp: { address: dns.does-not-exist.rthsdfvsdfg.com }
    - tls:
        name: dns.does-not-exist.rthsdfvsdfg.com
        address: dns.does-not-exist.rthsdfvsdfg.com:443
    - https: { name: dns.does-not-exist.rthsdfvsdfg.com }

tls:
    - ""
    - split:1
    - tlsfrag:1

fallback:
    # Nonexistent Outline Server
    - ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwNTprSzdEdHQ0MkJLOE9hRjBKYjdpWGFK@1.2.3.4:9999/?outline=1
    # Nonexistant Psiphon Config JSON
    - psiphon: {
        "PropagationChannelId":"ID1",
        "SponsorId":"ID2",
        "DisableLocalSocksProxy" : true,
        "DisableLocalHTTPProxy" : true,
        "EstablishTunnelTimeoutSeconds": 1,
        }
    # Nonexistant local socks5 proxy
    - socks5://192.168.1.10:1080
`)

	testDomains := []string{"www.google.com"}

	// Mock dependencies
	// mockResolverFactory := &MockResolverFactory{
	// 	NewResolverFunc: func(entry dnsEntryConfig) (dns.Resolver, bool, error) {
	// 		return nil, false, errors.New("mock resolver error")
	// 	},
	// }

	mockDialerFactory := &MockDialerFactory{
		NewStreamDialerFunc: func(ctx context.Context, baseDialer transport.StreamDialer, config string) (transport.StreamDialer, error) {
			return nil, errors.New("mock dialer error")
		},
	}

	mockPsiphonDialerFactory := &MockPsiphonDialerFactory{
		NewPsiphonDialerFunc: func(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error) {
			return nil, errors.New("mock psiphon dialer error")
		},
	}

	// Initialize a log writer
	var logBuffer bytes.Buffer
	logWriter := NewCancellableLogWriter(context.Background(), &logBuffer)

	streamDialer := &transport.TCPDialer{}
	packetDialer := &transport.UDPDialer{}
	finder := &StrategyFinder{
		TestTimeout:          5,
		LogWriter:            logWriter,
		StreamDialer:         streamDialer,
		PacketDialer:         packetDialer,
		ResolverFactory:      &DefaultResolverFactory{streamDialer, packetDialer},
		DialerFactory:        mockDialerFactory,
		PsiphonDialerFactory: mockPsiphonDialerFactory,
	}

	_, err := finder.NewDialer(context.Background(), testDomains, configBytes)
	logWriter.Flush()

	require.Error(t, err)
	require.Contains(t, err.Error(), "could not find a working fallback: all tests failed")

	// Check the content of the log writer
	expectedLogs := []string{
		"could not find working resolver",
		"could not find TLS strategy",
		"getStreamDialer failed",
		"getPsiphonDialer failed",
		"could not find a working fallback",
	}
	logContent := strings.Split(logBuffer.String(), "\n")
	assert.ElementsMatch(t, logContent, expectedLogs)
}
