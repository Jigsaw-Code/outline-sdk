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
	"io"
	"net"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSOCKS5Dialer_Dial(t *testing.T) {
	requestText := []byte("Request")
	responseText := []byte("Response")

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err, "Failed to create TCP listener: %v", err)
	defer listener.Close()

	var running sync.WaitGroup
	running.Add(2)

	// Server
	go func() {
		defer running.Done()
		clientConn, err := listener.AcceptTCP()
		require.NoError(t, err, "AcceptTCP failed: %v", err)

		defer clientConn.Close()
		// Client version (5) and 1 authentication method (0=no auth)
		err = iotest.TestReader(io.LimitReader(clientConn, 3), []byte{5, 1, 0})
		assert.NoError(t, err, "Request read failed: %v", err)

		// Version (5) and selected method (0)
		_, err = clientConn.Write([]byte{5, 0})
		assert.NoError(t, err, "Write failed: %v", err)

		// VER = 5, CMD = 1 (connect), RSV = 0, ATYP = 1 (IPv4), DST.ADDR, DST.PORT
		port := listener.Addr().(*net.TCPAddr).Port
		err = iotest.TestReader(io.LimitReader(clientConn, 10), []byte{5, 1, 0, 1, 127, 0, 0, 1, byte(port >> 8), byte(port & 0xFF)})
		assert.NoError(t, err, "Request read failed: %v", err)

		// VER = 5, REP = 0 (success), RSV = 0, ATYP = 1 (IPv4), DST.ADDR, DST.PORT
		_, err = clientConn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
		assert.NoError(t, err, "Write failed: %v", err)

		n, err := clientConn.Write(responseText)
		require.NoError(t, err)
		require.Equal(t, len(responseText), n)

		err = clientConn.CloseWrite()
		assert.NoError(t, err, "CloseWrite failed: %v", err)
	}()

	// Client
	go func() {
		defer running.Done()
		dialer, err := NewStreamDialer(&transport.TCPEndpoint{Address: listener.Addr().String()})
		require.NoError(t, err)
		serverConn, err := dialer.Dial(context.Background(), listener.Addr().String())
		require.NoError(t, err, "Dial failed")
		require.Equal(t, listener.Addr().String(), serverConn.RemoteAddr().String())
		defer serverConn.Close()

		n, err := serverConn.Write(requestText)
		require.NoError(t, err)
		require.Equal(t, len(requestText), n)
		assert.Nil(t, serverConn.CloseWrite())

		err = iotest.TestReader(serverConn, responseText)
		require.NoError(t, err, "Response read failed: %v", err)
	}()

	running.Wait()
}
