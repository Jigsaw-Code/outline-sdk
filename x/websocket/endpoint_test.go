// Copyright 2025 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package websocket

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
)

func Test_NewStreamEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	toTargetReader, toTargetWriter := io.Pipe()
	fromTargetReader, fromTargetWriter := io.Pipe()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(fortuna): support h2 and h3 on the server.
		require.Equal(t, "", r.TLS.NegotiatedProtocol)
		require.Equal(t, "HTTP/1.1", r.Proto)
		t.Log("Got stream request", "request", r)
		defer t.Log("Request done")

		clientConn, err := Upgrade(w, r, http.Header{})
		if err != nil {
			t.Log("Failed to accept Websocket connection", "error", err)
			http.Error(w, "Failed to accept Websocket connection", http.StatusBadGateway)
			return
		}
		defer clientConn.Close()

		// Handle client -> target.
		readClientDone := make(chan struct{})
		go func() {
			defer close(readClientDone)
			defer toTargetWriter.Close()
			_, err := io.Copy(toTargetWriter, clientConn)
			require.NoError(t, err)
		}()
		// Handle target -> client
		_, err = io.Copy(clientConn, fromTargetReader)
		require.NoError(t, err)
		<-readClientDone
	})
	mux.Handle("/tcp", http.StripPrefix("/tcp", handler))
	ts := httptest.NewUnstartedServer(mux)
	ts.EnableHTTP2 = true
	ts.StartTLS()
	defer ts.Close()

	// Run server functionality.
	go func() {
		for {
			// Fits "Request\n"
			req := make([]byte, 8)
			n, err := io.ReadFull(toTargetReader, req)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					require.NoError(t, err)
				}
				break
			}
			require.Equal(t, "Request\n", string(req[:n]))

			n, err = fromTargetWriter.Write([]byte("Response\n"))
			require.NoError(t, err)
			require.Equal(t, 9, n)
		}
	}()

	// TODO(fortuna): Support h2. We can force h2 on the client with the code below.
	// client := &http.Client{
	// 	Transport: &http2.Transport{
	// 		TLSClientConfig: ts.Client().Transport.(*http.Transport).TLSClientConfig,
	// 	},
	// }
	client := ts.Client()
	endpoint := &transport.TCPEndpoint{Address: ts.Listener.Addr().String()}
	connect, err := NewStreamEndpoint("wss"+ts.URL[5:]+"/tcp", endpoint, WithTLSConfig(client.Transport.(*http.Transport).TLSClientConfig))
	require.NoError(t, err)
	require.NotNil(t, connect)

	conn, err := connect(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conn)
	defer conn.Close()

	for i := 0; i < 10; i++ {
		n, err := conn.Write([]byte("Req"))
		require.NoError(t, err)
		require.Equal(t, 3, n)
		n, err = conn.Write([]byte("uest\n"))
		require.NoError(t, err)
		require.Equal(t, 5, n)

		resp := make([]byte, 9)
		n, err = conn.Read(resp)
		require.NoError(t, err)
		require.Equal(t, "Response\n", string(resp[:n]))
	}
	require.NoError(t, conn.CloseWrite())
}

func Test_NewPacketEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(fortuna): support h2 and h3 on the server.
		require.Equal(t, "", r.TLS.NegotiatedProtocol)
		require.Equal(t, "HTTP/1.1", r.Proto)
		clientConn, err := Upgrade(w, r, http.Header{})
		require.NoError(t, err)
		defer clientConn.Close()

		buf := make([]byte, 8)
		n, err := clientConn.Read(buf)
		require.NoError(t, err)
		require.Equal(t, []byte("Request"), buf[:n])

		n, err = clientConn.Write([]byte("Response"))
		require.NoError(t, err)
		require.Equal(t, 8, n)
	})
	mux.Handle("/udp", http.StripPrefix("/udp", handler))
	ts := httptest.NewUnstartedServer(mux)
	ts.EnableHTTP2 = true
	ts.StartTLS()
	defer ts.Close()

	client := ts.Client()
	endpoint := &transport.TCPEndpoint{Address: ts.Listener.Addr().String()}
	connect, err := NewPacketEndpoint("wss"+ts.URL[5:]+"/udp", endpoint, WithTLSConfig(client.Transport.(*http.Transport).TLSClientConfig))
	require.NoError(t, err)
	require.NotNil(t, connect)

	conn, err := connect(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conn)

	n, err := conn.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)

	buf := make([]byte, 9)
	n, err = conn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, []byte("Response"), buf[:n])
}

// Test_ConcurrentWritePacket tests if gorillaConn can concurrently write packets.
func Test_ConcurrentWritePacket(t *testing.T) {
	const numWrites = 100
	recved := make([]bool, numWrites)
	allFalse := make([]bool, numWrites)
	allTrue := slices.Repeat([]bool{true}, numWrites)
	var reqRecved sync.WaitGroup
	reqRecved.Add(numWrites)

	ts, conn := setupAndConnectToTestUDPWebSocketServer(t, func(svrConn transport.StreamConn) {
		defer svrConn.Close()
		scan := bufio.NewScanner(svrConn)
		for scan.Scan() {
			nStr, ok := strings.CutPrefix(scan.Text(), "write-")
			require.True(t, ok)
			n, err := strconv.Atoi(nStr)
			require.NoError(t, err)
			if n > numWrites {
				break
			}
			recved[n] = true
			reqRecved.Done()
		}
		require.NoError(t, scan.Err())
	})
	defer ts.Close()

	// Concurrenly writes "write-xxx\n" messages
	require.Equal(t, allFalse, recved)
	for i := range numWrites {
		go func() {
			_, err := fmt.Fprintf(conn, "write-%d\n", i)
			require.NoError(t, err)
		}()
	}
	reqRecved.Wait()
	require.NoError(t, conn.Close())
	require.Equal(t, allTrue, recved)
}

// Test_ConcurrentCloseWritePacket tests if gorillaConn can concurrently be closed while writing.
func Test_ConcurrentCloseWritePacket(t *testing.T) {
	t.Skip("TODO: figure out a good way to synchronize CloseWrite and Writes")

	const numWrites = 100
	var writesDone sync.WaitGroup
	writesDone.Add(numWrites)

	ts, conn := setupAndConnectToTestUDPWebSocketServer(t, func(svrConn transport.StreamConn) {
		writesDone.Wait()
		svrConn.Close()
	})
	defer ts.Close()

	// Concurrently Close while writing
	for range numWrites {
		go func() {
			defer writesDone.Done()
			fmt.Fprintf(conn, "message\n")
		}()
	}
	require.NoError(t, conn.Close())
	writesDone.Wait()
}

// Test_ConcurrentReadPacket tests if gorillaConn can concurrently receive packets.
func Test_ConcurrentReadPacket(t *testing.T) {
	const numReads = 100
	recved := make([]bool, numReads)
	allFalse := make([]bool, numReads)
	allTrue := slices.Repeat([]bool{true}, numReads)
	var readsDone, testDone sync.WaitGroup
	readsDone.Add(numReads)
	testDone.Add(1)
	defer testDone.Done()

	ts, conn := setupAndConnectToTestUDPWebSocketServer(t, func(svrConn transport.StreamConn) {
		defer svrConn.Close()
		for i := range numReads {
			_, err := fmt.Fprintf(svrConn, "read-%d\n", i)
			require.NoError(t, err)
		}
		testDone.Wait()
	})
	defer ts.Close()

	// Concurrently reads "read-xxx\n" messages
	require.Equal(t, allFalse, recved)
	for range numReads {
		go func() {
			defer readsDone.Done()
			scan := bufio.NewScanner(conn)
			for scan.Scan() {
				nStr, ok := strings.CutPrefix(scan.Text(), "read-")
				require.True(t, ok)
				n, err := strconv.Atoi(nStr)
				require.NoError(t, err)
				recved[n] = true
				break
			}
		}()
	}
	readsDone.Wait()
	require.NoError(t, conn.Close())
	require.Equal(t, allTrue, recved)
}

// Test_ConcurrentCloseReadPacket tests if gorillaConn can concurrently be closed while reading.
func Test_ConcurrentCloseReadPacket(t *testing.T) {
	t.Skip("TODO: figure out a good way to synchronize CloseRead and Reads")

	const numReads = 100
	var readsDone sync.WaitGroup
	readsDone.Add(numReads)

	ts, conn := setupAndConnectToTestUDPWebSocketServer(t, func(svrConn transport.StreamConn) {
		readsDone.Wait()
		svrConn.Close()
	})
	defer ts.Close()

	// Concurrently Close while reading
	for range numReads {
		go func() {
			defer readsDone.Done()
			io.ReadAll(conn)
		}()
	}
	require.NoError(t, conn.Close())
	readsDone.Wait()
}

// --- Test Helpers ---

func setupAndConnectToTestUDPWebSocketServer(t *testing.T, server func(transport.StreamConn)) (ts *httptest.Server, conn net.Conn) {
	mux := http.NewServeMux()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svrConn, err := Upgrade(w, r, http.Header{})
		require.NoError(t, err)
		server(svrConn)
	})
	mux.Handle("/udp", http.StripPrefix("/udp", handler))
	ts = httptest.NewTLSServer(mux)

	client := ts.Client()
	endpoint := &transport.TCPEndpoint{Address: ts.Listener.Addr().String()}
	connect, err := NewPacketEndpoint("wss"+ts.URL[5:]+"/udp", endpoint, WithTLSConfig(client.Transport.(*http.Transport).TLSClientConfig))
	require.NoError(t, err)
	conn, err = connect(context.Background())
	require.NoError(t, err)

	return
}
