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
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/transport/shadowsocks"
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
	Proxy    string `json:"proxy"`
	Resolver string `json:"resolver"`
	Proto    string `json:"proto"`
	Prefix   string `json:"prefix"`
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

type sessionConfig struct {
	Hostname  string
	Port      int
	CryptoKey *shadowsocks.EncryptionKey
	Prefix    Prefix
}

type Prefix []byte

func ConnectivityTest(request ConnectivityTestRequest) ([]ConnectivityTestResult, error) {
	var result ConnectivityTestResult
	accessKeyParameters, err := parseAccessKey(request.AccessKey)
	if err != nil {
		return nil, err
	}
	fmt.Printf("AccessKeyParameters: %v\n", accessKeyParameters)

	proxyIPs, err := net.DefaultResolver.LookupIP(context.Background(), "ip", accessKeyParameters.Hostname)
	if err != nil {
		return nil, err
	}
	fmt.Printf("ProxyIPs: %v\n", proxyIPs)

	// TODO: limit number of IPs. Or force an input IP?
	var results []ConnectivityTestResult
	for _, hostIP := range proxyIPs {
		proxyAddress := net.JoinHostPort(hostIP.String(), fmt.Sprint(accessKeyParameters.Port))
		fmt.Printf("ProxyAddress: %v\n", proxyAddress)

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
					log.Fatalf("Failed to create StreamDialer: %v", err)
				}
				resolver := &transport.StreamDialerEndpoint{Dialer: streamDialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverStreamConnectivity(context.Background(), resolver, resolverAddress)
				fmt.Printf("TestDuration: %v\n", testDuration)
				fmt.Printf("TestError: %v\n", testErr)

				result = ConnectivityTestResult{
					Proxy:      proxyAddress,
					Resolver:   resolverAddress,
					Proto:      "tcp",
					Prefix:     accessKeyParameters.Prefix.String(),
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
					log.Fatalf("Failed to create PacketDialer: %v", err)
				}
				resolver := &transport.PacketDialerEndpoint{Dialer: packetDialer, Address: resolverAddress}
				testDuration, testErr = connectivity.TestResolverPacketConnectivity(context.Background(), resolver, resolverAddress)
				fmt.Printf("TestDuration: %v\n", testDuration)
				fmt.Printf("TestError: %v\n", testErr)

				result = ConnectivityTestResult{
					Proxy:      proxyAddress,
					Resolver:   resolverAddress,
					Proto:      "udp",
					Prefix:     accessKeyParameters.Prefix.String(),
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
			}
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

func (p Prefix) String() string {
	runes := make([]rune, len(p))
	for i, b := range p {
		runes[i] = rune(b)
	}
	return string(runes)
}

func parseAccessKey(accessKey string) (*sessionConfig, error) {
	var config sessionConfig
	accessKeyURL, err := url.Parse(accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse access key: %w", err)
	}
	var portString string
	// Host is a <host>:<port> string
	config.Hostname, portString, err = net.SplitHostPort(accessKeyURL.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint address: %w", err)
	}
	config.Port, err = strconv.Atoi(portString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse port number: %w", err)
	}
	cipherInfoBytes, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(accessKeyURL.User.String())
	if err != nil {
		return nil, fmt.Errorf("failed to decode cipher info [%v]: %v", accessKeyURL.User.String(), err)
	}
	cipherName, secret, found := strings.Cut(string(cipherInfoBytes), ":")
	if !found {
		return nil, fmt.Errorf("invalid cipher info: no ':' separator")
	}
	config.CryptoKey, err = shadowsocks.NewEncryptionKey(cipherName, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	prefixStr := accessKeyURL.Query().Get("prefix")
	if len(prefixStr) > 0 {
		config.Prefix, err = ParseStringPrefix(prefixStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse prefix: %w", err)
		}
	}
	return &config, nil
}

func ParseStringPrefix(utf8Str string) (Prefix, error) {
	runes := []rune(utf8Str)
	rawBytes := make([]byte, len(runes))
	for i, r := range runes {
		if (r & 0xFF) != r {
			return nil, fmt.Errorf("character out of range: %d", r)
		}
		rawBytes[i] = byte(r)
	}
	return rawBytes, nil
}
