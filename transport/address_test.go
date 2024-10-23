// Copyright 2023 The Outline Authors
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
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMakeNetAddrType(t *testing.T) {
	for _, address := range []string{"example.com:53", "127.0.0.1:443", "[::1]:443"} {
		for _, network := range []string{"tcp", "udp"} {
			netAddr, err := MakeNetAddr(network, address)
			require.NoError(t, err)
			require.Equal(t, network, netAddr.Network())
			require.Equal(t, address, netAddr.String())
			if address == "example.com:53" {
				require.IsType(t, &domainAddr{}, netAddr)
			} else {
				switch network {
				case "udp":
					require.IsType(t, &net.UDPAddr{}, netAddr)
				case "tcp":
					require.IsType(t, &net.TCPAddr{}, netAddr)
				}
			}
		}
	}
}

func TestMakeNetAddrDomainCase(t *testing.T) {
	netAddr, err := MakeNetAddr("tcp", "Example.Com:83")
	require.NoError(t, err)
	require.Equal(t, "Example.Com:83", netAddr.String())
}

func TestMakeNetAddrIP4(t *testing.T) {
	netAddr, err := MakeNetAddr("tcp", "127.0.0.1:83")
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:83", netAddr.String())
}

func TestMakeNetAddrIP6(t *testing.T) {
	netAddr, err := MakeNetAddr("tcp", "[0000:0000:0000::0001]:83")
	require.NoError(t, err)
	require.Equal(t, "[::1]:83", netAddr.String())
}

func TestMakeNetAddrResolvePort(t *testing.T) {
	netAddr, err := MakeNetAddr("udp", "example.com:domain")
	require.NoError(t, err)
	require.Equal(t, "example.com:53", netAddr.String())
}
