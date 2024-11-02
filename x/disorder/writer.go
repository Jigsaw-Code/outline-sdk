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
	"net"
	"sync"
)

type disorderWriter struct {
	conn        net.Conn
	resetTTL    sync.Once
	prefixBytes int64
	oldTTL      int
}

var _ io.Writer = (*disorderWriter)(nil)

// TODO
// var _ io.ReaderFrom = (*splitWriterReaderFrom)(nil)

// TODO
func NewWriter(conn net.Conn, prefixBytes int64, oldTTL int) io.Writer {
	// TODO support ReaderFrom
	return &disorderWriter{
		conn:        conn,
		prefixBytes: prefixBytes,
		oldTTL:      oldTTL,
	}
}

func (w *disorderWriter) Write(data []byte) (written int, err error) {
	if 0 < w.prefixBytes && w.prefixBytes < int64(len(data)) {
		written, err = w.conn.Write(data[:w.prefixBytes])
		w.prefixBytes -= int64(written)
		if err != nil {
			return written, err
		}
		data = data[written:]
	}
	w.resetTTL.Do(func() {
		_, err = setTtl(w.conn, w.oldTTL)
	})
	if err != nil {
		return written, fmt.Errorf("setsockopt IPPROTO_IP/IP_TTL error: %w", err)
	}

	n, err := w.conn.Write(data)
	written += n
	w.prefixBytes -= int64(n)
	return written, err
}
