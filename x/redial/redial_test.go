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

package redial

import (
	"io"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/ipv4"
)

func TestSyn(t *testing.T) {
	conn, err := net.Dial("tcp4", "cloudflare.net.:80")
	conn.(*net.TCPConn).SetLinger(0)
	require.NoError(t, err)
	localAddr := conn.LocalAddr()
	remoteAddr := conn.RemoteAddr()
	log.Println("localAddr", localAddr)
	log.Println("remoteAddr", remoteAddr)
	conn4 := ipv4.NewConn(conn)
	conn4.SetTTL(1)
	conn.Close()
	dialer := net.Dialer{LocalAddr: localAddr}
	conn, err = dialer.Dial("tcp4", remoteAddr.String())
	require.NoError(t, err)

	_, err = conn.Write([]byte("HEAD / HTTP/1.1\r\nHost: meduza.io\r\n\r\n"))
	require.NoError(t, err)
	conn.(*net.TCPConn).CloseWrite()

	response, err := io.ReadAll(conn)
	require.NoError(t, err)
	log.Println(string(response))
	require.NoError(t, conn.Close())
}
