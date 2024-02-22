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

// Package mobileproxy provides convenience utilities to help applications run a local proxy
// and use that to configure their networking libraries.
//
// This package is suitable for use with Go Mobile, making it a convenient way to integrate with mobile apps.
package mobileproxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/config"
	"github.com/Jigsaw-Code/outline-sdk/x/httpproxy"
	"github.com/Jigsaw-Code/outline-sdk/x/smart"
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
}

// RunProxy runs a local web proxy that listens on localAddress, and handles proxy requests by
// establishing connections to requested destination using the [StreamDialer].
func RunProxy(localAddress string, dialer *StreamDialer) (*Proxy, error) {
	listener, err := net.Listen("tcp", localAddress)
	if err != nil {
		return nil, fmt.Errorf("could not listen on address %v: %v", localAddress, err)
	}

	proxyHandler := httpproxy.NewProxyHandler(dialer)
	proxyHandler.FallbackHandler = http.NotFoundHandler()
	server := &http.Server{Handler: proxyHandler}
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

// StreamDialer encapsulates the logic to create stream connections (like TCP).
type StreamDialer struct {
	transport.StreamDialer
}

// NewStreamDialerFromConfig creates a [StreamDialer] based on the given config.
// The config format is specified in https://pkg.go.dev/github.com/Jigsaw-Code/outline-sdk/x/config#hdr-Config_Format.
func NewStreamDialerFromConfig(transportConfig string) (*StreamDialer, error) {
	dialer, err := config.NewStreamDialer(transportConfig)
	if err != nil {
		return nil, err
	}
	return &StreamDialer{dialer}, nil
}

// LogWriter is used as a sink for logging.
type LogWriter io.StringWriter

// Adaptor to convert an [io.StringWriter] to a [io.Writer].
type stringToBytesWriter struct {
	w io.Writer
}

// WriteString implements [io.StringWriter].
func (w *stringToBytesWriter) WriteString(logText string) (int, error) {
	return io.WriteString(w.w, logText)
}

// NewStderrLogWriter creates a [LogWriter] that writes to the standard error output.
func NewStderrLogWriter() LogWriter {
	return &stringToBytesWriter{os.Stderr}
}

// Adaptor to convert an [io.Writer] to a [io.StringWriter].
type bytestoStringWriter struct {
	sw io.StringWriter
}

// Write implements [io.Writer].
func (w *bytestoStringWriter) Write(b []byte) (int, error) {
	return w.sw.WriteString(string(b))
}

func toWriter(logWriter LogWriter) io.Writer {
	if logWriter == nil {
		return nil
	}
	if w, ok := logWriter.(io.Writer); ok {
		return w
	}
	return &bytestoStringWriter{logWriter}
}

// NewSmartStreamDialer automatically selects a DNS and TLS strategy to use, and returns a [StreamDialer]
// that will use the selected strategy.
// It uses testDomains to find a strategy that works when accessing those domains.
// The strategies to search are given in the searchConfig. An example can be found in
// https://github.com/Jigsaw-Code/outline-sdk/x/examples/smart-proxy/config.json
func NewSmartStreamDialer(testDomains *StringList, searchConfig string, logWriter LogWriter) (*StreamDialer, error) {
	logBytesWriter := toWriter(logWriter)
	// TODO: inject the base dialer for tests.
	finder := smart.StrategyFinder{
		LogWriter:    logBytesWriter,
		TestTimeout:  5 * time.Second,
		StreamDialer: &transport.TCPDialer{},
		PacketDialer: &transport.UDPDialer{},
	}
	dialer, err := finder.NewDialer(context.Background(), testDomains.list, []byte(searchConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to find dialer: %w", err)
	}
	return &StreamDialer{dialer}, nil
}

// StringList allows us to pass a list of strings to the Go Mobile functions, since Go Mobile doesn't
// support slices as parameters.
type StringList struct {
	list []string
}

// Append adds the string value to the end of the list.
func (l *StringList) Append(value string) {
	l.list = append(l.list, value)
}

// NewListFromLines creates a StringList by splitting the input string on new lines.
func NewListFromLines(lines string) *StringList {
	return &StringList{list: strings.Split(lines, "\n")}
}
