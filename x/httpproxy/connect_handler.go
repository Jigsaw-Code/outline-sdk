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
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type handler struct {
	dialer transport.StreamDialer
}

var _ http.Handler = (*handler)(nil)

// ServeHTTP implements [http.Handler].ServeHTTP for CONNECT requests, using the internal [transport.StreamDialer].
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO(fortuna): For public services (not local), we need authentication and drain on failures to avoid fingerprinting.
	if r.Method != http.MethodConnect {
		http.Error(w, fmt.Sprintf("Method %v is not supported", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Validate the target address.
	_, portStr, err := net.SplitHostPort(r.Host)
	if err != nil {
		http.Error(w, "Authority is not a valid host:port", http.StatusBadRequest)
		return
	}
	if portStr == "" {
		// As per https://httpwg.org/specs/rfc9110.html#CONNECT.
		http.Error(w, "Port number must be specified", http.StatusBadRequest)
		return
	}

	// Dial the target.
	targetConn, err := h.dialer.Dial(r.Context(), r.Host)
	if err != nil {
		http.Error(w, "Failed to connect to target", http.StatusServiceUnavailable)
		return
	}
	defer targetConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	httpConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
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
// The resulting handler is currently vulnerable to probing attacks. It's ok as a localhost proxy
// but it may be vulnerable if used as a public proxy.
func NewConnectHandler(dialer transport.StreamDialer) http.Handler {
	return &handler{dialer}
}
