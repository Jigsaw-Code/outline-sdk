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
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"
	"github.com/Jigsaw-Code/outline-sdk/x/report"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

// var errorLog log.Logger = *log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

type JSONConnectivityEvents struct {
	ConnectInfo   []*JSONConnectInfo   `json:"connect_info,omitempty"`
	DnsInfo       []*JSONDNSInfo       `json:"dns_info,omitempty"`
	SystemDNSInfo []*JSONSystemDNSInfo `json:"system_dns_info,omitempty"`
}

type JSONConnectInfo struct {
	Network   string        `json:"network"`
	IP        string        `json:"ip"`
	Port      string        `json:"port"`
	Error     string        `json:"error,omitempty"`
	StartTime time.Time     `json:"start_time"`
	Duration  time.Duration `json:"duration_ms"`
	ConnError *errorJSON    `json:"conn_error,omitempty"`
}

type JSONDNSInfo struct {
	Host         string        `json:"host"`
	Resolver     string        `json:"resolver"`
	Network      string        `json:"network"`
	ResolverType string        `json:"resolver_type"`
	IPs          []string      `json:"ips"`
	RSCodes      []string      `json:"rs_codes"`
	Error        string        `json:"error,omitempty"`
	StartTime    time.Time     `json:"start_time"`
	Duration     time.Duration `json:"duration_ms"`
	ConnError    *errorJSON    `json:"conn_error,omitempty"`
}

type JSONSystemDNSInfo struct {
	Host      string        `json:"host"`
	IPs       []string      `json:"ips"`
	Error     string        `json:"error,omitempty"`
	StartTime time.Time     `json:"start_time"`
	Duration  time.Duration `json:"duration_ms"`
}

type errorJSON struct {
	Op         string `json:"op,omitempty"`
	PosixError string `json:"posix_error,omitempty"`
	Msg        string `json:"msg,omitempty"`
}

// Function to convert ConnectivityEvents to JSON-friendly structure
func ConvertToJSONFriendly(events *connectivity.ConnectivityEvents) *JSONConnectivityEvents {
	jsonEvents := &JSONConnectivityEvents{
		ConnectInfo:   make([]*JSONConnectInfo, len(events.ConnectInfo)),
		DnsInfo:       make([]*JSONDNSInfo, len(events.DnsInfo)),
		SystemDNSInfo: make([]*JSONSystemDNSInfo, len(events.SystemDNSInfo)),
	}

	for i, ci := range events.ConnectInfo {
		jsonEvents.ConnectInfo[i] = &JSONConnectInfo{
			Network:   ci.Network,
			IP:        ci.IP,
			Port:      ci.Port,
			StartTime: ci.StartTime,
			Duration:  time.Duration(ci.Duration.Milliseconds()),
			ConnError: makeErrorRecord(ci.ConnError),
		}
		if ci.Error != nil {
			jsonEvents.ConnectInfo[i].Error = ci.Error.Error()
		}
	}

	for i, di := range events.DnsInfo {
		ips := make([]string, len(di.IPs))
		for j, ip := range di.IPs {
			ips[j] = ip.IP.String()
		}
		rsCodes := make([]string, len(di.RSCodes))
		for j, code := range di.RSCodes {
			rsCodes[j] = code.String()
		}
		jsonEvents.DnsInfo[i] = &JSONDNSInfo{
			Host:         di.Host,
			Resolver:     di.Resolver,
			Network:      di.Network,
			ResolverType: di.ResolverType,
			IPs:          ips,
			RSCodes:      rsCodes,
			StartTime:    di.StartTime,
			Duration:     time.Duration(di.Duration.Milliseconds()),
			ConnError:    makeErrorRecord(di.ConnError),
		}
		if di.Error != nil {
			jsonEvents.DnsInfo[i].Error = di.Error.Error()
		}
	}

	for i, sdi := range events.SystemDNSInfo {
		ips := make([]string, len(sdi.IPs))
		for j, ip := range sdi.IPs {
			ips[j] = ip.IP.String()
		}
		jsonEvents.SystemDNSInfo[i] = &JSONSystemDNSInfo{
			Host:      sdi.Host,
			IPs:       ips,
			StartTime: sdi.StartTime,
			Duration:  time.Duration(sdi.Duration.Milliseconds()),
		}
		if sdi.Error != nil {
			jsonEvents.SystemDNSInfo[i].Error = sdi.Error.Error()
		}
	}

	return jsonEvents
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

// func (r connectivityReport) IsSuccess() bool {
// 	if r.Error == nil {
// 		return true
// 	} else {
// 		return false
// 	}
// }

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
	ctx, connectivityEvents := connectivity.SetupConnectivityTrace(context.Background())
	for _, resolverHost := range strings.Split(*resolverFlag, ",") {
		resolverHost := strings.TrimSpace(resolverHost)
		var result *connectivity.ConnectivityError
		var resolverAddress string
		for _, testType := range strings.Split(*testTypeFlag, ",") {
			testType = strings.TrimSpace(testType)
			var resolver dns.Resolver
			switch testType {
			case "do53-tcp":
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
				streamDialer, err := configToDialer.NewStreamDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				result, err = connectivity.TestStreamConnectivitywithHTTP(ctx, streamDialer, *domainFlag, *timeoutFlag, *methodFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			case "http3":
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
			if result == nil {
				success = true
			}
			debugLog.Printf("Test Type: %v Resolver Address: %v domain: %v result: %v", testType, resolverAddress, *domainFlag, result)
			//sanitizedConfig, err := config.SanitizeConfig(*transportFlag)
			// if err != nil {
			// 	log.Fatalf("Failed to sanitize config: %v", err)
			// }

			fmt.Printf("Connectivity Events: %+v\n", connectivityEvents)
			jsonFriendlyEvents := ConvertToJSONFriendly(connectivityEvents)
			var r report.Report = jsonFriendlyEvents
			if reportCollector != nil {
				err := reportCollector.Collect(context.Background(), r)
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
