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

package wait_stream

import (
	"errors"
	"fmt"
	"io"

	"github.com/Jigsaw-Code/outline-sdk/x/sockopt"
)

type waitStreamWriter struct {
	conn       io.Writer
	tcpOptions sockopt.TCPOptions
}

var _ io.Writer = (*waitStreamWriter)(nil)

func NewWriter(conn io.Writer, tcpOptions sockopt.TCPOptions) io.Writer {
	return &waitStreamWriter{
		conn:       conn,
		tcpOptions: tcpOptions,
	}
}

func (w *waitStreamWriter) Write(data []byte) (written int, err error) {
	written, err = w.conn.Write(data)

	// This may not be implemented, so it's best effort really.
	waitUntilBytesAreSentErr := w.tcpOptions.WaitUntilBytesAreSent()
	if waitUntilBytesAreSentErr != nil && !errors.Is(waitUntilBytesAreSentErr, errors.ErrUnsupported) {
		return written, fmt.Errorf("error when waiting for stream to send all bytes: %w", waitUntilBytesAreSentErr)
	}

	return
}
