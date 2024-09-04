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
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"

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

func MakeConnectivityError(op string, err error) *ConnectivityError {
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
		return MakeConnectivityError("connect", err), nil
	} else if errors.Is(err, dns.ErrSend) {
		return MakeConnectivityError("send", err), nil
	} else if errors.Is(err, dns.ErrReceive) {
		return MakeConnectivityError("receive", err), nil
	}
	return nil, nil
}

func TestStreamConnectivitywithHTTP(ctx context.Context, baseDialer transport.StreamDialer, domain string, timeout time.Duration, method string) error {
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
		return fmt.Errorf("HTTP request failed: %w", err)
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

	return nil
}

func TestPacketConnectivitywithHTTP3(ctx context.Context, baseDialer transport.PacketDialer, domain string, timeout time.Duration, method string) error {

	// Setup HTTP/3 RoundTripper
	http3RoundTripper := &http3.RoundTripper{
		TLSClientConfig: &ctls.Config{},
		QUICConfig:      &quic.Config{
			// Tracer: quictrace.NewTracer(),
		},
		Dial: func(ctx context.Context, addr string, tlsCfg *ctls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
			netAddr, err := net.ResolveUDPAddr("udp", addr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
			}
			dialer, err := baseDialer.DialPacket(ctx, addr)
			if err != nil {
				return nil, fmt.Errorf("failed to listen on packet: %w", err)
			}
			packetConn := dialer.(net.PacketConn)
			return quic.DialEarly(ctx, packetConn, netAddr, tlsCfg, cfg)
		},
	}
	defer http3RoundTripper.Close()

	httpClient := &http.Client{
		Transport: http3RoundTripper,
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
		return fmt.Errorf("HTTP request failed: %w", err)
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

	return nil
}
