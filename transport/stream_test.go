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

package transport

import (
	"context"
	"errors"
	"net"
	"sync"
	"syscall"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTCPStreamDialerIPv4(t *testing.T) {
	requestText := []byte("Request")
	responseText := []byte("Response")

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.Nilf(t, err, "Failed to create TCP listener: %v", err)
	defer listener.Close()

	var running sync.WaitGroup
	running.Add(2)

	// Server
	go func() {
		defer running.Done()
		clientConn, err := listener.AcceptTCP()
		require.Nilf(t, err, "AcceptTCP failed: %v", err)

		defer clientConn.Close()
		err = iotest.TestReader(clientConn, requestText)
		assert.Nilf(t, err, "Request read failed: %v", err)

		// This works on Linux, but on macOS it errors with "shutdown: socket is not connected" (syscall.ENOTCONN).
		// It seems that on macOS you cannot call CloseRead() if you've already received a FIN and read all the data.
		// TODO(fortuna): Consider wrapping StreamConns on macOS to make CloseRead a no-op if Read has returned io.EOF
		// or WriteTo has been called.
		// err = clientConn.CloseRead()
		// assert.Nilf(t, err, "clientConn.CloseRead failed: %v", err)

		_, err = clientConn.Write(responseText)
		assert.Nilf(t, err, "Write failed: %v", err)

		err = clientConn.CloseWrite()
		assert.Nilf(t, err, "CloseWrite failed: %v", err)
	}()

	// Client
	go func() {
		defer running.Done()
		dialer := &TCPStreamDialer{}
		dialer.Dialer.Control = func(network, address string, c syscall.RawConn) error {
			require.Equal(t, "tcp4", network)
			require.Equal(t, listener.Addr().String(), address)
			return nil
		}
		serverConn, err := dialer.Dial(context.Background(), listener.Addr().String())
		require.Nil(t, err, "Dial failed")
		require.Equal(t, listener.Addr().String(), serverConn.RemoteAddr().String())
		defer serverConn.Close()

		n, err := serverConn.Write(requestText)
		require.Nil(t, err)
		require.Equal(t, 7, n)
		assert.Nil(t, serverConn.CloseWrite())

		err = iotest.TestReader(serverConn, responseText)
		require.Nilf(t, err, "Response read failed: %v", err)
		// See CloseRead comment on the server go-routine.
		// err = serverConn.CloseRead()
		// assert.Nilf(t, err, "serverConn.CloseRead failed: %v", err)
	}()

	running.Wait()
}

func TestNewTCPStreamDialerAddress(t *testing.T) {
	errCancel := errors.New("cancelled")
	dialer := &TCPStreamDialer{}

	dialer.Dialer.Control = func(network, address string, c syscall.RawConn) error {
		require.Equal(t, "tcp4", network)
		require.Equal(t, "8.8.8.8:53", address)
		return errCancel
	}
	_, err := dialer.Dial(context.Background(), "8.8.8.8:53")
	require.ErrorIs(t, err, errCancel)

	dialer.Dialer.Control = func(network, address string, c syscall.RawConn) error {
		require.Equal(t, "tcp6", network)
		require.Equal(t, "[2001:4860:4860::8888]:53", address)
		return errCancel
	}
	_, err = dialer.Dial(context.Background(), "[2001:4860:4860::8888]:53")
	require.ErrorIs(t, err, errCancel)
}

func TestDialStreamEndpointAddr(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.Nil(t, err, "Failed to create TCP listener")
	defer listener.Close()

	endpoint := TCPEndpoint{Address: listener.Addr().String()}
	endpoint.Dialer.Control = func(network, address string, c syscall.RawConn) error {
		require.Equal(t, "tcp4", network)
		require.Equal(t, listener.Addr().String(), address)
		return nil
	}
	conn, err := endpoint.Connect(context.Background())
	require.Nil(t, err)
	require.Equal(t, listener.Addr().String(), conn.RemoteAddr().String())
	require.Nil(t, conn.Close())
}
