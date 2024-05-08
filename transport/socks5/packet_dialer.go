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

package socks5

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type packetConn struct {
	dstAddr net.Addr
	pc      net.Conn
	sc      transport.StreamConn
}

var _ net.Conn = (*packetConn)(nil)

func (c *packetConn) LocalAddr() net.Addr {
	// TODO: Is this right?
	return c.pc.LocalAddr()
}

func (c *packetConn) RemoteAddr() net.Addr {
	return c.dstAddr
}

func (c *packetConn) SetDeadline(t time.Time) error {
	return c.pc.SetDeadline(t)
}

func (c *packetConn) SetReadDeadline(t time.Time) error {
	return c.pc.SetReadDeadline(t)
}

func (c *packetConn) SetWriteDeadline(t time.Time) error {
	return c.pc.SetWriteDeadline(t)
}

func (c *packetConn) Read(b []byte) (int, error) {
	// TODO: read header
	return c.pc.Read(b)
}

func (c *packetConn) Write(b []byte) (int, error) {
	// TODO: write header
	return c.pc.Write(b)
}

func (c *packetConn) Close() error {
	return errors.Join(c.sc.Close(), c.pc.Close())
}

// DialPacket creates a packet [net.Conn] via SOCKS5.
func (d *Dialer) DialPacket(ctx context.Context, dstAddr string) (net.Conn, error) {
	netDstAddr, err := transport.MakeNetAddr("udp", dstAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse address: %w", err)
	}

	sc, bindAddr, err := d.request(ctx, CmdUDPAssociate, dstAddr)
	if err != nil {
		return nil, err
	}

	pc, err := d.pd.DialPacket(ctx, bindAddr)
	if err != nil {
		sc.Close()
		return nil, fmt.Errorf("failed to connect to packet endpoint: %w", err)
	}

	return &packetConn{netDstAddr, pc, sc}, nil
}
