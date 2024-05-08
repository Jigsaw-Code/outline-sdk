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

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

type PacketDialer struct {
	se   transport.StreamEndpoint
	pe   transport.PacketEndpoint
	cred *credentials
}

var _ transport.PacketDialer = (*PacketDialer)(nil)

// NewPacketDialer creates a [transport.PacketDialer] that routes connections to a SOCKS5
// proxy listening at the given endpoint.
func NewPacketDialer(streamEndpoint transport.StreamEndpoint, packetEndpoint transport.PacketEndpoint) (transport.PacketDialer, error) {
	if streamEndpoint == nil || packetEndpoint == nil {
		return nil, errors.New("must specify both endpoints")
	}
	return &PacketDialer{se: streamEndpoint, pe: packetEndpoint, cred: nil}, nil
}

// DialPacket creates a packet [net.Conn] via SOCKS5.
func (pd *PacketDialer) DialPacket(ctx context.Context, remoteAddr string) (net.Conn, error) {
	sc, err := pd.se.ConnectStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to stream endpoint: %w", err)
	}

	return nil, errors.ErrUnsupported
}
