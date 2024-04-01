// Copyright 2024 Jigsaw Operations LLC
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

package mobileproxy

import (
	"context"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/stretchr/testify/require"
)

func Test_stopStreamDialer_StopBeforeDial(t *testing.T) {
	d := &stopStreamDialer{
		Dialer: transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
			return nil, nil
		}),
	}
	d.StopConnections()
	conn, err := d.DialStream(context.Background(), "invalid:0")
	require.Nil(t, conn)
	require.ErrorIs(t, err, errStopped)
}

func Test_stopStreamDialer_StopDuringDial(t *testing.T) {
	dialStarted := make(chan struct{})
	resumeDial := make(chan struct{})
	d := &stopStreamDialer{
		Dialer: transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
			close(dialStarted)
			<-resumeDial
			return &net.TCPConn{}, nil
		}),
	}
	go func() {
		<-dialStarted
		d.StopConnections()
		close(resumeDial)
	}()
	conn, err := d.DialStream(context.Background(), "invalid:0")
	require.Nil(t, conn)
	require.ErrorIs(t, err, errStopped)
}

// netConnAdaptor converts a net.Conn to transport.StreamConn.
// TODO(fortuna): Consider moving it to the SDK.
type netConnAdaptor struct {
	net.Conn
	onceClose sync.Once
	closeErr  error
}

var _ transport.StreamConn = (*netConnAdaptor)(nil)

func (c *netConnAdaptor) CloseWrite() error {
	type WriteCloser interface {
		CloseWrite() error
	}
	if wc, ok := c.Conn.(WriteCloser); ok {
		return wc.CloseWrite()
	}
	return c.Close()
}

func (c *netConnAdaptor) CloseRead() error {
	type ReadCloser interface {
		CloseRead() error
	}
	if rc, ok := c.Conn.(ReadCloser); ok {
		return rc.CloseRead()
	}
	return nil
}

func (c *netConnAdaptor) Close() error {
	c.onceClose.Do(func() {
		c.closeErr = c.Conn.Close()
	})
	return c.closeErr
}

func asStreamConn(conn net.Conn) transport.StreamConn {
	if sc, ok := conn.(transport.StreamConn); ok {
		return sc
	}
	return &netConnAdaptor{Conn: conn}
}

func Test_stopStreamDialer_StopAfterDial(t *testing.T) {
	conn1, conn2 := net.Pipe()
	defer conn2.Close()
	d := &stopStreamDialer{
		Dialer: transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
			return asStreamConn(conn1), nil
		}),
	}
	conn, err := d.DialStream(context.Background(), "invalid:0")
	require.NotNil(t, conn)
	require.NoError(t, err)
	defer conn.Close()
	d.StopConnections()
	_, err = conn.Read([]byte{})
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

func Test_stopStreamDialer_StopAfterClose(t *testing.T) {
	conn1, conn2 := net.Pipe()
	defer conn2.Close()
	d := &stopStreamDialer{
		Dialer: transport.FuncStreamDialer(func(ctx context.Context, addr string) (transport.StreamConn, error) {
			return asStreamConn(conn1), nil
		}),
	}
	conn, err := d.DialStream(context.Background(), "invalid:0")
	require.NotNil(t, conn)
	require.NoError(t, err)
	err = conn.Close()
	require.NoError(t, err)
	d.StopConnections()
}
