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

	jsonEncoder := json.NewEncoder(os.Stdout)
	jsonEncoder.SetEscapeHTML(false)
	configToDialer := config.NewDefaultConfigToDialer()
	ctx, connectivityEvents := SetupConnectivityTrace(context.Background())
	for _, resolverHost := range strings.Split(*resolverFlag, ",") {
		resolverHost := strings.TrimSpace(resolverHost)
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
				_, err = connectivity.TestConnectivityWithResolver(ctx, resolver, *domainFlag)
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
				_, err = connectivity.TestConnectivityWithResolver(ctx, resolver, *domainFlag)
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
				_, err = connectivity.TestConnectivityWithResolver(ctx, resolver, *domainFlag)
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
				_, err = connectivity.TestConnectivityWithResolver(ctx, resolver, *domainFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			case "http":
				streamDialer, err := configToDialer.NewStreamDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				err = connectivity.TestStreamConnectivitywithHTTP(ctx, streamDialer, *domainFlag, *timeoutFlag, *methodFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			case "http3":
				packetDialer, err := configToDialer.NewPacketDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create PacketDialer: %v", err)
				}
				err = connectivity.TestPacketConnectivitywithHTTP3(ctx, packetDialer, *domainFlag, *timeoutFlag, *methodFlag)
				if err != nil {
					log.Fatalf("Connectivity test failed to run: %v", err)
				}
			default:
				log.Fatalf(`Invalid Test Type %v.`, testType)
			}
			sanitizedConfig, err := config.SanitizeConfig(*transportFlag)
			if err != nil {
				log.Fatalf("Failed to sanitize config: %v", err)
			}

			connReport := &ConnectivityReport{
				TestType:           testType,
				Transport:          sanitizedConfig,
				ConnectivityEvents: connectivityEvents,
			}
			var r report.Report = connReport
			if reportCollector != nil {
				err := reportCollector.Collect(context.Background(), r)
				if err != nil {
					debugLog.Printf("Failed to collect report: %v\n", err)
				}
			}
		}
	}
}
