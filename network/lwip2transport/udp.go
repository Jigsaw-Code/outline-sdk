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

	"github.com/Jigsaw-Code/outline-internal-sdk/network"
	lwip "github.com/eycorsican/go-tun2socks/core"
)

// Compilation guard against interface implementation
var _ lwip.UDPConnHandler = (*udpHandler)(nil)
var _ network.PacketResponseWriter = (*udpConnResponseWriter)(nil)

type udpHandler struct {
	mu       sync.Mutex                       // Protects the sessions field
	handler  network.PacketHandler            // A network stack neutral implementation of UDP handler
	sessions map[string]network.PacketSession // Maps local lwIP UDP socket (IPv4:port/[IPv6]:port) to PacketSession
	timeout  time.Duration                    // Session timeout is how long a session can last before closing
}

// newUDPHandler returns a lwIP UDP connection handler.
//
// `h` is a handler that handles UDP packets.
// `timeout` is the UDP read and write timeout.
func newUDPHandler(h network.PacketHandler, timeout time.Duration) *udpHandler {
	return &udpHandler{
		handler:  h,
		timeout:  timeout,
		sessions: make(map[string]network.PacketSession, 8),
	}
}

// Connect creats a new UDP session from the [transport.PacketServer]. It also schedule a timer with the timeout of the
// session; the session will be closed once the timer triggers.
func (h *udpHandler) Connect(tunConn lwip.UDPConn, _ *net.UDPAddr) error {
	laddr := tunConn.LocalAddr().String()

	// Lock the entire function to prevent from creating multiple sessions of one laddr
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.sessions[laddr]; ok {
		return fmt.Errorf("duplicated connection %v", laddr)
	}

	// Try to create a new session
	respw := &udpConnResponseWriter{tunConn}
	session, err := h.handler.NewSession(tunConn.LocalAddr(), respw)
	if err != nil {
		respw.Close()
		return err
	}
	h.sessions[laddr] = session

	// Clean-up the session after timeout
	time.AfterFunc(h.timeout, func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		delete(h.sessions, laddr)
		session.Close()
		respw.Close()
	})
	return nil
}

// ReceiveTo relays packets from the lwIP TUN device to the proxy. It's called by lwIP.
func (h *udpHandler) ReceiveTo(tunConn lwip.UDPConn, data []byte, destAddr *net.UDPAddr) error {
	h.mu.Lock()
	session, ok := h.sessions[tunConn.LocalAddr().String()]
	h.mu.Unlock()

	if !ok {
		return fmt.Errorf("connection %v->%v does not exist", tunConn.LocalAddr(), destAddr)
	}
	_, err := session.WriteRequest(data, destAddr)
	return err
}

// The PacketResponseWriter that will write responses to the lwip network stack.
type udpConnResponseWriter struct {
	lwip.UDPConn
}

// Write relays packets from the proxy to the lwIP TUN device. It's called by transport.PacketSession.
func (r *udpConnResponseWriter) Write(p []byte, from net.Addr) (int, error) {
	srcAddr, err := net.ResolveUDPAddr("udp", from.String())
	if err != nil {
		return 0, err
	}
	return r.WriteFrom(p, srcAddr)
}
