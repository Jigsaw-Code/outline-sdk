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

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
)

const packetMaxSize = 2048

// Compilation guard against interface implementation
var _ PacketHandler = (*packetListenerHandlerAdapter)(nil)
var _ PacketSession = (*packetListenerSessionAdapter)(nil)

type packetListenerHandlerAdapter struct {
	transport.PacketListener
}

type packetListenerSessionAdapter struct {
	net.PacketConn
	response     PacketResponseWriter
	listenerDone chan struct{}
}

// NewPacketHandlerFromPacketListener creates a new [PacketHandler] that uses the existing [transport.PacketListener].
// You can use this function if you already have an implementation of [transport.PacketListener] and would like to
// inject it into one of the network stacks (for example, network/lwip2transport) as UDP traffic handlers.
func NewPacketHandlerFromPacketListener(pl transport.PacketListener) (PacketHandler, error) {
	if pl == nil {
		return nil, errors.New("pl must not be nil")
	}
	return &packetListenerHandlerAdapter{pl}, nil
}

// NewSession implements [PacketHandler].NewSession function. It uses [transport.PacketListener].ListenPacket to create
// a [net.PacketConn], and constructs a new [PacketSession] that is based on this [net.PacketConn].
func (plh *packetListenerHandlerAdapter) NewSession(laddr net.Addr, w PacketResponseWriter) (PacketSession, error) {
	if w == nil {
		return nil, errors.New("w must not be nil")
	}
	conn, err := plh.ListenPacket(context.Background())
	if err != nil {
		return nil, err
	}
	session := &packetListenerSessionAdapter{
		PacketConn:   conn,
		response:     w,
		listenerDone: make(chan struct{}),
	}
	go session.listenPackets()
	return session, nil
}

// WriteRequest implements [PacketSession].WriteRequest function. It simply forwards the packet to the underlying
// [net.PacketConn].WriteTo function.
func (s *packetListenerSessionAdapter) WriteRequest(p []byte, to net.Addr) (n int, err error) {
	return s.WriteTo(p, to)
}

// Close implements [PacketSession].Close function. It closes the underlying [net.PacketConn] and wait for all
// goroutines to finish.
func (s *packetListenerSessionAdapter) Close() error {
	err := s.PacketConn.Close()
	<-s.listenerDone
	return err
}

// listenPackets reads packets from the [net.PacketConn] created by the [transport.PacketListener] until ReadFrom
// returns [io.EOF]. s.Close() will make ReadFrom returns [io.EOF] as well.
func (s *packetListenerSessionAdapter) listenPackets() {
	defer close(s.listenerDone)
	buf := make([]byte, 0, packetMaxSize)
	for {
		n, addr, err := s.ReadFrom(buf[:])
		if err != nil {
			return
		}
		if _, err := s.response.Write(buf[:n], addr); err != nil {
			return
		}
	}
}
