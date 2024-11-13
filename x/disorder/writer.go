// Copyright 2024 The Outline Authors
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

package disorder

import (
	"fmt"
	"io"

	"github.com/Jigsaw-Code/outline-sdk/x/sockopt"
)

type disorderWriter struct {
	conn             io.Writer
	tcpOptions       sockopt.TCPOptions
	writesToDisorder int
}

var _ io.Writer = (*disorderWriter)(nil)

func NewWriter(conn io.Writer, tcpOptions sockopt.TCPOptions, runAtPacketN int) io.Writer {
	// TODO: Support ReadFrom.
	return &disorderWriter{
		conn:             conn,
		tcpOptions:       tcpOptions,
		writesToDisorder: runAtPacketN,
	}
}

func (w *disorderWriter) Write(data []byte) (written int, err error) {
	if w.writesToDisorder == 0 {
		defaultHopLimit, err := w.tcpOptions.HopLimit()
		if err != nil {
			return 0, fmt.Errorf("failed to get the hop limit: %w", err)
		}

		// Setting number of hops to 1 will lead to data to get lost on host.
		err = w.tcpOptions.SetHopLimit(1)
		if err != nil {
			return 0, fmt.Errorf("failed to set the hop limit to 1: %w", err)
		}

		defer func() {
			// The packet with low hop limit was sent.
			// Make next calls send data normally.
			//
			// The packet with the low hop limit will get resent by the kernel later.
			// The network filters will receive data out of order.
			err = w.tcpOptions.SetHopLimit(defaultHopLimit)
			if err != nil {
				err = fmt.Errorf("failed to set the hop limit %d: %w", defaultHopLimit, err)
			}
		}()
	}

	// The packet will get lost at the first send, since the hop limit is too low.
	n, err := w.conn.Write(data)

	// TODO: Wait for queued data to be sent by the kernel to the socket.

	if w.writesToDisorder > -1 {
		w.writesToDisorder -= 1
	}
	return n, err
}
