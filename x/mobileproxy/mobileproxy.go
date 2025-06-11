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

// Package mobileproxy provides convenience utilities to help applications run a local proxy
// and use that to configure their networking libraries.
//
// This package is suitable for use with Go Mobile, making it a convenient way to integrate with mobile apps.
package mobileproxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/x/httpproxy"
)

// Proxy enables you to get the actual address bound by the server and stop the service when no longer needed.
type Proxy struct {
	host         string
	port         int
	proxyHandler *httpproxy.ProxyHandler
	server       *http.Server
}

// Address returns the IP and port the server is bound to.
func (p *Proxy) Address() string {
	return net.JoinHostPort(p.host, strconv.Itoa(p.port))
}

// Host returns the IP the server is bound to.
func (p *Proxy) Host() string {
	return p.host
}

// Port returns the port the server is bound to.
func (p *Proxy) Port() int {
	return p.port
}

// AddURLProxy sets up a URL-based proxy handler that activates when an incoming HTTP request matches
// the specified path prefix. The pattern must represent a path segment, which is checked against
// the path of the incoming request.
//
// This function is particularly useful for libraries or components that accept URLs but do not support proxy
// configuration directly. By leveraging AddURLProxy, such components can route requests through a proxy by
// constructing URLs in the format "http://${HOST}:${PORT}/${PATH}/${URL}", where "${URL}" is the target resource.
// For instance, using "http://localhost:8080/proxy/https://example.com" routes the request for "https://example.com"
// through a proxy at "http://localhost:8080/proxy".
//
// The path should start with a forward slash ('/') for clarity, but one will be added if missing.
//
// The function associates the given 'dialer' with the specified 'path', allowing different dialers to be used for
// different path-based proxies within the same application in the future. currently we only support one URL proxy.
func (p *Proxy) AddURLProxy(path string, dialer *StreamDialer) {
	if p.proxyHandler == nil {
		// Called after Stop. Warn and ignore.
		log.Println("Called Proxy.AddURLProxy after Stop")
		return
	}
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}
	// TODO(fortuna): Add support for multiple paths. I tried http.ServeMux, but it does request sanitization,
	// which breaks the URL extraction: https://pkg.go.dev/net/http#hdr-Request_sanitizing.
	// We can consider forking http.StripPrefix to provide a fallback instead of NotFound, and chaing them.
	p.proxyHandler.FallbackHandler = http.StripPrefix(path, httpproxy.NewPathHandler(dialer.StreamDialer))
}

// Stop gracefully stops the proxy service, waiting for at most timeout seconds before forcefully closing it.
// The function takes a timeoutSeconds number instead of a [time.Duration] so it's compatible with Go Mobile.
func (p *Proxy) Stop(timeoutSeconds int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	if err := p.server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown gracefully: %v", err)
		p.server.Close()
	}
	// Allow garbage collection in case the user keeps holding a reference to the Proxy.
	p.proxyHandler = nil
	p.server = nil
}

// RunProxy runs a local web proxy that listens on localAddress, and handles proxy requests by
// establishing connections to requested destination using the [StreamDialer].
func RunProxy(localAddress string, dialer *StreamDialer) (*Proxy, error) {
	listener, err := net.Listen("tcp", localAddress)
	if err != nil {
		return nil, fmt.Errorf("could not listen on address %v: %v", localAddress, err)
	}
	if dialer == nil {
		return nil, errors.New("dialer must not be nil. Please create and pass a valid StreamDialer")
	}

	// The default http.Server doesn't close hijacked connections or cancel in-flight request contexts during
	// shutdown. This can lead to lingering connections. We'll create a base context, propagated to requests,
	// that is cancelled on shutdown. This enables handlers to gracefully terminate requests and close connections.
	serverCtx, cancelCtx := context.WithCancelCause(context.Background())
	proxyHandler := httpproxy.NewProxyHandler(dialer)
	proxyHandler.FallbackHandler = http.NotFoundHandler()
	server := &http.Server{
		Handler: proxyHandler,
		BaseContext: func(l net.Listener) context.Context {
			return serverCtx
		},
	}
	server.RegisterOnShutdown(func() {
		cancelCtx(errors.New("server stopped"))
	})
	go server.Serve(listener)

	host, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return nil, fmt.Errorf("could not parse proxy address '%v': %v", listener.Addr().String(), err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse proxy port '%v': %v", portStr, err)
	}
	return &Proxy{
		host:         host,
		port:         port,
		server:       server,
		proxyHandler: proxyHandler,
	}, nil
}
