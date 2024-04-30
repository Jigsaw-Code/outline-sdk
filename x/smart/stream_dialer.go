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

package smart

import (
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

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
)

// To test one strategy:
// go run ./x/examples/smart-proxy -v -localAddr=localhost:1080 --transport="" --domain www.rferl.org  --config=<(echo '{"dns": [{"https": {"name": "doh.sb"}}]}')

type StrategyFinder struct {
	TestTimeout  time.Duration
	LogWriter    io.Writer
	StreamDialer transport.StreamDialer
	PacketDialer transport.PacketDialer
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

type httpsEntryJSON struct {
	// Domain name of the host.
	Name string `json:"name,omitempty"`
	// Host:port. Defaults to Name:443.
	Address string `json:"address,omitempty"`
}

type tlsEntryJSON struct {
	// Domain name of the host.
	Name string `json:"name,omitempty"`
	// Host:port. Defaults to Name:853.
	Address string `json:"address,omitempty"`
}

type udpEntryJSON struct {
	// Host:port.
	Address string `json:"address,omitempty"`
}

type tcpEntryJSON struct {
	// Host:port.
	Address string `json:"address,omitempty"`
}

type dnsEntryJSON struct {
	System *struct{}       `json:"system,omitempty"`
	HTTPS  *httpsEntryJSON `json:"https,omitempty"`
	TLS    *tlsEntryJSON   `json:"tls,omitempty"`
	UDP    *udpEntryJSON   `json:"udp,omitempty"`
	TCP    *tcpEntryJSON   `json:"tcp,omitempty"`
}

type configJSON struct {
	DNS []dnsEntryJSON `json:"dns,omitempty"`
	TLS []string       `json:"tls,omitempty"`
}

// newDNSResolverFromEntry creates a [dns.Resolver] based on the config, returning the resolver and
// a boolean indicating whether the resolver is secure (TLS, HTTPS) and a possible error.
func (f *StrategyFinder) newDNSResolverFromEntry(entry dnsEntryJSON) (dns.Resolver, bool, error) {
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

type smartResolver struct {
	dns.Resolver
	ID     string
	Secure bool
}

func (f *StrategyFinder) dnsConfigToResolver(dnsConfig []dnsEntryJSON) ([]*smartResolver, error) {
	if len(dnsConfig) == 0 {
		return nil, errors.New("no DNS config entry")
	}
	rts := make([]*smartResolver, 0, len(dnsConfig))
	for ei, entry := range dnsConfig {
		idBytes, err := json.Marshal(entry)
		if err != nil {
			return nil, fmt.Errorf("cannot serialize entry %v: %w", ei, err)
		}
		id := string(idBytes)
		resolver, isSecure, err := f.newDNSResolverFromEntry(entry)
		if err != nil {
			return nil, fmt.Errorf("failed to process entry %v: %w", ei, err)
		}
		rts = append(rts, &smartResolver{Resolver: resolver, ID: id, Secure: isSecure})
	}
	return rts, nil
}

func (f *StrategyFinder) findDNS(ctx context.Context, testDomains []string, dnsConfig []dnsEntryJSON) (dns.Resolver, error) {
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
		return nil, fmt.Errorf("could not find working resolver: %w", err)
	}
	f.log("🏆 selected DNS resolver %v in %0.2fs\n\n", resolver.ID, time.Since(raceStart).Seconds())
	return resolver.Resolver, nil
}

var configParser = config.NewDefaultConfigParser()

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
		tlsDialer, err := configParser.WrapStreamDialer(baseDialer, transportCfg)
		if err != nil {
			return nil, fmt.Errorf("WrapStreamDialer failed: %w", err)
		}
		for _, testDomain := range testDomains {
			startTime := time.Now()

			testAddr := net.JoinHostPort(testDomain, "443")
			f.logCtx(ctx, "🏃 run TLS: '%v' (domain: %v)\n", transportCfg, testDomain)

			ctx, cancel := context.WithTimeout(ctx, f.TestTimeout)
			defer cancel()
			testConn, err := tlsDialer.DialStream(ctx, testAddr)
			if err != nil {
				f.logCtx(ctx, "🏁 got TLS: '%v' (domain: %v), duration=%v, dial_error=%v ❌\n", transportCfg, testDomain, time.Since(startTime), err)
				return nil, err
			}
			tlsConn := tls.Client(testConn, &tls.Config{ServerName: testDomain})
			err = tlsConn.HandshakeContext(ctx)
			tlsConn.Close()
			if err != nil {
				f.logCtx(ctx, "🏁 got TLS: '%v' (domain: %v), duration=%v, handshake=%v ❌\n", transportCfg, testDomain, time.Since(startTime), err)
				return nil, err
			}
			f.logCtx(ctx, "🏁 got TLS: '%v' (domain: %v), duration=%v, status=ok ✅\n", transportCfg, testDomain, time.Since(startTime))
		}
		return &SearchResult{tlsDialer, transportCfg}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not find TLS strategy: %w", err)
	}
	f.log("🏆 selected TLS strategy '%v' in %0.2fs\n\n", result.Config, time.Since(raceStart).Seconds())
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

// NewDialer uses the config in configBytes to search for a strategy that unblocks DNS and TLS for all of the testDomains, returning a dialer with the found strategy.
// It returns an error if no strategy was found that unblocks the testDomains.
// The testDomains must be domains with a TLS service running on port 443.
func (f *StrategyFinder) NewDialer(ctx context.Context, testDomains []string, configBytes []byte) (transport.StreamDialer, error) {
	var parsedConfig configJSON
	err := json.Unmarshal(configBytes, &parsedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	// Make domain fully-qualified to prevent confusing domain search.
	testDomains = append(make([]string, 0, len(testDomains)), testDomains...)
	for di, domain := range testDomains {
		testDomains[di] = makeFullyQualified(domain)
	}

	resolver, err := f.findDNS(ctx, testDomains, parsedConfig.DNS)
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

	if len(parsedConfig.TLS) == 0 {
		return dnsDialer, nil
	}
	return f.findTLS(ctx, testDomains, dnsDialer, parsedConfig.TLS)
}
