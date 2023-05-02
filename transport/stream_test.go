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

	"github.com/stretchr/testify/require"
)

func TestNewTCPStreamDialerIPv4(t *testing.T) {
	requestText := []byte("Request")
	responseText := []byte("Response")

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 10)})
	require.Nil(t, err, "Failed to create TCP listener")

	var running sync.WaitGroup
	running.Add(1)
	go func() {
		defer running.Done()
		defer listener.Close()
		clientConn, err := listener.AcceptTCP()
		if err != nil {
			t.Errorf("AcceptTCP failed: %v", err)
			return
		}
		defer clientConn.Close()
		if err = iotest.TestReader(clientConn, requestText); err != nil {
			t.Errorf("Request read failed: %v", err)
			return
		}
		if err = clientConn.CloseRead(); err != nil {
			t.Errorf("CloseRead failed: %v", err)
			return
		}
		if _, err = clientConn.Write(responseText); err != nil {
			t.Errorf("Write failed: %v", err)
			return
		}
		if err = clientConn.CloseWrite(); err != nil {
			t.Errorf("CloseWrite failed: %v", err)
			return
		}
	}()

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

	serverConn.Write(requestText)
	serverConn.CloseWrite()

	require.Nil(t, iotest.TestReader(serverConn, responseText), "Response read failed")
	serverConn.CloseRead()

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
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 2)})
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
