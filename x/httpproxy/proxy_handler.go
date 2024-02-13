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
	"net/http"
	"net/url"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type proxyHandler struct {
	connectHandler http.Handler
	pathHandler    http.Handler
	forwardHandler http.Handler
}

// ServeHTTP implements [http.Handler].ServeHTTP for CONNECT and absolute URL requests, using the internal [transport.StreamDialer].
func (h *proxyHandler) ServeHTTP(proxyResp http.ResponseWriter, proxyReq *http.Request) {
	// TODO(fortuna): For public services (not local), we need authentication and drain on failures to avoid fingerprinting.
	if proxyReq.Method == http.MethodConnect {
		h.connectHandler.ServeHTTP(proxyResp, proxyReq)
		return
	}
	if proxyReq.URL.Host != "" {

		pathReqUrl, err := url.Parse(proxyReq.URL.Path)

		if err != nil && pathReqUrl.Scheme != "" {
			h.pathHandler.ServeHTTP(proxyResp, proxyReq)
			return
		}

		h.forwardHandler.ServeHTTP(proxyResp, proxyReq)
		return
	}
	http.Error(proxyResp, "Not Found", http.StatusNotFound)
}

// NewProxyHandler creates a [http.Handler] that works as a web proxy using the given dialer to deach the destination.
func NewProxyHandler(dialer transport.StreamDialer) http.Handler {
	return &proxyHandler{
		connectHandler: NewConnectHandler(dialer),
		pathHandler:    NewPathHandler(dialer),
		forwardHandler: NewForwardHandler(dialer),
	}
}
