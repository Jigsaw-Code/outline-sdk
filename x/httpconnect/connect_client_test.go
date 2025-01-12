// Copyright 2023 The Outline Authors
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

package httpconnect

import (
	"bufio"
	"context"
	"encoding/base64"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestConnectClientOk(t *testing.T) {
	t.Parallel()

	creds := base64.StdEncoding.EncodeToString([]byte("username:password"))

	targetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method, "Method")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("HTTP/1.1 200 OK\r\n"))
		require.NoError(t, err)
	}))
	defer targetSrv.Close()

	targetURL, err := url.Parse(targetSrv.URL)
	require.NoError(t, err)

	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodConnect, r.Method, "Method")
		require.Equal(t, targetURL.Host, r.Host, "Host")
		require.Equal(t, []string{"Basic " + creds}, r.Header["Proxy-Authorization"], "Proxy-Authorization")

		conn, err := net.Dial("tcp", targetURL.Host)
		require.NoError(t, err, "Dial")

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		require.NoError(t, err, "Write")

		rc := http.NewResponseController(w)
		err = rc.Flush()
		require.NoError(t, err, "Flush")

		clientConn, _, err := rc.Hijack()
		require.NoError(t, err, "Hijack")

		go func() {
			_, _ = io.Copy(conn, clientConn)
		}()
		_, _ = io.Copy(clientConn, conn)
	}))
	defer proxySrv.Close()

	proxyURL, err := url.Parse(proxySrv.URL)
	require.NoError(t, err, "Parse")

	dialer := &transport.TCPDialer{
		Dialer: net.Dialer{},
	}

	connClient, err := NewConnectClient(
		dialer,
		proxyURL.Host,
		WithHeaders(http.Header{"Proxy-Authorization": []string{"Basic " + creds}}),
	)
	require.NoError(t, err, "NewConnectClient")

	streamConn, err := connClient.DialStream(context.Background(), targetURL.Host)
	require.NoError(t, err, "DialStream")
	require.NotNil(t, streamConn, "StreamConn")

	req, err := http.NewRequest(http.MethodGet, targetSrv.URL, nil)
	require.NoError(t, err, "NewRequest")

	err = req.Write(streamConn)
	require.NoError(t, err, "Write")

	resp, err := http.ReadResponse(bufio.NewReader(streamConn), req)
	require.NoError(t, err, "ReadResponse")

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestConnectClientFail(t *testing.T) {
	t.Parallel()

	targetURL := "somehost:1234"

	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodConnect, r.Method, "Method")
		require.Equal(t, targetURL, r.Host, "Host")

		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("HTTP/1.1 400 Bad request\r\n\r\n"))
		require.NoError(t, err, "Write")
	}))
	defer proxySrv.Close()

	proxyURL, err := url.Parse(proxySrv.URL)
	require.NoError(t, err, "Parse")

	dialer := &transport.TCPDialer{
		Dialer: net.Dialer{},
	}

	connClient, err := NewConnectClient(
		dialer,
		proxyURL.Host,
	)
	require.NoError(t, err, "NewConnectClient")

	_, err = connClient.DialStream(context.Background(), targetURL)
	require.Error(t, err, "unexpected status code: 400")
}
