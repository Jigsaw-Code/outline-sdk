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

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	"github.com/miekg/dns"
)

// TestError captures the observed error of the connectivity test.
type TestError struct {
	// Which operation in the test that failed: "dial", "write" or "read"
	Op string
	// The POSIX error, when available
	PosixError string
	// The error observed for the action
	Err error
}

var _ error = (*TestError)(nil)

func (err *TestError) Error() string {
	return fmt.Sprintf("%v: %v", err.Op, err.Err)
}

func (err *TestError) Unwrap() error {
	return err.Err
}

// TestStreamDialerConnectivity uses the given [transport.StreamDialer] to resolve the test domain using the given DNS resolver.
// The context can be used to set a timeout or deadline, or to pass values to the dialer.
func TestStreamDialerConnectivity(ctx context.Context, dialer transport.StreamDialer, resolverAddress string, testDomain string) (time.Duration, error) {
	dial := func(ctx context.Context, address string) (net.Conn, error) { return dialer.Dial(ctx, resolverAddress) }
	return testDialer(ctx, dial, resolverAddress, testDomain)
}

// TestPacketDialerConnectivity uses the given [transport.PacketDialer] to resolve the test domain using the given DNS resolver.
// The context can be used to set a timeout or deadline, or to pass values to the listener.
func TestPacketDialerConnectivity(ctx context.Context, dialer transport.PacketDialer, resolverAddress string, testDomain string) (time.Duration, error) {
	dial := func(ctx context.Context, address string) (net.Conn, error) { return dialer.Dial(ctx, resolverAddress) }
	return testDialer(ctx, dial, resolverAddress, testDomain)
}

func isTimeout(err error) bool {
	var timeErr interface{ Timeout() bool }
	return errors.As(err, &timeErr) && timeErr.Timeout()
}

func makeTestError(op string, err error) error {
	var code string
	var errno syscall.Errno
	if errors.As(err, &errno) {
		code = errnoName(errno)
	} else if isTimeout(err) {
		code = "ETIMEDOUT"
	}
	return &TestError{Op: op, PosixError: code, Err: err}
}

func testDialer(ctx context.Context, dial func(context.Context, string) (net.Conn, error), resolverAddress string, testDomain string) (time.Duration, error) {
	deadline, ok := ctx.Deadline()
	if !ok {
		// Default deadline is 5 seconds.
		deadline = time.Now().Add(5 * time.Second)
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		// Releases the timer.
		defer cancel()
	}
	testTime := time.Now()
	testErr := func() error {
		conn, dialErr := dial(ctx, resolverAddress)
		if dialErr != nil {
			return makeTestError("dial", dialErr)
		}
		defer conn.Close()
		conn.SetDeadline(deadline)
		dnsConn := dns.Conn{Conn: conn}

		var dnsRequest dns.Msg
		dnsRequest.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)
		writeErr := dnsConn.WriteMsg(&dnsRequest)
		if writeErr != nil {
			return makeTestError("write", writeErr)
		}

		_, readErr := dnsConn.ReadMsg()
		if readErr != nil {
			return makeTestError("read", readErr)
		}
		return nil
	}()
	duration := time.Since(testTime)
	return duration, testErr
}
