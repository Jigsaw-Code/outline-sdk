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

// Package sockopt provides cross-platform ways to interact with socket options.
package sockopt

import (
	"fmt"
	"net"
	"net/netip"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// HasHopLimit enables manipulation of the hop limit option.
type HasHopLimit interface {
	// HopLimit returns the hop limit field value for outgoing packets.
	HopLimit() (int, error)
	// SetHopLimit sets the hop limit field value for future outgoing packets.
	SetHopLimit(hoplim int) error
}

// hopLimitOption implements HasHopLimit.
type hopLimitOption struct {
	hopLimit    func() (int, error)
	setHopLimit func(hoplim int) error
}

func (o *hopLimitOption) HopLimit() (int, error) {
	return o.hopLimit()
}

func (o *hopLimitOption) SetHopLimit(hoplim int) error {
	return o.setHopLimit(hoplim)
}

var _ HasHopLimit = (*hopLimitOption)(nil)

// TCPOptions represents options for TCP connections.
type TCPOptions interface {
	HasHopLimit
}

type tcpOptions struct {
	hopLimitOption
}

var _ TCPOptions = (*tcpOptions)(nil)

// newHopLimit creates a hopLimitOption from a [net.Conn]. Works for both TCP or UDP.
func newHopLimit(conn net.Conn) (*hopLimitOption, error) {
	addr, err := netip.ParseAddrPort(conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}
	opt := &hopLimitOption{}
	switch {
	case addr.Addr().Is4():
		ipConn := ipv4.NewConn(conn)
		opt.hopLimit = ipConn.TTL
		opt.setHopLimit = ipConn.SetTTL
	case addr.Addr().Is6():
		ipConn := ipv6.NewConn(conn)
		opt.hopLimit = ipConn.HopLimit
		opt.setHopLimit = ipConn.SetHopLimit
	default:
		return nil, fmt.Errorf("address is not IPv4 or IPv6 (%v)", addr.Addr().String())
	}
	return opt, nil
}

// NewTCPOptions creates a [TCPOptions] for the given [net.TCPConn].
func NewTCPOptions(conn *net.TCPConn) (TCPOptions, error) {
	hopLimit, err := newHopLimit(conn)
	if err != nil {
		return nil, err
	}
	return &tcpOptions{hopLimitOption: *hopLimit}, nil
}
