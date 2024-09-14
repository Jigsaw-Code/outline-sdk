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

package main

import (
	"context"
	ctls "crypto/tls"
	"errors"
	"net"
	"net/http/httptrace"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
	"github.com/Jigsaw-Code/outline-sdk/transport/tls"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"
	"golang.org/x/net/dns/dnsmessage"
)

type ConnectivityReport struct {
	TestType           string              `json:"test_type"`
	Transport          string              `json:"transport"`
	ConnectivityEvents *ConnectivityEvents `json:"connectivity_events"`
}

type ConnectivityEvents struct {
	ConnectInfo   []*ConnectInfo   `json:"connect_info,omitempty"`
	DnsInfo       []*DNSInfo       `json:"dns_info,omitempty"`
	SystemDNSInfo []*SystemDNSInfo `json:"system_dns_info,omitempty"`
}

func NewConnectivityEvents() *ConnectivityEvents {
	return &ConnectivityEvents{
		ConnectInfo:   []*ConnectInfo{},
		DnsInfo:       []*DNSInfo{},
		SystemDNSInfo: []*SystemDNSInfo{},
	}
}

type ConnectInfo struct {
	Network   string     `json:"network"`
	IP        string     `json:"ip"`
	Port      string     `json:"port"`
	Error     string     `json:"error,omitempty"`
	StartTime time.Time  `json:"start_time"`
	Duration  int64      `json:"duration_ms"`
	ConnError *errorJSON `json:"conn_error,omitempty"`
}

type DNSInfo struct {
	Host         string     `json:"host"`
	Resolver     string     `json:"resolver"`
	Network      string     `json:"network"`
	ResolverType string     `json:"resolver_type"`
	IPs          []string   `json:"ips"`
	RSCodes      []string   `json:"rs_codes"`
	Error        string     `json:"error,omitempty"`
	StartTime    time.Time  `json:"start_time"`
	Duration     int64      `json:"duration_ms"`
	ConnError    *errorJSON `json:"conn_error,omitempty"`
	TLSInfo      *TLSInfo   `json:"tls_info,omitempty"`
}

type TLSInfo struct {
	ServerName         string    `json:"server_name"`
	TLSversion         uint16    `json:"tls_version"`
	ALPN               string    `json:"alpn"`
	HandshakeStartTime time.Time `json:"handshake_start_time"`
	HandshakeDuration  int64     `json:"handshake_duration_ms"`
	Error              string    `json:"error,omitempty"`
	HandshakeComplete  bool      `json:"handshake_complete"`
}

type SystemDNSInfo struct {
	Host      string    `json:"host"`
	IPs       []string  `json:"ips"`
	Error     string    `json:"error,omitempty"`
	StartTime time.Time `json:"start_time"`
	Duration  int64     `json:"duration_ms"`
}

type HTTPSInfo struct {
	Host      string    `json:"host"`
	Method    string    `json:"method"`
	TLSInfo   *TLSInfo  `json:"tls_info,omitempty"`
	StartTime time.Time `json:"start_time"`
	Duration  int64     `json:"duration_ms"`
	Error     string    `json:"error,omitempty"`
}

type errorJSON struct {
	Op         string `json:"op,omitempty"`
	PosixError string `json:"posix_error,omitempty"`
	Msg        string `json:"msg,omitempty"`
}

func makeErrorRecord(result *connectivity.ConnectivityError) *errorJSON {
	if result == nil {
		return nil
	}
	var record = new(errorJSON)
	record.Op = result.Op
	record.PosixError = result.PosixError
	record.Msg = unwrapAll(result.Err).Error()
	return record
}

func unwrapAll(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// Converts a slice of net.IPAddr to a slice of strings
func ipAddrsToStrings(ipAddrs []net.IPAddr) []string {
	strs := make([]string, len(ipAddrs))
	for i, ipAddr := range ipAddrs {
		strs[i] = ipAddr.IP.String() // Convert net.IPAddr to string
	}
	return strs
}

func SetupConnectivityTrace(ctx context.Context) (context.Context, *ConnectivityEvents) {
	events := NewConnectivityEvents()

	t := &dns.DNSClientTrace{
		ResolverSetup: func(resolverType string, network string, addr string) {
			// Create a new DNSInfo event and add it to the list
			dnsInfo := &DNSInfo{
				ResolverType: resolverType,
				Resolver:     addr,
				Network:      network,
				StartTime:    time.Now(),
			}
			events.DnsInfo = append(events.DnsInfo, dnsInfo)
		},
		QuestionReady: func(question dnsmessage.Question) {
			// Find the last DNSInfo event without a host and set the host
			for i := len(events.DnsInfo) - 1; i >= 0; i-- {
				latest := events.DnsInfo[i]
				if latest.Host == "" {
					//fmt.Printf("host: %v\n", question.Name.String())
					latest.Host = question.Name.String()
					break
				} else {
					//fmt.Printf("host already set %v\n", latest.Host)
				}
			}
			//fmt.Printf("QuestionReady: DNS query for %s\n", question.Name.String())
		},
		ResponseDone: func(question dnsmessage.Question, msg *dnsmessage.Message, err error) {
			for i := len(events.DnsInfo) - 1; i >= 0; i-- {
				latest := events.DnsInfo[i]
				// If the host of the last DNSInfo event matches the question,
				// then update the event
				if latest.Host == question.Name.String() {
					latest.Duration = time.Since(latest.StartTime).Milliseconds()
					if err != nil {
						latest.Error = err.Error()
					} else {
						latest.RSCodes = append(latest.RSCodes, msg.RCode.String())
						for _, answer := range msg.Answers {
							switch rr := answer.Body.(type) {
							case *dnsmessage.AResource:
								ipv4 := net.IP(rr.A[:])
								latest.IPs = append(latest.IPs, ipv4.String())
							case *dnsmessage.AAAAResource:
								ipv6 := net.IP(rr.AAAA[:])
								latest.IPs = append(latest.IPs, ipv6.String())
							}
						}
					}
					break
				}
			}
		},
		WroteDone: func(q dnsmessage.Question, err error) {
			for i := len(events.DnsInfo) - 1; i >= 0; i-- {
				latest := events.DnsInfo[i]
				// If the host of the last DNSInfo event matches the question,
				// then update the event
				if latest.Host == q.Name.String() {
					if err != nil {
						latest.Error = err.Error()
						latest.ConnError = makeErrorRecord(connectivity.MakeConnectivityError("send", err))
					}
					latest.Duration = time.Since(latest.StartTime).Milliseconds()
					break
				}
			}
		},
		ReadDone: func(q dnsmessage.Question, err error) {
			for i := len(events.DnsInfo) - 1; i >= 0; i-- {
				// If the host of the last DNSInfo event matches the question,
				// then update the event
				latest := events.DnsInfo[i]
				if latest.Host == q.Name.String() {
					if err != nil {
						latest.Error = err.Error()
						latest.ConnError = makeErrorRecord(connectivity.MakeConnectivityError("receive", err))
					}
					latest.Duration = time.Since(latest.StartTime).Milliseconds()
					break
				}
			}
		},
	}

	ht := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			systemDNS := &SystemDNSInfo{
				Host:      info.Host,
				StartTime: time.Now(),
			}
			events.SystemDNSInfo = append(events.SystemDNSInfo, systemDNS)
			//fmt.Printf("DNS start: %+v\n", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			if len(events.SystemDNSInfo) > 0 {
				last := events.SystemDNSInfo[len(events.SystemDNSInfo)-1]
				last.Duration = time.Since(last.StartTime).Milliseconds()
				if info.Err != nil {
					last.Error = info.Err.Error()
				}
				last.IPs = ipAddrsToStrings(info.Addrs)
			}
			//fmt.Printf("DNS done: %v\n", info)
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
			//fmt.Printf("Connect start: %v %v\n", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			for i := len(events.ConnectInfo) - 1; i >= 0; i-- {
				last := events.ConnectInfo[i]
				ip, port, _ := net.SplitHostPort(addr)
				if last.Network == network && last.IP == ip && last.Port == port {
					events.ConnectInfo[i].Duration = time.Since(last.StartTime).Milliseconds()
					if err != nil {
						events.ConnectInfo[i].Error = err.Error()
					}
				}
			}
			//fmt.Printf("Connect done: %v %v %v\n", network, addr, err)
		},
		WroteHeaderField: func(key string, value []string) {
			//fmt.Printf("Wrote header field: %v %v\n", key, value)
		},
		WroteHeaders: func() {
			//fmt.Println("Wrote headers")
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			//fmt.Printf("Wrote request: %v\n", info)
		},
		GotFirstResponseByte: func() {
			//fmt.Println("Got first response byte")
		},
		TLSHandshakeStart: func() {
			for i := len(events.DnsInfo) - 1; i >= 0; i-- {
				last := events.DnsInfo[i]
				// if there is a DNSInfo event if type dot or doh without a TLSInfo, then create a TLSInfo event
				if last.TLSInfo == nil && (last.ResolverType == "dot" || last.ResolverType == "doh") {
					last.TLSInfo = &TLSInfo{}
					last.TLSInfo.HandshakeStartTime = time.Now()
					break
				}
				// TODO: if there are any open HTTP event, then this the
				// handshake for https connection
				// for i := len(events.HTTPInfo) - 1; i >= 0; i-- {
			}
		},
		TLSHandshakeDone: func(state ctls.ConnectionState, err error) {
			for i := len(events.DnsInfo) - 1; i >= 0; i-- {
				last := events.DnsInfo[i]
				// if there is a DNSInfo event if type dot or doh without a TLSInfo, then create a TLSInfo event
				if last.TLSInfo != nil && (last.ResolverType == "dot" || last.ResolverType == "doh") {
					last.TLSInfo.HandshakeDuration = time.Since(last.TLSInfo.HandshakeStartTime).Milliseconds()
					last.TLSInfo.ServerName = state.ServerName
					last.TLSInfo.TLSversion = state.Version
					last.TLSInfo.ALPN = state.NegotiatedProtocol
					last.TLSInfo.HandshakeComplete = state.HandshakeComplete
					if err != nil {
						last.TLSInfo.Error = err.Error()
					}
					//fmt.Printf("TLS handshake err: %v\n", err)
					break
				}
				// TODO: if there are any open HTTP event, then this the
				// handshake for https connection
				// for i := len(events.HTTPInfo) - 1; i >= 0; i-- {
			}
		},
	}

	tlsTrace := &tls.TLSClientTrace{
		TLSHandshakeStart: func() {
			for i := len(events.DnsInfo) - 1; i >= 0; i-- {
				last := events.DnsInfo[i]
				// if there is a DNSInfo event if type dot or doh without a TLSInfo, then create a TLSInfo event
				if last.TLSInfo == nil && (last.ResolverType == "dot" || last.ResolverType == "doh") {
					last.TLSInfo = &TLSInfo{}
					last.TLSInfo.HandshakeStartTime = time.Now()
					break
				}
			}
			// TODO: Add standalone TLSInfo event for tls: configs
			// such as SOCKs5 Over TLS
		},
		TLSHandshakeDone: func(state ctls.ConnectionState, err error) {
			for i := len(events.DnsInfo) - 1; i >= 0; i-- {
				last := events.DnsInfo[i]
				// if there is a DNSInfo event if type dot or doh without a TLSInfo, then create a TLSInfo event
				if last.TLSInfo != nil && (last.ResolverType == "dot" || last.ResolverType == "doh") {
					last.TLSInfo.HandshakeDuration = time.Since(last.TLSInfo.HandshakeStartTime).Milliseconds()
					last.TLSInfo.ServerName = state.ServerName
					last.TLSInfo.TLSversion = state.Version
					last.TLSInfo.ALPN = state.NegotiatedProtocol
					if err != nil {
						last.TLSInfo.Error = err.Error()
					}
					last.TLSInfo.HandshakeComplete = state.HandshakeComplete
					//fmt.Printf("TLS handshake err: %v\n", err)
					break
				}
			}
		},
	}

	socksTrace := &socks5.SOCKS5ClientTrace{
		RequestStarted: func(cmd byte, dstAddr string) {
			//fmt.Printf("SOCKS5 request started: cmd: %v address: %v\n", cmd, dstAddr)
		},
		RequestDone: func(network string, bindAddr string, err error) {
			// if err != nil {
			// 	fmt.Printf("SOCKS5 request failed! network: %v, bindAddr: %v, error: %v \n", network, bindAddr, err)
			// }
			//fmt.Printf("SOCKS5 request succeeded! network: %v, bindAddr: %v \n", network, bindAddr)
		},
	}

	ctx = httptrace.WithClientTrace(ctx, ht)
	ctx = dns.WithDNSClientTrace(ctx, t)
	ctx = tls.WithTLSClientTrace(ctx, tlsTrace)
	ctx = socks5.WithSOCKS5ClientTrace(ctx, socksTrace)
	return ctx, events
}
