// Copyright 2024 The Outline Authors
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

package disorder

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

var defaultTTL = 64

type disorderDialer struct {
	dialer     transport.StreamDialer
	splitPoint int64
}

var _ transport.StreamDialer = (*disorderDialer)(nil)

// NewStreamDialer creates a [transport.StreamDialer]
// It work almost the same as the other split dialer, however, it also manipulates socket TTL:
// * Before sending the first prefixBytes TTL is set to 1
// * This packet is dropped somewhere in the network and never reaches the server
// * TTL is restored
// * The next part of data is sent normally
// * Server notices the lost fragment and requests re-transmission
// Currently this only works with Linux kernel (for Windows/Mac a different implementation is required)
func NewStreamDialer(dialer transport.StreamDialer, prefixBytes int64) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	return &disorderDialer{dialer: dialer, splitPoint: prefixBytes}, nil
}

// DialStream implements [transport.StreamDialer].DialStream.
func (d *disorderDialer) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.dialer.DialStream(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}

	oldTTL, err := setHopLimit(innerConn, 1)
	if err != nil {
		return nil, fmt.Errorf("disorder strategy: failed to change ttl: %w", err)
	}

	dw := NewWriter(innerConn, d.splitPoint, oldTTL)

	return transport.WrapConn(innerConn, innerConn, dw), nil
}

// setHopLimit changes the socket TTL for IPv4 (or HopLimit for IPv6) and returns the old value
// socket must be `*net.TCPConn`
func setHopLimit(conn net.Conn, ttl int) (oldTTL int, err error) {
	addr, err := netip.ParseAddrPort(conn.RemoteAddr().String())
	if err != nil {
		return 0, fmt.Errorf("could not parse remote addr: %w", err)
	}

	switch {
	case addr.Addr().Is4():
		conn := ipv4.NewConn(conn)
		oldTTL, _ = conn.TTL()
		err = conn.SetTTL(ttl)
	case addr.Addr().Is6():
		conn := ipv6.NewConn(conn)
		oldTTL, _ = conn.HopLimit()
		err = conn.SetHopLimit(ttl)
	default:
		return 0, fmt.Errorf("unknown remote addr type (%v)", addr.Addr().String())
	}
	if err != nil {
		return 0, fmt.Errorf("failed to change TTL: %w", err)
	}

	if oldTTL == 0 {
		oldTTL = defaultTTL
	}

	return
}
