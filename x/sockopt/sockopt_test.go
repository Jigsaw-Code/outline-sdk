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

package sockopt

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTCPOptions(t *testing.T) {
	type Params struct {
		Net  string
		Addr string
	}
	for _, params := range []Params{{Net: "tcp4", Addr: "127.0.0.1:0"}, {Net: "tcp6", Addr: "[::1]:0"}} {
		l, err := net.Listen(params.Net, params.Addr)
		require.NoError(t, err)
		defer l.Close()

		conn, err := net.Dial("tcp", l.Addr().String())
		require.NoError(t, err)
		tcpConn, ok := conn.(*net.TCPConn)
		require.True(t, ok)

		opts, err := NewTCPOptions(tcpConn)
		require.NoError(t, err)

		require.NoError(t, opts.SetHopLimit(1))

		hoplim, err := opts.HopLimit()
		require.NoError(t, err)
		require.Equal(t, 1, hoplim)

		require.NoError(t, opts.SetHopLimit(20))

		hoplim, err = opts.HopLimit()
		require.NoError(t, err)
		require.Equal(t, 20, hoplim)
	}
}
