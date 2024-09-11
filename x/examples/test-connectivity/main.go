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
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"
	"github.com/Jigsaw-Code/outline-sdk/x/report"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

// var errorLog log.Logger = *log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

type connectivityReport struct {
	Test           testReport  `json:"test"`
	DNSQueries     []dnsReport `json:"dns_queries,omitempty"`
	TCPConnections []tcpReport `json:"tcp_connections,omitempty"`
}

type testReport struct {
	// Inputs
	Resolver  string `json:"resolver"`
	Proto     string `json:"proto"`
	Transport string `json:"transport"`

	// Observations
	Time       time.Time  `json:"time"`
	DurationMs int64      `json:"duration_ms"`
	Error      *errorJSON `json:"error"`
}

type dnsReport struct {
	QueryName  string    `json:"query_name"`
	Time       time.Time `json:"time"`
	DurationMs int64     `json:"duration_ms"`
	AnswerIPs  []string  `json:"answer_ips"`
	Error      string    `json:"error"`
}

type tcpReport struct {
	Hostname string    `json:"hostname"`
	IP       string    `json:"ip"`
	Port     string    `json:"port"`
	Error    string    `json:"error"`
	Time     time.Time `json:"time"`
	Duration int64     `json:"duration_ms"`
}

type errorJSON struct {
	// TODO: add Shadowsocks/Transport error
	Op string `json:"op,omitempty"`
	// Posix error, when available
	PosixError string `json:"posix_error,omitempty"`
	// TODO: remove IP addresses
	Msg string `json:"msg,omitempty"`
}

// Declare a mutex for thread-safe access to slices
var mutex sync.Mutex

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
	if r.Test.Error == nil {
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
func newTCPTraceDialer(
	onDNS func(ctx context.Context, domain string) func(di httptrace.DNSDoneInfo),
	onDial func(ctx context.Context, network, addr string, connErr error),
	onDialStart func(ctx context.Context, network, addr string),
) transport.StreamDialer {
	dialer := &transport.TCPDialer{}
	var onDNSDone func(di httptrace.DNSDoneInfo)
	return transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
			DNSStart: func(di httptrace.DNSStartInfo) {
				onDNSDone = onDNS(ctx, di.Host)
			},
			DNSDone: func(di httptrace.DNSDoneInfo) {
				if onDNSDone != nil {
					onDNSDone(di)
					onDNSDone = nil
				}
			},
			ConnectStart: func(network, addr string) {
				onDialStart(ctx, network, addr)
			},
			ConnectDone: func(network, addr string, connErr error) {
				onDial(ctx, network, addr, connErr)
			},
		})
		return dialer.DialStream(ctx, addr)
	})
}

func newUDPTraceDialer(
	onDNS func(ctx context.Context, domain string) func(di httptrace.DNSDoneInfo),
) transport.PacketDialer {
	dialer := &transport.UDPDialer{}
	var onDNSDone func(di httptrace.DNSDoneInfo)
	return transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
			DNSStart: func(di httptrace.DNSStartInfo) {
				onDNSDone = onDNS(ctx, di.Host)
			},
			DNSDone: func(di httptrace.DNSDoneInfo) {
				if onDNSDone != nil {
					onDNSDone(di)
					onDNSDone = nil
				}
			},
		})
		return dialer.DialPacket(ctx, addr)
	})
}

func main() {
	verboseFlag := flag.Bool("v", false, "Enable debug output")
	transportFlag := flag.String("transport", "", "Transport config")
	domainFlag := flag.String("domain", "example.com.", "Domain name to resolve in the test")
	resolverFlag := flag.String("resolver", "8.8.8.8,2001:4860:4860::8888", "Comma-separated list of addresses of DNS resolver to use for the test")
	protoFlag := flag.String("proto", "tcp,udp", "Comma-separated list of the protocols to test. Must be \"tcp\", \"udp\", or a combination of them")
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
	for _, resolverHost := range strings.Split(*resolverFlag, ",") {
		resolverHost := strings.TrimSpace(resolverHost)
		resolverAddress := net.JoinHostPort(resolverHost, "53")
		for _, proto := range strings.Split(*protoFlag, ",") {
			proto = strings.TrimSpace(proto)
			var resolver dns.Resolver
			var connectStart = make(map[string]time.Time)
			dnsReports := make([]dnsReport, 0)
			tcpReports := make([]tcpReport, 0)
			configToDialer := config.NewDefaultConfigToDialer()
			configToDialer.BaseStreamDialer = transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
				hostname, _, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				onDNS := func(ctx context.Context, domain string) func(di httptrace.DNSDoneInfo) {
					dnsStart := time.Now()
					return func(di httptrace.DNSDoneInfo) {
						report := dnsReport{
							QueryName:  hostname,
							Time:       dnsStart.UTC().Truncate(time.Second),
							DurationMs: time.Since(dnsStart).Milliseconds(),
						}
						if di.Err != nil {
							report.Error = di.Err.Error()
						}
						for _, ip := range di.Addrs {
							report.AnswerIPs = append(report.AnswerIPs, ip.IP.String())
						}
						mutex.Lock()
						dnsReports = append(dnsReports, report)
						mutex.Unlock()
					}
				}
				onDialStart := func(ctx context.Context, network, addr string) {
					connectStart[network+"|"+addr] = time.Now()
				}
				onDial := func(ctx context.Context, network, addr string, connErr error) {
					ip, port, err := net.SplitHostPort(addr)
					if err != nil {
						return
					}
					report := tcpReport{
						Hostname: hostname,
						IP:       ip,
						Port:     port,
						Time:     connectStart[network+"|"+addr].UTC().Truncate(time.Second),
						Duration: time.Since(connectStart[network+"|"+addr]).Milliseconds(),
					}
					if connErr != nil {
						report.Error = connErr.Error()
					}
					mutex.Lock()
					tcpReports = append(tcpReports, report)
					mutex.Unlock()
				}
				return newTCPTraceDialer(onDNS, onDial, onDialStart).DialStream(ctx, addr)
			})
			configToDialer.BasePacketDialer = transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
				hostname, _, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				onDNS := func(ctx context.Context, domain string) func(di httptrace.DNSDoneInfo) {
					dnsStart := time.Now()
					return func(di httptrace.DNSDoneInfo) {
						report := dnsReport{
							QueryName:  hostname,
							Time:       dnsStart.UTC().Truncate(time.Second),
							DurationMs: time.Since(dnsStart).Milliseconds(),
						}
						//fmt.Printf("DNS Done Info: %+v\n", di)
						if di.Err != nil {
							report.Error = di.Err.Error()
						}
						for _, ip := range di.Addrs {
							report.AnswerIPs = append(report.AnswerIPs, ip.IP.String())
						}
						mutex.Lock()
						dnsReports = append(dnsReports, report)
						mutex.Unlock()
					}
				}
				return newUDPTraceDialer(onDNS).DialPacket(ctx, addr)
				//return (&transport.UDPDialer{}).DialPacket(ctx, addr)
			})
			switch proto {
			case "tcp":
				streamDialer, err := configToDialer.NewStreamDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				resolver = dns.NewTCPResolver(streamDialer, resolverAddress)
			case "udp":
				packetDialer, err := configToDialer.NewPacketDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create PacketDialer: %v", err)
				}
				resolver = dns.NewUDPResolver(packetDialer, resolverAddress)
			default:
				log.Fatalf(`Invalid proto %v. Must be "tcp" or "udp"`, proto)
			}
			startTime := time.Now()
			result, err := connectivity.TestConnectivityWithResolver(context.Background(), resolver, *domainFlag)
			if err != nil {
				log.Fatalf("Connectivity test failed to run: %v", err)
			}
			testDuration := time.Since(startTime)
			if result == nil {
				success = true
			}
			debugLog.Printf("Test %v %v result: %v", proto, resolverAddress, result)
			sanitizedConfig, err := config.SanitizeConfig(*transportFlag)
			if err != nil {
				log.Fatalf("Failed to sanitize config: %v", err)
			}
			var r report.Report = connectivityReport{
				Test: testReport{
					Resolver:   resolverAddress,
					Proto:      proto,
					Time:       startTime.UTC().Truncate(time.Second),
					Transport:  sanitizedConfig,
					DurationMs: testDuration.Milliseconds(),
					Error:      makeErrorRecord(result),
				},
				DNSQueries:     dnsReports,
				TCPConnections: tcpReports,
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
