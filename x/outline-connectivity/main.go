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
	"os"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	"github.com/Jigsaw-Code/outline-internal-sdk/transport/shadowsocks"
	"github.com/Jigsaw-Code/outline-internal-sdk/x/connectivity"
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

// var errorLog log.Logger = *log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

func newStreamDialer(proxyAddress string, cryptoKey *shadowsocks.EncryptionKey, prefix []byte) (transport.StreamDialer, error) {
	dialer, err := shadowsocks.NewStreamDialer(&transport.TCPEndpoint{Address: proxyAddress}, cryptoKey)
	if err != nil {
		return nil, err
	}
	if len(prefix) > 0 {
		dialer.SaltGenerator = shadowsocks.NewPrefixSaltGenerator(prefix)
	}
	return dialer, nil
}

func newPacketDialer(proxyAddress string, cryptoKey *shadowsocks.EncryptionKey) (transport.PacketDialer, error) {
	listener, err := shadowsocks.NewPacketListener(&transport.UDPEndpoint{Address: proxyAddress}, cryptoKey)
	if err != nil {
		return nil, err
	}
	return &transport.PacketListenerDialer{Listener: listener}, nil
}

type jsonRecord struct {
	// Inputs
	Proxy    string `json:"proxy"`
	Resolver string `json:"resolver"`
	Proto    string `json:"proto"`
	Prefix   string `json:"prefix"`
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

func main() {
	verboseFlag := flag.Bool("v", false, "Enable debug output")
	accessKeyFlag := flag.String("key", "", "Outline access key")
	domainFlag := flag.String("domain", "example.com.", "Domain name to resolve in the test")
	resolverFlag := flag.String("resolver", "8.8.8.8,2001:4860:4860::8888", "Comma-separated list of addresses of DNS resolver to use for the test")
	protoFlag := flag.String("proto", "tcp,udp", "Comma-separated list of the protocols to test. Muse be \"tcp\", \"udp\", or a combination of them")

	flag.Parse()
	if *verboseFlag {
		debugLog = *log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	}
	if *accessKeyFlag == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Things to test:
	// - TCP working. Where's the error?
	// - UDP working
	// - Different server IPs
	// - Server IPv4 dial support
	// - Server IPv6 dial support

	config, err := parseAccessKey(*accessKeyFlag)
	if err != nil {
		log.Fatal(err.Error())
	}
	debugLog.Printf("Config: %+v", config)

	proxyIPs, err := net.DefaultResolver.LookupIP(context.Background(), "ip", config.Hostname)
	if err != nil {
		log.Fatalf("Failed to resolve host name: %v", err)
	}

	success := false
	jsonEncoder := json.NewEncoder(os.Stdout)
	jsonEncoder.SetEscapeHTML(false)
	// TODO: limit number of IPs. Or force an input IP?
	for _, hostIP := range proxyIPs {
		proxyAddress := net.JoinHostPort(hostIP.String(), fmt.Sprint(config.Port))
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
					dialer, err := newStreamDialer(proxyAddress, config.CryptoKey, config.Prefix)
					if err != nil {
						log.Fatalf("Failed to create StreamDialer: %v", err)
					}
					var resolver = transport.NewDialerEndpoint(dialer, resolverAddress)
					testDuration, testErr = connectivity.TestResolverStreamConnectivity(context.Background(), resolver, *domainFlag)
				case "udp":
					dialer, err := newPacketDialer(proxyAddress, config.CryptoKey)
					if err != nil {
						log.Fatalf("Failed to create PacketListener: %v", err)
					}
					resolver := transport.NewDialerEndpoint(dialer, resolverAddress)
					testDuration, testErr = connectivity.TestResolverPacketConnectivity(context.Background(), resolver, *domainFlag)
				default:
					log.Fatalf(`Invalid proto %v. Must be "tcp" or "udp"`, proto)
				}
				debugLog.Printf("Test error: %v", testErr)
				if testErr == nil {
					success = true
				}
				record := jsonRecord{
					Proxy:      proxyAddress,
					Resolver:   resolverAddress,
					Proto:      proto,
					Prefix:     config.Prefix.String(),
					Time:       testTime.UTC().Truncate(time.Second),
					DurationMs: testDuration.Milliseconds(),
					Error:      makeErrorRecord(testErr),
				}
				err = jsonEncoder.Encode(record)
				if err != nil {
					log.Fatalf("Failed to output JSON: %v", err)
				}
			}
		}
	}
	if !success {
		os.Exit(1)
	}
}
