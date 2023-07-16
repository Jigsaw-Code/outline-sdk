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
type PacketEndpoint = Endpoint[net.Conn]

// UDPEndpoint is a [PacketEndpoint] that connects to the given address via UDP
type UDPEndpoint struct {
	// The Dialer used to create the net.Conn on Connect().
	Dialer net.Dialer
	// The endpoint address (host:port) to pass to Dial.
	// If the host is a domain name, consider pre-resolving it to avoid resolution calls.
	Address string
}

var _ PacketEndpoint = (*UDPEndpoint)(nil)

// Connect implements [PacketEndpoint].Connect.
func (e UDPEndpoint) Connect(ctx context.Context) (net.Conn, error) {
	return e.Dialer.DialContext(ctx, "udp", e.Address)
}

// UDPDialer is a [PacketDialer] that uses the standard [net.Dialer] to dial.
// It provides a convenient way to use a [net.Dialer] when you need a [PacketDialer].
type UDPDialer struct {
	Dialer net.Dialer
}

var _ PacketDialer = (*UDPDialer)(nil)

// Dial implements [Dialer].Dial.
func (d *UDPDialer) Dial(ctx context.Context, addr string) (net.Conn, error) {
	return d.Dialer.DialContext(ctx, "udp", addr)
}

// PacketDialer provides a way to dial a destination and establish datagram connections.
type PacketDialer = Dialer[net.Conn]

// PacketListenerDialer is a [PacketDialer] that connects to the destination using the given [PacketListener].
type PacketListenerDialer struct {
	// The PacketListener that is used to create the net.PacketConn to bind on Dial. Must be non nil.
	Listener PacketListener
}

var _ PacketDialer = (*PacketListenerDialer)(nil)

type boundPacketConn struct {
	net.PacketConn
	remoteAddr net.Addr
}

var _ net.Conn = (*boundPacketConn)(nil)

// Dial implements [PacketDialer].Dial.
// The address is a host:port and the host must be a full IP address (not [::]) or a domain
// The address must be supported by the WriteTo call of the PacketConn
// returned by the PacketListener. For instance, a [net.UDPConn] only supports IP addresses, not domain names.
// If the host is a domain name, consider pre-resolving it to avoid resolution calls.
func (e PacketListenerDialer) Dial(ctx context.Context, address string) (net.Conn, error) {
	packetConn, err := e.Listener.ListenPacket(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create PacketConn: %w", err)
	}
	netAddr, err := MakeNetAddr("udp", address)
	if err != nil {
		return nil, err
	}
	return &boundPacketConn{
		PacketConn: packetConn,
		remoteAddr: netAddr,
	}, nil
}

// Read implements [net.Conn].Read.
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

// Write implements [net.Conn].Write.
func (c *boundPacketConn) Write(packet []byte) (int, error) {
	// This may return syscall.EINVAL if remoteAddr is a name like localhost or [::].
	n, err := c.PacketConn.WriteTo(packet, c.remoteAddr)
	return n, err
}

// RemoteAddr implements [net.Conn].RemoteAddr.
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

// ListenPacket implements [PacketListener].ListenPacket
func (l UDPPacketListener) ListenPacket(ctx context.Context) (net.PacketConn, error) {
	return l.ListenConfig.ListenPacket(ctx, "udp", l.Address)
}
