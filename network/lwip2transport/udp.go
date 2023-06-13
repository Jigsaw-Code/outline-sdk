// Copyright 2023 Google LLC
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
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	lwip "github.com/eycorsican/go-tun2socks/core"
)

// Compilation guard against interface implementation
var _ lwip.UDPConnHandler = (*udpHandler)(nil)

type udpHandler struct {
	// Protects the connections map
	sync.Mutex

	// Used to establish connections to the proxy
	listener transport.PacketListener

	// How long to wait for a packet from the proxy. Longer than this and the connection
	// is closed.
	timeout time.Duration

	// Maps local UDP addresses (IPv4:port/[IPv6]:port) to connections to the proxy.
	conns map[string]net.PacketConn
}

// newUDPHandler returns a lwIP UDP connection handler.
//
// `pl` is an UDP packet listener.
// `timeout` is the UDP read and write timeout.
func newUDPHandler(pl transport.PacketListener, timeout time.Duration) *udpHandler {
	return &udpHandler{
		listener: pl,
		timeout:  timeout,
		conns:    make(map[string]net.PacketConn, 8),
	}
}

func (h *udpHandler) Connect(tunConn lwip.UDPConn, target *net.UDPAddr) error {
	proxyConn, err := h.listener.ListenPacket(context.Background())
	if err != nil {
		return err
	}
	h.Lock()
	h.conns[tunConn.LocalAddr().String()] = proxyConn
	h.Unlock()
	go h.relayPacketsFromProxy(tunConn, proxyConn)
	return nil
}

// relayPacketsFromProxy relays packets from the proxy to the TUN device.
func (h *udpHandler) relayPacketsFromProxy(tunConn lwip.UDPConn, proxyConn net.PacketConn) {
	buf := lwip.NewBytes(lwip.BufSize)
	defer func() {
		h.close(tunConn)
		lwip.FreeBytes(buf)
	}()
	for {
		proxyConn.SetDeadline(time.Now().Add(h.timeout))
		n, sourceAddr, err := proxyConn.ReadFrom(buf)
		if err != nil {
			return
		}
		// No resolution will take place, the address sent by the proxy is a resolved IP.
		sourceUDPAddr, err := net.ResolveUDPAddr("udp", sourceAddr.String())
		if err != nil {
			return
		}
		_, err = tunConn.WriteFrom(buf[:n], sourceUDPAddr)
		if err != nil {
			return
		}
	}
}

// ReceiveTo relays packets from the TUN device to the proxy. It's called by tun2socks.
func (h *udpHandler) ReceiveTo(tunConn lwip.UDPConn, data []byte, destAddr *net.UDPAddr) error {
	h.Lock()
	proxyConn, ok := h.conns[tunConn.LocalAddr().String()]
	h.Unlock()
	if !ok {
		return fmt.Errorf("connection %v->%v does not exist", tunConn.LocalAddr(), destAddr)
	}
	proxyConn.SetDeadline(time.Now().Add(h.timeout))
	_, err := proxyConn.WriteTo(data, destAddr)
	return err
}

func (h *udpHandler) close(tunConn lwip.UDPConn) {
	laddr := tunConn.LocalAddr().String()
	tunConn.Close()
	h.Lock()
	defer h.Unlock()
	if proxyConn, ok := h.conns[laddr]; ok {
		proxyConn.Close()
		delete(h.conns, laddr)
	}
}
