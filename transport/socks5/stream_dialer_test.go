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

package socks5

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/things-go/go-socks5"
)

func TestSOCKS5Dialer_NewStreamDialerNil(t *testing.T) {
	dialer, err := NewStreamDialer(nil)
	require.Nil(t, dialer)
	require.Error(t, err)
}

func TestSOCKS5Dialer_BadConnection(t *testing.T) {
	dialer, err := NewStreamDialer(&transport.TCPEndpoint{Address: "127.0.0.0:0"})
	require.NotNil(t, dialer)
	require.NoError(t, err)
	_, err = dialer.DialStream(context.Background(), "example.com:443")
	require.Error(t, err)
}

func TestSOCKS5Dialer_BadAddress(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err, "Failed to create TCP listener: %v", err)
	defer listener.Close()

	dialer, err := NewStreamDialer(&transport.TCPEndpoint{Address: listener.Addr().String()})
	require.NotNil(t, dialer)
	require.NoError(t, err)

	_, err = dialer.DialStream(context.Background(), "noport")
	require.Error(t, err)
}

func TestSOCKS5Dialer_DialAddressTypes(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err, "Failed to create TCP listener: %v", err)
	defer listener.Close()

	testExchange(t, listener, "example.com:443", []byte("Request"), []byte("Response"), 0)
	testExchange(t, listener, "8.8.8.8:444", []byte("Request"), []byte("Response"), 0)
	testExchange(t, listener, "[2001:4860:4860::8888]:853", []byte("Request"), []byte("Response"), 0)
}

func TestSOCKS5Dialer_DialError(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err, "Failed to create TCP listener: %v", err)
	defer listener.Close()

	for _, replyCode := range []ReplyCode{
		ErrGeneralServerFailure,
		ErrConnectionNotAllowedByRuleset,
		ErrNetworkUnreachable,
		ErrHostUnreachable,
		ErrConnectionRefused,
		ErrTTLExpired,
		ErrCommandNotSupported,
		ErrAddressTypeNotSupported,
		ReplyCode(0xff),
	} {
		t.Run(fmt.Sprintf("ReplyCode=%v", replyCode), func(t *testing.T) {
			testExchange(t, listener, "example.com:443", nil, nil, ErrGeneralServerFailure)
		})
	}
}

func testExchange(tb testing.TB, listener *net.TCPListener, destAddr string, request []byte, response []byte, replyCode ReplyCode) {
	var running sync.WaitGroup
	running.Add(2)

	// Client
	go func() {
		defer running.Done()
		dialer, err := NewStreamDialer(&transport.TCPEndpoint{Address: listener.Addr().String()})
		require.NoError(tb, err)
		serverConn, err := dialer.DialStream(context.Background(), destAddr)
		if replyCode != 0 {
			require.ErrorIs(tb, err, replyCode)
			var extractedReplyCode ReplyCode
			require.True(tb, errors.As(err, &extractedReplyCode))
			require.Equal(tb, replyCode, extractedReplyCode)
			return
		}
		require.NoError(tb, err, "Dial failed")
		require.Equal(tb, listener.Addr().String(), serverConn.RemoteAddr().String())
		defer serverConn.Close()

		n, err := serverConn.Write(request)
		require.NoError(tb, err)
		require.Equal(tb, len(request), n)
		assert.NoError(tb, serverConn.CloseWrite())

		err = iotest.TestReader(serverConn, response)
		require.NoError(tb, err, "Response read failed: %v", err)
	}()

	// Server
	go func() {
		defer running.Done()
		clientConn, err := listener.AcceptTCP()
		require.NoError(tb, err, "AcceptTCP failed: %v", err)
		defer clientConn.Close()

		// See https://datatracker.ietf.org/doc/html/rfc1928#autoid-3
		// This reads method and connect requests at once, demonstrating they are both sent before a server response.
		// Method request: VER = 5, NMETHODS = 1, METHODS = 0 (no auth)
		// Connect request: VER = 5, CMD = 1, RSV = 0, ATYP, DST.ADDR, DST.PORT
		expected := []byte{5, 1, 0, 5, 1, 0}
		expected, err = appendSOCKS5Address(expected, destAddr)
		require.NoError(tb, err)
		err = iotest.TestReader(io.LimitReader(clientConn, int64(len(expected))), expected)
		assert.NoError(tb, err, "Request read failed: %v", err)

		// Write the method and connect responses
		// Method response: VER = 5, METHOD = 0
		// Connect response: VER = 5, REP, RSV = 0, ATYP = 1 (IPv4), BND.ADDR, BND.PORT
		_, err = clientConn.Write([]byte{5, 0, 5, byte(replyCode), 0, 1, 0, 0, 0, 0, 0, 0})
		assert.NoError(tb, err, "Write failed: %v", err)

		if request != nil {
			err = iotest.TestReader(clientConn, request)
			assert.NoError(tb, err, "Request read failed: %v", err)
		}

		if response != nil {
			n, err := clientConn.Write(response)
			require.NoError(tb, err)
			require.Equal(tb, len(response), n)
		}

		// There's a race condition here. If the replyCode is an error, the client may close
		// the connection before we have a chance to close the write, resulting in the error
		// "shutdown: transport endpoint is not connected". For that reason we don't treat the
		// error as fatal.
		err = clientConn.CloseWrite()
		if err != nil {
			tb.Logf("CloseWrite failed: %v", err)
		}
	}()

	running.Wait()
}

func TestConnectWithoutAuth(t *testing.T) {
	// Create a SOCKS5 server
	server := socks5.NewServer()

	// Create SOCKS5 proxy on localhost with a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go func() {
		err := server.Serve(listener)
		defer listener.Close()
		t.Log("server is listening...")
		require.NoError(t, err)
	}()

	// wait for server to start
	time.Sleep(10 * time.Millisecond)

	address := listener.Addr().String()

	// Create a SOCKS5 client
	dialer, err := NewStreamDialer(&transport.TCPEndpoint{Address: address})
	require.NotNil(t, dialer)
	require.NoError(t, err)

	_, err = dialer.DialStream(context.Background(), address)
	require.NoError(t, err)
}

func TestConnectWithAuth(t *testing.T) {
	// Create a SOCKS5 server
	cator := socks5.UserPassAuthenticator{
		Credentials: socks5.StaticCredentials{
			"testusername": "testpassword",
		},
	}
	server := socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{cator}),
	)

	// Create SOCKS5 proxy on localhost with a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()

	// Create SOCKS5 proxy on localhost port 8001
	go func() {
		err := server.Serve(listener)
		defer listener.Close()
		require.NoError(t, err)
	}()
	// wait for server to start
	time.Sleep(10 * time.Millisecond)

	dialer, err := NewStreamDialer(&transport.TCPEndpoint{Address: address})
	require.NotNil(t, dialer)
	require.NoError(t, err)
	err = dialer.SetCredentials([]byte("testusername"), []byte("testpassword"))
	require.NoError(t, err)
	_, err = dialer.DialStream(context.Background(), address)
	require.NoError(t, err)

	// Try to connect with incorrect credentials
	err = dialer.SetCredentials([]byte("testusername"), []byte("wrongpassword"))
	require.NoError(t, err)
	_, err = dialer.DialStream(context.Background(), address)
	require.Error(t, err)
}
