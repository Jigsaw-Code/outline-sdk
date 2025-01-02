// Copyright 2025 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package websocket

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/coder/websocket"
)

func NewCoderNetConnStreamEndpoint(url string, httpClient *http.Client) (func(context.Context) (transport.StreamConn, error), error) {
	return newEndpoint(url, httpClient, func(c *websocket.Conn) transport.StreamConn {
		return &netToStreamConn{websocket.NetConn(context.Background(), c, websocket.MessageBinary)}
	})
}

func NewCoderNetConnPacketEndpoint(url string, httpClient *http.Client) (func(context.Context) (net.Conn, error), error) {
	return newEndpoint(url, httpClient, func(c *websocket.Conn) net.Conn {
		return websocket.NetConn(context.Background(), c, websocket.MessageBinary)
	})

}

func newEndpoint[ConnType any](endpointURL string, httpClient *http.Client, wsToConn func(*websocket.Conn) ConnType) (func(context.Context) (ConnType, error), error) {
	url, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("url is invalid: %w", err)
	}

	options := &websocket.DialOptions{
		HTTPClient: httpClient,
		HTTPHeader: http.Header(map[string][]string{"User-Agent": {fmt.Sprintf("Outline (%s; %s; %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())}}),
	}
	return func(ctx context.Context) (ConnType, error) {
		var zero ConnType
		conn, _, err := websocket.Dial(ctx, url.String(), options)

		if err != nil {
			return zero, err
		}
		return wsToConn(conn), nil
	}, nil
}

// netToStreamConn converts a [net.Conn] to a [transport.StreamConn].
type netToStreamConn struct {
	net.Conn
}

var _ transport.StreamConn = (*netToStreamConn)(nil)

func (c *netToStreamConn) CloseRead() error {
	// Do nothing.
	return nil
}

func (c *netToStreamConn) CloseWrite() error {
	return c.Close()
}
