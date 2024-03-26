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

// Package mobileproxy provides convenience utilities to help applications run a local proxy
// and use that to configure their networking libraries.
//
// This package is suitable for use with Go Mobile, making it a convenient way to integrate with mobile apps.
package mobileproxy

import (
	"container/list"
	"context"
	"sync"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

// stopConn is a [transport.StreamConn] that can be stopped and has a close handler.
type stopConn struct {
	transport.StreamConn
	onceClose sync.Once
	closeErr  error
	// OnClose is called when Close is called. Close must be called only once.
	OnClose func()
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
	Dialer transport.StreamDialer
	conns  list.List
	mu     sync.Mutex
}

var _ transport.StreamDialer = (*stopStreamDialer)(nil)

// DialStream implements [transport.StreamDialer].
func (d *stopStreamDialer) DialStream(ctx context.Context, addr string) (transport.StreamConn, error) {
	conn, err := d.Dialer.DialStream(ctx, addr)
	if err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	e := d.conns.PushBack(nil)
	sConn := &stopConn{
		StreamConn: conn,
		OnClose: func() {
			d.mu.Lock()
			d.conns.Remove(e)
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
	for e := d.conns.Front(); e != nil; e = e.Next() {
		e.Value.(func())()
	}
}
