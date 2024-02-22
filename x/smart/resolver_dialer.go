// Copyright 2024 Jigsaw Operations LLC
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
	"errors"
	"fmt"
	"net/netip"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/dns/dnsmessage"
)

func resolveIP(ctx context.Context, resolver dns.Resolver, rrType dnsmessage.Type, hostname string) ([]netip.Addr, error) {
	ips := []netip.Addr{}
	q, err := dns.NewQuestion(hostname, rrType)
	if err != nil {
		return nil, err
	}
	response, err := resolver.Query(ctx, *q)
	if err != nil {
		return nil, err
	}
	if response.RCode != dnsmessage.RCodeSuccess {
		return nil, fmt.Errorf("got %v (%d)", response.RCode.String(), response.RCode)
	}
	for _, answer := range response.Answers {
		if answer.Header.Type != rrType {
			continue
		}
		if rr, ok := answer.Body.(*dnsmessage.AResource); ok {
			ips = append(ips, netip.AddrFrom4(rr.A))
		}
		if rr, ok := answer.Body.(*dnsmessage.AAAAResource); ok {
			ips = append(ips, netip.AddrFrom16(rr.AAAA))
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("no ips found")
	}
	return ips, nil
}

func newResolverStreamDialer(resolver dns.Resolver, dialer transport.StreamDialer) transport.StreamDialer {
	return &transport.HappyEyeballsStreamDialer{
		Dialer: dialer,
		Resolve: transport.NewParallelHappyEyeballsResolveFunc(
			func(ctx context.Context, hostname string) ([]netip.Addr, error) {
				return resolveIP(ctx, resolver, dnsmessage.TypeAAAA, hostname)
			},
			func(ctx context.Context, hostname string) ([]netip.Addr, error) {
				return resolveIP(ctx, resolver, dnsmessage.TypeA, hostname)
			},
		),
	}
}
