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
	"log"
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
	ConnectionAddress string
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

func TestStreamConnectivityWithDNS(ctx context.Context, dialer transport.StreamDialer, resolverAddress string, testDomain string) (*ConnectivityResult, error) {
	result := &ConnectivityResult{}
	captureDialer := transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
		conn, err := dialer.DialStream(ctx, addr)
		if conn != nil {
			result.ConnectionAddress = conn.RemoteAddr().String()
			log.Println("address", result.ConnectionAddress)
		}
		return conn, err
	})
	resolver := dns.NewTCPResolver(captureDialer, resolverAddress)
	var err error
	result.Error, err = TestConnectivityWithResolver(ctx, resolver, testDomain)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func TestPacketConnectivityWithDNS(ctx context.Context, dialer transport.PacketDialer, resolverAddress string, testDomain string) (*ConnectivityResult, error) {
	result := &ConnectivityResult{}
	captureDialer := transport.FuncPacketDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		conn, err := dialer.DialPacket(ctx, addr)
		if conn != nil {
			// This doesn't work with the PacketListenerDialer we use because it returns the address of the target, not the proxy.
			// TODO(fortuna): make PLD use the first hop address or try something else.
			result.ConnectionAddress = conn.RemoteAddr().String()
			log.Println("address", result.ConnectionAddress)
		}
		return conn, err
	})
	resolver := dns.NewUDPResolver(captureDialer, resolverAddress)
	var err error
	result.Error, err = TestConnectivityWithResolver(ctx, resolver, testDomain)
	if err != nil {
		return nil, err
	}
	return result, nil
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
