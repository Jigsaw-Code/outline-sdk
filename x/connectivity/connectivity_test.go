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
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StreamDialer Tests
func TestTestStreamDialerConnectivityOk(t *testing.T) {
	// TODO(fortuna): Run a local resolver and make test not depend on an external server.
	dialer := &transport.TCPStreamDialer{}
	_, err := TestStreamDialerConnectivity(context.Background(), dialer, "8.8.8.8:53", "example.com")
	require.Nil(t, err)
}

// TODO: Move this to the SDK.
func runTestTCPServer(tb testing.TB, handle func(conn *net.TCPConn), running *sync.WaitGroup) *net.TCPListener {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IP{127, 0, 0, 1}})
	tb.Logf("Listening on %v", listener.Addr().String())
	require.Nil(tb, err)

	running.Add(1)
	go func() {
		defer running.Done()
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				assert.ErrorIs(tb, err, net.ErrClosed)
				break
			}
			defer conn.Close()
			handle(conn)
		}
	}()

	return listener
}

func TestTestStreamDialerConnectivityRefused(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IP{127, 0, 0, 1}})
	require.Nil(t, err)
	// Close right away to ensure the port is closed. The OS will likely not reuse it soon enough.
	require.Nil(t, listener.Close())

	dialer := &transport.TCPStreamDialer{}
	_, err = TestStreamDialerConnectivity(context.Background(), dialer, listener.Addr().String(), "anything")
	var testErr *TestError
	require.ErrorAs(t, err, &testErr)
	require.Equal(t, "dial", testErr.Op)
	require.Equal(t, "ECONNREFUSED", testErr.PosixError)

	var sysErr *os.SyscallError
	require.ErrorAs(t, err, &sysErr)
	expectedSyscall := "connect"
	if runtime.GOOS == "windows" {
		expectedSyscall = "connectex"
	}
	require.Equal(t, expectedSyscall, sysErr.Syscall)

	var errno syscall.Errno
	require.ErrorAs(t, sysErr.Err, &errno)
	require.Equal(t, "ECONNREFUSED", errnoName(errno))
}

func TestTestStreamDialerConnectivityReset(t *testing.T) {
	var running sync.WaitGroup
	listener := runTestTCPServer(t, func(conn *net.TCPConn) {
		// Wait for some data from client. We read one byte to unblock the client write.
		// With localhost sockets, the OS may short-circuit communication with a pipe in a way that
		// not reading any data may keep the client blocked on the write, causing inconsistent
		// TestErr.Op results across OSes.
		_, err := conn.Read(make([]byte, 1))
		require.ErrorIs(t, err, nil)
		// This forces a reset when the connection is closed and there's data not acknowledged.
		conn.SetLinger(0)
		require.Nil(t, conn.Close())
	}, &running)
	defer listener.Close()

	dialer := &transport.TCPStreamDialer{}
	_, err := TestStreamDialerConnectivity(context.Background(), dialer, listener.Addr().String(), "anything")

	var testErr *TestError
	require.ErrorAs(t, err, &testErr)
	require.Equalf(t, "read", testErr.Op, "Wrong test operation. Error: %v", testErr.Err)
	require.Equal(t, "ECONNRESET", testErr.PosixError)

	var sysErr *os.SyscallError
	require.ErrorAs(t, err, &sysErr)
	expectedSyscall := "read"
	if runtime.GOOS == "windows" {
		expectedSyscall = "wsarecv"
	}
	require.Equalf(t, expectedSyscall, sysErr.Syscall, "Wrong system call. Error: %v", sysErr)

	var errno syscall.Errno
	require.ErrorAs(t, err, &errno)
	require.Equal(t, "ECONNRESET", errnoName(errno))
}

func TestTestStreamDialerConnectivityTimeout(t *testing.T) {
	var running sync.WaitGroup
	var timeout sync.WaitGroup
	timeout.Add(1)
	listener := runTestTCPServer(t, func(conn *net.TCPConn) {
		defer conn.Close()
		timeout.Wait()
	}, &running)
	defer listener.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	dialer := &transport.TCPStreamDialer{}
	_, err := TestStreamDialerConnectivity(ctx, dialer, listener.Addr().String(), "anything")

	var testErr *TestError
	require.ErrorAs(t, err, &testErr)
	assert.Equalf(t, "read", testErr.Op, "Wrong test operation. Error: %v", testErr.Err)

	assert.ErrorContains(t, err, "i/o timeout")
	assert.True(t, isTimeout(err))
	assert.Equalf(t, "ETIMEDOUT", testErr.PosixError, "Wrong posix error code. Error: %#v, %v", testErr.Err, testErr.Err.Error())

	timeout.Done()
	listener.Close()
	running.Wait()
}

// PacketDialer tests

func TestTestPacketPacketConnectivityOk(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.ErrorIs(t, err, nil)
	defer server.Close()

	go func() {
		buf := make([]byte, 512)
		n, clientAddr, err := server.ReadFrom(buf)
		require.ErrorIs(t, err, nil)
		var request dns.Msg
		err = request.Unpack(buf[:n])
		require.ErrorIs(t, err, nil)

		var response dns.Msg
		response.SetReply(&request)
		responseBytes, err := response.Pack()
		require.ErrorIs(t, err, nil)
		_, err = server.WriteTo(responseBytes, clientAddr)
		require.ErrorIs(t, err, nil)
	}()

	dialer := &transport.UDPPacketDialer{}
	_, err = TestPacketDialerConnectivity(context.Background(), dialer, server.LocalAddr().String(), "example.com")
	require.ErrorIs(t, err, nil)
}

// TODO: Add more tests
