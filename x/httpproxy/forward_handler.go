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
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type forwardHandler struct {
	client http.Client
}

var _ http.Handler = (*forwardHandler)(nil)

func (h *forwardHandler) ServeHTTP(proxyResp http.ResponseWriter, proxyReq *http.Request) {
	if proxyReq.URL.Host == "" {
		http.Error(proxyResp, "Must specify an absolute request target", http.StatusNotFound)
		return
	}
	// We create a new request that uses a relative path + Host header, instead of the absolute URL in the proxy request.
	targetReq, err := http.NewRequestWithContext(proxyReq.Context(), proxyReq.Method, proxyReq.URL.String(), proxyReq.Body)
	if err != nil {
		http.Error(proxyResp, "Error creating target request", http.StatusInternalServerError)
		return
	}
	for key, values := range proxyReq.Header {
		for _, value := range values {
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
	_, err = io.Copy(proxyResp, targetResp.Body)
	if err != nil {
		http.Error(proxyResp, "Failed write response", http.StatusServiceUnavailable)
		return
	}
}

// NewForwardHandler creates a [http.Handler] that handles absolute HTTP requests using the given [http.Client].
func NewForwardHandler(dialer transport.StreamDialer) http.Handler {
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !strings.HasPrefix(network, "tcp") {
			return nil, fmt.Errorf("protocol not supported: %v", network)
		}
		return dialer.DialStream(ctx, addr)
	}
	return &forwardHandler{http.Client{Transport: &http.Transport{DialContext: dialContext}}}
}
