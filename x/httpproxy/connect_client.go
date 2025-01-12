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
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/Jigsaw-Code/outline-sdk/transport"
	"net"
	"net/http"
)

// connectClient is a [transport.StreamDialer] implementation that sends a CONNECT request
type connectClient struct {
	endpoint transport.StreamEndpoint

	// proxyAuth is the Proxy-Authorization header value. If empty, the header is not sent
	proxyAuth string
}

var _ transport.StreamDialer = (*connectClient)(nil)

type ConnectClientOption func(c *connectClient)

func NewConnectClient(endpoint transport.StreamEndpoint, opts ...ConnectClientOption) (transport.StreamDialer, error) {
	if endpoint == nil {
		return nil, errors.New("endpoint must not be nil")
	}

	cc := &connectClient{
		endpoint: endpoint,
	}

	for _, opt := range opts {
		opt(cc)
	}

	return cc, nil
}

// WithProxyAuthorization - sets the Proxy-Authorization header value
func WithProxyAuthorization(proxyAuth string) ConnectClientOption {
	return func(c *connectClient) {
		c.proxyAuth = proxyAuth
	}
}

// DialStream - connects using the endpoint and sends a CONNECT request to the remoteAddr, closes the connection if the request fails
func (c *connectClient) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	conn, err := c.endpoint.ConnectStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to endpoint: %w", err)
	}

	err = c.sendConnectRequest(ctx, remoteAddr, conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("sendConnectRequest: %w", err)
	}

	return conn, nil
}

func (c *connectClient) sendConnectRequest(ctx context.Context, remoteAddr string, conn transport.StreamConn) error {
	_, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to parse remote address: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodConnect, "http://"+remoteAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if c.proxyAuth != "" {
		req.Header.Add("Proxy-Authorization", c.proxyAuth)
	}

	err = req.Write(conn)
	if err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
