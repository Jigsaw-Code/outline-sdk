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

package smart

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/configurl"
)

// To test one strategy:
// go run -C ./x/examples/smart-proxy/ . -v -localAddr=localhost:1080 --transport="" --domain www.rferl.org  --config=<(echo '{"dns": [{"https": {"name": "doh.sb"}}]}')

// CancellableLogWriter is a log writer that can be cancelled.
type CancellableLogWriter struct {
	Writer	io.Writer
	ctx 	context.Context
	logMu   sync.Mutex
}

func NewCancellableLogWriter(ctx context.Context, writer io.Writer) *CancellableLogWriter {
	return &CancellableLogWriter{Writer: writer, ctx: ctx}
}

// Only log if context is not done
func (f *CancellableLogWriter) logCtx(ctx context.Context, format string, a ...any) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	f.log(format, a...)
}

func (f *CancellableLogWriter) log(format string, a ...any) {
	if f.Writer != nil {
		f.logMu.Lock()
		defer f.logMu.Unlock()
		fmt.Fprintf(f.Writer, format, a...)
	}
}

// ResolverFactory creates a dns.Resolver from a dnsEntryConfig.
type ResolverFactory interface {
	NewResolver(entry dnsEntryConfig) (dns.Resolver, bool, error)
}

// DialerFactory creates a transport.StreamDialer from a string config.
type DialerFactory interface {
	NewStreamDialer(ctx context.Context, baseDialer transport.StreamDialer, config string) (transport.StreamDialer, error)
}

// PsiphonDialerFactory creates a transport.StreamDialer from a psiphon config.
type PsiphonDialerFactory interface {
	NewPsiphonDialer(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error)
}

// TestRunner runs tests against a dialer.
type TestRunner interface {
	TestDialer(ctx context.Context, dialer transport.StreamDialer, testDomains []string, transportCfg string) error
}

// ConfigParser parses the config.
type ConfigParser interface {
	ParseConfig(configBytes []byte) (configConfig, error)
}

type StrategyFinder struct {
	TestTimeout  time.Duration
	LogWriter    *CancellableLogWriter
	StreamDialer transport.StreamDialer
	PacketDialer transport.PacketDialer
	ResolverFactory ResolverFactory
	DialerFactory DialerFactory
	PsiphonDialerFactory PsiphonDialerFactory
	TestRunner TestRunner
	ConfigParser ConfigParser
}

func NewStrategyFinder(ctx context.Context, streamDialer transport.StreamDialer, packetDialer transport.PacketDialer, logWriter io.Writer) *StrategyFinder {
	cancellableLogWriter := NewCancellableLogWriter(ctx, logWriter)
	return &StrategyFinder{
		TestTimeout:  5 * time.Second,
		LogWriter:    cancellableLogWriter,
		StreamDialer: streamDialer,
		PacketDialer: packetDialer,
		ResolverFactory: &DefaultResolverFactory{streamDialer, packetDialer},
		DialerFactory: &DefaultDialerFactory{},
		PsiphonDialerFactory: &DefaultPsiphonDialerFactory{},
		TestRunner: &DefaultTestRunner{cancellableLogWriter},
		ConfigParser: &DefaultConfigParser{},
	}
}

func (f *StrategyFinder) log(format string, a ...any) {
	if f.LogWriter != nil {
		f.LogWriter.log(format, a...)
	}
}

type httpsEntryConfig struct {
	// Domain name of the host.
	Name string `yaml:"name,omitempty"`
	// Host:port. Defaults to Name:443.
	Address string `yaml:"address,omitempty"`
}

type tlsEntryConfig struct {
	// Domain name of the host.
	Name string `yaml:"name,omitempty"`
	// Host:port. Defaults to Name:853.
	Address string `yaml:"address,omitempty"`
}

type udpEntryConfig struct {
	// Host:port.
	Address string `yaml:"address,omitempty"`
}

type tcpEntryConfig struct {
	// Host:port.
	Address string `yaml:"address,omitempty"`
}

type dnsEntryConfig struct {
	System *struct{}         `yaml:"system,omitempty"`
	HTTPS  *httpsEntryConfig `yaml:"https,omitempty"`
	TLS    *tlsEntryConfig   `yaml:"tls,omitempty"`
	UDP    *udpEntryConfig   `yaml:"udp,omitempty"`
	TCP    *tcpEntryConfig   `yaml:"tcp,omitempty"`
}

type fallbackEntryStructConfig struct {
	Psiphon any	`yaml:"psiphon,omitempty"`
	// As we allow more fallback types beyond psiphon they will be added here
}

// This contains either a configURL string or a fallbackEntryStructConfig
// It is parsed into the correct type later
type fallbackEntryConfig any

type configConfig struct {
	DNS      []dnsEntryConfig 		`yaml:"dns,omitempty"`
	TLS      []string         		`yaml:"tls,omitempty"`
	Fallback []fallbackEntryConfig 	`yaml:"fallback,omitempty"`
}

// mapToAny marshalls a map into a struct. It's a helper for parsers that want to
// map config maps into their config structures.
func mapToAny(in map[string]any, out any) error {
	newMap := make(map[string]any)
	for k, v := range in {
		if len(k) > 0 && k[0] == '$' {
			// Skip $ keys
			continue
		}
		newMap[k] = v
	}
	yamlText, err := yaml.Marshal(newMap)
	if err != nil {
		return fmt.Errorf("error marshaling to YAML: %w", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(yamlText), yaml.DisallowUnknownField())
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("error decoding YAML: %w", err)
	}
	return nil
}

// DefaultResolverFactory is the default implementation of ResolverFactory.
type DefaultResolverFactory struct {
	StreamDialer transport.StreamDialer
	PacketDialer transport.PacketDialer
}

func (f *DefaultResolverFactory) NewResolver(entry dnsEntryConfig) (dns.Resolver, bool, error) {
	if entry.System != nil {
		return nil, false, nil
	} else if cfg := entry.HTTPS; cfg != nil {
		if cfg.Name == "" {
			return nil, true, errors.New("https entry has empty server name")
		}
		serverAddr := cfg.Address
		if serverAddr == "" {
			serverAddr = cfg.Name
		}
		_, port, err := net.SplitHostPort(serverAddr)
		if err != nil {
			serverAddr = net.JoinHostPort(serverAddr, "443")
			port = "443"
		}
		dohURL := url.URL{Scheme: "https", Host: net.JoinHostPort(cfg.Name, port), Path: "/dns-query"}
		return dns.NewHTTPSResolver(f.StreamDialer, serverAddr, dohURL.String()), true, nil
	} else if cfg := entry.TLS; cfg != nil {
		if cfg.Name == "" {
			return nil, true, errors.New("tls entry has empty server name")
		}
		serverAddr := cfg.Address
		if serverAddr == "" {
			serverAddr = cfg.Name
		}
		_, _, err := net.SplitHostPort(serverAddr)
		if err != nil {
			serverAddr = net.JoinHostPort(serverAddr, "853")
		}
		return dns.NewTLSResolver(f.StreamDialer, serverAddr, cfg.Name), true, nil
	} else if cfg := entry.TCP; cfg != nil {
		if cfg.Address == "" {
			return nil, false, errors.New("tcp entry has empty server address")
		}
		host, port, err := net.SplitHostPort(cfg.Address)
		if err != nil {
			host = cfg.Address
			port = "53"
		}
		serverAddr := net.JoinHostPort(host, port)
		return dns.NewTCPResolver(f.StreamDialer, serverAddr), false, nil
	} else if cfg := entry.UDP; cfg != nil {
		if cfg.Address == "" {
			return nil, false, errors.New("udp entry has empty server address")
		}
		host, port, err := net.SplitHostPort(cfg.Address)
		if err != nil {
			host = cfg.Address
			port = "53"
		}
		serverAddr := net.JoinHostPort(host, port)
		return dns.NewUDPResolver(f.PacketDialer, serverAddr), false, nil
	} else {
		return nil, false, errors.New("invalid DNS entry")
	}
}

// DefaultDialerFactory is the default implementation of DialerFactory.
type DefaultDialerFactory struct{}

func (f *DefaultDialerFactory) NewStreamDialer(ctx context.Context, baseDialer transport.StreamDialer, config string) (transport.StreamDialer, error) {
	var configModule = configurl.NewDefaultProviders()
	configModule.StreamDialers.BaseInstance = baseDialer
	return configModule.NewStreamDialer(ctx, config)
}

// DefaultPsiphonDialerFactory is the default implementation of PsiphonDialerFactory.
type DefaultPsiphonDialerFactory struct{}

func (f *DefaultPsiphonDialerFactory) NewPsiphonDialer(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error) {
	return newPsiphonDialer(ctx, psiphonJSON, psiphonSignature)
}

// DefaultTestRunner is the default implementation of TestRunner.
type DefaultTestRunner struct{
	logWriter *CancellableLogWriter
}

func (f *DefaultTestRunner) TestDialer(ctx context.Context, dialer transport.StreamDialer, testDomains []string, transportCfg string) error {
	for _, testDomain := range testDomains {
		startTime := time.Now()

		testAddr := net.JoinHostPort(testDomain, "443")
		f.logWriter.logCtx(ctx, "🏃 running test: '%v' (domain: %v)\n", transportCfg, testDomain)

		ctx, cancel := context.WithTimeout(ctx, 5 * time.Second)
		defer cancel()
		testConn, err := dialer.DialStream(ctx, testAddr)
		if err != nil {
			f.logWriter.logCtx(ctx, "🏁 failed to dial: '%v' (domain: %v), duration=%v, dial_error=%v ❌\n", transportCfg, testDomain, time.Since(startTime), err)
			return err
		}
		tlsConn := tls.Client(testConn, &tls.Config{ServerName: testDomain})
		err = tlsConn.HandshakeContext(ctx)
		tlsConn.Close()
		if err != nil {
			f.logWriter.logCtx(ctx, "🏁 failed TLS handshake: '%v' (domain: %v), duration=%v, handshake=%v ❌\n", transportCfg, testDomain, time.Since(startTime), err)
			return err
		}
		f.logWriter.logCtx(ctx, "🏁 success: '%v' (domain: %v), duration=%v, status=ok ✅\n", transportCfg, testDomain, time.Since(startTime))
	}
	return nil
}

// DefaultConfigParser is the default implementation of ConfigParser.
type DefaultConfigParser struct{}

func (f *DefaultConfigParser) ParseConfig(configBytes []byte) (configConfig, error) {
	var parsedConfig configConfig
	var configMap map[string]any
	err := yaml.Unmarshal(configBytes, &configMap)
	if err != nil {
		return configConfig{}, fmt.Errorf("failed to unmarshal config to map: %w", err)
	}
	err = mapToAny(configMap, &parsedConfig)
	if err != nil {
		return configConfig{}, fmt.Errorf("failed to parse config: %w", err)
	}

	// Iterate through fallback field and convert individual elements to strings or fallbackEntryStructConfig
	for i, fallbackElement := range parsedConfig.Fallback {
		switch v := fallbackElement.(type) {
		case string:
			parsedConfig.Fallback[i] = v
		case map[string]any:
			var fallbackEntry fallbackEntryStructConfig
			err := mapToAny(v, &fallbackEntry)
			if err != nil {
				return configConfig{}, fmt.Errorf("failed to parse fallback config: %w", err)
			}
			parsedConfig.Fallback[i] = fallbackEntry
		default:
			return configConfig{}, fmt.Errorf("unknown fallback type: %v", v)
		}
	}

	return parsedConfig, nil
}

// Takes a (potentially very long) psiphon config and outputs
// a short signature string for logging identification purposes
// with only the PropagationChannelId and SponsorId (required fields)
// ex: {PropagationChannelId: FFFFFFFFFFFFFFFF, SponsorId: FFFFFFFFFFFFFFFF}
// If the config does not contains these fields
// output the whole config as a string
func (f *StrategyFinder) getPsiphonConfigSignature(psiphonJSON []byte) string {
	var psiphonConfig map[string]any
	if err := json.Unmarshal(psiphonJSON, &psiphonConfig); err != nil {
		return string(psiphonJSON)
	}

	propagationChannelId, ok1 := psiphonConfig["PropagationChannelId"].(string)
	sponsorId, ok2 := psiphonConfig["SponsorId"].(string)

	if ok1 && ok2 {
		return fmt.Sprintf("{PropagationChannelId: %v, SponsorId: %v}", propagationChannelId, sponsorId)
	}
	return string(psiphonJSON)
}

type smartResolver struct {
	dns.Resolver
	ID     string
	Secure bool
}

func (f *StrategyFinder) dnsConfigToResolver(dnsConfig []dnsEntryConfig) ([]*smartResolver, error) {
	if len(dnsConfig) == 0 {
		return nil, errors.New("no DNS config entry")
	}
	rts := make([]*smartResolver, 0, len(dnsConfig))
	for ei, entry := range dnsConfig {
		idBytes, err := yaml.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("cannot serialize entry %v: %w", ei, err)
		}
		id := string(idBytes)
		resolver, isSecure, err := f.ResolverFactory.NewResolver(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to process entry %v: %w", ei, err)
		}
		rts = append(rts, &smartResolver{Resolver: resolver, ID: id, Secure: isSecure})
	}
	return rts, nil
}

func (f *StrategyFinder) testDialer(ctx context.Context, dialer transport.StreamDialer, testDomains []string, transportCfg string) error {
	return f.TestRunner.TestDialer(ctx, dialer, testDomains, transportCfg)
}

func (f *StrategyFinder) findDNS(ctx context.Context, testDomains []string, dnsConfig []dnsEntryConfig) (dns.Resolver, error) {
	resolvers, err := f.dnsConfigToResolver(dnsConfig)
	if err != nil {
		return nil, err
	}

	ctx, searchDone := context.WithCancel(ctx)
	defer searchDone()
	raceStart := time.Now()
	resolver, err := raceTests(ctx, 250*time.Millisecond, resolvers, func(resolver *smartResolver) (*smartResolver, error) {
		for _, testDomain := range testDomains {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			f.LogWriter.logCtx(ctx, "🏃 run DNS: %v (domain: %v)\n", resolver.ID, testDomain)
			startTime := time.Now()
			ips, err := testDNSResolver(ctx, f.TestTimeout, resolver, testDomain)
			duration := time.Since(startTime)

			status := "ok ✅"
			if err != nil {
				status = fmt.Sprintf("%v ❌", err)
			}
			// Only output log if the search is not done yet.
			f.LogWriter.logCtx(ctx, "🏁 got DNS: %v (domain: %v), duration=%v, ips=%v, status=%v\n", resolver.ID, testDomain, duration, ips, status)

			if err != nil {
				return nil, err
			}
		}
		return resolver, nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not find working resolver: %w", err)
	}
	f.LogWriter.logCtx(ctx, "🏆 selected DNS resolver %v in %0.2fs\n\n", resolver.ID, time.Since(raceStart).Seconds())
	return resolver.Resolver, nil
}

func (f *StrategyFinder) findTLS(ctx context.Context, testDomains []string, baseDialer transport.StreamDialer, tlsConfig []string) (transport.StreamDialer, error) {
	if len(tlsConfig) == 0 {
		return nil, errors.New("config for TLS is empty. Please specify at least one transport")
	}

	ctx, searchDone := context.WithCancel(ctx)
	defer searchDone()
	raceStart := time.Now()
	type SearchResult struct {
		Dialer transport.StreamDialer
		Config string
	}
	result, err := raceTests(ctx, 250*time.Millisecond, tlsConfig, func(transportCfg string) (*SearchResult, error) {
		tlsDialer, err := f.DialerFactory.NewStreamDialer(ctx, baseDialer, transportCfg)
		if err != nil {
			return nil, fmt.Errorf("WrapStreamDialer failed: %w", err)
		}

		err = f.testDialer(ctx, tlsDialer, testDomains, transportCfg)
		if err != nil {
			return nil, err
		}

		return &SearchResult{tlsDialer, transportCfg}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not find TLS strategy: %w", err)
	}
	f.LogWriter.logCtx(ctx, "🏆 selected TLS strategy '%v' in %0.2fs\n\n", result.Config, time.Since(raceStart).Seconds())
	tlsDialer := result.Dialer
	return transport.FuncStreamDialer(func(ctx context.Context, raddr string) (transport.StreamConn, error) {
		_, portStr, err := net.SplitHostPort(raddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse address: %w", err)
		}
		portNum, err := net.DefaultResolver.LookupPort(ctx, "tcp", portStr)
		if err != nil {
			return nil, fmt.Errorf("could not resolve port: %w", err)
		}
		selectedDialer := baseDialer
		if portNum == 443 || portNum == 853 {
			selectedDialer = tlsDialer
		}
		return selectedDialer.DialStream(ctx, raddr)
	}), nil
}

// Return the fastest fallback dialer that is able to access all the testDomans
func (f *StrategyFinder) findFallback(ctx context.Context, testDomains []string, fallbackConfigs []fallbackEntryConfig) (transport.StreamDialer, error) {
	if len(fallbackConfigs) == 0 {
		return nil, errors.New("no fallback was specified")
	}

	raceCtx, searchDone := context.WithCancel(ctx)
	defer searchDone()
	raceStart := time.Now()
	type SearchResult struct {
		Dialer transport.StreamDialer
		Config fallbackEntryConfig
	}

	fallback, err := raceTests(raceCtx, 250*time.Millisecond, fallbackConfigs, func(fallbackConfig fallbackEntryConfig) (*SearchResult, error) {
		switch v := fallbackConfig.(type) {
		case string:
			configUrl := v
			dialer, err := f.DialerFactory.NewStreamDialer(raceCtx, nil, configUrl)
			if err != nil {
				return nil, fmt.Errorf("getStreamDialer failed: %w", err)
			}

			err = f.testDialer(raceCtx, dialer, testDomains, configUrl)
			if err != nil {
				return nil, err
			}

			return &SearchResult{dialer, fallbackConfig}, nil
		case fallbackEntryStructConfig:
			if v.Psiphon != nil {
				psiphonCfg := v.Psiphon
				psiphonJSON, err := json.Marshal(psiphonCfg)
				if err != nil {
					f.LogWriter.logCtx(ctx, "Error marshaling to JSON: %v, %v\n", psiphonCfg, err)
				}
				psiphonSignature := f.getPsiphonConfigSignature(psiphonJSON)

				dialer, err := f.PsiphonDialerFactory.NewPsiphonDialer(ctx, psiphonJSON, psiphonSignature)
				if err != nil {
					return nil, fmt.Errorf("getPsiphonDialer failed: %w", err)
				}

				err = f.testDialer(raceCtx, dialer, testDomains, psiphonSignature)
				if err != nil {
					return nil, err
				}

				return &SearchResult{dialer, psiphonSignature}, nil
			} else {
				return nil, fmt.Errorf("unknown fallback type: %v", v)
			}
		default:
			return nil, fmt.Errorf("unknown fallback type: %v\n", v)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("could not find a working fallback: %w", err)
	}
	f.LogWriter.logCtx(ctx, "🏆 selected fallback '%v' in %0.2fs\n\n", fallback.Config, time.Since(raceStart).Seconds())

	return fallback.Dialer, nil
}

// Attempts to create a new Dialer using only proxyless (DNS and TLS) strategies
func (f *StrategyFinder) newProxylessDialer(ctx context.Context, testDomains []string, config configConfig) (transport.StreamDialer, error) {
	resolver, err := f.findDNS(ctx, testDomains, config.DNS)
	if err != nil {
		return nil, err
	}
	var dnsDialer transport.StreamDialer
	if resolver == nil {
		if _, ok := f.StreamDialer.(*transport.TCPDialer); !ok {
			return nil, fmt.Errorf("cannot use system resolver with base dialer of type %T", f.StreamDialer)
		}
		dnsDialer = f.StreamDialer
	} else {
		resolver = newSimpleLRUCacheResolver(resolver, 100)
		dnsDialer, err = dns.NewStreamDialer(resolver, f.StreamDialer)
		if err != nil {
			return nil, fmt.Errorf("dns.NewStreamDialer failed: %w", err)
		}
	}

	if len(config.TLS) == 0 {
		return dnsDialer, nil
	}
	return f.findTLS(ctx, testDomains, dnsDialer, config.TLS)
}

// NewDialer uses the config in configBytes to search for a strategy that unblocks DNS and TLS for all of the testDomains, returning a dialer with the found strategy.
// It returns an error if no strategy was found that unblocks the testDomains.
// The testDomains must be domains with a TLS service running on port 443.
func (f *StrategyFinder) NewDialer(ctx context.Context, testDomains []string, configBytes []byte) (transport.StreamDialer, error) {
	parsedConfig, err := f.ConfigParser.ParseConfig(configBytes)
	if err != nil {
		return nil, err
	}

	// Make domain fully-qualified to prevent confusing domain search.
	testDomains = append(make([]string, 0, len(testDomains)), testDomains...)
	for di, domain := range testDomains {
		testDomains[di] = makeFullyQualified(domain)
	}

	dialer, err := f.newProxylessDialer(ctx, testDomains, parsedConfig)
	if err != nil {
		return f.findFallback(ctx, testDomains, parsedConfig.Fallback)
	}
	return dialer, err
}
