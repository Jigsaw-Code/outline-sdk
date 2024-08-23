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

package connectivity

import (
	"context"
	ctls "crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/trace"
	"golang.org/x/net/dns/dnsmessage"
)

// ConnectivityError captures the observed error of the connectivity test.
type ConnectivityError struct {
	// Which operation in the test that failed: "connect", "send" or "receive"
	Op string
	// The POSIX error, when available
	PosixError string
	// The error observed for the action
	Err error
}

var _ error = (*ConnectivityError)(nil)

func (err *ConnectivityError) Error() string {
	return fmt.Sprintf("%v: %v", err.Op, err.Err)
}

func (err *ConnectivityError) Unwrap() error {
	return err.Err
}

func isTimeout(err error) bool {
	var timeErr interface{ Timeout() bool }
	return errors.As(err, &timeErr) && timeErr.Timeout()
}

func makeConnectivityError(op string, err error) *ConnectivityError {
	// An early close on the connection may cause an "unexpected EOF" error. That's an application-layer error,
	// not triggered by a syscall error so we don't capture an error code.
	// TODO: figure out how to standardize on those errors.
	var code string
	var errno syscall.Errno
	if errors.As(err, &errno) {
		code = errnoName(errno)
	} else if isTimeout(err) {
		code = "ETIMEDOUT"
	}
	return &ConnectivityError{Op: op, PosixError: code, Err: err}
}

// TestConnectivityWithResolver tests weather we can get a response from the given [Resolver]. It can be used
// to test connectivity of its underlying [transport.StreamDialer] or [transport.PacketDialer].
// Invalid tests that cannot assert connectivity will return (nil, error).
// Valid tests will return (*ConnectivityError, nil), where *ConnectivityError will be nil if there's connectivity or
// a structure with details of the error found.
func TestConnectivityWithResolver(ctx context.Context, resolver dns.Resolver, testDomain string) (*ConnectivityError, error) {
	if _, ok := ctx.Deadline(); !ok {
		// Default deadline is 5 seconds.
		deadline := time.Now().Add(5 * time.Second)
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		// Releases the timer.
		defer cancel()
	}
	q, err := dns.NewQuestion(testDomain, dnsmessage.TypeA)
	if err != nil {
		return nil, fmt.Errorf("question creation failed: %w", err)
	}

	// Pass this context to your DNS resolver function
	_, err = resolver.Query(ctx, *q)

	if errors.Is(err, dns.ErrBadRequest) {
		return nil, err
	}
	if errors.Is(err, dns.ErrDial) {
		return makeConnectivityError("connect", err), nil
	} else if errors.Is(err, dns.ErrSend) {
		return makeConnectivityError("send", err), nil
	} else if errors.Is(err, dns.ErrReceive) {
		return makeConnectivityError("receive", err), nil
	}
	return nil, nil
}

func TestStreamConnectivitywithHTTP(ctx context.Context, baseDialer transport.StreamDialer, domain string, timeout time.Duration, method string) (*ConnectivityError, error) {
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
	httpClient := &http.Client{
		Transport: &http.Transport{DialContext: dialContext},
		Timeout:   time.Duration(timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(method, domain, nil)
	if err != nil {
		log.Fatalln("Failed to create request:", err)
	}
	// TODO: Add this as test param
	// headerText := strings.Join(headersFlag, "\r\n") + "\r\n\r\n"
	// h, err := textproto.NewReader(bufio.NewReader(strings.NewReader(headerText))).ReadMIMEHeader()
	// if err != nil {
	// 	log.Fatalf("invalid header line: %v", err)
	// }
	// for name, values := range h {
	// 	for _, value := range values {
	// 		req.Header.Add(name, value)
	// 	}
	// }

	req = req.WithContext(ctx)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		fmt.Printf("%v: %v\n", k, v)
	}

	fmt.Printf("StatusCode %v\n", resp.StatusCode)

	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		log.Fatalf("Read of page body failed: %v\n", err)
	}

	return nil, nil
}

func AddLoggerTrace(ctx context.Context) context.Context {
	t := &trace.DNSClientTrace{
		QuestionSent: func(question dnsmessage.Question) {
			fmt.Println("DNS query started for", question.Name.String())
		},
		ResponsDone: func(question dnsmessage.Question, msg *dnsmessage.Message, err error) {
			if err != nil {
				fmt.Printf("DNS query for %s failed: %v\n", question.Name.String(), err)
			} else {
				// Prepare to collect IP addresses
				var ips []string

				// Iterate over the answer section
				for _, answer := range msg.Answers {
					switch rr := answer.Body.(type) {
					case *dnsmessage.AResource:
						// Handle IPv4 addresses - convert [4]byte to IP string
						ipv4 := net.IP(rr.A[:]) // Convert [4]byte to net.IP
						ips = append(ips, ipv4.String())
					case *dnsmessage.AAAAResource:
						// Handle IPv6 addresses - convert [16]byte to IP string
						ipv6 := net.IP(rr.AAAA[:]) // Convert [16]byte to net.IP
						ips = append(ips, ipv6.String())
					}
				}

				// Print all resolved IP addresses
				if len(ips) > 0 {
					fmt.Printf("Resolved IPs for %s: %v\n", question.Name.String(), ips)
				} else {
					fmt.Printf("No IPs found for %s\n", question.Name.String())
				}
			}
		},
		ConnectDone: func(network, addr string, err error) {
			if err != nil {
				fmt.Printf("%v Connection to %s failed: %v\n", network, addr, err)
			} else {
				fmt.Printf("%v Connection to %s succeeded\n", network, addr)
			}
		},
		WroteDone: func(err error) {
			if err != nil {
				fmt.Printf("Write failed: %v\n", err)
			} else {
				fmt.Println("Write succeeded")
			}
		},
		ReadDone: func(err error) {
			if err != nil {
				fmt.Printf("Read failed: %v\n", err)
			} else {
				fmt.Println("Read succeeded")
			}
		},
	}

	// Variables to store the timestamps
	var startTLS time.Time

	ht := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			fmt.Printf("DNS start: %v\n", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			fmt.Printf("DNS done: %v\n", info)
		},
		ConnectStart: func(network, addr string) {
			fmt.Printf("Connect start: %v %v\n", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			fmt.Printf("Connect done: %v %v %v\n", network, addr, err)
		},
		GotFirstResponseByte: func() {
			fmt.Println("Got first response byte")
		},
		WroteHeaderField: func(key string, value []string) {
			fmt.Printf("Wrote header field: %v %v\n", key, value)
		},
		WroteHeaders: func() {
			fmt.Println("Wrote headers")
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			fmt.Printf("Wrote request: %v\n", info)
		},
		TLSHandshakeStart: func() {
			startTLS = time.Now()
		},
		TLSHandshakeDone: func(state ctls.ConnectionState, err error) {
			if err != nil {
				fmt.Printf("TLS handshake failed: %v\n", err)
			}
			fmt.Printf("SNI: %v\n", state.ServerName)
			fmt.Printf("TLS version: %v\n", state.Version)
			fmt.Printf("ALPN: %v\n", state.NegotiatedProtocol)
			fmt.Printf("TLS handshake took %v seconds.\n", time.Since(startTLS).Seconds())
		},
	}

	tlsTrace := &trace.TLSClientTrace{
		TLSHandshakeStart: func() {
			fmt.Println("TLS handshake started")
			startTLS = time.Now()
		},
		TLSHandshakeDone: func(state ctls.ConnectionState, err error) {
			if err != nil {
				fmt.Printf("TLS handshake failed: %v\n", err)
			}
			fmt.Printf("SNI: %v\n", state.ServerName)
			fmt.Printf("TLS version: %v\n", state.Version)
			fmt.Printf("ALPN: %v\n", state.NegotiatedProtocol)
			fmt.Printf("TLS handshake took %v seconds.\n", time.Since(startTLS).Seconds())
		},
	}

	ctx = httptrace.WithClientTrace(ctx, ht)
	ctx = trace.WithDNSClientTrace(ctx, t)
	ctx = trace.WithTLSClientTrace(ctx, tlsTrace)
	return ctx
}
