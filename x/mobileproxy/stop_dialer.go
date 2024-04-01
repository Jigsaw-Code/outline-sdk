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
	"container/list"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

var errStopped = errors.New("dialer stopped")

// stopConn is a [transport.StreamConn] that can be stopped and has a close handler.
type stopConn struct {
	transport.StreamConn
	onceClose sync.Once
	closeErr  error
	// OnClose is called when Close is called. Close must be called only once.
	OnClose func()
}

// WriteTo implements [io.WriterTo], to make sure we don't hide the functionality from the
// underlying connection.
func (c *stopConn) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, c.StreamConn)
}

// ReadFrom implements [io.ReaderFrom], to make sure we don't hide the functionality from
// the underlying connection.
func (c *stopConn) ReadFrom(r io.Reader) (int64, error) {
	// Prefer ReadFrom if requested. Otherwise io.Copy prefers WriteTo.
	if rf, ok := c.StreamConn.(io.ReaderFrom); ok {
		return rf.ReadFrom(r)
	}
	return io.Copy(c.StreamConn, r)
}

// Close implements [transport.StreamConn].
func (c *stopConn) Close() error {
	c.Stop()
	c.OnClose()
	return c.closeErr
}

// Stop stops the connection, calling its underlying Close if it wasn't called yet, without calling OnClose.
func (c *stopConn) Stop() {
	c.onceClose.Do(func() {
		c.closeErr = c.StreamConn.Close()
	})
}

// stopStreamDialer is a [transport.StreamDialer] that provides a method to close all open connections.
type stopStreamDialer struct {
	Dialer       transport.StreamDialer
	stopped      bool
	cleanupFuncs list.List
	mu           sync.Mutex
}

var _ transport.StreamDialer = (*stopStreamDialer)(nil)

// DialStream implements [transport.StreamDialer].
func (d *stopStreamDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Capture any stop that happened before DialStream.
	if d.stopped {
		return nil, errStopped
	}

	// Register dial cancelation to capture a stop during the DialStream.
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if !errors.Is(ctx.Err(), context.Canceled) {
			cancel()
		}
	}()
	cleanDialEl := d.cleanupFuncs.PushBack(func() {
		cancel()
	})

	// We release the lock for the dial because it's a slow operation and to allow for parallel dials.
	d.mu.Unlock()
	conn, err := d.Dialer.DialStream(ctx, addr)
	d.mu.Lock()

	// Clean up dial cancelation.
	d.cleanupFuncs.Remove(cleanDialEl)

	if err != nil {
		return nil, err
	}

	// Dialer may not pay attention to context cancellation, so we check if we are stopped here.
	if d.stopped {
		conn.Close()
		return nil, errStopped
	}

	// We have a connection. Register cleanup to capture stop during the connection lifetime.
	e := d.cleanupFuncs.PushBack(nil)
	sConn := &stopConn{
		StreamConn: conn,
		OnClose: func() {
			d.mu.Lock()
			d.cleanupFuncs.Remove(e)
			d.mu.Unlock()
		},
	}
	e.Value = sConn.Stop
	return sConn, nil
}

// StopConnections stops all active connections created by this [stopStreamDialer].
func (d *stopStreamDialer) StopConnections() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopped = true
	for e := d.cleanupFuncs.Front(); e != nil; e = e.Next() {
		e.Value.(func())()
	}
}
