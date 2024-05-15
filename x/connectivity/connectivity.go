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
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/dns/dnsmessage"
)

// ConnectivityResult captures the observed result of the connectivity test.
type ConnectivityResult struct {
	// Lists each connection attempt
	Attempts []ConnectionAttempt
	// Address of the connection that was selected
	Endpoint string
	// Start time of the main test
	StartTime time.Time
	// Duration of the main test
	Duration time.Duration
	// result error
	Error *ConnectivityError
}

type ConnectionAttempt struct {
	Address string
	// Start time of the connection attempt
	StartTime time.Time
	// Duration of the connection attempt
	Duration time.Duration
	// Observed error
	Error *ConnectivityError
}

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

type WrapStreamDialer func(baseDialer transport.StreamDialer) (transport.StreamDialer, error)

var ErrNoMoreIPs = errors.New("no more ips to try")

// TestStreamConnectivityWithDNS tests weather we can get a response from a DNS resolver at resolverAddress over a stream connection. It sends testDomain as the query.
// It uses the baseDialer to create a first-hop connection to the proxy, and the wrap to apply the transport.
// The baseDialer is typically TCPDialer, but it can be replaced for remote measurements.
func TestStreamConnectivityWithDNS(ctx context.Context, baseDialer transport.StreamDialer, wrap WrapStreamDialer, resolverAddress string, testDomain string) (*ConnectivityResult, error) {
	testResult := &ConnectivityResult{}
	connectResult := &testResult.Attempts
	ipIndex := 0
	done := make(chan bool)
	proceed := make(chan bool, 1)
	var waitGroup sync.WaitGroup
	// Create a new context for canceling goroutines
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	proceed <- true
	testResult.StartTime = time.Now()
loop:
	for {
		select {
		case <-done:
			break loop
		case <-proceed:
			waitGroup.Add(1)
			attempt := &ConnectionAttempt{}
			go func(attempt *ConnectionAttempt) {
				defer waitGroup.Done()
				attempt.StartTime = time.Now()
				interceptDialer := transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
					// Captures the address of the first hop, before resolution.
					testResult.Endpoint = addr
					host, port, err := net.SplitHostPort(addr)
					if err != nil {
						cancel()
						done <- true
						return nil, err
					}
					ips, err := (&net.Resolver{PreferGo: false}).LookupHost(ctx, host)
					if err != nil {
						cancel()
						done <- true
						return nil, err
					}
					var conn transport.StreamConn
					if ipIndex < len(ips) {
						// proceed to setting up the next test
						proceed <- true
						ip := ips[ipIndex]
						//fmt.Printf("Trying address %v\n", ip)
						ipIndex++
						addr := net.JoinHostPort(ip, port)
						attempt.Address = addr
						// TODO: pass timeout paramter as argument
						ipCtx, cancelWithTimeout := context.WithTimeout(ctx, 5*time.Second)
						defer cancelWithTimeout()
						conn, err = baseDialer.DialStream(ipCtx, addr)
						if err != nil {
							return nil, err
						}
						return conn, err
					} else {
						// stop iterating
						done <- true
						return nil, ErrNoMoreIPs
					}
				})
				dialer, err := wrap(interceptDialer)
				if err != nil {
					attempt.Duration = time.Since(attempt.StartTime)
					attempt.Error.Err = err
					*connectResult = append(*connectResult, *attempt)
					return
				}
				resolverConn, err := dialer.DialStream(ctx, resolverAddress)
				if err != nil {
					// Do not include cencelled errors in the result
					if errors.Is(err, context.Canceled) {
						return
					}
					// Do not include ErrNoMoreIPs type error in the attempt result
					if errors.Is(err, ErrNoMoreIPs) {
						return
					}
					attempt.Duration = time.Since(attempt.StartTime)
					attempt.Error = makeConnectivityError("connect", err)
					*connectResult = append(*connectResult, *attempt)
					// CHANGE: populate main test result error field with
					// one of attempt errors if non of the attempts succeeded
					testResult.Error = attempt.Error
					return
				}
				resolver := dns.NewTCPResolver(transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
					return resolverConn, nil
				}), resolverAddress)
				// I am ignoring the error returned by TestConnectivityWithResolver
				// because I am already capturing the error in the attempt. Not sure
				// if this is the right approach.
				attempt.Error, _ = TestConnectivityWithResolver(ctx, resolver, testDomain)
				attempt.Duration = time.Since(attempt.StartTime)
				*connectResult = append(*connectResult, *attempt)
				if attempt.Error == nil {
					testResult.Error = nil
					// test has succeeded; cancel the rest of the goroutines
					cancel()
				} else {
					// CHANGE: populate main test result error field with
					// one of attempt errors if non of the attempts succeeded
					testResult.Error = attempt.Error
				}
			}(attempt)
		}
	}
	waitGroup.Wait()
	testResult.Duration = time.Since(testResult.StartTime)
	return testResult, nil
}

type WrapPacketDialer func(baseDialer transport.PacketDialer) (transport.PacketDialer, error)

// TestPacketConnectivityWithDNS tests weather we can get a response from a DNS resolver at resolverAddress over a packet connection. It sends testDomain as the query.
// It uses the baseDialer to create a first-hop connection to the proxy, and the wrap to apply the transport.
// The baseDialer is typically UDPDialer, but it can be replaced for remote measurements.
func TestPacketConnectivityWithDNS(ctx context.Context, baseDialer transport.PacketDialer, wrap WrapPacketDialer, resolverAddress string, testDomain string) (*ConnectivityResult, error) {
	testResult := &ConnectivityResult{}
	connectResult := &testResult.Attempts
	ipIndex := 0
	done := make(chan bool)
	proceed := make(chan bool, 1)
	var waitGroup sync.WaitGroup
	// Create a new context for canceling goroutines
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	proceed <- true
	testResult.StartTime = time.Now()
loop:
	for {
		select {
		case <-done:
			break loop
		case <-proceed:
			waitGroup.Add(1)
			attempt := &ConnectionAttempt{}
			go func(attempt *ConnectionAttempt) {
				defer waitGroup.Done()
				attempt.StartTime = time.Now()
				interceptDialer := transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
					// Captures the address of the first hop, before resolution.
					testResult.Endpoint = addr
					host, port, err := net.SplitHostPort(addr)
					if err != nil {
						cancel()
						done <- true
						return nil, err
					}
					ips, err := (&net.Resolver{PreferGo: false}).LookupHost(ctx, host)
					if err != nil {
						cancel()
						done <- true
						return nil, err
					}
					var conn net.Conn
					if ipIndex < len(ips) {
						// proceed to setting up the next test
						proceed <- true
						ip := ips[ipIndex]
						ipIndex++
						//fmt.Printf("Trying address %v\n", ip)
						addr = net.JoinHostPort(ip, port)
						attempt.Address = addr
						conn, err = baseDialer.DialPacket(ctx, addr)
						if err != nil {
							return nil, err
						}
						return conn, err
					} else {
						// stop iterating
						done <- true
						return nil, ErrNoMoreIPs
					}
				})
				dialer, err := wrap(interceptDialer)
				if err != nil {
					fmt.Println("wrap failed...")
					*connectResult = append(*connectResult, *attempt)
					return
				}
				resolverConn, err := dialer.DialPacket(ctx, resolverAddress)
				if err != nil {
					// Do not include cencelled errors in the result
					// This never gets triggered in PacketDialer since
					// connect is not a blocking operation; we can remove it later
					if errors.Is(err, context.Canceled) {
						return
					}
					// Do not include ErrNoMoreIPs type error in the attempt result
					if errors.Is(err, ErrNoMoreIPs) {
						return
					}
					attempt.Duration = time.Since(attempt.StartTime)
					attempt.Error = makeConnectivityError("connect", err)
					*connectResult = append(*connectResult, *attempt)
					// CHANGE: populate main test result error field with
					// one of attempt errors if non of the attempts succeeded
					testResult.Error = attempt.Error
					return
				}
				resolver := dns.NewUDPResolver(transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
					return resolverConn, nil
				}), resolverAddress)
				//resolver := dns.NewUDPResolver(dialer, resolverAddress)
				attempt.Error, err = TestConnectivityWithResolver(ctx, resolver, testDomain)
				if err != nil {
					//fmt.Printf("Test failed: %v\n", err)
					return
				}
				attempt.Duration = time.Since(attempt.StartTime)
				*connectResult = append(*connectResult, *attempt)
				if attempt.Error == nil {
					testResult.Error = nil
					// test has succeeded; cancel the rest of the goroutines
					cancel()
				} else {
					// CHANGE: populate main test result error field with
					// one of attempt errors if non of the attempts succeeded
					testResult.Error = attempt.Error
					//return
				}
			}(attempt)
		}
	}
	// TODO: error is always being returned as nil; must change this
	waitGroup.Wait()
	testResult.Duration = time.Since(testResult.StartTime)
	return testResult, nil
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

	_, err = resolver.Query(ctx, *q)

	if errors.Is(err, dns.ErrBadRequest) {
		// maybe change this to include err in report?
		return nil, err
	}
	// If the connection is force cancelled,
	// we don't want to report an error.
	if errors.Is(err, net.ErrClosed) {
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
