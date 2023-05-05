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
)

var debugLog log.Logger = *log.New(io.Discard, "", 0)

// var errorLog log.Logger = *log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

// captureConnErrors wraps a net.Conn to capture the errors that happen.
// TODO(fortuna): Move away from net.Resolver and do resolution ourselves, so we don't need this hack.
func captureConnErrors(conn net.Conn, connErrors *ConnErrors) net.Conn {
	cec := captureErrorConn{Conn: conn, errors: connErrors}
	// This is a hack needed by net.Resolver.Dial, because it decides on the packet or stream path
	// based on whether the returned net.Conn is a net.PacketConn. See
	// https://cs.opensource.google/go/go/+/refs/heads/master:src/net/dnsclient_unix.go;l=186;drc=4badad8d477ffd7a6b762c35bc69aed82faface7
	if pc, ok := conn.(net.PacketConn); ok {
		return &captureErrorPacketConn{PacketConn: pc, captureErrorConn: cec}
	}
	return &cec
}

type ConnErrors struct {
	writeErr error
	readErr  error
}

type captureErrorConn struct {
	net.Conn
	errors *ConnErrors
}

var _ net.Conn = (*captureErrorConn)(nil)

func (c *captureErrorConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err != nil {
		debugLog.Printf("Read error: %#v", debugError(err))
	}
	c.errors.readErr = err
	return n, err
}

func (c *captureErrorConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if err != nil {
		debugLog.Printf("Write error: %#v", debugError(err))
	}
	c.errors.writeErr = err
	return n, err
}

type captureErrorPacketConn struct {
	net.PacketConn
	captureErrorConn
}

func makeTCPDialer(proxyAddress string, cryptoKey *shadowsocks.EncryptionKey, prefix []byte) (DNSDial, error) {
	proxyDialer, err := shadowsocks.NewStreamDialer(&transport.TCPEndpoint{Address: proxyAddress}, cryptoKey)
	if err != nil {
		return nil, err
	}
	if len(prefix) > 0 {
		proxyDialer.SaltGenerator = shadowsocks.NewPrefixSaltGenerator(prefix)
	}
	return func(ctx context.Context, resolverAddress string) (net.Conn, error) {
		return proxyDialer.Dial(ctx, resolverAddress)
	}, nil
}

func makeUDPDialer(proxyAddress string, cryptoKey *shadowsocks.EncryptionKey) (DNSDial, error) {
	proxyListener, err := shadowsocks.NewPacketListener(&transport.UDPEndpoint{Address: proxyAddress}, cryptoKey)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context, resolverAddress string) (net.Conn, error) {
		endpoint := transport.PacketListenerEndpoint{Listener: proxyListener, Address: resolverAddress}
		return endpoint.Connect(ctx)
	}, nil
}

type DNSDial func(ctx context.Context, resolverAddress string) (net.Conn, error)

func testResolver(dnsDial DNSDial, resolverAddress, domain string) error {
	var dialErr error
	var connErrors ConnErrors
	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network string, address string) (net.Conn, error) {
			var conn net.Conn
			conn, dialErr = dnsDial(ctx, resolverAddress)
			if dialErr != nil {
				debugLog.Printf("Dial failed: %v", debugError(dialErr))
				return nil, dialErr
			}
			// We wrap the net.Conn to capture the read and write errors, since the DNSErrors used by the Resolver
			// doed not expose the underlying error objects.
			return captureConnErrors(conn, &connErrors), nil
		},
	}

	ips, dnsErr := resolver.LookupIP(context.Background(), "ip4", domain)
	if dnsErr != nil {
		debugLog.Printf("DNS error: %#v", dnsErr)
		if dialErr != nil {
			return &testError{Op: "dial", Err: dialErr}
		}
		if connErrors.writeErr != nil {
			return &testError{Op: "write", Err: connErrors.writeErr}
		}
		if connErrors.readErr != nil {
			return &testError{Op: "read", Err: connErrors.readErr}
		}
		return dnsErr
	}
	debugLog.Printf("DNS Resolution succeeded: %v", ips)
	return nil
}

func debugError(err error) string {
	// var netErr *net.OpError
	var syscallErr *os.SyscallError
	// errors.As(err, &netErr)
	errors.As(err, &syscallErr)
	return fmt.Sprintf("%#v %#v %v", err, syscallErr, err.Error())
}

type testError struct {
	Op  string
	Err error
}

func (err *testError) Error() string {
	return fmt.Sprintf("%v: %v", err.Op, err.Err)
}

func (err *testError) Unwrap() error {
	return err.Err
}

type jsonRecord struct {
	Proxy      string     `json:"proxy"`
	Resolver   string     `json:"resolver"`
	Proto      string     `json:"proto"`
	Prefix     string     `json:"prefix"`
	Time       time.Time  `json:"time"`
	DurationMs int64      `json:"duration_ms"`
	Error      *errorJSON `json:"error"`
}

type errorJSON struct {
	// TODO: add Shadowsocks/Transport error
	Op string
	// TODO: remove IP addresses
	Msg string
}

func makeErrorRecord(err error) *errorJSON {
	if err == nil {
		return nil
	}
	var record = new(errorJSON)
	var testErr *testError
	if errors.As(err, &testErr) {
		record.Op = testErr.Op
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
				// var dnsClient := dns.Client{Net: proto}
				var dnsDial DNSDial
				if proto == "tcp" {
					dnsDial, err = makeTCPDialer(proxyAddress, config.CryptoKey, config.Prefix)
				} else {
					dnsDial, err = makeUDPDialer(proxyAddress, config.CryptoKey)
				}
				if err != nil {
					log.Fatalf("Failed to create DNS resolver: %#v", err)
				}
				testTime := time.Now()
				testErr := testResolver(dnsDial, resolverAddress, *domainFlag)
				debugLog.Printf("Test error: %v", testErr)
				duration := time.Since(testTime)
				if testErr == nil {
					success = true
				}
				record := jsonRecord{
					Time:       testTime.UTC().Truncate(time.Second),
					DurationMs: duration.Milliseconds(),
					Proxy:      proxyAddress,
					Resolver:   resolverAddress,
					Proto:      proto,
					Prefix:     config.Prefix.String(),
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
