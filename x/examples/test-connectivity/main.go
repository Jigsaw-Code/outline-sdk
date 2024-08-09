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

var debugLog log.Logger = *log.New(os.Stderr, "", 0)

//var errorLog log.Logger = *log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

type connectivityReport struct {
	// Inputs
	Resolver string `json:"resolver"`
	Proto    string `json:"proto"`
	// TODO(fortuna): add sanitized transport config.
	Transport string `json:"transport"`

	// The result for the connection.
	Result connectivityResult `json:"result"`

	// Observations
	StartTime  time.Time  `json:"start_time"`
	DurationMs int64      `json:"duration_ms"`
	Error      *errorJSON `json:"error"`
}

type connectivityResult struct {
	Endpoint *addressJSON            `json:"endpoint,omitempty"`
	Attempts []connectionAttemptJSON `json:"attempts,omitempty"`
}

type connectionAttemptJSON struct {
	Address    *addressJSON `json:"address,omitempty"`
	StartTime  time.Time    `json:"start_time"`
	DurationMs int64        `json:"duration_ms"`
	Error      *errorJSON   `json:"error"`
}

type errorJSON struct {
	// TODO: add Shadowsocks/Transport error
	Op string `json:"op,omitempty"`
	// Posix error, when available
	PosixError string `json:"posix_error,omitempty"`
	// TODO: remove IP addresses
	Msg string `json:"msg,omitempty"`
}

type addressJSON struct {
	Host        string `json:"host"`
	Port        string `json:"port"`
	AddressType IPType `json:"ip_type,omitempty"`
}

type IPType string

const (
	IPv4   IPType = "v4"
	IPv6   IPType = "v6"
	Domain IPType = ""
)

func newAddressJSON(address string) (addressJSON, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return addressJSON{}, err
	}
	ipType, err := checkIPVersion(host)
	if err != nil {
		// If the address is not an IP, it is a domain name
		return addressJSON{Host: host, Port: port, AddressType: Domain}, nil
	}
	return addressJSON{host, port, ipType}, nil
}

func checkIPVersion(ipStr string) (IPType, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return Domain, fmt.Errorf("invalid ip address: %s", ipStr)
	}
	if ip.To4() != nil {
		return IPv4, nil
	} else {
		return IPv6, nil
	}
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
	transportFlag := flag.String("transport", "", "Transport config")
	baseTransportFlag := flag.String("base-transport", "", "Remote Transport config")
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
	configToDialer := config.NewDefaultConfigToDialer()
	baseDialer, err := configToDialer.NewStreamDialer(*baseTransportFlag)
	if err != nil {
		log.Fatalf("Could not create dialer: %v\n", err)
	}
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %w", err)
		}
		if !strings.HasPrefix(network, "tcp") {
			return nil, fmt.Errorf("protocol not supported: %v", network)
		}
		return baseDialer.DialStream(ctx, net.JoinHostPort(host, port))
	}
	// TODO: provide timeout as input argument to cli
	httpClient := &http.Client{
		Transport: &http.Transport{DialContext: dialContext},
		Timeout:   30 * time.Second,
	}

	var reportCollector report.Collector
	if *reportToFlag != "" {
		collectorURL, err := url.Parse(*reportToFlag)
		if err != nil {
			debugLog.Printf("Failed to parse collector URL: %v", err)
		}
		remoteCollector := &report.RemoteCollector{
			CollectorURL: collectorURL,
			HttpClient:   httpClient,
		}
		retryCollector := &report.RetryCollector{
			Collector:    remoteCollector,
			MaxRetry:     1,
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
			var testResult *connectivity.ConnectivityResult
			var testErr error
			startTime := time.Now()
			switch proto {
			case "tcp":
				base, err := configToDialer.NewStreamDialer(*baseTransportFlag)
				if err != nil {
					log.Fatalf("Could not create base dialer: %v\n", err)
				}
				wrap := func(baseDialer transport.StreamDialer) (transport.StreamDialer, error) {
					c := config.NewDefaultConfigToDialer()
					c.BaseStreamDialer = baseDialer
					return c.NewStreamDialer(*transportFlag)
				}
				testResult, testErr = connectivity.TestStreamConnectivityWithDNS(context.Background(), base, wrap, resolverAddress, *domainFlag)
			case "udp":
				base, err := configToDialer.NewPacketDialer(*baseTransportFlag)
				if err != nil {
					log.Fatalf("Could not create base dialer: %v\n", err)
				}
				wrap := func(baseDialer transport.PacketDialer) (transport.PacketDialer, error) {
					c := config.NewDefaultConfigToDialer()
					c.BasePacketDialer = baseDialer
					return c.NewPacketDialer(*transportFlag)
				}
				testResult, testErr = connectivity.TestPacketConnectivityWithDNS(context.Background(), base, wrap, resolverAddress, *domainFlag)
			default:
				log.Fatalf(`Invalid proto %v. Must be "tcp" or "udp"`, proto)
			}
			if testErr != nil {
				//log.Fatalf("Connectivity test failed to run: %v", testErr)
				debugLog.Printf("Connectivity test failed to run: %v", testErr)
			}
			testDuration := time.Since(startTime)
			if testResult.Error == nil {
				success = true
			}
			debugLog.Printf("Test %v %v result: %v", proto, resolverAddress, testResult)
			sanitizedConfig, err := config.SanitizeConfig(*transportFlag)
			if err != nil {
				log.Fatalf("Failed to sanitize config: %v", err)
			}
			r := connectivityReport{
				Resolver:  resolverAddress,
				Proto:     proto,
				StartTime: startTime.UTC().Truncate(time.Second),
				// TODO(fortuna): Add sanitized config:
				Transport:  sanitizedConfig,
				DurationMs: testDuration.Milliseconds(),
				Error:      makeErrorRecord(testResult.Error),
			}
			addressJSON, err := newAddressJSON(testResult.Endpoint)
			if err == nil {
				r.Result.Endpoint = &addressJSON
			}
			for _, cr := range testResult.Attempts {
				cj := connectionAttemptJSON{}
				addressJSON, err := newAddressJSON(cr.Address)
				if err == nil {
					cj.Address = &addressJSON
				}
				cj.StartTime = cr.StartTime.UTC().Truncate(time.Second)
				cj.DurationMs = cr.Duration.Milliseconds()
				if cr.Error != nil {
					cj.Error = makeErrorRecord(cr.Error)
				}
				r.Result.Attempts = append(r.Result.Attempts, cj)
			}

			if reportCollector != nil && r.Result.Attempts != nil {
				// do not report if there are no attempts
				fmt.Println("sending the report....")
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
