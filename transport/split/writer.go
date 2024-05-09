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

package split

import (
	"fmt"
	"io"
	"net"
	"net/netip"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type splitWriter struct {
	writer      io.Writer
	prefixBytes int64
}

var _ io.Writer = (*splitWriter)(nil)

type splitWriterReaderFrom struct {
	*splitWriter
	rf io.ReaderFrom
}

// var _ io.ReaderFrom = (*splitWriterReaderFrom)(nil)

// NewWriter creates a [io.Writer] that ensures the byte sequence is split at prefixBytes.
// A write will end right after byte index prefixBytes - 1, before a write starting at byte index prefixBytes.
// For example, if you have a write of [0123456789] and prefixBytes = 3, you will get writes [012] and [3456789].
// If the input writer is a [io.ReaderFrom], the output writer will be too.
func NewWriter(writer io.Writer, prefixBytes int64) io.Writer {
	sw := &splitWriter{writer, prefixBytes}
	// if rf, ok := writer.(io.ReaderFrom); ok {
	// 	return &splitWriterReaderFrom{sw, rf}
	// }
	return sw
}

// func (w *splitWriterReaderFrom) ReadFrom(source io.Reader) (int64, error) {
// 	reader := io.MultiReader(io.LimitReader(source, w.prefixBytes), source)
// 	written, err := w.rf.ReadFrom(reader)
// 	w.prefixBytes -= written
// 	return written, err
// }

func (w *splitWriter) Write(data []byte) (written int, err error) {
	if 0 < w.prefixBytes && w.prefixBytes < int64(len(data)) {
		fmt.Println("Split!")
		var ttl int
		tc, ok := w.writer.(*net.TCPConn)
		if !ok {
			return 0, fmt.Errorf("not tcp")
		}
		ap, err := netip.ParseAddrPort(tc.RemoteAddr().String())
		if err != nil {
			return 0, fmt.Errorf("invalid address: %w", err)
		}

		if ap.Addr().Is4() {
			conn4 := ipv4.NewConn(tc)
			ttl, _ = conn4.TTL()
			conn4.SetTTL(1)
		} else if ap.Addr().Is6() {
			conn6 := ipv6.NewConn(tc)
			ttl, _ = conn6.HopLimit()
			conn6.SetHopLimit(1)
		} else {
			return 0, fmt.Errorf("invalid IP: %w", err)
		}
		written, err = tc.Write(data[:w.prefixBytes])
		if ap.Addr().Is4() {
			conn4 := ipv4.NewConn(tc)
			conn4.SetTTL(ttl)
		} else if ap.Addr().Is6() {
			conn6 := ipv6.NewConn(tc)
			conn6.SetHopLimit(ttl)
		}
		w.prefixBytes -= int64(written)
		if err != nil {
			return written, err
		}
		data = data[written:]
	}
	n, err := w.writer.Write(data)
	written += n
	w.prefixBytes -= int64(n)
	return written, err
}
