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

// Package smart provides utilities to dynamically find serverless strategies for circumvention.
package smart

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"time"
	"unicode"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"golang.org/x/net/dns/dnsmessage"
)

// To test one strategy:
// go run ./x/examples/smart-proxy -v -localAddr=localhost:1080 --transport="" --domain www.rferl.org  --config=<(echo '{"dns": [{"https": {"name": "doh.sb"}}]}')

// mixCase randomizes the case of the domain letters.
func mixCase(domain string) string {
	var mixed []rune
	for _, r := range domain {
		if rand.Intn(2) == 0 {
			mixed = append(mixed, unicode.ToLower(r))
		} else {
			mixed = append(mixed, unicode.ToUpper(r))
		}
	}
	return string(mixed)
}

func evaluateNetResolver(ctx context.Context, resolver *net.Resolver, testDomain string) ([]net.IP, error) {
	requestDomain := mixCase(testDomain)
	_, err := lookupCNAME(ctx, requestDomain)
	if err != nil {
		return nil, fmt.Errorf("could not get cname: %w", err)
	}
	ips, err := resolver.LookupIP(ctx, "ip", requestDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup IPs: %w", err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no ip answer")
	}
	for _, ip := range ips {
		if ip.IsLoopback() {
			return nil, fmt.Errorf("localhost ip: %v", ip) // -1
		}
		if ip.IsPrivate() {
			return nil, fmt.Errorf("private ip: %v", ip) // -1
		}
		if ip.IsUnspecified() {
			return nil, fmt.Errorf("zero ip: %v", ip) // -1
		}
		// TODO: consider validating the IPs: fingerprint, hardcoded ground truth, trusted response, TLS connection.
	}
	return ips, nil
}

func getIPs(answers []dnsmessage.Resource) []net.IP {
	var ips []net.IP
	for _, answer := range answers {
		if answer.Header.Type != dnsmessage.TypeA && answer.Header.Type != dnsmessage.TypeAAAA {
			continue
		}
		var ip net.IP
		switch rr := answer.Body.(type) {
		case *dnsmessage.AResource:
			ip = net.IP(rr.A[:])
		case *dnsmessage.AAAAResource:
			ip = net.IP(rr.AAAA[:])
		default:
			continue
		}
		ips = append(ips, ip)
	}
	return ips
}

func evaluateAddressResponse(response dnsmessage.Message, requestDomain string) ([]net.IP, error) {
	if response.RCode != dnsmessage.RCodeSuccess {
		return nil, fmt.Errorf("rcode is not success: %v", response.RCode)
	}
	if len(response.Answers) == 0 {
		return nil, errors.New("no answers")
	}
	ips := getIPs(response.Answers)
	if len(ips) == 0 {
		return ips, fmt.Errorf("no ip answer: %v", response.Answers)
	}
	for _, ip := range ips {
		if ip.IsLoopback() {
			return nil, fmt.Errorf("localhost ip: %v", ip)
		}
		if ip.IsPrivate() {
			return nil, fmt.Errorf("private ip: %v", ip)
		}
		if ip.IsUnspecified() {
			return nil, fmt.Errorf("zero ip: %v", ip)
		}
	}
	// All popular recursive resolvers we tested maintain the domain case of the request.
	// Note that this is not the case of authoritative resolvers. Some of them will return
	// a fully normalized domain name, or normalize part of it.
	if response.Answers[0].Header.Name.String() != requestDomain {
		return ips, fmt.Errorf("domain mismatch: got %v, expected %v", response.Answers[0].Header.Name, requestDomain)
	}
	return ips, nil
}

func evaluateCNAMEResponse(response dnsmessage.Message, requestDomain string) error {
	if response.RCode != dnsmessage.RCodeSuccess {
		return fmt.Errorf("rcode is not success: %v", response.RCode)
	}
	if len(response.Answers) == 0 {
		var numSOA int
		for _, answer := range response.Authorities {
			if _, ok := answer.Body.(*dnsmessage.SOAResource); ok {
				numSOA++
			}
		}
		if numSOA != 1 {
			return fmt.Errorf("SOA records is %v, expected 1", numSOA)
		}
		return nil
	}
	var cname string
	for _, answer := range response.Answers {
		if answer.Header.Type != dnsmessage.TypeCNAME {
			return fmt.Errorf("bad answer type: %v", answer.Header.Type)
		}
		if rr, ok := answer.Body.(*dnsmessage.CNAMEResource); ok {
			if cname != "" {
				return fmt.Errorf("found too many CNAMEs: %v %v", cname, rr.CNAME)
			}
			cname = rr.CNAME.String()
		}
	}
	if cname == "" {
		return fmt.Errorf("no CNAME in answers")
	}
	return nil
}

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

func (f *StrategyFinder) testDNSResolver(baseCtx context.Context, resolver *smartResolver, testDomain string) ([]net.IP, error) {
	// We special case the system resolver, since we can't get a dns.RoundTripper.
	if resolver.Resolver == nil {
		ctx, cancel := context.WithTimeout(baseCtx, f.TestTimeout)
		defer cancel()
		return evaluateNetResolver(ctx, new(net.Resolver), testDomain)
	}

	requestDomain := mixCase(testDomain)

	q, err := dns.NewQuestion(requestDomain, dnsmessage.TypeA)
	if err != nil {
		return nil, fmt.Errorf("failed to create question: %v", err)
	}
	ctxA, cancelA := context.WithTimeout(baseCtx, f.TestTimeout)
	defer cancelA()
	response, err := resolver.Query(ctxA, *q)
	if err != nil {
		return nil, fmt.Errorf("request for A query failed: %w", err)
	}

	if resolver.Secure {
		// For secure DNS, we just need to check if we can communicate with it.
		// No need to analyze content, since it is protected by TLS.
		return getIPs(response.Answers), nil
	}

	ips, err := evaluateAddressResponse(*response, requestDomain)
	if err != nil {
		return ips, fmt.Errorf("failed A test: %w", err)
	}

	// TODO(fortuna): Consider testing whether we can establish a TCP connection to ip:443.

	// Run CNAME test, which helps in case the resolver returns a public IP, as is the
	// case in China.
	q, err = dns.NewQuestion(requestDomain, dnsmessage.TypeCNAME)
	if err != nil {
		return nil, fmt.Errorf("failed to create question: %v", err)
	}
	ctxCNAME, cancelCNAME := context.WithTimeout(baseCtx, f.TestTimeout)
	defer cancelCNAME()
	response, err = resolver.Query(ctxCNAME, *q)
	if err != nil {
		return nil, fmt.Errorf("request for CNAME query failed: %w", err)
	}
	err = evaluateCNAMEResponse(*response, requestDomain)
	if err != nil {
		return nil, fmt.Errorf("failed CNAME test: %w", err)
	}
	return ips, nil
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

// newDNSResolverFromEntry creates a [dns.Resolver] based on the config, returning the resolver
// a boolean indicating whether the resolver is secure (TLS, HTTPS) and a possible error.
func (f *StrategyFinder) newDNSResolverFromEntry(entry dnsEntryJSON) (dns.Resolver, bool, error) {
	if entry.System != nil {
		return nil, false, nil
	} else if cfg := entry.HTTPS; cfg != nil {
		if cfg.Name == "" {
			return nil, true, fmt.Errorf("https entry has empty server name")
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
			return nil, true, fmt.Errorf("tls entry has empty server name")
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
			return nil, false, fmt.Errorf("tcp entry has empty server address")
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
			return nil, false, fmt.Errorf("udp entry has empty server address")
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

// Returns a [context.Context] that is already done.
func newDoneContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func (f *StrategyFinder) findDNS(testDomains []string, dnsConfig []dnsEntryJSON) (dns.Resolver, error) {
	resolvers, err := f.dnsConfigToResolver(dnsConfig)
	if err != nil {
		return nil, err
	}
	type testResult struct {
		Resolver *smartResolver
		Err      error
	}
	// Communicates the result of each test.
	resultChan := make(chan testResult)
	// Indicates to tests that the search is done, so they don't get stuck writing to the results channel that will no longer be read.
	searchCtx, searchDone := context.WithCancel(context.Background())
	defer searchDone()
	// Used to space out each test. The initial value is done because there's no wait needed.
	waitCtx := newDoneContext()
	// Next entry to start testing.
	nextResolver := 0
	// How many test entries are not done.
	resolversToTest := len(resolvers)
	for resolversToTest > 0 {
		if nextResolver == len(resolvers) {
			// No more tests to start. Make sure the select doesn't trigger on waitCtx.
			waitCtx = searchCtx
		}
		select {
		case <-waitCtx.Done():
			// Start a new test.
			entry := resolvers[nextResolver]
			nextResolver++
			var waitDone context.CancelFunc
			waitCtx, waitDone = context.WithTimeout(searchCtx, 250*time.Millisecond)
			go func(resolver *smartResolver, testDone context.CancelFunc) {
				defer testDone()
				for _, testDomain := range testDomains {
					select {
					case <-searchCtx.Done():
						return
					default:
					}
					f.log("🏃 run dns: %v (domain: %v)\n", resolver.ID, testDomain)
					startTime := time.Now()
					ips, err := f.testDNSResolver(searchCtx, resolver, testDomain)
					duration := time.Since(startTime)
					status := "ok ✅"
					if err != nil {
						status = fmt.Sprintf("%v ❌", err)
					}
					f.log("🏁 got dns: %v (domain: %v), duration=%v, ips=%v, status=%v\n", resolver.ID, testDomain, duration, ips, status)
					if err != nil {
						select {
						case <-searchCtx.Done():
							return
						case resultChan <- testResult{Resolver: resolver, Err: err}:
							return
						}
					}
				}
				select {
				case <-searchCtx.Done():
				case resultChan <- testResult{Resolver: resolver, Err: nil}:
				}
			}(entry, waitDone)

		case result := <-resultChan:
			resolversToTest--
			// Process the result of a test.
			if result.Err != nil {
				continue
			}
			f.log("✅ selected resolver %v\n", result.Resolver.ID)
			// Tested all domains on this resolver. Unwrap and return.
			return result.Resolver.Resolver, nil
		}
	}
	return nil, errors.New("could not find working resolver")
}

func (f *StrategyFinder) findTLS(testDomains []string, baseDialer transport.StreamDialer, tlsConfig []string) (transport.StreamDialer, error) {
	if len(tlsConfig) == 0 {
		return nil, errors.New("config for TLS is empty. Please specify at least one transport")
	}
	for _, transportCfg := range tlsConfig {
		for di, testDomain := range testDomains {
			testAddr := net.JoinHostPort(testDomain, "443")
			f.log("  tls=%v (domain: %v)", transportCfg, testDomain)

			tlsDialer, err := config.WrapStreamDialer(baseDialer, transportCfg)
			if err != nil {
				f.log("; wrap_error=%v ❌\n", err)
				break
			}
			ctx, cancel := context.WithTimeout(context.Background(), f.TestTimeout)
			defer cancel()
			testConn, err := tlsDialer.DialStream(ctx, testAddr)
			if err != nil {
				f.log("; dial_error=%v ❌\n", err)
				break
			}
			tlsConn := tls.Client(testConn, &tls.Config{ServerName: testDomain})
			err = tlsConn.HandshakeContext(ctx)
			tlsConn.Close()
			if err != nil {
				f.log("; handshake=%v ❌\n", err)
				break
			}
			f.log("; status=ok ✅\n")
			if di+1 < len(testDomains) {
				// More domains to test
				continue
			}
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
	}
	return nil, errors.New("could not find TLS strategy")
}

// makeFullyQualified makes the domain fully-qualified, ending on a dot (".").
// This is useful in domain resolution to avoid ambiguity with local domains
// and domain search.
func makeFullyQualified(domain string) string {
	if len(domain) > 0 && domain[len(domain)-1] == '.' {
		return domain
	}
	return domain + "."
}

// NewDialer uses the config in configBytes to search for a strategy that unblocks all of the testDomains, returning a dialer with the found strategy.
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

	resolver, err := f.findDNS(testDomains, parsedConfig.DNS)
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
		resolver = newCacheResolver(resolver, 100)
		dnsDialer, err = dns.NewStreamDialer(resolver, f.StreamDialer)
		if err != nil {
			return nil, fmt.Errorf("dns.NewStreamDialer failed: %w", err)
		}
	}

	if len(parsedConfig.TLS) == 0 {
		return dnsDialer, nil
	}
	return f.findTLS(testDomains, dnsDialer, parsedConfig.TLS)
}
