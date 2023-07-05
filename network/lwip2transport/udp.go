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

package lwip2transport

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Jigsaw-Code/outline-internal-sdk/transport"
	lwip "github.com/eycorsican/go-tun2socks/core"
)

// Compilation guard against interface implementation
var _ lwip.UDPConnHandler = (*udpHandler)(nil)
var _ transport.PacketResponseWriter = (*udpResponseWriter)(nil)

type udpHandler struct {
	mu      sync.Mutex // protect the session field
	server  transport.PacketServer
	session transport.PacketHandler
	respw   *udpResponseWriter

	// How long to wait for a packet from the proxy. Longer than this and the connection
	// is closed.
	timeout time.Duration
}

type udpResponseWriter struct {
	lwip.UDPConn
}

// newUDPHandler returns a lwIP UDP connection handler.
//
// `pkt` is a server that handles UDP packets.
// `timeout` is the UDP read and write timeout.
func newUDPHandler(pkt transport.PacketServer, timeout time.Duration) *udpHandler {
	return &udpHandler{
		server:  pkt,
		timeout: timeout,
	}
}

// Connect creats a new UDP session from the [transport.PacketServer]. It also schedule a timer with the timeout of the
// session; the session will be closed once the timer triggers.
func (h *udpHandler) Connect(tunConn lwip.UDPConn, _ *net.UDPAddr) (err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.respw = &udpResponseWriter{tunConn}
	h.session, err = h.server.NewSession(tunConn.LocalAddr())
	if err != nil {
		h.respw.Close()
		return
	}
	time.AfterFunc(h.timeout, func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		h.session.Close()
		h.respw.Close()
		h.session = nil
	})
	return
}

// ReceiveTo relays packets from the lwIP TUN device to the proxy. It's called by lwIP.
func (h *udpHandler) ReceiveTo(tunConn lwip.UDPConn, data []byte, destAddr *net.UDPAddr) error {
	h.mu.Lock()
	session := h.session
	respw := h.respw
	h.mu.Unlock()

	if session == nil {
		return fmt.Errorf("connection %v->%v does not exist", tunConn.LocalAddr(), destAddr)
	}
	return session.ServePacket(respw, &transport.PacketRequest{
		Body: data,
		Dest: destAddr,
	})
}

// Write relays packets from the proxy to the lwIP TUN device. It's called by [transport.PacketHandler].
func (r *udpResponseWriter) Write(p []byte, from net.Addr) (int, error) {
	srcAddr, err := net.ResolveUDPAddr("udp", from.String())
	if err != nil {
		return 0, err
	}
	return r.WriteFrom(p, srcAddr)
}
