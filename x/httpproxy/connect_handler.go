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
	"net/http"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type handler struct {
	dialer transport.StreamDialer
}

var _ http.Handler = (*handler)(nil)

// handleConnect handles the HTTP CONNECT method.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		http.Error(w, fmt.Sprintf("Method %v is not supported", r.Method), http.StatusMethodNotAllowed)
		return
	}

	// Dial the target
	targetConn, err := h.dialer.Dial(r.Context(), r.Host)
	if err != nil {
		http.Error(w, "Failed to connect to target", http.StatusServiceUnavailable)
		return
	}
	defer targetConn.Close()

	// Inform the client that the connection has been established
	httpConn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		http.Error(w, "Failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer httpConn.Close()

	httpConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))

	// Relay data between client and target in both directions
	go func() {
		io.Copy(targetConn, httpConn)
		targetConn.CloseWrite()
	}()
	io.Copy(httpConn, targetConn)
	httpConn.Close()
}

func NewConnectHandler(dialer transport.StreamDialer) http.Handler {
	return &handler{dialer}
}
