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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

// var errorLog log.Logger = *log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

type jsonRecord struct {
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

func sendReport(record jsonRecord, collectorURL string) error {
	jsonData, err := json.Marshal(record)
	if err != nil {
		log.Fatalf("Error encoding JSON: %s\n", err)
		return err
	}

	req, err := http.NewRequest("POST", collectorURL, bytes.NewReader(jsonData))
	if err != nil {
		debugLog.Printf("Error creating the HTTP request: %s\n", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending the HTTP request: %s\n", err)
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		debugLog.Printf("Error reading the HTTP response body: %s\n", err)
		return err
	}
	debugLog.Printf("Response: %s\n", respBody)
	return nil
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
			record := jsonRecord{
				Resolver: resolverAddress,
				Proto:    proto,
				Time:     testTime.UTC().Truncate(time.Second),
				// TODO(fortuna): Add tracing to get more detailed info:
				// Proxy:    proxyAddress,
				// Prefix:   config.Prefix.String(),
				DurationMs: testDuration.Milliseconds(),
				Error:      makeErrorRecord(testErr),
			}
			err := jsonEncoder.Encode(record)
			if err != nil {
				log.Fatalf("Failed to output JSON: %v", err)
			}
			// Send error report to collector if specified
			if *reportToFlag != "" {
				var samplingRate float64
				if success {
					samplingRate = *reportSuccessFlag
				} else {
					samplingRate = *reportFailureFlag
				}
				// Generate a random number between 0 and 1
				random := rand.Float64()
				if random < samplingRate {
					err = sendReport(record, *reportToFlag)
					if err != nil {
						log.Fatalf("Report failed: %v", err)
					} else {
						fmt.Println("Report sent")
					}
				} else {
					fmt.Println("Report was not sent this time")
				}
			}
		}
	}
	if !success {
		os.Exit(1)
	}
}
