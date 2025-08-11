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
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/configurl"
	"github.com/goccy/go-yaml"
)

// To test one strategy:
// go run -C ./x/examples/smart-proxy/ . -v -localAddr=localhost:1080 --transport="" --domain www.rferl.org  --config=<(echo '{"dns": [{"https": {"name": "doh.sb"}}]}')

type StrategyFinder struct {
	TestTimeout  time.Duration
	LogWriter    io.Writer
	StreamDialer transport.StreamDialer
	PacketDialer transport.PacketDialer
	Cache        StrategyResultCache
	logMu        sync.Mutex
}

func (f *StrategyFinder) log(format string, a ...any) {
	if f.LogWriter != nil {
		f.logMu.Lock()
		defer f.logMu.Unlock()
		fmt.Fprintf(f.LogWriter, format, a...)
	}
}

// Only log if context is not done
func (f *StrategyFinder) logCtx(ctx context.Context, format string, a ...any) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	f.log(format, a...)
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
	Psiphon any `yaml:"psiphon,omitempty"`
	// As we allow more fallback types beyond psiphon they will be added here
}

// This contains either a configURL string or a fallbackEntryStructConfig
// It is parsed into the correct type later
type fallbackEntryConfig any

type configConfig struct {
	DNS      []dnsEntryConfig      `yaml:"dns,omitempty"`
	TLS      []string              `yaml:"tls,omitempty"`
	Fallback []fallbackEntryConfig `yaml:"fallback,omitempty"`
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

// newDNSResolverFromEntry creates a [dns.Resolver] based on the config, returning the resolver and
// a boolean indicating whether the resolver is secure (TLS, HTTPS) and a possible error.
func (f *StrategyFinder) newDNSResolverFromEntry(entry dnsEntryConfig) (dns.Resolver, bool, error) {
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

// Takes a (potentially very long) psiphon config and outputs
// a short signature string for logging identification purposes
// with only the PropagationChannelId and SponsorId (required fields)
// ex: {PropagationChannelId: FFFFFFFFFFFFFFFF, SponsorId: FFFFFFFFFFFFFFFF, [...]}
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
		return fmt.Sprintf("Psiphon: {PropagationChannelId: %v, SponsorId: %v, [...]}", propagationChannelId, sponsorId)
	}
	return string(psiphonJSON)
}

type smartResolver struct {
	dns.Resolver
	ID     string
	Secure bool
	Config dnsEntryConfig
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
		resolver, isSecure, err := f.newDNSResolverFromEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to process entry %v: %w", ei, err)
		}
		rts = append(rts, &smartResolver{Resolver: resolver, ID: id, Secure: isSecure, Config: entry})
	}
	return rts, nil
}

// Test that a dialer is able to access all the given test domains. Returns nil if all tests succeed
func (f *StrategyFinder) testDialer(ctx context.Context, dialer transport.StreamDialer, testDomains []string, transportCfg string) error {
	for _, testDomain := range testDomains {
		startTime := time.Now()

		testAddr := net.JoinHostPort(testDomain, "443")
		f.logCtx(ctx, "🏃 running test: '%v' (domain: %v)\n", transportCfg, testDomain)

		testCtx, cancel := context.WithTimeout(ctx, f.TestTimeout)
		defer cancel()

		// Dial

		testConn, err := dialer.DialStream(testCtx, testAddr)
		if err != nil {
			f.logCtx(ctx, "🏁 failed to dial: '%v' (domain: %v), duration=%v, dial_error=%v ❌\n", transportCfg, testDomain, time.Since(startTime), err)
			return err
		}

		// TLS Connection

		tlsConn := tls.Client(testConn, &tls.Config{ServerName: testDomain})
		defer tlsConn.Close()
		err = tlsConn.HandshakeContext(testCtx)
		if err != nil {
			f.logCtx(ctx, "🏁 failed TLS handshake: '%v' (domain: %v), duration=%v, handshake=%v ❌\n", transportCfg, testDomain, time.Since(startTime), err)
			return err
		}

		// HTTPS Get

		req, err := http.NewRequestWithContext(testCtx, http.MethodHead, "https://"+testDomain, nil)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		if err := req.Write(tlsConn); err != nil {
			f.logCtx(ctx, "🏁 failed to write HTTP request: '%v' (domain: %v), duration=%v, error=%v ❌\n", transportCfg, testDomain, time.Since(startTime), err)
			return err
		}

		resp, err := http.ReadResponse(bufio.NewReader(tlsConn), req)
		if err != nil {
			f.logCtx(ctx, "🏁 failed to read HTTP response: '%v' (domain: %v), duration=%v, error=%v ❌\n", transportCfg, testDomain, time.Since(startTime), err)
			return err
		}
		defer resp.Body.Close()

		// Many bare domains return i.e. 301 redirects, so we don't validate anything about the response here, just that the request succeeded.

		f.logCtx(ctx, "🏁 success: '%v' (domain: %v), duration=%v, status=ok ✅\n", transportCfg, testDomain, time.Since(startTime))
	}
	return nil
}

func (f *StrategyFinder) findDNS(ctx context.Context, testDomains []string, dnsConfig []dnsEntryConfig) (dns.Resolver, *dnsEntryConfig, error) {
	resolvers, err := f.dnsConfigToResolver(dnsConfig)
	if err != nil {
		return nil, nil, err
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

			f.logCtx(ctx, "🏃 run DNS: %v (domain: %v)\n", resolver.ID, testDomain)
			startTime := time.Now()
			ips, err := testDNSResolver(ctx, f.TestTimeout, resolver, testDomain)
			duration := time.Since(startTime)

			status := "ok ✅"
			if err != nil {
				status = fmt.Sprintf("%v ❌", err)
			}
			// Only output log if the search is not done yet.
			f.logCtx(ctx, "🏁 got DNS: %v (domain: %v), duration=%v, ips=%v, status=%v\n", resolver.ID, testDomain, duration, ips, status)

			if err != nil {
				return nil, err
			}
		}
		return resolver, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("could not find working resolver: %w", err)
	}
	f.log("🏆 selected DNS resolver %v in %0.2fs\n\n", resolver.ID, time.Since(raceStart).Seconds())
	return resolver.Resolver, &resolver.Config, nil
}

func (f *StrategyFinder) findTLS(
	ctx context.Context, testDomains []string, baseDialer transport.StreamDialer, tlsConfig []string,
) (transport.StreamDialer, string, error) {
	if len(tlsConfig) == 0 {
		return nil, "", errors.New("config for TLS is empty. Please specify at least one transport")
	}
	var configModule = configurl.NewDefaultProviders()
	configModule.StreamDialers.BaseInstance = baseDialer

	searchCtx, searchDone := context.WithCancel(ctx)
	defer searchDone()
	raceStart := time.Now()
	type SearchResult struct {
		Dialer transport.StreamDialer
		Config string
	}
	result, err := raceTests(searchCtx, 250*time.Millisecond, tlsConfig, func(transportCfg string) (*SearchResult, error) {
		tlsDialer, err := configModule.NewStreamDialer(searchCtx, transportCfg)
		if err != nil {
			f.logCtx(searchCtx, "❌ dialer creation failed: %v, error=%v\n", transportCfg, err)
			return nil, fmt.Errorf("NewStreamDialer failed: %w", err)
		}

		err = f.testDialer(searchCtx, tlsDialer, testDomains, transportCfg)
		if err != nil {
			return nil, err
		}

		return &SearchResult{tlsDialer, transportCfg}, nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("could not find TLS strategy: %w", err)
	}
	f.log("🏆 selected TLS strategy '%v' in %0.2fs\n\n", result.Config, time.Since(raceStart).Seconds())
	tlsDialer := result.Dialer
	return transport.FuncStreamDialer(func(searchCtx context.Context, raddr string) (transport.StreamConn, error) {
		_, portStr, err := net.SplitHostPort(raddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse address: %w", err)
		}
		portNum, err := net.DefaultResolver.LookupPort(searchCtx, "tcp", portStr)
		if err != nil {
			return nil, fmt.Errorf("could not resolve port: %w", err)
		}
		selectedDialer := baseDialer
		if portNum == 443 || portNum == 853 {
			selectedDialer = tlsDialer
		}
		return selectedDialer.DialStream(searchCtx, raddr)
	}), result.Config, nil
}

type SearchResult struct {
	Dialer          transport.StreamDialer
	Config          fallbackEntryConfig
	ConfigSignature string
}

// Make a fallback dialer (either from a configurl or a Psiphon config)
// Returns a stream dialer, config signature, error
// In case of an error the stream dialer can be nil, but the string is always set.
func (f *StrategyFinder) makeDialerFromConfig(ctx context.Context, configModule *configurl.ProviderContainer, fallbackConfig fallbackEntryConfig) (transport.StreamDialer, string, error) {
	switch v := fallbackConfig.(type) {
	case string:
		configUrl := v
		dialer, err := configModule.NewStreamDialer(ctx, configUrl)
		if err != nil {
			return nil, v, fmt.Errorf("getStreamDialer failed: %w", err)
		}
		return dialer, v, nil

	case fallbackEntryStructConfig:
		if v.Psiphon != nil {
			psiphonCfg := v.Psiphon

			psiphonJSON, err := json.Marshal(psiphonCfg)
			if err != nil {
				f.logCtx(ctx, "Error marshaling to JSON: %v, %v\n", psiphonCfg, err)
			}

			psiphonSignature := f.getPsiphonConfigSignature(psiphonJSON)
			dialer, err := newPsiphonDialer(f, ctx, psiphonJSON)
			if err != nil {
				return nil, psiphonSignature, fmt.Errorf("newPsiphonDialer failed: %w", err)
			}
			return dialer, psiphonSignature, nil
		} else {
			return nil, fmt.Sprintf("Unknown Config: %v", fallbackConfig), fmt.Errorf("unknown fallback type: %v", fallbackConfig)
		}
	default:
		return nil, fmt.Sprintf("Unknown Config: %v", fallbackConfig), fmt.Errorf("unknown fallback type: %v", fallbackConfig)
	}
}

// Return the fastest fallback dialer that is able to access all the testDomans
func (f *StrategyFinder) findFallback(
	ctx context.Context, testDomains []string, fallbackConfigs []fallbackEntryConfig,
) (transport.StreamDialer, fallbackEntryConfig, error) {
	if len(fallbackConfigs) == 0 {
		return nil, nil, errors.New("attempted to find fallback but no fallback configuration was specified")
	}

	raceCtx, searchDone := context.WithCancel(ctx)
	defer searchDone()
	raceStart := time.Now()

	configModule := configurl.NewDefaultProviders()

	fallback, err := raceTests(raceCtx, 250*time.Millisecond, fallbackConfigs, func(fallbackConfig fallbackEntryConfig) (*SearchResult, error) {
		dialer, configSignature, err := f.makeDialerFromConfig(raceCtx, configModule, fallbackConfig)
		if err != nil {
			f.logCtx(raceCtx, "❌ Failed to start dialer: %v %v\n", configSignature, err)
			return nil, err
		}

		err = f.testDialer(raceCtx, dialer, testDomains, configSignature)
		if err != nil {
			return nil, err
		}

		return &SearchResult{dialer, fallbackConfig, configSignature}, nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("could not find a working fallback: %w", err)
	}
	f.log("🏆 selected fallback '%v' in %0.2fs\n\n", fallback.ConfigSignature, time.Since(raceStart).Seconds())

	return fallback.Dialer, fallback.Config, nil
}

// Attempts to create a new Dialer using only proxyless (DNS and TLS) strategies
func (f *StrategyFinder) newProxylessDialer(
	ctx context.Context, testDomains []string, config configConfig,
) (transport.StreamDialer, *dnsEntryConfig, string, error) {
	resolver, dnsConfig, err := f.findDNS(ctx, testDomains, config.DNS)
	if err != nil {
		return nil, nil, "", err
	}
	var dnsDialer transport.StreamDialer
	if resolver == nil {
		if _, ok := f.StreamDialer.(*transport.TCPDialer); !ok {
			return nil, nil, "", fmt.Errorf("cannot use system resolver with base dialer of type %T", f.StreamDialer)
		}
		dnsDialer = f.StreamDialer
	} else {
		resolver = newSimpleLRUCacheResolver(resolver, 100)
		dnsDialer, err = dns.NewStreamDialer(resolver, f.StreamDialer)
		if err != nil {
			return nil, nil, "", fmt.Errorf("dns.NewStreamDialer failed: %w", err)
		}
	}

	if len(config.TLS) == 0 {
		return dnsDialer, dnsConfig, "", nil
	}
	sd, tlsConfig, err := f.findTLS(ctx, testDomains, dnsDialer, config.TLS)
	if err != nil {
		return nil, nil, "", err
	}
	return sd, dnsConfig, tlsConfig, err
}

func (f *StrategyFinder) parseConfig(configBytes []byte) (configConfig, error) {
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

// rankStrategiesFromCache reads a winningStrategy from the cache and adjust the input config accordingly.
// It returns the adjusted ranked config, and optionally a first2Try config that the caller should prioritize.
func (f *StrategyFinder) rankStrategiesFromCache(
	input configConfig,
) (ranked configConfig, first2Try fallbackEntryConfig) {
	data, ok := f.Cache.Get(winningStrategyCacheKey)
	if !ok {
		return input, nil
	}

	cachedCfg, err := f.parseConfig(data)
	if err != nil {
		return input, nil
	}

	f.log("💾 resume strategy from cache\n")
	winner := winningConfig(cachedCfg)

	if fbCfg, ok := winner.getFallbackIfExclusive(&input); ok {
		return input, fbCfg
	}
	winner.promoteProxylessToFront(&input)
	return input, nil
}

// NewDialer uses the config in configBytes to search for a strategy that unblocks DNS and TLS for all of the testDomains, returning a dialer with the found strategy.
// It returns an error if no strategy was found that unblocks the testDomains.
// The testDomains must be domains with a TLS service running on port 443.
func (f *StrategyFinder) NewDialer(ctx context.Context, testDomains []string, configBytes []byte) (transport.StreamDialer, error) {
	// Parse the config and make sure it's valid
	inputConfig, err := f.parseConfig(configBytes)
	if err != nil {
		return nil, err
	}

	// Make domain fully-qualified to prevent confusing domain search.
	testDomains = append(make([]string, 0, len(testDomains)), testDomains...)
	for di, domain := range testDomains {
		testDomains[di] = makeFullyQualified(domain)
	}

	// Fast resume the winning strategy from the cache
	if f.Cache != nil {
		rankedConfig, first2Try := f.rankStrategiesFromCache(inputConfig)
		if first2Try != nil {
			if dialer, _, err := f.findFallback(ctx, testDomains, []fallbackEntryConfig{first2Try}); err == nil {
				return dialer, nil
			}
		}
		inputConfig = rankedConfig
	}

	// Find a working strategy and persist it to the cache
	var winner winningConfig
	dialer, dnsConf, tlsConf, err := f.newProxylessDialer(ctx, testDomains, inputConfig)
	if err == nil {
		winner = newProxylessWinningConfig(dnsConf, tlsConf)
	} else if inputConfig.Fallback != nil {
		var fbConf fallbackEntryConfig
		dialer, fbConf, err = f.findFallback(ctx, testDomains, inputConfig.Fallback)
		if err == nil {
			winner = newFallbackWinningConfig(fbConf)
		}
	}

	// Persist the potential winner to cache
	if f.Cache != nil {
		var data []byte = nil
		if err == nil {
			data, err = winner.toYAML()
		}
		f.Cache.Put(winningStrategyCacheKey, data)
		if data != nil {
			f.log("💾 strategy stored to cache\n")
		} else {
			f.log("💾 strategy cache cleared\n")
		}
	}

	return dialer, err
}
