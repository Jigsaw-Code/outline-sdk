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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/gorilla/websocket"
)

func userAgentString() string {
	return fmt.Sprintf("Outline (%s; %s; %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())
}

// NewStreamEndpoint creates a new Websocket Stream Endpoint. Streams are sent over
// Websockets, with each Write becoming a separate message. Half-close is supported:
// CloseRead will not close the Websocket connection, while CloseWrite sends a Websocket
// close but continues reading until a close is received from the server.
func NewStreamEndpoint(urlStr string, sd transport.StreamDialer, tlsConfig *tls.Config) (func(context.Context) (transport.StreamConn, error), error) {
	return newEndpoint(urlStr, sd, tlsConfig, func(gc *gorillaConn) transport.StreamConn { return gc })
}

// NewPacketEndpoint creates a new Websocket Packet Endpoint. Each packet is exchanged as a Websocket message.
func NewPacketEndpoint(urlStr string, sd transport.StreamDialer, tlsConfig *tls.Config) (func(context.Context) (net.Conn, error), error) {
	return newEndpoint(urlStr, sd, tlsConfig, func(gc *gorillaConn) net.Conn { return gc })
}

func newEndpoint[ConnType net.Conn](urlStr string, sd transport.StreamDialer, tlsConfig *tls.Config, wsToConn func(*gorillaConn) ConnType) (func(context.Context) (ConnType, error), error) {
	_, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("url is invalid: %w", err)
	}

	wsDialer := &websocket.Dialer{
		TLSClientConfig: tlsConfig,
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if !strings.HasPrefix(network, "tcp") {
				return nil, fmt.Errorf("websocket dialer does not support network type %v", network)
			}
			return sd.DialStream(ctx, addr)
		},
	}
	headers := http.Header(map[string][]string{"User-Agent": {userAgentString()}})
	return func(ctx context.Context) (ConnType, error) {
		var zero ConnType
		wsConn, _, err := wsDialer.DialContext(ctx, urlStr, headers)
		if err != nil {
			return zero, err
		}
		gConn := &gorillaConn{wsConn: wsConn}
		wsConn.SetCloseHandler(func(code int, text string) error {
			gConn.readErr = io.EOF
			return nil
		})
		return wsToConn(gConn), nil
	}, nil
}

type gorillaConn struct {
	wsConn        *websocket.Conn
	writeErr      error
	readErr       error
	pendingReader io.Reader
}

var _ transport.StreamConn = (*gorillaConn)(nil)

func (c *gorillaConn) LocalAddr() net.Addr {
	return c.wsConn.LocalAddr()
}

func (c *gorillaConn) RemoteAddr() net.Addr {
	return c.wsConn.RemoteAddr()
}

func (c *gorillaConn) SetDeadline(deadline time.Time) error {
	return errors.Join(c.wsConn.SetReadDeadline(deadline), c.wsConn.SetWriteDeadline(deadline))
}

func (c *gorillaConn) SetReadDeadline(deadline time.Time) error {
	return c.wsConn.SetReadDeadline(deadline)
}

func (c *gorillaConn) SetWriteDeadline(deadline time.Time) error {
	return c.wsConn.SetWriteDeadline(deadline)
}

func (c *gorillaConn) Read(buf []byte) (int, error) {
	if c.readErr != nil {
		return 0, c.readErr
	}
	if c.pendingReader != nil {
		n, err := c.pendingReader.Read(buf)
		if c.readErr != nil {
			return n, c.readErr
		}
		if !errors.Is(err, io.EOF) {
			return n, err
		}
		c.pendingReader = nil
	}

	msgType, reader, err := c.wsConn.NextReader()
	if c.readErr != nil {
		return 0, c.readErr
	}
	if err != nil {
		return 0, err
	}
	if msgType != websocket.BinaryMessage {
		return 0, errors.New("read message is not binary")
	}

	c.pendingReader = reader
	return reader.Read(buf)
}

func (c *gorillaConn) Write(buf []byte) (int, error) {
	err := c.wsConn.WriteMessage(websocket.BinaryMessage, buf)
	if err != nil {
		if c.writeErr != nil {
			return 0, c.writeErr
		}
		return 0, err
	}
	return len(buf), nil
}

func (c *gorillaConn) CloseRead() error {
	c.readErr = net.ErrClosed
	c.wsConn.SetReadDeadline(time.Now())
	return nil
}

func (c *gorillaConn) CloseWrite() error {
	// Send close message.
	message := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	c.wsConn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
	c.writeErr = net.ErrClosed
	c.wsConn.SetWriteDeadline(time.Now())
	return nil
}

func (c *gorillaConn) Close() error {
	return c.wsConn.Close()
}
