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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/coder/websocket"
)

func NewCoderRWStreamEndpoint(url string, httpClient *http.Client) (func(context.Context) (transport.StreamConn, error), error) {
	return newEndpoint(url, httpClient, func(wsConn *websocket.Conn) transport.StreamConn {
		return &wsToStreamConn{wsConn: wsConn}
	})
}

// func NewCoderNetConnPacketEndpoint(url string, httpClient *http.Client) (func(context.Context) (net.Conn, error), error) {
// 	return newEndpoint(url, httpClient, func(c *websocket.Conn) net.Conn {
// 		return websocket.NetConn(context.Background(), c, websocket.MessageBinary)
// 	})

// }

// func newEndpoint[ConnType any](endpointURL string, httpClient *http.Client, wsToConn func(*websocket.Conn) ConnType) (func(context.Context) (ConnType, error), error) {
// 	url, err := url.Parse(endpointURL)
// 	if err != nil {
// 		return nil, fmt.Errorf("url is invalid: %w", err)
// 	}

// 	options := &websocket.DialOptions{
// 		HTTPClient: httpClient,
// 		HTTPHeader: http.Header(map[string][]string{"User-Agent": {fmt.Sprintf("Outline (%s; %s; %s)", runtime.GOOS, runtime.GOARCH, runtime.Version())}}),
// 	}
// 	return func(ctx context.Context) (ConnType, error) {
// 		var zero ConnType
// 		conn, _, err := websocket.Dial(ctx, url.String(), options)

// 		if err != nil {
// 			return zero, err
// 		}
// 		return wsToConn(conn), nil
// 	}, nil
// }

// wsToStreamConn converts a [websocket.Conn] to a [transport.StreamConn].
type wsToStreamConn struct {
	wsConn     *websocket.Conn
	reader     io.Reader
	writer     io.WriteCloser
	readerErr  error
	writerErr  error
	readerOnce sync.Once
	writerOnce sync.Once
}

var _ transport.StreamConn = (*wsToStreamConn)(nil)

func (c *wsToStreamConn) LocalAddr() net.Addr {
	return websocketAddr{}
}

func (c *wsToStreamConn) RemoteAddr() net.Addr {
	return websocketAddr{}
}

func (c *wsToStreamConn) SetDeadline(time.Time) error {
	return errors.ErrUnsupported
}

func (c *wsToStreamConn) SetReadDeadline(time.Time) error {
	return errors.ErrUnsupported
}

func (c *wsToStreamConn) Read(buf []byte) (int, error) {
	c.readerOnce.Do(func() {
		slog.Info("Initializing Websocket Reader")
		// We use a single message with unbounded size for the entire TCP stream.
		c.wsConn.SetReadLimit(-1)
		msgType, reader, err := c.wsConn.Reader(context.Background())
		slog.Info("Got Websocket Reader")
		if err != nil {
			c.readerErr = fmt.Errorf("failed to get websocket reader: %w", err)
			return
		}
		if msgType != websocket.MessageBinary {
			c.readerErr = errors.New("message type is not binary")
			return
		}
		c.reader = reader
	})
	slog.Info("Read", "readerErr", c.readerErr)
	defer slog.Info("Read done")
	if c.readerErr != nil {
		return 0, c.readerErr
	}
	return c.reader.Read(buf)
}

func (c *wsToStreamConn) CloseRead() error {
	c.wsConn.CloseRead(context.Background())
	return nil
}

func (c *wsToStreamConn) SetWriteDeadline(time.Time) error {
	return errors.ErrUnsupported
}

func (c *wsToStreamConn) Write(buf []byte) (int, error) {
	c.writerOnce.Do(func() {
		slog.Info("Initializing Websocket Writer")
		writer, err := c.wsConn.Writer(context.Background(), websocket.MessageBinary)
		if err != nil {
			c.writerErr = fmt.Errorf("failed to get websocket reader: %w", err)
			return
		}
		c.writer = writer
	})
	slog.Info("Write", "writeErr", c.writerErr)

	if c.writerErr != nil {
		return 0, c.writerErr
	}
	n, err := c.writer.Write(buf)
	slog.Info("Write done", "n", n, "err", err)
	return n, err
}

func (c *wsToStreamConn) CloseWrite() error {
	return c.writer.Close()
}

func (c *wsToStreamConn) Close() error {
	return c.wsConn.Close(websocket.StatusNormalClosure, "")
}

type websocketAddr struct {
}

func (a websocketAddr) Network() string {
	return "websocket"
}

func (a websocketAddr) String() string {
	return "websocket/unknown-addr"
}
