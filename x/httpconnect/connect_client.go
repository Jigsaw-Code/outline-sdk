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
	"context"
	"errors"
	"fmt"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"io"
	"net"
	"net/http"
)

// connectClient is a [transport.StreamDialer] implementation that dials [proxyAddr] with the given [dialer]
// and sends a CONNECT request to the dialed proxy.
type connectClient struct {
	dialer    transport.StreamDialer
	proxyAddr string

	headers http.Header
}

var _ transport.StreamDialer = (*connectClient)(nil)

type ClientOption func(c *connectClient)

func NewConnectClient(dialer transport.StreamDialer, proxyAddr string, opts ...ClientOption) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("dialer must not be nil")
	}
	_, _, err := net.SplitHostPort(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy address %s: %w", proxyAddr, err)
	}

	cc := &connectClient{
		dialer:    dialer,
		proxyAddr: proxyAddr,
		headers:   make(http.Header),
	}

	for _, opt := range opts {
		opt(cc)
	}

	return cc, nil
}

// WithHeaders appends the given [headers] to the CONNECT request
func WithHeaders(headers http.Header) ClientOption {
	return func(c *connectClient) {
		c.headers = headers.Clone()
	}
}

// DialStream - connects to the proxy and sends a CONNECT request to it, closes the connection if the request fails
func (cc *connectClient) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := cc.dialer.DialStream(ctx, cc.proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial proxy %s: %w", cc.proxyAddr, err)
	}

	conn, err := cc.doConnect(ctx, remoteAddr, innerConn)
	if err != nil {
		_ = innerConn.Close()
		return nil, fmt.Errorf("doConnect %s: %w", remoteAddr, err)
	}

	return conn, nil
}

func (cc *connectClient) doConnect(ctx context.Context, remoteAddr string, conn transport.StreamConn) (transport.StreamConn, error) {
	_, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote address %s: %w", remoteAddr, err)
	}

	pr, pw := io.Pipe()

	req, err := http.NewRequestWithContext(ctx, http.MethodConnect, "http://"+remoteAddr, pr) // TODO: HTTPS support
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.ContentLength = -1 // -1 means length unknown
	mergeHeaders(req.Header, cc.headers)

	tr := &http.Transport{
		// TODO: HTTP/2 support with [http2.ConfigureTransport]
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return conn, nil
		},
	}

	hc := http.Client{
		Transport: tr,
	}

	resp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return &pipeConn{
		reader:     resp.Body,
		writer:     pw,
		StreamConn: conn,
	}, nil
}

func mergeHeaders(dst http.Header, src http.Header) {
	for k, v := range src {
		dst[k] = append(dst[k], v...)
	}
}
