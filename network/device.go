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

package network

import (
	"fmt"
	"syscall"
)

var (
	// ErrMsgSize is the error returned by a Write on a network device. It means that the size of the message to be
	// sent is bigger than the maximum message size the device can process.
	ErrMsgSize = fmt.Errorf("packet size is too big: %w", syscall.EMSGSIZE)
)

// IPDevice is a generic network device that reads and writes IP packets. It extends the [io.ReadWriteCloser]
// interface. For better memory efficiency, we also recommend that you implement the [io.ReaderFrom] and [io.WriterTo]
// interfaces if possible.
//
// Some examples of IPDevices are a virtual network adapter or a local IP proxy.
type IPDevice interface {
	// Close closes this device. Any future Read will return io.EOF and Write will return ErrClosed.
	Close() error

	// Read reads an IP packet from this device into p, returning the number of bytes read. It blocks until a full IP
	// packet has been received. Note that an IP packet might be fragmented, and we will not reassemble it. Instead, we
	// will simply return the fragmented packets to the caller.
	//
	// If len(p) is smaller than the incoming IP packet, only len(p) bytes will be copied to p, the excess bytes are
	// discarded (this aligns with the socket recvfrom function), and nil error will be returned. You can Use MTU to get
	// the maximum size of the packets.
	//
	// If the returned error is nil, it means that Read has completed successfully and that an IP packet has been copied
	// into p.
	Read(p []byte) (int, error)

	// Write writes an IP packet p to this device and returns the number of bytes written. Similar to Read, large IP
	// packets must be fragmented by the caller, and len(p) must not exceed the maximum buffer size returned by MTU.
	//
	// Write will return (0, ErrMsgSize) if len(p) > MTU(). This aligns with the socket sendto function.
	//
	// If the returned error is nil, it means that Write has completed successfully and that the entire packet has been
	// written to the destination.
	//
	// If only a portion of the packet has been written, Write will return a non-nil error as well as the number of bytes
	// written (< len(p)).
	Write(b []byte) (int, error)

	// MTU returns the size of the Maximum Transmission Unit for this device, which is the maximum size of a single IP
	// packet that can be received/sent.
	MTU() int
}
