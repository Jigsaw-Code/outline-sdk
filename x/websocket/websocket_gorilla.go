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
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/gorilla/websocket"
)

func NewGorillatreamEndpoint(urlStr string, tlsConfig *tls.Config) (func(context.Context) (transport.StreamConn, error), error) {
	_, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("url is invalid: %w", err)
	}

	headers := http.Header(map[string][]string{"User-Agent": {fmt.Sprintf("Outline (%s; %s; %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())}})
	dialer := &websocket.Dialer{TLSClientConfig: tlsConfig}
	return func(ctx context.Context) (transport.StreamConn, error) {
		wsConn, _, err := dialer.DialContext(ctx, urlStr, headers)
		if err != nil {
			return nil, err
		}
		gConn := &gorillaConn{wsConn: wsConn}
		wsConn.SetCloseHandler(func(code int, text string) error {
			gConn.readErr = io.EOF
			return nil
		})
		return gConn, nil
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
	// TODO: handle pending data.
	if c.readErr != nil {
		return 0, c.readErr
	}
	if c.pendingReader != nil {
		n, err := c.pendingReader.Read(buf)
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
