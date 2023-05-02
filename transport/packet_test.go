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

package transport

import (
	"context"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUDPEndpointIPv4(t *testing.T) {
	const serverAddr = "127.0.0.10:8888"
	ep := &UDPEndpoint{Address: serverAddr}
	ep.Dialer.Control = func(network, address string, c syscall.RawConn) error {
		require.Equal(t, "udp4", network)
		require.Equal(t, serverAddr, address)
		return nil
	}
	conn, err := ep.Connect(context.Background())
	require.Nil(t, err)
	require.Equal(t, serverAddr, conn.RemoteAddr().String())
}

func TestUDPEndpointIPv6(t *testing.T) {
	const serverAddr = "[::1]:8888"
	ep := &UDPEndpoint{Address: serverAddr}
	ep.Dialer.Control = func(network, address string, c syscall.RawConn) error {
		require.Equal(t, "udp6", network)
		require.Equal(t, serverAddr, address)
		return nil
	}
	conn, err := ep.Connect(context.Background())
	require.Nil(t, err)
	require.Equal(t, serverAddr, conn.RemoteAddr().String())
}
