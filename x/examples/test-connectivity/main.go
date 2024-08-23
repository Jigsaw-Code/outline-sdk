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
	testTypeFlag := flag.String("test-type", "do53-tcp,do53-udp,doh,dot,http", "Type of test to run")
	transportFlag := flag.String("transport", "", "Transport config")
	domainFlag := flag.String("domain", "example.com.", "Domain name to resolve in the DNS test and to fetch in the HTTP test")
	methodFlag := flag.String("method", "GET", "HTTP method to use in the HTTP test")
	timeoutFlag := flag.Duration("timeout", 10*time.Second, "Timeout for the test; default is 10s")
	resolverFlag := flag.String("resolver", "8.8.8.8,2001:4860:4860::8888", "Comma-separated list of addresses of DNS resolver to use for the test")
	resolverNameFlag := flag.String("resolver-name", "one.one.one.one", "Name of the resolver to use for the test")
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
	ctx := connectivity.AddLoggerTrace(context.Background())
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
			default:
				log.Fatalf(`Invalid Test Type %v. Must be "tcp" or "udp"`, testType)
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
