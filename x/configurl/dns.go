// Copyright 2024 The Outline Authors
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

package configurl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/dns/dnsmessage"
)

func wrapStreamDialerWithDO53(innerSD func() (transport.StreamDialer, error), innerPD func() (transport.PacketDialer, error), configURL *url.URL) (transport.StreamDialer, error) {
	sd, err := innerSD()
	if err != nil {
		return nil, err
	}
	pd, err := innerPD()
	if err != nil {
		return nil, err
	}
	query := configURL.Opaque
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	var address string
	for key, values := range values {
		switch strings.ToLower(key) {
		case "address":
			if len(values) != 1 {
				return nil, fmt.Errorf("address option must has one value, found %v", len(values))
			}
			address = values[0]
		default:
			return nil, fmt.Errorf("unsupported option %v", key)

		}
	}
	if address == "" {
		return nil, errors.New("must set an address")
	}
	_, _, err = net.SplitHostPort(address)
	if err != nil {
		address = net.JoinHostPort(address, "53")
	}
	udpResolver := dns.NewUDPResolver(pd, address)
	tcpResolver := dns.NewTCPResolver(sd, address)
	resolver := dns.FuncResolver(func(ctx context.Context, q dnsmessage.Question) (*dnsmessage.Message, error) {
		msg, err := udpResolver.Query(ctx, q)
		if err != nil {
			return nil, err
		}
		if !msg.Header.Truncated {
			return msg, nil
		}
		// If the message is truncated, retry over TCP.
		// See https://datatracker.ietf.org/doc/html/rfc1123#page-75.
		return tcpResolver.Query(ctx, q)
	})
	return dns.NewStreamDialer(resolver, sd)
}

func wrapStreamDialerWithDOH(innerSD func() (transport.StreamDialer, error), innerPD func() (transport.PacketDialer, error), configURL *url.URL) (transport.StreamDialer, error) {
	query := configURL.Opaque
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, err
	}
	sd, err := innerSD()
	if err != nil {
		return nil, err
	}

	var name, address string
	for key, values := range values {
		switch strings.ToLower(key) {
		case "address":
			if len(values) != 1 {
				return nil, fmt.Errorf("address option must has one value, found %v", len(values))
			}
			address = values[0]
		case "name":
			if len(values) != 1 {
				return nil, fmt.Errorf("name option must has one value, found %v", len(values))
			}
			name = values[0]
		default:
			return nil, fmt.Errorf("unsupported option %v", key)

		}
	}
	if name == "" {
		return nil, errors.New("must set a name")
	}
	if address == "" {
		address = name
	}
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		address = net.JoinHostPort(address, "443")
		port = "443"
	}
	dohURL := url.URL{Scheme: "https", Host: net.JoinHostPort(name, port), Path: "/dns-query"}
	resolver := dns.NewHTTPSResolver(sd, address, dohURL.String())
	return dns.NewStreamDialer(resolver, sd)
}
