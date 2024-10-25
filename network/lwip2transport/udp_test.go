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

package lwip2transport

import (
	"errors"
	"net"
	"net/netip"
	"testing"

	"github.com/Jigsaw-Code/outline-sdk/network"
	"github.com/stretchr/testify/require"
)

// Make sure we can successfully Close the request sender and response receiver wihout deadlock
func TestUDPResponseWriterCloseNoDeadlock(t *testing.T) {
	proxy := &noopSingleSessionPacketProxy{}
	h := newUDPHandler(proxy)

	// Create one and only one session in the proxy
	localAddr := net.UDPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:60127"))
	destAddr := net.UDPAddrFromAddrPort(netip.MustParseAddrPort("1.2.3.4:4321"))
	err := h.ReceiveTo(&noopLwIPUDPConn{localAddr}, []byte{}, destAddr)
	require.NoError(t, err)

	// Close this single session (i.e. the request sender), it will close proxy.respWriter
	// udpHandler must make sure only one `Close()` is called, and there should be no deadlocks
	err = proxy.respWriter.Close()
	require.NoError(t, err)
	require.Exactly(t, 1, proxy.closeCnt)
}

/********** Test Utilities **********/

type noopSingleSessionPacketProxy struct {
	closeCnt   int
	respWriter network.PacketResponseReceiver
}

func (p *noopSingleSessionPacketProxy) NewSession(respWriter network.PacketResponseReceiver) (network.PacketRequestSender, error) {
	if p.respWriter != nil {
		return nil, errors.New("don't support multiple sessions in this proxy")
	}
	p.respWriter = respWriter
	return p, nil
}

func (p *noopSingleSessionPacketProxy) Close() error {
	p.closeCnt++
	return p.respWriter.Close()
}

func (p *noopSingleSessionPacketProxy) WriteTo([]byte, netip.AddrPort) (int, error) {
	return 0, nil
}

type noopLwIPUDPConn struct {
	localAddr *net.UDPAddr
}

func (*noopLwIPUDPConn) Close() error {
	return nil
}

func (conn *noopLwIPUDPConn) LocalAddr() *net.UDPAddr {
	return conn.localAddr
}

func (*noopLwIPUDPConn) ReceiveTo(data []byte, addr *net.UDPAddr) error {
	return nil
}

func (*noopLwIPUDPConn) WriteFrom(data []byte, addr *net.UDPAddr) (int, error) {
	return 0, nil
}
