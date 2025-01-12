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

package httpproxy

import (
	"context"
	"encoding/base64"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestConnectClient(t *testing.T) {
	t.Parallel()

	host := "host:1234"
	creds := base64.StdEncoding.EncodeToString([]byte("username:password"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodConnect, r.Method, "Method")
		require.Equal(t, host, r.Host, "Host")
		require.Equal(t, []string{"Basic " + creds}, r.Header["Proxy-Authorization"], "Proxy-Authorization")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		require.NoError(t, err, "Write")
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	require.NoError(t, err, "Parse")

	endpoint := &transport.TCPEndpoint{
		Dialer:  net.Dialer{},
		Address: u.Host,
	}

	connClient, err := NewConnectClient(endpoint, WithProxyAuthorization("Basic "+creds))
	require.NoError(t, err, "NewConnectClient")

	streamConn, err := connClient.DialStream(context.Background(), host)
	require.NoError(t, err, "DialStream")
	require.NotNil(t, streamConn, "StreamConn")
}
