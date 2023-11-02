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
	"time"
)

// StreamConn is a [net.Conn] that allows for closing only the reader or writer end of it, supporting half-open state.
type StreamConn interface {
	// Read reads data from the connection.
	// Read can be made to time out and return an error after a fixed
	// time limit; see SetDeadline and SetReadDeadline.
	Read(b []byte) (n int, err error)

	// ReadFrom writes data from a io.Reader to the connection.
	// ReadFrom can be made to time out and return an error after a fixed
	// time limit; see SetDeadline and SetWriteDeadline.
	ReadFrom(r io.Reader) (n int64, err error)

	// Closes the Read end of the connection, allowing for the release of resources.
	// No more reads should happen.
	CloseRead() error

	// Closes the Write end of the connection. An EOF or FIN signal can be
	// sent to the connection target.
	CloseWrite() error

	// Close closes the connection.
	// Any blocked Read or ReadFrom operations will be unblocked and return errors.
	Close() error

	// LocalAddr returns the local network address, if known.
	LocalAddr() net.Addr

	// RemoteAddr returns the remote network address, if known.
	RemoteAddr() net.Addr

	// SetDeadline sets the read and write deadlines associated
	// with the connection. It is equivalent to calling both
	// SetReadDeadline and SetWriteDeadline.
	//
	// A deadline is an absolute time after which I/O operations
	// fail instead of blocking. The deadline applies to all future
	// and pending I/O, not just the immediately following call to
	// Read or Write. After a deadline has been exceeded, the
	// connection can be refreshed by setting a deadline in the future.
	//
	// If the deadline is exceeded a call to Read or Write or to other
	// I/O methods will return an error that wraps os.ErrDeadlineExceeded.
	// This can be tested using errors.Is(err, os.ErrDeadlineExceeded).
	// The error's Timeout method will return true, but note that there
	// are other possible errors for which the Timeout method will
	// return true even if the deadline has not been exceeded.
	//
	// An idle timeout can be implemented by repeatedly extending
	// the deadline after successful Read or Write calls.
	//
	// A zero value for t means I/O operations will not time out.
	SetDeadline(t time.Time) error

	// SetReadDeadline sets the deadline for future Read calls
	// and any currently-blocked Read call.
	// A zero value for t means Read will not time out.
	SetReadDeadline(t time.Time) error

	// SetWriteDeadline sets the deadline for future Write calls
	// and any currently-blocked Write call.
	// Even if write times out, it may return n > 0, indicating that
	// some of the data was successfully written.
	// A zero value for t means Write will not time out.
	SetWriteDeadline(t time.Time) error
}

type duplexConnAdaptor struct {
	StreamConn
	r  io.Reader
	rf io.ReaderFrom
}

var _ StreamConn = (*duplexConnAdaptor)(nil)

func (dc *duplexConnAdaptor) Read(b []byte) (int, error) {
	return dc.r.Read(b)
}
func (dc *duplexConnAdaptor) CloseRead() error {
	return dc.StreamConn.CloseRead()
}
func (dc *duplexConnAdaptor) ReadFrom(r io.Reader) (int64, error) {
	return dc.rf.ReadFrom(r)
}
func (dc *duplexConnAdaptor) CloseWrite() error {
	return dc.StreamConn.CloseWrite()
}

// WrapConn wraps an existing [StreamConn] with a new [io.Reader] and [io.Writer], but preserves the original
// [StreamConn].CloseRead and [StreamConn].CloseWrite.
func WrapConn(c StreamConn, r io.Reader, rf io.ReaderFrom) StreamConn {
	conn := c
	// We special-case duplexConnAdaptor to avoid multiple levels of nesting.
	if a, ok := c.(*duplexConnAdaptor); ok {
		conn = a.StreamConn
	}
	return &duplexConnAdaptor{StreamConn: conn, r: r, rf: rf}
}

// StreamEndpoint represents an endpoint that can be used to establish stream connections (like TCP) to a fixed
// destination.
type StreamEndpoint interface {
	// Connect establishes a connection with the endpoint, returning the connection.
	Connect(ctx context.Context) (StreamConn, error)
}

// TCPEndpoint is a [StreamEndpoint] that connects to the specified address using the specified [StreamDialer].
type TCPEndpoint struct {
	// The Dialer used to create the net.Conn on Connect().
	Dialer net.Dialer
	// The endpoint address (host:port) to pass to Dial.
	// If the host is a domain name, consider pre-resolving it to avoid resolution calls.
	Address string
}

var _ StreamEndpoint = (*TCPEndpoint)(nil)

// Connect implements [StreamEndpoint].Connect.
func (e *TCPEndpoint) Connect(ctx context.Context) (StreamConn, error) {
	conn, err := e.Dialer.DialContext(ctx, "tcp", e.Address)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}

// StreamDialerEndpoint is a [StreamEndpoint] that connects to the specified address using the specified
// [StreamDialer].
type StreamDialerEndpoint struct {
	Dialer  StreamDialer
	Address string
}

var _ StreamEndpoint = (*StreamDialerEndpoint)(nil)

// Connect implements [StreamEndpoint].Connect.
func (e *StreamDialerEndpoint) Connect(ctx context.Context) (StreamConn, error) {
	return e.Dialer.Dial(ctx, e.Address)
}

// StreamDialer provides a way to dial a destination and establish stream connections.
type StreamDialer interface {
	// Dial connects to `raddr`.
	// `raddr` has the form "host:port", where "host" can be a domain name or IP address.
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
