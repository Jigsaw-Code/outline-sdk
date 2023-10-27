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

package httpproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
)

type sanitizeErrorDialer struct {
	transport.StreamDialer
}

func isCancelledError(err error) bool {
	if err == nil {
		return false
	}
	// Works around the fact that DNS doesn't return typed errors.
	return errors.Is(err, context.Canceled) || strings.HasSuffix(err.Error(), "operation was canceled")
}

func (d *sanitizeErrorDialer) Dial(ctx context.Context, addr string) (transport.StreamConn, error) {
	conn, err := d.StreamDialer.Dial(ctx, addr)
	if isCancelledError(err) {
		return nil, context.Canceled
	}
	if err != nil {
		return nil, errors.New("base dial failed")
	}
	return conn, nil
}

type connectHandler struct {
	dialer *sanitizeErrorDialer
}

var _ http.Handler = (*connectHandler)(nil)

func (h *connectHandler) ServeHTTP(proxyResp http.ResponseWriter, proxyReq *http.Request) {
	if proxyReq.Method != http.MethodConnect {
		proxyResp.Header().Add("Allow", "CONNECT")
		http.Error(proxyResp, fmt.Sprintf("Method %v is not supported", proxyReq.Method), http.StatusMethodNotAllowed)
		return
	}
	// Validate the target address.
	_, portStr, err := net.SplitHostPort(proxyReq.Host)
	if err != nil {
		http.Error(proxyResp, fmt.Sprintf("Authority \"%v\" is not a valid host:port", proxyReq.Host), http.StatusBadRequest)
		return
	}
	if portStr == "" {
		// As per https://httpwg.org/specs/rfc9110.html#CONNECT.
		http.Error(proxyResp, "Port number must be specified", http.StatusBadRequest)
		return
	}

	// Dial the target.
	transportConfig := proxyReq.Header.Get("Transport")
	dialer, err := config.WrapStreamDialer(h.dialer, transportConfig)
	if err != nil {
		// Because we sanitize the base dialer error, it's safe to return error details here.
		http.Error(proxyResp, fmt.Sprintf("Invalid config in Transport header: %v", err), http.StatusBadRequest)
		return
	}
	targetConn, err := dialer.Dial(proxyReq.Context(), proxyReq.Host)
	if err != nil {
		http.Error(proxyResp, fmt.Sprintf("Failed to connect to %v: %v", proxyReq.Host, err), http.StatusServiceUnavailable)
		return
	}
	defer targetConn.Close()

	hijacker, ok := proxyResp.(http.Hijacker)
	if !ok {
		http.Error(proxyResp, "Webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	httpConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(proxyResp, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer httpConn.Close()

	// Inform the client that the connection has been established.
	httpConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))

	// Relay data between client and target in both directions.
	go func() {
		io.Copy(targetConn, httpConn)
		targetConn.CloseWrite()
	}()
	io.Copy(httpConn, targetConn)
	// httpConn is closed by the defer httpConn.Close() above.
}

// NewConnectHandler creates a [http.Handler] that handles CONNECT requests and forwards
// the requests using the given [transport.StreamDialer].
//
// Clients can specify a Transport header with a value of a transport config as specified in
// the [config] package to specify the transport for a given request.
//
// The resulting handler is currently vulnerable to probing attacks. It's ok as a localhost proxy
// but it may be vulnerable if used as a public proxy.
func NewConnectHandler(dialer transport.StreamDialer) http.Handler {
	// We sanitize the errors from the input Dialer because we don't want to leak sensitive details
	// of the base dialer (e.g. access key credentials) to the user.
	return &connectHandler{&sanitizeErrorDialer{dialer}}
}
