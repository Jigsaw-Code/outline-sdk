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

package dnsextra

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/dns/dnsmessage"
)

type resolverStreamDialer struct {
	resolver dns.Resolver
	dialer   transport.StreamDialer
}

var _ transport.StreamDialer = (*resolverStreamDialer)(nil)

// Returns a [context.Context] that is already done.
func newDoneContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func (d *resolverStreamDialer) lookupIPv4(ctx context.Context, domain string) ([]net.IP, error) {
	ips := []net.IP{}
	q, err := dns.NewQuestion(domain, dnsmessage.TypeA)
	if err != nil {
		return nil, err
	}
	response, err := d.resolver.Query(ctx, *q)
	if err != nil {
		return nil, err
	}
	if response.RCode != dnsmessage.RCodeSuccess {
		return nil, fmt.Errorf("got %v (%d)", response.RCode.String(), response.RCode)
	}
	for _, answer := range response.Answers {
		if answer.Header.Type != dnsmessage.TypeA {
			continue
		}
		if rr, ok := answer.Body.(*dnsmessage.AResource); ok {
			ips = append(ips, net.IP(rr.A[:]))
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("no ips found")
	}
	return ips, nil
}

// MakeFullyQualified makes the domain fully-qualified, ending on a dot (".").
// This is useful in domain resolution to avoid ambiguity with local domains
// and domain search.
func MakeFullyQualified(domain string) string {
	if len(domain) > 0 && domain[len(domain)-1] == '.' {
		return domain
	}
	return domain + "."
}

func (d *resolverStreamDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %w", err)
	}
	var ips []net.IP
	ip := net.ParseIP(host)
	if ip != nil {
		ips = []net.IP{ip}
	} else {
		// TODO: Implement standard Happy Eyeballs v2.
		// Need to properly sort addresses.
		// We don't do domain search.
		fqdn := MakeFullyQualified(host)
		ips, err = d.lookupIPv4(ctx, fqdn)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup IPv4 ips: %w", err)
		}
	}
	type dialResult struct {
		Conn transport.StreamConn
		Err  error
	}
	// Communicates the result of each dial.
	resultChan := make(chan dialResult)
	// Indicates to attempts that the search is done, so they don't get stuck.
	searchCtx, searchDone := context.WithCancel(context.Background())
	defer searchDone()
	// Used to space out each attempt. The initial value is done because there's no wait needed.
	waitCtx := newDoneContext()
	// Next entry to start dialing.
	next := 0
	// How many connection attempts are not done.
	toTry := len(ips)
	var dialErr error
	for toTry > 0 {
		if next == len(ips) {
			waitCtx = searchCtx
		}
		select {
		case <-waitCtx.Done():
			// Start a new attempt.
			ip := ips[next]
			next++
			var waitDone context.CancelFunc
			waitCtx, waitDone = context.WithTimeout(searchCtx, 250*time.Millisecond)
			go func(ip net.IP, waitDone context.CancelFunc) {
				defer waitDone()
				conn, err := d.dialer.DialStream(ctx, net.JoinHostPort(ip.String(), port))
				select {
				case <-searchCtx.Done():
					if conn != nil {
						conn.Close()
					}
				case resultChan <- dialResult{Conn: conn, Err: err}:
				}
			}(ip, waitDone)

		case result := <-resultChan:
			toTry--
			if result.Err != nil {
				dialErr = errors.Join(dialErr, result.Err)
				continue
			}
			return result.Conn, nil
		}
	}
	return nil, dialErr
}

func NewStreamDialer(resolver dns.Resolver, dialer transport.StreamDialer) transport.StreamDialer {
	return &resolverStreamDialer{resolver, dialer}
}
