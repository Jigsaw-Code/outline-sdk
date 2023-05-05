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
	"fmt"
	"net"
)

// PacketEndpoint represents an endpoint that can be used to established packet connections (like UDP) to a fixed destination.
type PacketEndpoint interface {
	// Connect creates a connection bound to an endpoint, returning the connection.
	Connect(ctx context.Context) (net.Conn, error)
}

// UDPEndpoint is a [PacketEndpoint] that connects to the given address via UDP
type UDPEndpoint struct {
	// The Dialer used to create the net.Conn on Connect().
	Dialer net.Dialer
	// The endpoint address (host:port) to pass to Dial.
	// If the host is a domain name, consider pre-resolving it to avoid resolution calls.
	Address string
}

var _ PacketEndpoint = (*UDPEndpoint)(nil)

// Connect implements [PacketEndpoint.Connect].
func (e UDPEndpoint) Connect(ctx context.Context) (net.Conn, error) {
	return e.Dialer.DialContext(ctx, "udp", e.Address)
}

// PacketListenerEndpoint is a [PacketEndpoint] that connects to the given address using the given [PacketListener].
type PacketListenerEndpoint struct {
	// The Dialer used to create the net.Conn on Connect(). Must be non nil.
	Listener PacketListener
	// The endpoint address (host:port) to bind the connection to.
	// If the host is a domain name, consider pre-resolving it to avoid resolution calls.
	Address string
}

var _ PacketEndpoint = (*PacketListenerEndpoint)(nil)

type boundPacketConn struct {
	net.PacketConn
	remoteAddr net.Addr
}

var _ net.Conn = (*boundPacketConn)(nil)

func (e PacketListenerEndpoint) Connect(ctx context.Context) (net.Conn, error) {
	packetConn, err := e.Listener.ListenPacket(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create PacketConn: %#v", err)
	}
	netAddr, err := MakeNetAddr("udp", e.Address)
	if err != nil {
		return nil, err
	}
	return &boundPacketConn{
		PacketConn: packetConn,
		remoteAddr: netAddr,
	}, nil
}

func (c *boundPacketConn) Read(packet []byte) (int, error) {
	for {
		n, remoteAddr, err := c.PacketConn.ReadFrom(packet)
		if err != nil {
			return n, err
		}
		if remoteAddr.String() != c.remoteAddr.String() {
			continue
		}
		return n, nil
	}
}

func (c *boundPacketConn) Write(packet []byte) (int, error) {
	n, err := c.PacketConn.WriteTo(packet, c.remoteAddr)
	return n, err
}

func (c *boundPacketConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

// PacketListener provides a way to create a local unbound packet connection to send packets to different destinations.
type PacketListener interface {
	// ListenPacket creates a PacketConn that can be used to relay packets (such as UDP) through some proxy.
	ListenPacket(ctx context.Context) (net.PacketConn, error)
}

// UDPPacketListener is a [PacketListener] that uses the standard [net.ListenConfig].ListenPacket to listen.
type UDPPacketListener struct {
	net.ListenConfig
	// The local address to bind to, as specified in [net.ListenPacket].
	Address string
}

var _ PacketListener = (*UDPPacketListener)(nil)

func (l UDPPacketListener) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	return l.ListenConfig.ListenPacket(ctx, "udp", l.Address)
}
