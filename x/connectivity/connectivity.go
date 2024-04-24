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
	"syscall"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/dns"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/dns/dnsmessage"
)

// ConnectivityResult captures the observed result of the connectivity test.
type ConnectivityResult struct {
	// Address we connected to
	Connections []ConnectionResult
	// Address of the connection that was selected
	SelectedAddress string
	// Observed error
	Error *ConnectivityError
}

type ConnectionResult struct {
	// Address we connected to
	Address string
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

// TestStreamConnectivityWithDNS tests weather we can get a response from a DNS resolver at resolverAddress over a stream connection. It sends testDomain as the query.
// It uses the baseDialer to create a first-hop connection to the proxy, and the wrap to apply the transport.
// The baseDialer is typically TCPDialer, but it can be replaced for remote measurements.
func TestStreamConnectivityWithDNS(ctx context.Context, baseDialer transport.StreamDialer, wrap WrapStreamDialer, resolverAddress string, testDomain string) (*ConnectivityResult, error) {
	testResult := &ConnectivityResult{
		Connections: make([]ConnectionResult, 0),
	}
	interceptDialer := transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := (&net.Resolver{PreferGo: false}).LookupHost(ctx, host)
		var conn transport.StreamConn
		for _, ip := range ips {
			addr := net.JoinHostPort(ip, port)
			connResult := ConnectionResult{Address: addr}
			deadline := time.Now().Add(5 * time.Second)
			ipCtx, cancel := context.WithDeadline(ctx, deadline)
			defer cancel()
			conn, err = baseDialer.DialStream(ipCtx, addr)
			if err != nil {
				connResult.Error = makeConnectivityError("connect", err)
			}
			testResult.Connections = append(testResult.Connections, connResult)
			if err == nil {
				testResult.SelectedAddress = addr
				break
			}
		}
		return conn, err
	})
	dialer, err := wrap(interceptDialer)
	if err != nil {
		return nil, err
	}
	resolverConn, err := dialer.DialStream(ctx, resolverAddress)
	if err != nil {
		return nil, err
	}
	resolver := dns.NewTCPResolver(transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		return resolverConn, nil
	}), resolverAddress)
	testResult.Error, err = TestConnectivityWithResolver(ctx, resolver, testDomain)
	if err != nil {
		return nil, err
	}
	return testResult, nil
}

type WrapPacketDialer func(baseDialer transport.PacketDialer) (transport.PacketDialer, error)

// TestPacketConnectivityWithDNS tests weather we can get a response from a DNS resolver at resolverAddress over a packet connection. It sends testDomain as the query.
// It uses the baseDialer to create a first-hop connection to the proxy, and the wrap to apply the transport.
// The baseDialer is typically UDPDialer, but it can be replaced for remote measurements.
func TestPacketConnectivityWithDNS(ctx context.Context, baseDialer transport.PacketDialer, wrap WrapPacketDialer, resolverAddress string, testDomain string) (*ConnectivityResult, error) {
	testResult := &ConnectivityResult{}
	interceptDialer := transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := (&net.Resolver{PreferGo: false}).LookupHost(ctx, host)
		var conn net.Conn
		for _, ip := range ips {
			addr := net.JoinHostPort(ip, port)
			connResult := ConnectionResult{Address: addr}
			conn, err = baseDialer.DialPacket(ctx, addr)
			if err != nil {
				connResult.Error = makeConnectivityError("connect", err)
			}
			testResult.Connections = append(testResult.Connections, connResult)
			if err == nil {
				testResult.SelectedAddress = addr
				break
			}
		}
		return conn, err
	})
	dialer, err := wrap(interceptDialer)
	if err != nil {
		return nil, err
	}
	resolver := dns.NewUDPResolver(dialer, resolverAddress)
	testResult.Error, err = TestConnectivityWithResolver(ctx, resolver, testDomain)
	return testResult, err
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
