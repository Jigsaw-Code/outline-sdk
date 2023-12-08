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

	"github.com/Jigsaw-Code/outline-sdk/transport"
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
	// TODO(fortuna): get details from trace
	// Proxy    string `json:"proxy"`
	// Prefix   string `json:"prefix"`
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

func makeErrorRecord(err error) *errorJSON {
	if err == nil {
		return nil
	}
	var record = new(errorJSON)
	var testErr *connectivity.TestError
	if errors.As(err, &testErr) {
		record.Op = testErr.Op
		record.PosixError = testErr.PosixError
		record.Msg = unwrapAll(testErr).Error()
	} else {
		record.Msg = err.Error()
	}
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
			testTime := time.Now()
			var testErr error
			var testDuration time.Duration
			switch proto {
			case "tcp":
				streamDialer, err := config.NewStreamDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				resolver := &transport.StreamDialerEndpoint{Dialer: streamDialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverStreamConnectivity(context.Background(), resolver, *domainFlag)
			case "udp":
				packetDialer, err := config.NewPacketDialer(*transportFlag)
				if err != nil {
					log.Fatalf("Failed to create PacketDialer: %v", err)
				}
				resolver := &transport.PacketDialerEndpoint{Dialer: packetDialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverPacketConnectivity(context.Background(), resolver, *domainFlag)
			default:
				log.Fatalf(`Invalid proto %v. Must be "tcp" or "udp"`, proto)
			}
			debugLog.Printf("Test error: %v", testErr)
			if testErr == nil {
				success = true
			}
			var r report.Report = connectivityReport{
				Resolver: resolverAddress,
				Proto:    proto,
				Time:     testTime.UTC().Truncate(time.Second),
				// TODO(fortuna): Add tracing to get more detailed info:
				// Proxy:    proxyAddress,
				// Prefix:   config.Prefix.String(),
				DurationMs: testDuration.Milliseconds(),
				Error:      makeErrorRecord(testErr),
			}
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
			c := report.SamplingCollector{
				Collector:       retryCollector,
				SuccessFraction: *reportSuccessFlag,
				FailureFraction: *reportFailureFlag,
			}
			err = c.Collect(context.Background(), r)
			if err != nil {
				debugLog.Printf("Failed to collect report: %v\n", err)
			}
		}
		if !success {
			os.Exit(1)
		}
	}
}
