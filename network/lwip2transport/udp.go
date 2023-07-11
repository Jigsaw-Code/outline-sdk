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

	"github.com/Jigsaw-Code/outline-internal-sdk/network"
	lwip "github.com/eycorsican/go-tun2socks/core"
)

// Compilation guard against interface implementation
var _ lwip.UDPConnHandler = (*udpHandler)(nil)
var _ network.PacketResponseReceiver = (*udpConnResponseWriter)(nil)

type udpHandler struct {
	mu      sync.Mutex                             // Protects the sessions field
	handler network.PacketProxy                    // A network stack neutral implementation of UDP handler
	senders map[string]network.PacketRequestSender // Maps local lwIP UDP socket (IPv4:port/[IPv6]:port) to PacketSession
}

// newUDPHandler returns a lwIP UDP connection handler.
//
// `h` is a handler that handles UDP packets.
// `timeout` is the UDP read and write timeout.
func newUDPHandler(h network.PacketProxy) *udpHandler {
	return &udpHandler{
		handler: h,
		senders: make(map[string]network.PacketRequestSender, 8),
	}
}

// Connect creats a new UDP session from the [transport.PacketServer]. It also schedule a timer with the timeout of the
// session; the session will be closed once the timer triggers.
func (h *udpHandler) Connect(tunConn lwip.UDPConn, _ *net.UDPAddr) error {
	return h.newSession(tunConn)
}

// ReceiveTo relays packets from the lwIP TUN device to the proxy. It's called by lwIP.
func (h *udpHandler) ReceiveTo(tunConn lwip.UDPConn, data []byte, destAddr *net.UDPAddr) error {
	laddr := tunConn.LocalAddr().String()

	h.mu.Lock()
	reqSender, ok := h.senders[laddr]
	h.mu.Unlock()

	if !ok {
		return fmt.Errorf("no session found for local address %v", laddr)
	}
	_, err := reqSender.WriteTo(data, destAddr)
	return err
}

// newSession creates a new UDP session related to conn.
func (h *udpHandler) newSession(conn lwip.UDPConn) error {
	laddr := conn.LocalAddr().String()

	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.senders[laddr]; ok {
		return fmt.Errorf("session already exists for local address %v", laddr)
	}

	respWriter := &udpConnResponseWriter{conn, h}

	// TODO: we can move this out of h.mu.Lock() to increase performance
	//       but that requires additional handling of early arrived packets
	reqSender, err := h.handler.NewSession(respWriter)
	if err != nil {
		respWriter.Close()
		return err
	}

	h.senders[laddr] = reqSender
	return nil
}

// closeSession cleans up resources related to conn.
func (h *udpHandler) closeSession(conn lwip.UDPConn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	laddr := conn.LocalAddr().String()
	if reqSender, ok := h.senders[laddr]; ok {
		reqSender.Close()
		delete(h.senders, laddr)
	}
}

// The PacketResponseWriter that will write responses to the lwip network stack.
type udpConnResponseWriter struct {
	conn lwip.UDPConn
	h    *udpHandler
}

// Write relays packets from the proxy to the lwIP TUN device. It's called by transport.PacketSession.
func (r *udpConnResponseWriter) WriteFrom(p []byte, source net.Addr) (int, error) {
	srcAddr, err := net.ResolveUDPAddr("udp", source.String())
	if err != nil {
		return 0, err
	}
	return r.WriteFrom(p, srcAddr)
}

func (r *udpConnResponseWriter) Close() error {
	err := r.conn.Close()
	r.h.closeSession(r.conn)
	return err
}
