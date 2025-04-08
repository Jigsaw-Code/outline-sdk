package smart

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/configurl"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_InvalidConfig(t *testing.T) {
	config := `
dns:
  - randomkey: {}
`
	configBytes := []byte(config)
	finder := &StrategyFinder{}
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
	finder := &StrategyFinder{}
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
	finder := &StrategyFinder{}
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
	finder := &StrategyFinder{}
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

	finder := &StrategyFinder{}
	config := []byte(`{"ClientPlatform": "outline", "ClientVersion": "1"}`)
	expected := `{"ClientPlatform": "outline", "ClientVersion": "1"}`
	actual := finder.getPsiphonConfigSignature(config)
	require.Equal(t, expected, actual)
}

func Test_getPsiphonConfigSignature_InvalidJson(t *testing.T) {
	finder := &StrategyFinder{}
	config := []byte(`invalid json`)
	expected := `invalid json`
	actual := finder.getPsiphonConfigSignature(config)
	require.Equal(t, expected, actual)
}

// MockStreamDialer is a mock implementation of transport.StreamDialer.
type MockStreamDialer struct {
	mock.Mock
}

func (m *MockStreamDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	args := m.Called(ctx, addr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(transport.StreamConn), args.Error(1)
}

// MockStreamConn is a mock implementation of transport.StreamConn.
type MockStreamConn struct {
	mock.Mock
	net.Conn
}

func (m *MockStreamConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStreamConn) CloseWrite() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStreamConn) CloseRead() error {
	args := m.Called()
	return args.Error(0)
}

// MockResolver is a mock implementation of dns.Resolver.
type MockResolver struct {
	mock.Mock
}

func (m *MockResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	args := m.Called(ctx, network, host)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]net.IP), args.Error(1)
}

func TestCreateFallbackDialers_EmptyConfig(t *testing.T) {
	finder := &StrategyFinder{}
	_, err := finder.createFallbackDialers(context.Background(), []fallbackEntryConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no fallback was specified")
}

func TestCreateFallbackDialers_InvalidConfig(t *testing.T) {
	finder := &StrategyFinder{}
	_, err := finder.createFallbackDialers(context.Background(), []fallbackEntryConfig{123})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown fallback type")
}

func TestCreateFallbackDialers_StringConfig(t *testing.T) {
	mockDialer := &MockStreamDialer{}
	mockDialer.On("DialStream", mock.Anything, mock.Anything).Return(&MockStreamConn{}, nil)
	defer mockDialer.AssertExpectations(t)

	finder := &StrategyFinder{}
	// Override the configurl.NewStreamDialer function to return our mock dialer
	oldNewStreamDialer := configurl.NewDefaultProviders().NewStreamDialer
	configurl.NewDefaultProviders().NewStreamDialer = func(ctx context.Context, config string) (transport.StreamDialer, error) {
		return mockDialer, nil
	}
	defer func() {
		configurl.NewDefaultProviders().NewStreamDialer = oldNewStreamDialer
	}()

	dialers, err := finder.createFallbackDialers(context.Background(), []fallbackEntryConfig{"testConfig"})
	require.NoError(t, err)
	require.Len(t, dialers, 1)
	require.Equal(t, "testConfig", dialers[0].Config)
	require.Equal(t, mockDialer, dialers[0].Dialer)
}

func TestCreateFallbackDialers_PsiphonConfig(t *testing.T) {
	mockDialer := &MockStreamDialer{}
	mockDialer.On("DialStream", mock.Anything, mock.Anything).Return(&MockStreamConn{}, nil)
	defer mockDialer.AssertExpectations(t)

	finder := &StrategyFinder{}
	// Override the newPsiphonDialer function to return our mock dialer
	oldNewPsiphonDialer := newPsiphonDialer
	newPsiphonDialer = func(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error) {
		return mockDialer, nil
	}
	defer func() {
		newPsiphonDialer = oldNewPsiphonDialer
	}()

	dialers, err := finder.createFallbackDialers(context.Background(), []fallbackEntryConfig{fallbackEntryStructConfig{Psiphon: map[string]any{
		"PropagationChannelId": "FFFFFFFFFFFFFFFF",
		"SponsorId":            "FFFFFFFFFFFFFFFF",
	}}})
	require.NoError(t, err)
	require.Len(t, dialers, 1)
	require.Contains(t, dialers[0].Config, "PropagationChannelId")
	require.Contains(t, dialers[0].Config, "SponsorId")
	require.Equal(t, mockDialer, dialers[0].Dialer)
}

func TestCreateFallbackDialers_PsiphonConfig_Error(t *testing.T) {
	finder := &StrategyFinder{}
	// Override the newPsiphonDialer function to return an error
	oldNewPsiphonDialer := newPsiphonDialer
	newPsiphonDialer = func(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error) {
		return nil, errors.New("psiphon error")
	}
	defer func() {
		newPsiphonDialer = oldNewPsiphonDialer
	}()

	_, err := finder.createFallbackDialers(context.Background(), []fallbackEntryConfig{fallbackEntryStructConfig{Psiphon: map[string]any{
		"PropagationChannelId": "FFFFFFFFFFFFFFFF",
		"SponsorId":            "FFFFFFFFFFFFFFFF",
	}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "getPsiphonDialer failed")
}

func TestTestFallbackDialers_Success(t *testing.T) {
	mockDialer1 := &MockStreamDialer{}
	mockDialer1.On("DialStream", mock.Anything, mock.Anything).Return(&MockStreamConn{}, nil)
	defer mockDialer1.AssertExpectations(t)

	mockDialer2 := &MockStreamDialer{}
	mockDialer2.On("DialStream", mock.Anything, mock.Anything).Return(&MockStreamConn{}, nil)
	defer mockDialer2.AssertExpectations(t)

	finder := &StrategyFinder{TestTimeout: 1 * time.Second}
	dialers := []fallbackDialer{
		{mockDialer1, "config1"},
		{mockDialer2, "config2"},
	}
	dialer, err := finder.testFallbackDialers(context.Background(), []string{"test.com"}, dialers)
	require.NoError(t, err)
	require.NotNil(t, dialer)
	require.Equal(t, mockDialer1, dialer)
}

func TestTestFallbackDialers_Failure(t *testing.T) {
	mockDialer1 := &MockStreamDialer{}
	mockDialer1.On("DialStream", mock.Anything, mock.Anything).Return(nil, errors.New("dial error"))
	defer mockDialer1.AssertExpectations(t)

	mockDialer2 := &MockStreamDialer{}
	mockDialer2.On("DialStream", mock.Anything, mock.Anything).Return(nil, errors.New("dial error"))
	defer mockDialer2.AssertExpectations(t)

	finder := &StrategyFinder{TestTimeout: 1 * time.Second}
	dialers := []fallbackDialer{
		{mockDialer1, "config1"},
		{mockDialer2, "config2"},
	}
	_, err := finder.testFallbackDialers(context.Background(), []string{"test.com"}, dialers)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not find a working fallback")
}

func TestCreateDNSDialer_SystemResolver(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	dialer, err := finder.createDNSDialer(context.Background(), []dnsEntryConfig{{System: &struct{}{}}})
	require.NoError(t, err)
	require.Equal(t, mockStreamDialer, dialer)
}

func TestCreateDNSDialer_CustomResolver(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	mockResolver := &MockResolver{}
	mockResolver.On("LookupIP", mock.Anything, mock.Anything, mock.Anything).Return([]net.IP{net.ParseIP("1.1.1.1")}, nil)
	defer mockResolver.AssertExpectations(t)

	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	// Override the findDNS function to return our mock resolver
	oldFindDNS := finder.findDNS
	finder.findDNS = func(ctx context.Context, testDomains []string, dnsConfig []dnsEntryConfig) (dns.Resolver, error) {
		return mockResolver, nil
	}
	defer func() {
		finder.findDNS = oldFindDNS
	}()

	dialer, err := finder.createDNSDialer(context.Background(), []dnsEntryConfig{{HTTPS: &httpsEntryConfig{Name: "doh.sb"}}})
	require.NoError(t, err)
	require.NotNil(t, dialer)
}

func TestCreateDNSDialer_FindDNSError(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	// Override the findDNS function to return an error
	oldFindDNS := finder.findDNS
	finder.findDNS = func(ctx context.Context, testDomains []string, dnsConfig []dnsEntryConfig) (dns.Resolver, error) {
		return nil, errors.New("findDNS error")
	}
	defer func() {
		finder.findDNS = oldFindDNS
	}()

	_, err := finder.createDNSDialer(context.Background(), []dnsEntryConfig{{HTTPS: &httpsEntryConfig{Name: "doh.sb"}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "findDNS error")
}

func TestNewProxylessDialer_DNSOnly(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	// Override the createDNSDialer function to return our mock dialer
	oldCreateDNSDialer := finder.createDNSDialer
	finder.createDNSDialer = func(ctx context.Context, dnsConfig []dnsEntryConfig) (transport.StreamDialer, error) {
		return mockStreamDialer, nil
	}
	defer func() {
		finder.createDNSDialer = oldCreateDNSDialer
	}()

	dialer, err := finder.newProxylessDialer(context.Background(), []string{"test.com"}, configConfig{DNS: []dnsEntryConfig{{System: &struct{}{}}}})
	require.NoError(t, err)
	require.Equal(t, mockStreamDialer, dialer)
}

func TestNewProxylessDialer_DNSAndTLS(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	mockTLSStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	// Override the createDNSDialer function to return our mock dialer
	oldCreateDNSDialer := finder.createDNSDialer
	finder.createDNSDialer = func(ctx context.Context, dnsConfig []dnsEntryConfig) (transport.StreamDialer, error) {
		return mockStreamDialer, nil
	}
	defer func() {
		finder.createDNSDialer = oldCreateDNSDialer
	}()
	// Override the findTLS function to return our mock dialer
	oldFindTLS := finder.findTLS
	finder.findTLS = func(ctx context.Context, testDomains []string, baseDialer transport.StreamDialer, tlsConfig []string) (transport.StreamDialer, error) {
		return mockTLSStreamDialer, nil
	}
	defer func() {
		finder.findTLS = oldFindTLS
	}()

	dialer, err := finder.newProxylessDialer(context.Background(), []string{"test.com"}, configConfig{DNS: []dnsEntryConfig{{System: &struct{}{}}}, TLS: []string{""}})
	require.NoError(t, err)
	require.Equal(t, mockTLSStreamDialer, dialer)
}

func TestNewProxylessDialer_CreateDNSError(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	// Override the createDNSDialer function to return an error
	oldCreateDNSDialer := finder.createDNSDialer
	finder.createDNSDialer = func(ctx context.Context, dnsConfig []dnsEntryConfig) (transport.StreamDialer, error) {
		return nil, errors.New("createDNSDialer error")
	}
	defer func() {
		finder.createDNSDialer = oldCreateDNSDialer
	}()

	_, err := finder.newProxylessDialer(context.Background(), []string{"test.com"}, configConfig{DNS: []dnsEntryConfig{{System: &struct{}{}}}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "createDNSDialer error")
}

func TestFindFallback_CreateFallbackDialersError(t *testing.T) {
	finder := &StrategyFinder{}
	// Override the createFallbackDialers function to return an error
	oldCreateFallbackDialers := finder.createFallbackDialers
	finder.createFallbackDialers = func(ctx context.Context, fallbackConfigs []fallbackEntryConfig) ([]fallbackDialer, error) {
		return nil, errors.New("createFallbackDialers error")
	}
	defer func() {
		finder.createFallbackDialers = oldCreateFallbackDialers
	}()

	_, err := finder.findFallback(context.Background(), []string{"test.com"}, []fallbackEntryConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "createFallbackDialers error")
}

func TestFindFallback_TestFallbackDialersError(t *testing.T) {
	finder := &StrategyFinder{}
	// Override the testFallbackDialers function to return an error
	oldTestFallbackDialers := finder.testFallbackDialers
	finder.testFallbackDialers = func(ctx context.Context, testDomains []string, dialers []fallbackDialer) (transport.StreamDialer, error) {
		return nil, errors.New("testFallbackDialers error")
	}
	defer func() {
		finder.testFallbackDialers = oldTestFallbackDialers
	}()

	// Override the createFallbackDialers function to return a valid dialer
	oldCreateFallbackDialers := finder.createFallbackDialers
	finder.createFallbackDialers = func(ctx context.Context, fallbackConfigs []fallbackEntryConfig) ([]fallbackDialer, error) {
		return []fallbackDialer{{Dialer: &MockStreamDialer{}, Config: "test"}}, nil
	}
	defer func() {
		finder.createFallbackDialers = oldCreateFallbackDialers
	}()

	_, err := finder.findFallback(context.Background(), []string{"test.com"}, []fallbackEntryConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "testFallbackDialers error")
}

func TestNewDialer_ProxylessSuccess(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	// Override the newProxylessDialer function to return our mock dialer
	oldNewProxylessDialer := finder.newProxylessDialer
	finder.newProxylessDialer = func(ctx context.Context, testDomains []string, config configConfig) (transport.StreamDialer, error) {
		return mockStreamDialer, nil
	}
	defer func() {
		finder.newProxylessDialer = oldNewProxylessDialer
	}()

	dialer, err := finder.NewDialer(context.Background(), []string{"test.com"}, []byte(`dns: []`))
	require.NoError(t, err)
	require.Equal(t, mockStreamDialer, dialer)
}

func TestNewDialer_FallbackSuccess(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	// Override the newProxylessDialer function to return an error
	oldNewProxylessDialer := finder.newProxylessDialer
	finder.newProxylessDialer = func(ctx context.Context, testDomains []string, config configConfig) (transport.StreamDialer, error) {
		return nil, errors.New("newProxylessDialer error")
	}
	defer func() {
		finder.newProxylessDialer = oldNewProxylessDialer
	}()
	// Override the findFallback function to return our mock dialer
	oldFindFallback := finder.findFallback
	finder.findFallback = func(ctx context.Context, testDomains []string, fallbackConfigs []fallbackEntryConfig) (transport.StreamDialer, error) {
		return mockStreamDialer, nil
	}
	defer func() {
		finder.findFallback = oldFindFallback
	}()

	dialer, err := finder.NewDialer(context.Background(), []string{"test.com"}, []byte(`dns: []`))
	require.NoError(t, err)
	require.Equal(t, mockStreamDialer, dialer)
}

func TestNewDialer_ParseConfigError(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}

	_, err := finder.NewDialer(context.Background(), []string{"test.com"}, []byte(`invalid yaml`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal config to map")
}

func TestNewDialer_MakeFullyQualified(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer}
	// Override the newProxylessDialer function to return our mock dialer
	oldNewProxylessDialer := finder.newProxylessDialer
	finder.newProxylessDialer = func(ctx context.Context, testDomains []string, config configConfig) (transport.StreamDialer, error) {
		require.Equal(t, []string{"test.com.", "www.google.com."}, testDomains)
		return mockStreamDialer, nil
	}
	defer func() {
		finder.newProxylessDialer = oldNewProxylessDialer
	}()

	_, err := finder.NewDialer(context.Background(), []string{"test.com", "www.google.com"}, []byte(`dns: []`))
	require.NoError(t, err)
}

func TestTestDialer_Success(t *testing.T) {
	mockDialer := &MockStreamDialer{}
	mockConn := &MockStreamConn{}
	mockConn.On("Close").Return(nil)
	mockConn.On("CloseWrite").Return(nil)
	mockConn.On("CloseRead").Return(nil)
	mockDialer.On("DialStream", mock.Anything, "test.com:443").Return(mockConn, nil)
	defer mockDialer.AssertExpectations(t)
	defer mockConn.AssertExpectations(t)

	finder := &StrategyFinder{TestTimeout: 1 * time.Second, LogWriter: io.Discard}
	err := finder.testDialer(context.Background(), mockDialer, []string{"test.com"}, "test")
	require.NoError(t, err)
}

func TestTestDialer_DialError(t *testing.T) {
	mockDialer := &MockStreamDialer{}
	mockDialer.On("DialStream", mock.Anything, "test.com:443").Return(nil, errors.New("dial error"))
	defer mockDialer.AssertExpectations(t)

	finder := &StrategyFinder{TestTimeout: 1 * time.Second, LogWriter: io.Discard}
	err := finder.testDialer(context.Background(), mockDialer, []string{"test.com"}, "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "dial error")
}

func TestTestDialer_HandshakeError(t *testing.T) {
	mockDialer := &MockStreamDialer{}
	mockConn := &MockStreamConn{}
	mockConn.On("Close").Return(nil)
	mockConn.On("CloseWrite").Return(nil)
	mockConn.On("CloseRead").Return(nil)
	mockDialer.On("DialStream", mock.Anything, "test.com:443").Return(mockConn, nil)
	defer mockDialer.AssertExpectations(t)
	defer mockConn.AssertExpectations(t)

	// Override the tls.Client function to return an error
	oldTLSClient := tlsClient
	tlsClient = func(conn net.Conn, config *tlsConfig) net.Conn {
		return &errorConn{err: errors.New("handshake error")}
	}
	defer func() {
		tlsClient = oldTLSClient
	}()

	finder := &StrategyFinder{TestTimeout: 1 * time.Second, LogWriter: io.Discard}
	err := finder.testDialer(context.Background(), mockDialer, []string{"test.com"}, "test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "handshake error")
}

// errorConn is a net.Conn that always returns an error on HandshakeContext.
type errorConn struct {
	net.Conn
	err error
}

func (c *errorConn) HandshakeContext(ctx context.Context) error {
	return c.err
}

// tlsConfig is a type alias for tls.Config to allow us to override the tls.Client function.
type tlsConfig = struct {
	ServerName string
}

// tlsClient is a type alias for tls.Client to allow us to override it in tests.
var tlsClient = func(conn net.Conn, config *tlsConfig) net.Conn {
	return conn
}

func TestNewDNSResolverFromEntry_InvalidEntry(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	_, _, err := finder.newDNSResolverFromEntry(dnsEntryConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid DNS entry")
}

func TestNewDNSResolverFromEntry_SystemEntry(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	resolver, secure, err := finder.newDNSResolverFromEntry(dnsEntryConfig{System: &struct{}{}})
	require.NoError(t, err)
	require.False(t, secure)
	require.Nil(t, resolver)
}

func TestNewDNSResolverFromEntry_HTTTPS_EmptyName(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	_, _, err := finder.newDNSResolverFromEntry(dnsEntryConfig{HTTPS: &httpsEntryConfig{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "https entry has empty server name")
}

func TestNewDNSResolverFromEntry_HTTPS_Success(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer, PacketDialer: &transport.UDPDialer{}}
	resolver, secure, err := finder.newDNSResolverFromEntry(dnsEntryConfig{HTTPS: &httpsEntryConfig{Name: "doh.sb"}})
	require.NoError(t, err)
	require.True(t, secure)
	require.NotNil(t, resolver)
}

func TestNewDNSResolverFromEntry_TLS_EmptyName(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	_, _, err := finder.newDNSResolverFromEntry(dnsEntryConfig{TLS: &tlsEntryConfig{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "tls entry has empty server name")
}

func TestNewDNSResolverFromEntry_TLS_Success(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer, PacketDialer: &transport.UDPDialer{}}
	resolver, secure, err := finder.newDNSResolverFromEntry(dnsEntryConfig{TLS: &tlsEntryConfig{Name: "dot.sb"}})
	require.NoError(t, err)
	require.True(t, secure)
	require.NotNil(t, resolver)
}

func TestNewDNSResolverFromEntry_TCP_EmptyAddress(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	_, _, err := finder.newDNSResolverFromEntry(dnsEntryConfig{TCP: &tcpEntryConfig{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "tcp entry has empty server address")
}

func TestNewDNSResolverFromEntry_TCP_Success(t *testing.T) {
	mockStreamDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockStreamDialer, PacketDialer: &transport.UDPDialer{}}
	resolver, secure, err := finder.newDNSResolverFromEntry(dnsEntryConfig{TCP: &tcpEntryConfig{Address: "1.1.1.1"}})
	require.NoError(t, err)
	require.False(t, secure)
	require.NotNil(t, resolver)
}

func TestNewDNSResolverFromEntry_UDP_EmptyAddress(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	_, _, err := finder.newDNSResolverFromEntry(dnsEntryConfig{UDP: &udpEntryConfig{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "udp entry has empty server address")
}

func TestNewDNSResolverFromEntry_UDP_Success(t *testing.T) {
	mockPacketDialer := &transport.UDPDialer{}
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: mockPacketDialer}
	resolver, secure, err := finder.newDNSResolverFromEntry(dnsEntryConfig{UDP: &udpEntryConfig{Address: "1.1.1.1"}})
	require.NoError(t, err)
	require.False(t, secure)
	require.NotNil(t, resolver)
}

func TestDNSConfigToResolver_EmptyConfig(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	_, err := finder.dnsConfigToResolver([]dnsEntryConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no DNS config entry")
}

func TestDNSConfigToResolver_InvalidEntry(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	_, err := finder.dnsConfigToResolver([]dnsEntryConfig{{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid DNS entry")
}

func TestDNSConfigToResolver_Success(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	resolvers, err := finder.dnsConfigToResolver([]dnsEntryConfig{
		{System: &struct{}{}},
		{HTTPS: &httpsEntryConfig{Name: "doh.sb"}},
	})
	require.NoError(t, err)
	require.Len(t, resolvers, 2)
	require.NotNil(t, resolvers[0].Resolver)
	require.False(t, resolvers[0].Secure)
	require.NotNil(t, resolvers[1].Resolver)
	require.True(t, resolvers[1].Secure)
}

func TestFindTLS_EmptyConfig(t *testing.T) {
	finder := &StrategyFinder{StreamDialer: &MockStreamDialer{}, PacketDialer: &transport.UDPDialer{}}
	_, err := finder.findTLS(context.Background(), []string{"test.com"}, &MockStreamDialer{}, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "config for TLS is empty")
}

func TestFindTLS_Success(t *testing.T) {
	mockBaseDialer := &MockStreamDialer{}
	mockTLSStreamDialer := &MockStreamDialer{}
	mockTLSStreamDialer.On("DialStream", mock.Anything, mock.Anything).Return(&MockStreamConn{}, nil)
	defer mockTLSStreamDialer.AssertExpectations(t)

	finder := &StrategyFinder{StreamDialer: mockBaseDialer, TestTimeout: 1 * time.Second}
	// Override the configurl.NewStreamDialer function to return our mock dialer
	oldNewStreamDialer := configurl.NewDefaultProviders().NewStreamDialer
	configurl.NewDefaultProviders().NewStreamDialer = func(ctx context.Context, config string) (transport.StreamDialer, error) {
		return mockTLSStreamDialer, nil
	}
	defer func() {
		configurl.NewDefaultProviders().NewStreamDialer = oldNewStreamDialer
	}()

	dialer, err := finder.findTLS(context.Background(), []string{"test.com"}, mockBaseDialer, []string{"testConfig"})
	require.NoError(t, err)
	require.NotNil(t, dialer)
}

func TestFindTLS_WrapStreamDialerError(t *testing.T) {
	mockBaseDialer := &MockStreamDialer{}
	finder := &StrategyFinder{StreamDialer: mockBaseDialer, TestTimeout: 1 * time.Second}
	// Override the configurl.NewStreamDialer function to return an error
	oldNewStreamDialer := configurl.NewDefaultProviders().NewStreamDialer
	configurl.NewDefaultProviders().NewStreamDialer = func(ctx context.Context, config string) (transport.StreamDialer, error) {
		return nil, errors.New("WrapStreamDialer error")
	}
	defer func() {
		configurl.NewDefaultProviders().NewStreamDialer = oldNewStreamDialer
	}()

	_, err := finder.findTLS(context.Background(), []string{"test.com"}, mockBaseDialer, []string{"testConfig"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "WrapStreamDialer failed")
}

func TestFindTLS_TestDialerError(t *testing.T) {
	mockBaseDialer := &MockStreamDialer{}
	mockTLSStreamDialer := &MockStreamDialer{}
	mockTLSStreamDialer.On("DialStream", mock.Anything, mock.Anything).Return(nil, errors.New("dial error"))
	defer mockTLSStreamDialer.AssertExpectations(t)

	finder := &StrategyFinder{StreamDialer: mockBaseDialer, TestTimeout: 1 * time.Second, LogWriter: io.Discard}
	// Override the configurl.NewStreamDialer function to return our mock dialer
	oldNewStreamDialer := configurl.NewDefaultProviders().NewStreamDialer
	configurl.NewDefaultProviders().NewStreamDialer = func(ctx context.Context, config string) (transport.StreamDialer, error) {
		return mockTLSStreamDialer, nil
	}
	defer func() {
		configurl.NewDefaultProviders().NewStreamDialer = oldNewStreamDialer
	}()

	_, err := finder.findTLS(context.Background(), []string{"test.com"}, mockBaseDialer, []string{"testConfig"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "dial error")
}

func TestFindTLS_DialerFunc(t *testing.T) {
	mockBaseDialer := &MockStreamDialer{}
	mockBaseDialer.On("DialStream", mock.Anything, "test.com:80").Return(&MockStreamConn{}, nil)
	mockBaseDialer.On("DialStream", mock.Anything, "test.com:443").Return(&MockStreamConn{}, nil)
	mockBaseDialer.On("DialStream", mock.Anything, "test.com:853").Return(&MockStreamConn{}, nil)
	defer mockBaseDialer.AssertExpectations(t)

	mockTLSStreamDialer := &MockStreamDialer{}
	mockTLSStreamDialer.On("DialStream", mock.Anything, "test.com:443").Return(&MockStreamConn{}, nil)
	mockTLSStreamDialer.On("DialStream", mock.Anything, "test.com:853").Return(&MockStreamConn{}, nil)
	defer mockTLSStreamDialer.AssertExpectations(t)

	finder := &StrategyFinder{StreamDialer: mockBaseDialer, TestTimeout: 1 * time.Second, LogWriter: io.Discard}
	// Override the configurl.NewStreamDialer function to return our mock dialer
	oldNewStreamDialer := configurl.NewDefaultProviders().NewStreamDialer
	configurl.NewDefaultProviders().NewStreamDialer = func(ctx context.Context, config string) (transport.StreamDialer, error) {
		return mockTLSStreamDialer, nil
	}
	defer func() {
		configurl.NewDefaultProviders().NewStreamDialer = oldNewStreamDialer
	}()

	dialer, err := finder.findTLS(context.Background(), []string{"test.com"}, mockBaseDialer, []string{"testConfig"})
	require.NoError(t, err)
	require.NotNil(t, dialer)

	_, err = dialer.DialStream(context.Background(), "test.com:80")
	require.NoError(t, err)

	_, err = dialer.DialStream(context.Background(), "test.com:443")
	require.NoError(t, err)

	_, err = dialer.DialStream(context.Background(), "test.com:853")
	require.NoError(t, err)
}

func TestFindTLS_InvalidAddress(t *testing.T) {
	mockBaseDialer := &MockStreamDialer{}
	mockTLSStreamDialer := &MockStreamDialer{}
	mockTLSStreamDialer.On("DialStream", mock.Anything, mock.Anything).Return(&MockStreamConn{}, nil)
	defer mockTLSStreamDialer.AssertExpectations(t)

	finder := &StrategyFinder{StreamDialer: mockBaseDialer, TestTimeout: 1 * time.Second, LogWriter: io.Discard}
	// Override the configurl.NewStreamDialer function to return our mock dialer
	oldNewStreamDialer := configurl.NewDefaultProviders().NewStreamDialer
	configurl.NewDefaultProviders().NewStreamDialer = func(ctx context.Context, config string) (transport.StreamDialer, error) {
		return mockTLSStreamDialer, nil
	}
	defer func() {
		configurl.NewDefaultProviders().NewStreamDialer = oldNewStreamDialer
	}()

	dialer, err := finder.findTLS(context.Background(), []string{"test.com"}, mockBaseDialer, []string{"testConfig"})
	require.NoError(t, err)
	require.NotNil(t, dialer)

	_, err = dialer.DialStream(context.Background(), "test.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse address")
}

func TestFindTLS_InvalidPort(t *testing.T) {
	mockBaseDialer := &MockStreamDialer{}
	mockTLSStreamDialer := &MockStreamDialer{}
	mockTLSStreamDialer.On("DialStream", mock.Anything, mock.Anything).Return(&MockStreamConn{}, nil)
	defer mockTLSStreamDialer.AssertExpectations(t)

	finder := &StrategyFinder{StreamDialer: mockBaseDialer, TestTimeout: 1 * time.Second, LogWriter: io.Discard}
	// Override the configurl.NewStreamDialer function to return our mock dialer
	oldNewStreamDialer := configurl.NewDefaultProviders().NewStreamDialer
	configurl.NewDefaultProviders().NewStreamDialer = func(ctx context.Context, config string) (transport.StreamDialer, error) {
		return mockTLSStreamDialer, nil
	}
	defer func() {
		configurl.NewDefaultProviders().NewStreamDialer = oldNewStreamDialer
	}()

	dialer, err := finder.findTLS(context.Background(), []string{"test.com"}, mockBaseDialer, []string{"testConfig"})
	require.NoError(t, err)
	require.NotNil(t, dialer)

	_, err = dialer.DialStream(context.Background(), "test.com:invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not resolve port")