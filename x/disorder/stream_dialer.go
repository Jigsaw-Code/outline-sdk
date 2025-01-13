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

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/sockopt"
)

type disorderDialer struct {
	dialer          transport.StreamDialer
	disorderPacketN int
}

var _ transport.StreamDialer = (*disorderDialer)(nil)

// NewStreamDialer creates a [transport.StreamDialer].
// It works like this:
// * Wait for disorderPacketN'th call to Write. All Write requests before and after the target packet are written normally.
// * Send the disorderPacketN'th packet with TTL == 1.
// * This packet is dropped somewhere in the network and never reaches the server.
// * TTL is restored.
// * The next part of data is sent normally.
// * Server notices the lost fragment and requests re-transmission of lost packet.
func NewStreamDialer(dialer transport.StreamDialer, disorderPacketN int) (transport.StreamDialer, error) {
	if dialer == nil {
		return nil, errors.New("argument dialer must not be nil")
	}
	if disorderPacketN < 0 {
		return nil, fmt.Errorf("disorder argument must be >= 0, got %d", disorderPacketN)
	}
	return &disorderDialer{dialer: dialer, disorderPacketN: disorderPacketN}, nil
}

// DialStream implements [transport.StreamDialer].DialStream.
func (d *disorderDialer) DialStream(ctx context.Context, remoteAddr string) (transport.StreamConn, error) {
	innerConn, err := d.dialer.DialStream(ctx, remoteAddr)
	if err != nil {
		return nil, err
	}

	tcpInnerConn, ok := innerConn.(*net.TCPConn)
	if !ok {
		return nil, fmt.Errorf("disorder strategy: expected base dialer to return TCPConn")
	}
	tcpOptions, err := sockopt.NewTCPOptions(tcpInnerConn)
	if err != nil {
		return nil, err
	}

	dw := NewWriter(innerConn, tcpOptions, d.disorderPacketN)

	return transport.WrapConn(innerConn, innerConn, dw), nil
}
