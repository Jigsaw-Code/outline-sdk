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

package network

import "net"

// PacketHandler handles UDP traffic from the network stack. When a user creates a new local UDP socket, the network
// stack calls PacketHandler.NewSession function. The network stack then uses the returned [PacketSession] to send and
// receive UDP packets to and from different servers.
//
// Multiple goroutines may invoke methods on a PacketHandler simultaneously.
type PacketHandler interface {
	// NewSession creates a PacketSession that can send and receive UDP packets. The laddr specifies the local socket
	// endpoint. The PacketResponseWriter w receives packets from remote servers. The PacketSession must use
	// PacketResponseWriter.Write to send UDP response packets.
	NewSession(laddr net.Addr, w PacketResponseWriter) (PacketSession, error)
}

// PacketSession represents a UDP socket session. It can be used to send and receive UDP packets. To send a UDP packet,
// call the WriteRequest function. To receive a UDP packet, the PacketSession will use the PacketResponseWriter object
// passed to the PacketHandler.NewSession function.
//
// Multiple goroutines may invoke methods on a PacketSession simultaneously.
type PacketSession interface {
	// WriteRequest sends a UDP request packet to the target server identified by `to`; and the payload of the packet is
	// stored in `p`; if `p` is empty, no requests will be made to the target server. WriteRequest returns the number of
	// bytes written from p (0 <= n <= len(p)) and any error encountered that caused the function to stop early.
	//
	// `p` must not be modified, and it must not be referenced after WriteRequest returns.
	WriteRequest(p []byte, to net.Addr) (n int, err error)

	// Close ends the session. After Close returns, you must not call WriteRequest any more. The session will also not
	// use the PacketResponseWriter that was passed to PacketHandler.NewSession.
	Close() error
}

// PacketResponseWriter is used to receive UDP response packets. It is usually implemented by the network stack. When
// creating a new UDP session, a valid instance PacketResponseWriter is passed to the PacketHandler.NewSession
// function. Then the [PacketSession] must use this instance to send UDP responses all the time; and it must stop
// using this instance after PacketSession.Close is called. This allows us to receive responses without sending any
// requests to a [PacketSession].
//
// Multiple goroutines may invoke methods on a PacketResponseWriter simultaneously.
type PacketResponseWriter interface {
	// Write is a callback function that is called by a PacketSession when a UDP response packet is received. The `from`
	// identifies the remote server that sent the packet; and the `p` contains the packet payload. Write returns the
	// number of bytes written from p (0 <= n <= len(p)) and any error encountered that caused the function to stop
	// early.
	//
	// `p` must not be modified, and it must not be referenced after WriteRequest returns.
	Write(p []byte, from net.Addr) (n int, err error)
}
