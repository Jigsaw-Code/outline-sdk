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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport/socks5"
	"github.com/Jigsaw-Code/outline-sdk/transport/tls"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"
	"github.com/Jigsaw-Code/outline-sdk/x/report"
	"golang.org/x/net/dns/dnsmessage"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

// var errorLog log.Logger = *log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

type connectivityReport struct {
	// Inputs
	Resolver string `json:"resolver"`
	Proto    string `json:"proto"`
	// TODO(fortuna): add sanitized transport config.
	Transport string `json:"transport"`

	// Observations
	Time       time.Time  `json:"time"`
	DurationMs int64      `json:"duration_ms"`
	Error      *errorJSON `json:"error"`
}

type errorJSON struct {
	// TODO: add Shadowsocks/Transport error
	Op string `json:"op,omitempty"`
	// Posix error, when available
	PosixError string `json:"posix_error,omitempty"`
	// TODO: remove IP addresses
	Msg string `json:"msg,omitempty"`
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

func (r connectivityReport) IsSuccess() bool {
	if r.Error == nil {
		return true
	} else {
		return false
	}
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [flags...]\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	verboseFlag := flag.Bool("v", false, "Enable debug output")
	testTypeFlag := flag.String("test-type", "do53-tcp,do53-udp,doh,dot,http,http3", "Type of test to run")
	transportFlag := flag.String("transport", "", "Transport config")
	domainFlag := flag.String("domain", "example.com.", "Domain name to resolve in the DNS test and to fetch in the HTTP test")
	methodFlag := flag.String("method", "GET", "HTTP method to use in the HTTP test")
	timeoutFlag := flag.Duration("timeout", 10*time.Second, "Timeout for the test; default is 10s")
	resolverFlag := flag.String("resolver", "8.8.8.8,2001:4860:4860::8888", "Comma-separated list of addresses of DNS resolver to use for the test")
	resolverNameFlag := flag.String("resolver-name", "dns.google.com", "Name of the resolver to use for the test")
	reportToFlag := flag.String("report-to", "", "URL to send JSON error reports to")
	reportSuccessFlag := flag.Float64("report-success-rate", 0.1, "Report success to collector with this probability - must be between 0 and 1")
	reportFailureFlag := flag.Float64("report-failure-rate", 1, "Report failure to collector with this probability - must be between 0 and 1")

	flag.Parse()

	// Perform custom range validation for sampling rate
	if *reportSuccessFlag < 0.0 || *reportSuccessFlag > 1.0 {
		fmt.Println("Error: report-success-rate must be between 0 and 1.")
		flag.Usage()
		return
	}

	if *reportFailureFlag < 0.0 || *reportFailureFlag > 1.0 {
		fmt.Println("Error: report-failure-rate must be between 0 and 1.")
		flag.Usage()
		return
	}

	if *verboseFlag {
		debugLog = *log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	}

	var reportCollector report.Collector
	if *reportToFlag != "" {
		collectorURL, err := url.Parse(*reportToFlag)
		if err != nil {
			debugLog.Printf("Failed to parse collector URL: %v", err)
		}
		remoteCollector := &report.RemoteCollector{
			CollectorURL: collectorURL,
			HttpClient:   &http.Client{Timeout: 10 * time.Second},
		}
		retryCollector := &report.RetryCollector{
			Collector:    remoteCollector,
			MaxRetry:     3,
			InitialDelay: 1 * time.Second,
		}
		reportCollector = &report.SamplingCollector{
			Collector:       retryCollector,
			SuccessFraction: *reportSuccessFlag,
			FailureFraction: *reportFailureFlag,
		}
	} else {
		reportCollector = &report.WriteCollector{Writer: os.Stdout}
	}

	// Things to test:
	// - TCP working. Where's the error?
	// - UDP working
	// - Different server IPs
	// - Server IPv4 dial support
	// - Server IPv6 dial support

	success := false
	jsonEncoder := json.NewEncoder(os.Stdout)
	jsonEncoder.SetEscapeHTML(false)
	configToDialer := config.NewDefaultConfigToDialer()
	ctx := SetupConnectivityTrace(context.Background())
	for _, resolverHost := range strings.Split(*resolverFlag, ",") {
		resolverHost := strings.TrimSpace(resolverHost)
		var result *connectivity.ConnectivityError
		var resolverAddress string
		for _, testType := range strings.Split(*testTypeFlag, ",") {
			testType = strings.TrimSpace(testType)
			var resolver dns.Resolver
			var Protocol string
			startTime := time.Now()
			switch testType {
			case "do53-tcp":
				Protocol = "tcp"
				streamDialer, err := configToDialer.NewStreamDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				resolverAddress = net.JoinHostPort(resolverHost, "53")
				resolver = dns.NewTCPResolver(streamDialer, resolverAddress)
				result, err = connectivity.TestConnectivityWithResolver(ctx, resolver, *domainFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			case "do53-udp":
				Protocol = "udp"
				packetDialer, err := configToDialer.NewPacketDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create PacketDialer: %v", err)
				}
				resolverAddress = net.JoinHostPort(resolverHost, "53")
				resolver = dns.NewUDPResolver(packetDialer, resolverAddress)
				result, err = connectivity.TestConnectivityWithResolver(ctx, resolver, *domainFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			case "doh":
				Protocol = "tcp"
				streamDialer, err := configToDialer.NewStreamDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				resolverAddress = net.JoinHostPort(resolverHost, "443")
				resolver = dns.NewHTTPSResolver(streamDialer, resolverAddress, "https://"+resolverAddress+"/dns-query")
				result, err = connectivity.TestConnectivityWithResolver(ctx, resolver, *domainFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			case "dot":
				Protocol = "tcp"
				streamDialer, err := configToDialer.NewStreamDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				resolverAddress = net.JoinHostPort(resolverHost, "853")
				resolver = dns.NewTLSResolver(streamDialer, resolverAddress, *resolverNameFlag)
				result, err = connectivity.TestConnectivityWithResolver(ctx, resolver, *domainFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			case "http":
				Protocol = "tcp"
				streamDialer, err := configToDialer.NewStreamDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				result, err = connectivity.TestStreamConnectivitywithHTTP(ctx, streamDialer, *domainFlag, *timeoutFlag, *methodFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			case "http3":
				Protocol = "udp"
				packetDialer, err := configToDialer.NewPacketDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create PacketDialer: %v", err)
				}
				result, err = connectivity.TestPacketConnectivitywithHTTP3(ctx, packetDialer, *domainFlag, *timeoutFlag, *methodFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			default:
				log.Fatalf(`Invalid Test Type %v.`, testType)
			}
			testDuration := time.Since(startTime)
			if result == nil {
				success = true
			}
			debugLog.Printf("Test Type: %v Resolver Address: %v domain: %v result: %v", testType, resolverAddress, *domainFlag, result)
			sanitizedConfig, err := config.SanitizeConfig(*transportFlag)
			if err != nil {
				log.Fatalf("Failed to sanitize config: %v", err)
			}
			var r report.Report = connectivityReport{
				Resolver: resolverAddress,
				Proto:    Protocol, // change this
				Time:     startTime.UTC().Truncate(time.Second),
				// TODO(fortuna): Add sanitized config:
				Transport:  sanitizedConfig,
				DurationMs: testDuration.Milliseconds(),
				Error:      makeErrorRecord(result),
			}
			if reportCollector != nil {
				err = reportCollector.Collect(context.Background(), r)
				if err != nil {
					debugLog.Printf("Failed to collect report: %v\n", err)
				}
			}
		}
		if !success {
			os.Exit(1)
		}
	}
}

func SetupConnectivityTrace(ctx context.Context) context.Context {
	t := &dns.DNSClientTrace{
		QuestionSent: func(question dnsmessage.Question) {
			fmt.Println("DNS query started for", question.Name.String())
		},
		ResponsDone: func(question dnsmessage.Question, msg *dnsmessage.Message, err error) {
			if err != nil {
				fmt.Printf("DNS query for %s failed: %v\n", question.Name.String(), err)
			} else {
				// Prepare to collect IP addresses
				var ips []string

				// Iterate over the answer section
				for _, answer := range msg.Answers {
					switch rr := answer.Body.(type) {
					case *dnsmessage.AResource:
						// Handle IPv4 addresses - convert [4]byte to IP string
						ipv4 := net.IP(rr.A[:]) // Convert [4]byte to net.IP
						ips = append(ips, ipv4.String())
					case *dnsmessage.AAAAResource:
						// Handle IPv6 addresses - convert [16]byte to IP string
						ipv6 := net.IP(rr.AAAA[:]) // Convert [16]byte to net.IP
						ips = append(ips, ipv6.String())
					}
				}

				// Print all resolved IP addresses
				if len(ips) > 0 {
					fmt.Printf("Resolved IPs for %s: %v\n", question.Name.String(), ips)
				} else {
					fmt.Printf("No IPs found for %s\n", question.Name.String())
				}
			}
		},
		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				fmt.Printf("%v Connection to %s failed: %v\n", network, addr, err)
			} else {
				fmt.Printf("%v Connection to %s succeeded\n", network, addr)
			}
		},
		WroteDone: func(err error) {
			if err != nil {
				fmt.Printf("Write failed: %v\n", err)
			} else {
				fmt.Println("Write succeeded")
			}
		},
		ReadDone: func(err error) {
			if err != nil {
				fmt.Printf("Read failed: %v\n", err)
			} else {
				fmt.Println("Read succeeded")
			}
		},
	}

	// Variables to store the timestamps
	var startTLS time.Time

	ht := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			fmt.Printf("DNS start: %v\n", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			fmt.Printf("DNS done: %v\n", info)
		},
		ConnectStart: func(network, addr string) {
			fmt.Printf("Connect start: %v %v\n", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			fmt.Printf("Connect done: %v %v %v\n", network, addr, err)
		},
		GotFirstResponseByte: func() {
			fmt.Println("Got first response byte")
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
	return ctx
}
