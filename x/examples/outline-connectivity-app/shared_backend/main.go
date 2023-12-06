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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
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
	if r.Error == nil {
		return true
	} else {
		return false
	}
}

type ConnectivityTestError struct {
	// TODO: add Shadowsocks/Transport error
	Op string `json:"operation"`
	// Posix error, when available
	PosixError string `json:"posixError"`
	// TODO: remove IP addresses
	Msg string `json:"message"`
}

type ConnectivityTestRequest struct {
	AccessKey string                         `json:"accessKey"`
	Domain    string                         `json:"domain"`
	Resolvers []string                       `json:"resolvers"`
	Protocols ConnectivityTestProtocolConfig `json:"protocols"`
	ReportTo  string                         `json:"reportTo"`
}

func ConnectivityTest(request ConnectivityTestRequest) ([]ConnectivityTestResult, error) {
	var result ConnectivityTestResult
	var results []ConnectivityTestResult

	transportConfig := replaceSSKeyWithHash(request.AccessKey)

	for _, resolverHost := range request.Resolvers {
		resolverHost := strings.TrimSpace(resolverHost)
		resolverAddress := net.JoinHostPort(resolverHost, "53")
		fmt.Printf("ResolverAddress: %v\n", resolverAddress)

		if request.Protocols.TCP {
			testTime := time.Now()
			var testErr error
			var testDuration time.Duration

			streamDialer, err := config.NewStreamDialer(request.AccessKey)
			if err != nil {
				//log.Fatalf("Failed to create StreamDialer: %v", err)
				testDuration = time.Duration(0)
				testErr = err
			} else {
				resolver := &transport.StreamDialerEndpoint{Dialer: streamDialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverStreamConnectivity(context.Background(), resolver, resolverAddress)
				fmt.Printf("TestDuration: %v\n", testDuration)
				fmt.Printf("TestError: %v\n", testErr)
			}
			result = ConnectivityTestResult{
				Transport:  transportConfig,
				Resolver:   resolverAddress,
				Proto:      "tcp",
				Time:       testTime.UTC().Truncate(time.Second),
				DurationMs: testDuration.Milliseconds(),
				Error:      makeErrorRecord(testErr),
			}
			results = append(results, result)
		}

		if request.Protocols.UDP {
			testTime := time.Now()
			var testErr error
			var testDuration time.Duration

			packetDialer, err := config.NewPacketDialer(request.AccessKey)
			if err != nil {
				//log.Fatalf("Failed to create PacketDialer: %v", err)
				testDuration = time.Duration(0)
				testErr = err
			} else {
				resolver := &transport.PacketDialerEndpoint{Dialer: packetDialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverPacketConnectivity(context.Background(), resolver, resolverAddress)
				fmt.Printf("TestDuration: %v\n", testDuration)
				fmt.Printf("TestError: %v\n", testErr)
			}

			result = ConnectivityTestResult{
				Transport:  transportConfig,
				Resolver:   resolverAddress,
				Proto:      "udp",
				Time:       testTime.UTC().Truncate(time.Second),
				DurationMs: testDuration.Milliseconds(),
				Error:      makeErrorRecord(testErr),
			}
			results = append(results, result)
		}
	}
	for _, result := range results {
		fmt.Printf("Result: %v\n", result)
		var r report.Report = result
		u, err := url.Parse(request.ReportTo)
		if err != nil {
			log.Printf("Expected no error, but got: %v", err)
			//return results, errors.New("failed to parse collector URL")
		}
		fmt.Println("Parsed URL: ", u.String())
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
				//return results, errors.New("failed to collect report")
			}
		}
	}

	return results, nil
}

type PlatformMetadata struct {
	OS string `json:"operatingSystem"`
}

func Platform() PlatformMetadata {
	return PlatformMetadata{OS: runtime.GOOS}
}

func makeErrorRecord(err error) *ConnectivityTestError {
	if err == nil {
		return nil
	}
	var record = new(ConnectivityTestError)
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

// hashKey hashes the given key and returns the hexadecimal representation
func hashKey(key string) string {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	fullHash := hex.EncodeToString(hasher.Sum(nil))
	return fullHash[:10] // Truncate the hash to 10 characters

}

// replaceSSKeyWithHash function replaces the key value with its hash in parts that start with "ss://"
func replaceSSKeyWithHash(input string) string {
	// Split the string into parts
	parts := strings.Split(input, "|")

	// Iterate through each part
	for i, part := range parts {
		if strings.HasPrefix(part, "ss://") {
			// Find the key part and replace it with its hash
			keyStart := strings.Index(part, "//") + 2
			keyEnd := strings.Index(part, "@")
			if keyStart != -1 && keyEnd != -1 && keyEnd > keyStart {
				key := part[keyStart:keyEnd]
				hashedKey := hashKey(key)
				parts[i] = part[:keyStart] + hashedKey + part[keyEnd:]
			}
		}
	}

	// Join the parts back into a string
	return strings.Join(parts, "|")
}
