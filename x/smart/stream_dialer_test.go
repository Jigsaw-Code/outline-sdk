package smart

import (
    "context"
    "errors"
    "fmt"
    "testing"

    "github.com/goccy/go-yaml"
    "github.com/Jigsaw-Code/outline-sdk/dns"
    "github.com/Jigsaw-Code/outline-sdk/transport"
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
    NewStreamDialerFunc func(ctx context.Context, config string) (transport.StreamDialer, error)
}

func (m *MockDialerFactory) NewStreamDialer(ctx context.Context, config string) (transport.StreamDialer, error) {
    return m.NewStreamDialerFunc(ctx, config)
}

// MockPsiphonDialerFactory is a mock implementation of PsiphonDialerFactory.
type MockPsiphonDialerFactory struct {
    NewPsiphonDialerFunc func(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error)
}

func (m *MockPsiphonDialerFactory) NewPsiphonDialer(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error) {
    return m.NewPsiphonDialerFunc(ctx, psiphonJSON, psiphonSignature)
}

// MockTestRunner is a mock implementation of TestRunner.
type MockTestRunner struct {
    TestDialerFunc func(ctx context.Context, dialer transport.StreamDialer, testDomains []string, transportCfg string) error
}

func (m *MockTestRunner) TestDialer(ctx context.Context, dialer transport.StreamDialer, testDomains []string, transportCfg string) error {
    return m.TestDialerFunc(ctx, dialer, testDomains, transportCfg)
}

// MockConfigParser is a mock implementation of ConfigParser.
type MockConfigParser struct {
    ParseConfigFunc func(configBytes []byte) (configConfig, error)
}

func (m *MockConfigParser) ParseConfig(configBytes []byte) (configConfig, error) {
    return m.ParseConfigFunc(configBytes)
}

func TestParseConfig_InvalidConfig(t *testing.T) {
    config := `
dns:
  - randomkey: {}
`
    configBytes := []byte(config)
    finder := NewStrategyFinder(nil, nil, nil, nil)
    _, err := finder.ConfigParser.ParseConfig(configBytes)
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
    parsedConfig, err := finder.ConfigParser.ParseConfig(configBytes)
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
                Psiphon: map[string]any {
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
    parsedConfig, err := finder.ConfigParser.ParseConfig(configBytes)
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
    # We get censored DNS responses when we send queries to an IP in China.
    - udp: { address: china.cn }
    # We get censored DNS responses when we send queries to a resolver in Iran.
    - udp: { address: ns1.tic.ir }
    - tcp: { address: ns1.tic.ir }
    # We get censored DNS responses when we send queries to an IP in Turkmenistan.
    - udp: { address: tmcell.tm }
    # We get censored DNS responses when we send queries to a resolver in Russia.
    - udp: { address: dns1.transtelecom.net. }
    # Testing captive portal.
    - tls:
        name: captive-portal.badssl.com
        address: captive-portal.badssl.com:443
    # Testing forged TLS certificate.
    - https: { name: mitm-software.badssl.com }

tls:
    - ""
    - split:1
    - split:2
    - split:5
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
    mockResolverFactory := &MockResolverFactory{
        NewResolverFunc: func(entry dnsEntryConfig) (dns.Resolver, bool, error) {
            return nil, false, errors.New("mock resolver error")
        },
    }

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

    mockTestRunner := &MockTestRunner{
        TestDialerFunc: func(ctx context.Context, dialer transport.StreamDialer, testDomains []string, transportCfg string) error {
            return errors.New("mock test runner error")
        },
    }

    mockConfigParser := &MockConfigParser{
        ParseConfigFunc: func(configBytes []byte) (configConfig, error) {
            var parsedConfig configConfig
            err := yaml.Unmarshal(configBytes, &parsedConfig)
            if err != nil {
                return configConfig{}, fmt.Errorf("failed to unmarshal config to map: %w", err)
            }
            return parsedConfig, nil
        },
    }

    finder := &StrategyFinder{
        TestTimeout:            5,
        LogWriter:            NewCancellableLogWriter(context.Background(), nil),
        StreamDialer:         &transport.TCPDialer{},
        PacketDialer:         &transport.UDPDialer{},
        ResolverFactory:        mockResolverFactory,
        DialerFactory:          mockDialerFactory,
        PsiphonDialerFactory: mockPsiphonDialerFactory,
        TestRunner:             mockTestRunner,
        ConfigParser:           mockConfigParser,
    }

    _, err := finder.NewDialer(context.Background(), testDomains, configBytes)
    require.Error(t, err)
    require.Contains(t, err.Error(), "could not find a working fallback: all tests failed")
}
