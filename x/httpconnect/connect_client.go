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
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/http2"
	"io"
	"net"
	"net/http"
)

// ConnectClient is a [transport.StreamDialer] implementation that dials proxyAddr with the given dialer and sends a CONNECT request to the dialed proxy.
// By default, the client uses "http", but it can be changed to "https" with the [WithHTTPS] option.
type ConnectClient struct {
	dialer    transport.StreamDialer
	proxyAddr string
	scheme    string
	tlsConfig *tls.Config
	headers   http.Header
}

var _ transport.StreamDialer = (*ConnectClient)(nil)

type ClientOption func(c *ConnectClient)

func NewConnectClient(dialer transport.StreamDialer, proxyAddr string, opts ...ClientOption) (*ConnectClient, error) {
	if dialer == nil {
		return nil, errors.New("dialer must not be nil")
	}
	_, _, err := net.SplitHostPort(proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse proxy address %s: %w", proxyAddr, err)
	}

	cc := &ConnectClient{
		dialer:    dialer,
		proxyAddr: proxyAddr,
		scheme:    "http",
	}

	for _, opt := range opts {
		opt(cc)
	}

	return cc, nil
}

// WithHTTPS sets the scheme to "https" and the given tlsConfig to the transport
func WithHTTPS(tlsConfig *tls.Config) ClientOption {
	return func(c *ConnectClient) {
		c.scheme = "https"
		c.tlsConfig = tlsConfig.Clone()
	}
}

// WithHeaders appends the given headers to the CONNECT request
func WithHeaders(headers http.Header) ClientOption {
	return func(c *ConnectClient) {
		c.headers = headers.Clone()
	}
}

// DialStream - connects to the proxy and sends a CONNECT request to it, closes the connection if the request fails
func (cc *ConnectClient) DialStream(ctx context.Context, remoteAddr string) (streamConn transport.StreamConn, err error) {
	_, _, err = net.SplitHostPort(remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote address %s: %w", remoteAddr, err)
	}

	innerConn, err := cc.dialer.DialStream(ctx, cc.proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial proxy %s: %w", cc.proxyAddr, err)
	}
	defer func() {
		if err != nil {
			_ = innerConn.Close()
		}
	}()

	roundTripper, err := cc.buildTransport(innerConn)
	if err != nil {
		return nil, fmt.Errorf("failed to build roundTripper: %w", err)
	}

	reader, writer, err := doConnect(ctx, roundTripper, cc.scheme, remoteAddr, cc.headers)
	if err != nil {
		return nil, fmt.Errorf("doConnect %s: %w", remoteAddr, err)
	}

	return &pipeConn{
		reader:     reader,
		writer:     writer,
		StreamConn: innerConn,
	}, nil
}

func (cc *ConnectClient) buildTransport(conn transport.StreamConn) (http.RoundTripper, error) {
	tr := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return conn, nil
		},
		TLSClientConfig: cc.tlsConfig,
	}

	err := http2.ConfigureTransport(tr)
	if err != nil {
		return nil, fmt.Errorf("failed to configure transport for HTTP/2: %w", err)
	}

	return tr, nil
}

func doConnect(
	ctx context.Context,
	roundTripper http.RoundTripper,
	scheme, remoteAddr string,
	headers http.Header,
) (io.ReadCloser, io.WriteCloser, error) {
	if scheme != "http" && scheme != "https" {
		return nil, nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}

	pr, pw := io.Pipe()
	remoteURL := fmt.Sprintf("%s://%s", scheme, remoteAddr)
	req, err := http.NewRequestWithContext(ctx, http.MethodConnect, remoteURL, pr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.ContentLength = -1 // -1 means unknown length
	mergeHeaders(req.Header, headers)

	hc := http.Client{
		Transport: roundTripper,
	}
	resp, err := hc.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("do: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, pw, nil
}

func mergeHeaders(dst http.Header, src http.Header) {
	for k, v := range src {
		dst[k] = append(dst[k], v...)
	}
}
