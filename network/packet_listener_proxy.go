// Copyright 2023 Jigsaw Operations LLC
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

package network

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/internal/slicepool"
	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
)

// this was the buffer size used before, we may consider update it in the future
const packetMaxSize = 2048

// packetBufferPool is used to create buffers to read UDP response packets
var packetBufferPool = slicepool.MakePool(packetMaxSize)

// Compilation guard against interface implementation
var _ PacketProxy = (*packetListenerProxyAdapter)(nil)
var _ PacketRequestSender = (*packetListenerRequestSender)(nil)

type packetListenerProxyAdapter struct {
	listener         transport.PacketListener
	writeIdleTimeout time.Duration
}

type packetListenerRequestSender struct {
	mu     sync.Mutex // Protects closed and timer function calls
	closed bool

	proxyConn        net.PacketConn
	writeIdleTimeout time.Duration
	writeIdleTimer   *time.Timer
}

// NewPacketProxyFromPacketListener creates a new [PacketProxy] that uses the existing [transport.PacketListener]
// to create connections to a proxy. You can use this function if you already have an implementation of
// [transport.PacketListener] and would like to inject it into one of the network stacks (for example,
// network/lwip2transport) as UDP traffic handlers.
func NewPacketProxyFromPacketListener(pl transport.PacketListener) (PacketProxy, error) {
	if pl == nil {
		return nil, errors.New("pl must not be nil")
	}
	return &packetListenerProxyAdapter{
		listener:         pl,
		writeIdleTimeout: 30 * time.Second,
	}, nil
}

// NewSession implements [PacketProxy].NewSession function. It uses [transport.PacketListener].ListenPacket to create
// a [net.PacketConn], and constructs a new [PacketRequestSender] that is based on this [net.PacketConn].
func (proxy *packetListenerProxyAdapter) NewSession(respWriter PacketResponseReceiver) (PacketRequestSender, error) {
	if respWriter == nil {
		return nil, errors.New("respWriter must not be nil")
	}
	proxyConn, err := proxy.listener.ListenPacket(context.Background())
	if err != nil {
		return nil, err
	}
	reqSender := &packetListenerRequestSender{
		proxyConn:        proxyConn,
		writeIdleTimeout: proxy.writeIdleTimeout,
	}

	// Terminate the session after timeout with no outgoing writes (deadline is refreshed by WriteTo)
	reqSender.writeIdleTimer = time.AfterFunc(reqSender.writeIdleTimeout, func() {
		reqSender.Close()
	})

	// Relay incoming UDP responses from the proxy asynchronously until EOF, session expiration or error
	go func() {
		defer respWriter.Close()

		// Allocate buffer from slicepool, because `go build -gcflags="-m"` shows a local array will escape to heap
		slice := packetBufferPool.LazySlice()
		buf := slice.Acquire()
		defer slice.Release()

		for {
			n, srcAddr, err := proxyConn.ReadFrom(buf)
			if err != nil {
				// Ignore some specific recoverable errors
				if errors.Is(err, io.ErrShortBuffer) {
					continue
				}
				return
			}
			if _, err := respWriter.WriteFrom(buf[:n], srcAddr); err != nil {
				return
			}
		}
	}()

	return reqSender, nil
}

// WriteTo implements [PacketRequestSender].WriteTo function. It simply forwards the packet to the underlying
// [net.PacketConn].WriteTo function.
func (s *packetListenerRequestSender) WriteTo(p []byte, destination net.Addr) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return 0, ErrClosed
	}
	s.writeIdleTimer.Reset(s.writeIdleTimeout)
	return s.proxyConn.WriteTo(p, destination)
}

// Close implements [PacketRequestSender].Close function. It closes the underlying [net.PacketConn]. This will also
// terminate the goroutine created in NewSession because s.conn.ReadFrom will return [io.EOF].
func (s *packetListenerRequestSender) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}
	s.closed = true
	s.writeIdleTimer.Stop()
	return s.proxyConn.Close()
}
