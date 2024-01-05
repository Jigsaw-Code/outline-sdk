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

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/miekg/dns"
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

// Resolver encapsulates the DNS resolution logic for connectivity tests.
type Resolver func(context.Context) (net.Conn, error)

func (r Resolver) connect(ctx context.Context) (*dns.Conn, error) {
	conn, err := r(ctx)
	if err != nil {
		return nil, err
	}
	return &dns.Conn{Conn: conn}, nil
}

// NewTCPResolver creates a [Resolver] to test StreamDialers.
func NewTCPResolver(dialer transport.StreamDialer, resolverAddr string) Resolver {
	endpoint := transport.StreamDialerEndpoint{Dialer: dialer, Address: resolverAddr}
	return Resolver(func(ctx context.Context) (net.Conn, error) {
		return endpoint.Connect(ctx)
	})
}

// NewUDPResolver creates a [Resolver] to test PacketDialers.
func NewUDPResolver(dialer transport.PacketDialer, resolverAddr string) Resolver {
	endpoint := transport.PacketDialerEndpoint{Dialer: dialer, Address: resolverAddr}
	return Resolver(func(ctx context.Context) (net.Conn, error) {
		return endpoint.Connect(ctx)
	})
}

func isTimeout(err error) bool {
	var timeErr interface{ Timeout() bool }
	return errors.As(err, &timeErr) && timeErr.Timeout()
}

func makeConnectivityError(op string, err error) *ConnectivityError {
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
func TestConnectivityWithResolver(ctx context.Context, resolver Resolver, testDomain string) (*ConnectivityError, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		// Default deadline is 5 seconds.
		deadline = time.Now().Add(5 * time.Second)
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		// Releases the timer.
		defer cancel()
	}

	dnsConn, err := resolver.connect(ctx)
	if err != nil {
		return makeConnectivityError("connect", err), nil
	}
	defer dnsConn.Close()
	dnsConn.SetDeadline(deadline)

	var dnsRequest dns.Msg
	dnsRequest.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)
	if err = dnsConn.WriteMsg(&dnsRequest); err != nil {
		return makeConnectivityError("send", err), nil
	}

	if _, err = dnsConn.ReadMsg(); err != nil {
		// An early close on the connection may cause a "unexpected EOF" error. That's an application-layer error,
		// not triggered by a syscall error so we don't capture an error code.
		// TODO: figure out how to standardize on those errors.
		return makeConnectivityError("receive", err), nil
	}
	return nil, nil
}
