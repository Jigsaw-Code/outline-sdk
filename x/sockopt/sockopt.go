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
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type HasWaitUntilBytesAreSent interface {
	// Wait until all bytes are sent to the socket.
	// Returns ErrUnsupported if the platform doesn't support it.
	// May return a different error.
	WaitUntilBytesAreSent() error
	// Checks if the OS supports waiting until the bytes are sent
	OsSupportsWaitingUntilBytesAreSent() bool
}

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
	HasWaitUntilBytesAreSent
	HasHopLimit
}

type tcpOptions struct {
	hopLimitOption

	conn *net.TCPConn

	// Timeout after which we return an error
	waitingTimeout time.Duration
	// Delay between checking the socket
	waitingDelay time.Duration
}

var _ TCPOptions = (*tcpOptions)(nil)

func (o *tcpOptions) SetWaitingTimeout(timeout time.Duration) {
	o.waitingTimeout = timeout
}

func (o *tcpOptions) SetWaitingDelay(delay time.Duration) {
	o.waitingDelay = delay
}

func (o *tcpOptions) OsSupportsWaitingUntilBytesAreSent() bool {
	return isConnectionSendingBytesImplemented()
}

func (o *tcpOptions) WaitUntilBytesAreSent() error {
	startTime := time.Now()
	for time.Since(startTime) < o.waitingTimeout {
		isSendingBytes, err := isConnectionSendingBytes(o.conn)
		if err != nil {
			return err
		}
		if !isSendingBytes {
			return nil
		}

		time.Sleep(o.waitingDelay)
	}
	return fmt.Errorf("waiting for socket to send all bytes: timeout exceeded")
}

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
	return &tcpOptions{
		hopLimitOption: *hopLimit,
		conn:           conn,
		waitingTimeout: 10 * time.Millisecond,
		waitingDelay:   100 * time.Microsecond,
	}, nil
}
