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

package shared_backend

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/connectivity"
	"github.com/Jigsaw-Code/outline-sdk/x/report"

	_ "golang.org/x/mobile/bind"
)

type ConnectivityTestProtocolConfig struct {
	TCP bool `json:"tcp"`
	UDP bool `json:"udp"`
}

type ConnectivityTestResult struct {
	// Inputs
	Transport string `json:"transport"`
	Resolver  string `json:"resolver"`
	Proto     string `json:"proto"`
	// Observations
	Time       time.Time              `json:"time"`
	DurationMs int64                  `json:"durationMs"`
	Error      *ConnectivityTestError `json:"error"`
}

func (r ConnectivityTestResult) IsSuccess() bool {
	return r.Error == nil
}

type ConnectivityTestError struct {
	// TODO: add Shadowsocks/Transport error
	Op string `json:"op,omitempty"`
	// Posix error, when available
	PosixError string `json:"posixError,omitempty"`
	// TODO: remove IP addresses
	Msg string `json:"message,omitempty"`
}

type ConnectivityTestRequest struct {
	Transport string                         `json:"transport"`
	Domain    string                         `json:"domain"`
	Resolvers []string                       `json:"resolvers"`
	Protocols ConnectivityTestProtocolConfig `json:"protocols"`
	ReportTo  string                         `json:"reportTo"`
}

func ConnectivityTest(request ConnectivityTestRequest) ([]ConnectivityTestResult, error) {
	var result ConnectivityTestResult
	var results []ConnectivityTestResult
	var conTestResult *ConnectivityTestError
	var conError *connectivity.ConnectivityError
	var testDuration time.Duration

	sanitizedConfig, err := config.SanitizeConfig(request.Transport)
	if err != nil {
		log.Fatalf("Failed to sanitize config: %v", err)
	}

	for _, resolverHost := range request.Resolvers {
		resolverHost := strings.TrimSpace(resolverHost)
		resolverAddress := net.JoinHostPort(resolverHost, "53")

		if request.Protocols.TCP {
			testTime := time.Now()
			streamDialer, err := config.NewStreamDialer(request.Transport)
			if err != nil {
				testDuration = time.Duration(0)
				conTestResult = &ConnectivityTestError{Msg: err.Error()}
			} else {
				resolver := dns.NewTCPResolver(streamDialer, resolverAddress)
				conError, err = connectivity.TestConnectivityWithResolver(context.Background(), resolver, request.Domain)
				if err != nil {
					testDuration = time.Duration(0)
					conTestResult = &ConnectivityTestError{Msg: err.Error()}
					log.Printf("TCP question failed: %v\n", err)
				} else {
					testDuration = time.Since(testTime)
					if conError == nil {
						conTestResult = nil
					} else {
						conTestResult = &ConnectivityTestError{conError.Op, conError.PosixError, unwrapAll(conError).Error()}
					}
				}
			}
			result = ConnectivityTestResult{
				Transport:  sanitizedConfig,
				Resolver:   resolverAddress,
				Proto:      "tcp",
				Time:       testTime.UTC().Truncate(time.Second),
				DurationMs: testDuration.Milliseconds(),
				Error:      conTestResult,
			}
			results = append(results, result)
		}

		if request.Protocols.UDP {
			testTime := time.Now()
			packetDialer, err := config.NewPacketDialer(request.Transport)
			if err != nil {
				testDuration = time.Duration(0)
				conTestResult = &ConnectivityTestError{Msg: err.Error()}
			} else {
				resolver := dns.NewUDPResolver(packetDialer, resolverAddress)
				conError, err = connectivity.TestConnectivityWithResolver(context.Background(), resolver, request.Domain)
				if err != nil {
					log.Printf("UDP question failed: %v\n", err)
					testDuration = time.Duration(0)
					conTestResult = &ConnectivityTestError{Msg: err.Error()}
				} else {
					testDuration = time.Since(testTime)
					if conError == nil {
						conTestResult = nil
					} else {
						conTestResult = &ConnectivityTestError{conError.Op, conError.PosixError, unwrapAll(conError).Error()}
					}
				}
			}
			result = ConnectivityTestResult{
				Transport:  sanitizedConfig,
				Resolver:   resolverAddress,
				Proto:      "udp",
				Time:       testTime.UTC().Truncate(time.Second),
				DurationMs: testDuration.Milliseconds(),
				Error:      conTestResult,
			}
			results = append(results, result)
		}
	}
	reportResults(results, request.ReportTo)
	return results, nil
}

func reportResults(results []ConnectivityTestResult, reportTo string) {
	for _, result := range results {
		var r report.Report = result
		u, err := url.Parse(reportTo)
		if err != nil {
			log.Printf("Expected no error, but got: %v", err)
		}
		if u.String() != "" {
			remoteCollector := &report.RemoteCollector{
				CollectorURL: u,
				HttpClient:   &http.Client{Timeout: 10 * time.Second},
			}
			retryCollector := &report.RetryCollector{
				Collector:    remoteCollector,
				MaxRetry:     3,
				InitialDelay: 1 * time.Second,
			}
			c := report.SamplingCollector{
				Collector:       retryCollector,
				SuccessFraction: 0.1,
				FailureFraction: 1.0,
			}
			err = c.Collect(context.Background(), r)
			if err != nil {
				log.Printf("Failed to collect report: %v\n", err)
			}
		}
	}
}

type PlatformMetadata struct {
	OS string `json:"operatingSystem"`
}

func Platform() PlatformMetadata {
	return PlatformMetadata{OS: runtime.GOOS}
}

func unwrapAll(err error) error {
	if err == nil {
		return nil
	}
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}
