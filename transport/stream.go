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
)

// StreamConn is a [net.Conn] that allows for closing only the reader or writer end of
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
type StreamEndpoint = Endpoint[StreamConn]

// TCPEndpoint is a [StreamEndpoint] that connects to the given address using the given [StreamDialer].
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

// StreamDialer provides a way to dial a destination and establish stream connections.
type StreamDialer = Dialer[StreamConn]

// TCPDialer is a [StreamDialer] that uses the standard [net.Dialer] to dial.
// It provides a convenient way to use a [net.Dialer] when you need a [StreamDialer].
type TCPDialer struct {
	Dialer net.Dialer
}

var _ StreamDialer = (*TCPDialer)(nil)

// Dial implements [Dialer].Dial.
func (d *TCPDialer) Dial(ctx context.Context, addr string) (StreamConn, error) {
	conn, err := d.Dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	return conn.(*net.TCPConn), nil
}
