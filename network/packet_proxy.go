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

// PacketProxy handles UDP traffic from the upstream network stack. The upstream network stack uses the NewSession
// function to create a new UDP session that can send or receive UDP packets from PacketProxy.
//
// Multiple goroutines may invoke methods on a PacketProxy simultaneously.
type PacketProxy interface {
	// NewSession function tells the PacketProxy that a new UDP socket session has been started (using socket as an
	// example, a session will be started by calling the bind() function). The PacketProxy then creates a
	// PacketRequestSender object to handle requests from this session, and it also uses the PacketResponseReceiver
	// to send responses back to the upstream network stack.
	//
	// Note that, it is possible for a session to receive UDP packets without sending any requests.
	NewSession(PacketResponseReceiver) (PacketRequestSender, error)
}

// PacketRequestSender sends UDP request packets to the [PacketProxy]. It should be implemented by the [PacketProxy],
// who must implement the WriteTo method to process the request packets. PacketRequestSender is typically called by an
// upstream component (such as a network stack). Once the Close method is called, there will be no more requests sent
// to the sender, and all resources can be freed.
//
// Multiple goroutines may invoke methods on a PacketRequestSender simultaneously.
type PacketRequestSender interface {
	// WriteTo sends a UDP request packet to the [PacketProxy]. The packet is destined for the remote server identified
	// by `destination`; and the payload of the packet is stored in `p`. If `p` is empty, the request packet will be
	// ignored. WriteTo returns the number of bytes written from `p` and any error encountered that caused the function
	// to stop early.
	//
	// `p` must not be modified, and it must not be referenced after WriteTo returns.
	WriteTo(p []byte, destination net.Addr) (int, error)

	// Close indicates that the sender is no longer accepting new requests. Any future attempts to call WriteTo on the
	// sender will fail with ErrClosed.
	Close() error
}

// PacketResponseReceiver receives UDP response packets from the [PacketProxy]. It is usually implemented by the
// upstream component (such as a network stack). When creating a new UDP session, a valid instance
// PacketResponseReceiver is passed to the PacketProxy.NewSession function. Then the [PacketProxy] must use this
// instance to send UDP responses all the time; and it must call the Close function to indicate there will be no more
// responses send to this receiver.
//
// Multiple goroutines may invoke methods on a PacketResponseReceiver simultaneously.
type PacketResponseReceiver interface {
	// WriteFrom is a callback function that is called by a PacketProxy when a UDP response packet is received. The
	// `source` identifies the remote server that sent the packet; and the `p` contains the packet payload. WriteFrom
	// returns the number of bytes written from `p` and any error encountered that caused the function to stop early.
	//
	// `p` must not be modified, and it must not be referenced after WriteFrom returns.
	WriteFrom(p []byte, source net.Addr) (int, error)

	// Close indicates that the receiver is no longer accepting new responses. Any future attempts to call WriteFrom on
	// the receiver will fail with ErrClosed.
	Close() error
}
