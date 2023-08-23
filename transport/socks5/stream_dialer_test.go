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
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	_, err = dialer.Dial(context.Background(), "example.com:443")
	require.Error(t, err)
}

func TestSOCKS5Dialer_BadAddress(t *testing.T) {
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err, "Failed to create TCP listener: %v", err)
	defer listener.Close()

	dialer, err := NewStreamDialer(&transport.TCPEndpoint{Address: listener.Addr().String()})
	require.NotNil(t, dialer)
	require.NoError(t, err)

	_, err = dialer.Dial(context.Background(), "noport")
	require.Error(t, err)
}

func TestSOCKS5Dialer_Dial(t *testing.T) {
	requestText := []byte("Request")
	responseText := []byte("Response")

	for _, destAddress := range []string{"example.com:443", "8.8.8.8:444", "[2001:4860:4860::8888]:853"} {
		t.Run(fmt.Sprintf("addr=%v", destAddress), func(t *testing.T) {
			listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
			require.NoError(t, err, "Failed to create TCP listener: %v", err)
			defer listener.Close()

			var running sync.WaitGroup
			running.Add(2)

			// Client
			go func() {
				defer running.Done()
				dialer, err := NewStreamDialer(&transport.TCPEndpoint{Address: listener.Addr().String()})
				require.NoError(t, err)
				serverConn, err := dialer.Dial(context.Background(), destAddress)
				require.NoError(t, err, "Dial failed")
				require.Equal(t, listener.Addr().String(), serverConn.RemoteAddr().String())
				defer serverConn.Close()

				n, err := serverConn.Write(requestText)
				require.NoError(t, err)
				require.Equal(t, len(requestText), n)
				assert.NoError(t, serverConn.CloseWrite())

				err = iotest.TestReader(serverConn, responseText)
				require.NoError(t, err, "Response read failed: %v", err)
			}()

			// Server
			go func() {
				defer running.Done()
				clientConn, err := listener.AcceptTCP()
				require.NoError(t, err, "AcceptTCP failed: %v", err)
				defer clientConn.Close()

				// See https://datatracker.ietf.org/doc/html/rfc1928#autoid-3
				// This reads method and connect requests at once, demonstrating they are both sent before a server response.
				// Method request: VER = 5, NMETHODS = 1, METHODS = 0 (no auth)
				// Connect request: VER = 5, CMD = 1, RSV = 0, ATYP, DST.ADDR, DST.PORT
				expected := []byte{5, 1, 0, 5, 1, 0}
				expected, err = appendSOCKS5Address(expected, destAddress)
				require.NoError(t, err)
				err = iotest.TestReader(io.LimitReader(clientConn, int64(len(expected))), expected)
				assert.NoError(t, err, "Request read failed: %v", err)

				// Write the method and connect responses
				// Method response: VER = 5, METHOD = 0
				// Connect response: VER = 5, REP = 0 (success), RSV = 0, ATYP = 1 (IPv4), BND.ADDR, BND.PORT
				_, err = clientConn.Write([]byte{5, 0, 5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
				assert.NoError(t, err, "Write failed: %v", err)

				err = iotest.TestReader(clientConn, requestText)
				assert.NoError(t, err, "Request read failed: %v", err)

				n, err := clientConn.Write(responseText)
				require.NoError(t, err)
				require.Equal(t, len(responseText), n)

				err = clientConn.CloseWrite()
				assert.NoError(t, err, "CloseWrite failed: %v", err)
			}()

			running.Wait()
		})
	}
}
