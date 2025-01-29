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
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	// TODO(fortuna): Implement the test with gorilla instead.
	"github.com/coder/websocket"
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
		clientConn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Log("Failed to accept Websocket connection", "error", err)
			http.Error(w, "Failed to accept Websocket connection", http.StatusBadGateway)
			return
		}
		clientConn.SetReadLimit(-1)
		defer clientConn.CloseNow()

		// Handle client -> target.
		readClientDone := make(chan struct{})
		go func() {
			defer close(readClientDone)
			defer clientConn.CloseRead(r.Context())
			for {
				msgType, msg, err := clientConn.Read(r.Context())
				if err != nil {
					if !errors.Is(err, io.EOF) {
						t.Log("Failed to read from client", "error", err)
						clientConn.Close(websocket.StatusInternalError, "failed to read from client")
					}
					break
				}
				require.Equal(t, websocket.MessageBinary, msgType)
				if _, err := toTargetWriter.Write(msg); err != nil {
					t.Log("Failed to write to target", "error", err)
					clientConn.Close(websocket.StatusInternalError, "failed to write message to target")
					break
				}
			}
		}()
		// Handle target -> client
		func() {
			// About 2 MTUs
			buf := make([]byte, 3000)
			for {
				n, err := fromTargetReader.Read(buf)
				if err != nil {
					if !errors.Is(err, io.EOF) {
						t.Log("Failed to read from target", "error", err)
						clientConn.Close(websocket.StatusInternalError, "failed to read message from target")
					}
					break
				}
				read := buf[:n]
				if err := clientConn.Write(r.Context(), websocket.MessageBinary, read); err != nil {
					t.Log("Failed to write to client", "error", err)
					clientConn.Close(websocket.StatusInternalError, "failed to write message to client")
					break
				}
			}
		}()
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
	connect, err := NewStreamEndpoint("wss"+ts.URL[5:]+"/tcp", &transport.TCPDialer{}, client.Transport.(*http.Transport).TLSClientConfig)
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
		clientConn, err := websocket.Accept(w, r, nil)
		require.NoError(t, err)
		defer clientConn.CloseNow()

		msgType, msg, err := clientConn.Read(r.Context())
		require.NoError(t, err)
		require.Equal(t, websocket.MessageBinary, msgType)
		require.Equal(t, []byte("Request"), msg)

		err = clientConn.Write(r.Context(), websocket.MessageBinary, []byte("Response"))
		require.NoError(t, err)

		clientConn.Close(websocket.StatusNormalClosure, "")
	})
	mux.Handle("/udp", http.StripPrefix("/udp", handler))
	ts := httptest.NewUnstartedServer(mux)
	ts.EnableHTTP2 = true
	ts.StartTLS()
	defer ts.Close()

	client := ts.Client()
	connect, err := NewPacketEndpoint("wss"+ts.URL[5:]+"/udp", &transport.TCPDialer{}, client.Transport.(*http.Transport).TLSClientConfig)
	require.NoError(t, err)
	require.NotNil(t, connect)

	conn, err := connect(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conn)

	n, err := conn.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)

	resp, err := io.ReadAll(conn)
	require.NoError(t, err)
	require.Equal(t, []byte("Response"), resp)
}
