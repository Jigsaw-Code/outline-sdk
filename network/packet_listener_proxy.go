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
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
)

const packetMaxSize = 2048

// Compilation guard against interface implementation
var _ PacketProxy = (*packetListenerProxyAdapter)(nil)
var _ PacketRequestSender = (*packetListenerRequestSender)(nil)

type packetListenerProxyAdapter struct {
	listener transport.PacketListener
	timeout  time.Duration
}

type packetListenerRequestSender struct {
	conn    net.PacketConn
	timeout time.Duration
}

// NewPacketProxyFromPacketListener creates a new [PacketProxy] that uses the existing [transport.PacketListener].
// You can use this function if you already have an implementation of [transport.PacketListener] and would like to
// inject it into one of the network stacks (for example, network/lwip2transport) as UDP traffic handlers.
func NewPacketProxyFromPacketListener(pl transport.PacketListener) (PacketProxy, error) {
	if pl == nil {
		return nil, errors.New("pl must not be nil")
	}
	return &packetListenerProxyAdapter{
		listener: pl,
		timeout:  30 * time.Second,
	}, nil
}

// NewSession implements [PacketProxy].NewSession function. It uses [transport.PacketListener].ListenPacket to create
// a [net.PacketConn], and constructs a new [PacketRequestSender] that is based on this [net.PacketConn].
func (proxy *packetListenerProxyAdapter) NewSession(respWriter PacketResponseReceiver) (PacketRequestSender, error) {
	if respWriter == nil {
		return nil, errors.New("respWriter must not be nil")
	}
	conn, err := proxy.listener.ListenPacket(context.Background())
	if err != nil {
		return nil, err
	}
	reqSender := &packetListenerRequestSender{conn, proxy.timeout}

	// Read UDP responses asynchronously
	go func() {
		defer respWriter.Close()
		buf := make([]byte, 0, packetMaxSize)
		for {
			conn.SetReadDeadline(time.Now().Add(proxy.timeout))
			n, addr, err := conn.ReadFrom(buf[:])
			if err != nil {
				return
			}
			if _, err := respWriter.WriteFrom(buf[:n], addr); err != nil {
				return
			}
		}
	}()

	return reqSender, nil
}

// WriteTo implements [PacketRequestSender].WriteTo function. It simply forwards the packet to the underlying
// [net.PacketConn].WriteTo function.
func (s *packetListenerRequestSender) WriteTo(p []byte, destination net.Addr) (int, error) {
	s.conn.SetWriteDeadline(time.Now().Add(s.timeout))
	return s.conn.WriteTo(p, destination)
}

// Close implements [PacketRequestSender].Close function. It closes the underlying [net.PacketConn]. This will also
// terminate the goroutine created in NewSession because s.conn.ReadFrom will return [io.EOF].
func (s *packetListenerRequestSender) Close() error {
	return s.conn.Close()
}
