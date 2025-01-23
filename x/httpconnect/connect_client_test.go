// Copyright 2025 The Outline Authors
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
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/httpproxy"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func Test_ConnectClient_HTTP_Ok(t *testing.T) {
	t.Parallel()

	creds := base64.StdEncoding.EncodeToString([]byte("username:password"))

	targetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method, "Method")
	}))
	defer targetSrv.Close()

	targetURL, err := url.Parse(targetSrv.URL)
	require.NoError(t, err)

	tcpDialer := &transport.TCPDialer{Dialer: net.Dialer{}}
	connectHandler := httpproxy.NewConnectHandler(tcpDialer)
	proxySrv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "Basic "+creds, request.Header.Get("Proxy-Authorization"))
		connectHandler.ServeHTTP(writer, request)
	}))
	defer proxySrv.Close()

	proxyURL, err := url.Parse(proxySrv.URL)
	require.NoError(t, err, "Parse")

	connClient, err := NewConnectClient(
		tcpDialer,
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

func Test_ConnectClient_HTTP_Fail(t *testing.T) {
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

	connClient, err := NewConnectClient(
		&transport.TCPDialer{
			Dialer: net.Dialer{},
		},
		proxyURL.Host,
	)
	require.NoError(t, err, "NewConnectClient")

	_, err = connClient.DialStream(context.Background(), targetURL)
	require.Error(t, err, "unexpected status code: 400")
}

func Test_ConnectClient_HTTP2_Ok(t *testing.T) {
	t.Parallel()

	targetSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method, "Method")
		w.Header().Set("Content-Type", "text/plain")
		_, err := w.Write([]byte("Hello, world!"))
		require.NoError(t, err)
	}))
	defer targetSrv.Close()

	targetURL, err := url.Parse(targetSrv.URL)
	require.NoError(t, err)

	tcpDialer := &transport.TCPDialer{Dialer: net.Dialer{}}
	proxySrv := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "HTTP/2.0", request.Proto, "Proto")
		require.Equal(t, http.MethodConnect, request.Method, "Method")
		require.Equal(t, targetURL.Host, request.URL.Host, "Host")

		conn, err := tcpDialer.DialStream(request.Context(), request.URL.Host)
		require.NoError(t, err, "DialStream")

		writer.WriteHeader(http.StatusOK)
		writer.(http.Flusher).Flush()

		go func() {
			_, _ = io.Copy(conn, request.Body)
			require.NoError(t, err, "io.Copy")
		}()

		_, _ = io.Copy(writer, conn)
		require.NoError(t, err, "io.Copy")
	}))
	proxySrv.EnableHTTP2 = true
	proxySrv.StartTLS()
	defer proxySrv.Close()

	proxyURL, err := url.Parse(proxySrv.URL)
	require.NoError(t, err, "Parse")

	certs := x509.NewCertPool()
	for _, c := range proxySrv.TLS.Certificates {
		roots, err := x509.ParseCertificates(c.Certificate[len(c.Certificate)-1])
		require.NoError(t, err, "x509.ParseCertificates")
		for _, root := range roots {
			certs.AddCert(root)
		}
	}

	connClient, err := NewConnectClient(
		tcpDialer,
		proxyURL.Host,
		WithHTTPS(&tls.Config{RootCAs: certs}),
	)
	require.NoError(t, err, "NewConnectClient")

	streamConn, err := connClient.DialStream(context.Background(), targetURL.Host)
	require.NoError(t, err, "DialStream")
	require.NotNil(t, streamConn, "StreamConn")

	req, err := http.NewRequest(http.MethodGet, targetSrv.URL, nil)
	require.NoError(t, err, "NewRequest")
	req.Header.Add("Connection", "close")

	err = req.Write(streamConn)
	require.NoError(t, err, "Write")

	rd := bufio.NewReader(streamConn)
	resp, err := http.ReadResponse(rd, req)
	require.NoError(t, err, "ReadResponse")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "ReadAll")
	require.Equal(t, "Hello, world!", string(body))

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
