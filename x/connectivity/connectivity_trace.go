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

package connectivity

import (
	"context"
	ctls "crypto/tls"
	"fmt"
	"net"
	"net/http/httptrace"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
	"github.com/Jigsaw-Code/outline-sdk/transport/tls"
	"golang.org/x/net/dns/dnsmessage"
)

type ConnectivityEvents struct {
	ConnectInfo   []*ConnectInfo
	DnsInfo       []*DNSInfo
	SystemDNSInfo []*SystemDNSInfo
}

func NewConnectivityEvents() *ConnectivityEvents {
	return &ConnectivityEvents{
		ConnectInfo:   []*ConnectInfo{},
		DnsInfo:       []*DNSInfo{},
		SystemDNSInfo: []*SystemDNSInfo{},
	}
}

func SetupConnectivityTrace(ctx context.Context) (context.Context, *ConnectivityEvents) {
	events := NewConnectivityEvents()

	t := &dns.DNSClientTrace{
		ResolverSetup: func(resolverType string, network string, addr string) {
			dnsInfo := &DNSInfo{
				ResolverType: resolverType,
				Resolver:     addr,
				Network:      network,
				StartTime:    time.Now(),
			}
			events.DnsInfo = append(events.DnsInfo, dnsInfo)
		},
		QuestionReady: func(question dnsmessage.Question) {
			// Assuming last DNSInfo is related to this event
			if len(events.DnsInfo) > 0 {
				last := events.DnsInfo[len(events.DnsInfo)-1]
				last.Host = question.Name.String()
			}
			fmt.Printf("QuestionReady: DNS query for %s\n", question.Name.String())
		},
		ResponseDone: func(question dnsmessage.Question, msg *dnsmessage.Message, err error) {
			if len(events.DnsInfo) > 0 {
				last := events.DnsInfo[len(events.DnsInfo)-1]
				last.Duration = time.Since(last.StartTime)
				if err != nil {
					last.Error = err
					fmt.Printf("ResponseDone: DNS query for %s failed: %v\n", question.Name.String(), err)
				} else {
					last.RSCodes = append(last.RSCodes, msg.RCode)
					for _, answer := range msg.Answers {
						switch rr := answer.Body.(type) {
						case *dnsmessage.AResource:
							ipv4 := net.IP(rr.A[:])
							last.IPs = append(last.IPs, net.IPAddr{IP: ipv4})
						case *dnsmessage.AAAAResource:
							ipv6 := net.IP(rr.AAAA[:])
							last.IPs = append(last.IPs, net.IPAddr{IP: ipv6})
						}
					}
				}
			}
		},
		WroteDone: func(err error) {
			if err != nil {
				if len(events.DnsInfo) > 0 {
					last := events.DnsInfo[len(events.DnsInfo)-1]
					last.Error = err
					last.ConnError = makeConnectivityError("send", err)
					last.Duration = time.Since(last.StartTime)
				}
			}
		},
		ReadDone: func(err error) {
			if err != nil {
				if len(events.DnsInfo) > 0 {
					last := events.DnsInfo[len(events.DnsInfo)-1]
					last.Error = err
					last.ConnError = makeConnectivityError("receive", err)
					last.Duration = time.Since(last.StartTime)
				}
			}
		},
	}

	// Variables to store the timestamps
	var startTLS time.Time

	ht := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			systemDNS := &SystemDNSInfo{
				Host:      info.Host,
				StartTime: time.Now(),
			}
			events.SystemDNSInfo = append(events.SystemDNSInfo, systemDNS)
			fmt.Printf("DNS start: %+v\n", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if len(events.SystemDNSInfo) > 0 {
				last := events.SystemDNSInfo[len(events.SystemDNSInfo)-1]
				last.Duration = time.Since(last.StartTime)
				last.Error = info.Err
				last.IPs = info.Addrs
			}
			fmt.Printf("DNS done: %v\n", info)
		},
		ConnectStart: func(network, addr string) {
			ip, port, _ := net.SplitHostPort(addr)
			connectInfo := &ConnectInfo{
				StartTime: time.Now(),
				Network:   network,
				IP:        ip,
				Port:      port,
			}
			events.ConnectInfo = append(events.ConnectInfo, connectInfo)
			fmt.Printf("Connect start: %v %v\n", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			if len(events.ConnectInfo) > 0 {
				last := events.ConnectInfo[len(events.ConnectInfo)-1]
				last.Duration = time.Since(last.StartTime)
				last.Error = err
			}
			fmt.Printf("Connect done: %v %v %v\n", network, addr, err)
		},
		WroteHeaderField: func(key string, value []string) {
			fmt.Printf("Wrote header field: %v %v\n", key, value)
		},
		WroteHeaders: func() {
			fmt.Println("Wrote headers")
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			fmt.Printf("Wrote request: %v\n", info)
		},
		GotFirstResponseByte: func() {
			fmt.Println("Got first response byte")
		},
		TLSHandshakeStart: func() {
			startTLS = time.Now()
		},
		TLSHandshakeDone: func(state ctls.ConnectionState, err error) {
			if err != nil {
				fmt.Printf("TLS handshake failed: %v\n", err)
			}
			fmt.Printf("SNI: %v\n", state.ServerName)
			fmt.Printf("TLS version: %v\n", state.Version)
			fmt.Printf("ALPN: %v\n", state.NegotiatedProtocol)
			fmt.Printf("TLS handshake took %v seconds.\n", time.Since(startTLS).Seconds())
		},
	}

	tlsTrace := &tls.TLSClientTrace{
		TLSHandshakeStart: func() {
			fmt.Println("TLS handshake started")
			startTLS = time.Now()
		},
		TLSHandshakeDone: func(state ctls.ConnectionState, err error) {
			if err != nil {
				fmt.Printf("TLS handshake failed: %v\n", err)
			}
			fmt.Printf("SNI: %v\n", state.ServerName)
			fmt.Printf("TLS version: %v\n", state.Version)
			fmt.Printf("ALPN: %v\n", state.NegotiatedProtocol)
			fmt.Printf("TLS handshake took %v seconds.\n", time.Since(startTLS).Seconds())
		},
	}

	socksTrace := &socks5.SOCKS5ClientTrace{
		RequestStarted: func(cmd byte, dstAddr string) {
			fmt.Printf("SOCKS5 request started: cmd: %v address: %v\n", cmd, dstAddr)
		},
		RequestDone: func(network string, bindAddr string, err error) {
			if err != nil {
				fmt.Printf("SOCKS5 request failed! network: %v, bindAddr: %v, error: %v \n", network, bindAddr, err)
			}
			fmt.Printf("SOCKS5 request succeeded! network: %v, bindAddr: %v \n", network, bindAddr)
		},
	}

	ctx = httptrace.WithClientTrace(ctx, ht)
	ctx = dns.WithDNSClientTrace(ctx, t)
	ctx = tls.WithTLSClientTrace(ctx, tlsTrace)
	ctx = socks5.WithSOCKS5ClientTrace(ctx, socksTrace)
	return ctx, events
}
