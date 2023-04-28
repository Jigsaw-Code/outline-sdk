// Copyright 2019 Jigsaw Operations LLC
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

package transport

import (
	"context"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

// StreamConn is a net.Conn that allows for closing only the reader or writer end of
// it, supporting half-open state.
type StreamConn interface {
	net.Conn
	// Closes the Read end of the connection, allowing for the release of resources.
	// No more reads should happen.
	CloseRead() error
	// Closes the Write end of the connection. An EOF or FIN signal may be
	// sent to the connection target.
	CloseWrite() error
}

type duplexConnAdaptor struct {
	StreamConn
	r io.Reader
	w io.Writer
}

var _ StreamConn = (*duplexConnAdaptor)(nil)

func (dc *duplexConnAdaptor) Read(b []byte) (int, error) {
	return dc.r.Read(b)
}
func (dc *duplexConnAdaptor) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, dc.r)
}
func (dc *duplexConnAdaptor) CloseRead() error {
	return dc.StreamConn.CloseRead()
}
func (dc *duplexConnAdaptor) Write(b []byte) (int, error) {
	return dc.w.Write(b)
}
func (dc *duplexConnAdaptor) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(dc.w, r)
}
func (dc *duplexConnAdaptor) CloseWrite() error {
	return dc.StreamConn.CloseWrite()
}

// WrapDuplexConn wraps an existing [StreamConn] with new Reader and Writer, but
// preserving the original [StreamConn.CloseRead] and [StreamConn.CloseWrite].
func WrapConn(c StreamConn, r io.Reader, w io.Writer) StreamConn {
	conn := c
	// We special-case duplexConnAdaptor to avoid multiple levels of nesting.
	if a, ok := c.(*duplexConnAdaptor); ok {
		conn = a.StreamConn
	}
	return &duplexConnAdaptor{StreamConn: conn, r: r, w: w}
}

// StreamEndpoint represents an endpoint that can be used to established stream connections (like TCP) to a fixed destination.
type StreamEndpoint interface {
	// Connect establishes a connection with the endpoint, returning the connection.
	Connect(ctx context.Context) (StreamConn, error)
}

// TCPEndpoint is a [StreamEndpoint] that connects to the given address using the given [StreamDialer].
type TCPEndpoint struct {
	// The Dialer used to create the net.Conn on Connect().
	Dialer net.Dialer
	// The endpoint address (host:port) to pass to Dial.
	// If the host is a domain name, consider pre-resolving it to avoid resolution calls.
	Address string
}

var _ StreamEndpoint = (*TCPEndpoint)(nil)

// Connect implements [StreamEndpoint.Connect].
func (e *TCPEndpoint) Connect(ctx context.Context) (StreamConn, error) {
	conn, err := e.Dialer.DialContext(ctx, "tcp", e.Address)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}

// StreamDialer provides a way to dial a destination and establish stream connections.
type StreamDialer interface {
	// Dial connects to `raddr`.
	// `raddr` has the form `host:port`, where `host` can be a domain name or IP address.
	Dial(ctx context.Context, raddr string) (StreamConn, error)
}

// TCPStreamDialer is a [StreamDialer] that uses the standard [net.Dialer] to dial.
// It provides a convenient way to use a [net.Dialer] when you need a [StreamDialer].
type TCPStreamDialer struct {
	Dialer net.Dialer
}

var _ StreamDialer = (*TCPStreamDialer)(nil)

func (d *TCPStreamDialer) Dial(ctx context.Context, addr string) (StreamConn, error) {
	conn, err := d.Dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}

type pipeStreamConn struct {
	Reader     *io.PipeReader
	Writer     *io.PipeWriter
	localAddr  net.Addr
	remoteAddr net.Addr
	timerMu    sync.Mutex
	readTimer  *time.Timer
	writeTimer *time.Timer
}

var _ StreamConn = (*pipeStreamConn)(nil)

func (c *pipeStreamConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *pipeStreamConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *pipeStreamConn) Read(b []byte) (int, error) {
	n, err := c.Reader.Read(b)
	if err == io.ErrClosedPipe {
		err = net.ErrClosed
	}
	return n, err
}

func (c *pipeStreamConn) CloseRead() error {
	return c.Reader.Close()
}

func (c *pipeStreamConn) Write(b []byte) (int, error) {
	n, err := c.Writer.Write(b)
	if err == io.ErrClosedPipe {
		err = net.ErrClosed
	}
	return n, err
}

func (c *pipeStreamConn) CloseWrite() error {
	return c.Writer.Close()
}

func (c *pipeStreamConn) Close() error {
	c.Reader.Close()
	c.Writer.Close()
	return nil
}

func (c *pipeStreamConn) SetReadDeadline(t time.Time) error {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	if c.readTimer != nil {
		if !c.readTimer.Stop() {
			<-c.readTimer.C
		}
	}
	c.readTimer = time.AfterFunc(time.Until(t), func() { c.Reader.CloseWithError(os.ErrDeadlineExceeded) })
	return nil
}

func (c *pipeStreamConn) SetWriteDeadline(t time.Time) error {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	if c.writeTimer != nil {
		if !c.writeTimer.Stop() {
			<-c.writeTimer.C
		}
	}
	c.writeTimer = time.AfterFunc(time.Until(t), func() { c.Writer.CloseWithError(os.ErrDeadlineExceeded) })
	return nil
}

func (c *pipeStreamConn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}
