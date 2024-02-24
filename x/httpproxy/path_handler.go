// Copyright 2024 Jigsaw Operations LLC
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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type pathHandler struct {
	client http.Client
}

var _ http.Handler = (*pathHandler)(nil)

func (h *pathHandler) ServeHTTP(proxyResp http.ResponseWriter, proxyReq *http.Request) {
	requestURL := strings.TrimPrefix(proxyReq.URL.Path, "/")
	if requestURL == "" {
		http.Error(proxyResp, "Empty URL", http.StatusBadRequest)
		return
	}
	if proxyReq.URL.RawQuery != "" {
		requestURL += "?" + proxyReq.URL.RawQuery
	}
	targetURL, err := url.Parse(requestURL)
	if err != nil {
		http.Error(proxyResp, "Invalid target URL", http.StatusBadRequest)
		return
	}
	// We create a new request that uses the path of the proxy request.
	targetReq, err := http.NewRequestWithContext(proxyReq.Context(), proxyReq.Method, targetURL.String(), proxyReq.Body)
	if err != nil {
		http.Error(proxyResp, "Error creating target request", http.StatusInternalServerError)
		return
	}
	for key, values := range proxyReq.Header {
		for _, value := range values {
			// Host header is set by the HTTP client in client.Do.
			targetReq.Header.Add(key, value)
		}
	}
	targetResp, err := h.client.Do(targetReq)
	if err != nil {
		http.Error(proxyResp, "Failed to fetch destination", http.StatusServiceUnavailable)
		return
	}
	defer targetResp.Body.Close()
	for key, values := range targetResp.Header {
		for _, value := range values {
			proxyResp.Header().Add(key, value)
		}
	}
	proxyResp.WriteHeader(targetResp.StatusCode)
	_, err = io.Copy(proxyResp, targetResp.Body)
	if err != nil {
		http.Error(proxyResp, "Failed write response", http.StatusServiceUnavailable)
		return
	}
}

// NewPathHandler creates a [http.Handler] that resolves the URL path as an absolute URL using the given [http.Client].
func NewPathHandler(dialer transport.StreamDialer) http.Handler {
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !strings.HasPrefix(network, "tcp") {
			return nil, fmt.Errorf("protocol not supported: %v", network)
		}
		return dialer.DialStream(ctx, addr)
	}
	return &pathHandler{http.Client{Transport: &http.Transport{DialContext: dialContext}}}
}
