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

package ip

import (
	"context"
)

// IPDevice is a generic network device that reads and writes IP packets.
type IPDevice interface {
	// Close closes this device. Any future ReadPacket or WritePacket operations
	// will return errors.
	Close() error

	// ReadPacket reads an IP packet from this device using the provided context
	// `ctx`. The implementation decides where to get the IP packet from, such as
	// a virtual network adapter or a local proxy.
	//
	// The provided `ctx` must be non-nil. If the `ctx` expires before the
	// operation is complete, an error is returned.
	//
	// If the returned error is nil, it means that ReadPacket has completed
	// successfully and that an entire IP packet has been read and returned. It
	// won't return if only a portion of the packet is read.
	ReadPacket(ctx context.Context) ([]byte, error)

	// WritePacket writes an IP packet `b` to this device using the provided
	// context `ctx`. The implementation decides what to do with the packet, such
	// as sending it to the Internet or interpreting it as TCP/UDP packets.
	//
	// The provided `ctx` must be non-nil. If the `ctx` expires before the
	// operation is complete, an error is returned.
	//
	// If the returned error is nil, it means that WritePacket has completed
	// successfully and that the entire packet has been written to the
	// destination. It won't return if only a portion of the packet has been
	// processed.
	WritePacket(ctx context.Context, b []byte) error
}
