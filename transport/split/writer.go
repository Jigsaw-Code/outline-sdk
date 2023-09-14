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
	"errors"
	"io"
)

type SplitWriter struct {
	writer      io.Writer
	prefixBytes int64
}

var _ io.Writer = (*SplitWriter)(nil)
var _ io.ReaderFrom = (*SplitWriter)(nil)

// NewWriter creates a [io.Writer] that ensures the byte sequence is split at prefixBytes,
// meaning a write will end right after byte index prefixBytes - 1, before a write starting at byte index prefixBytes.
// For example, if you have a write of [0123456789] and prefixBytes = 3, you will get writes [012] and [3456789].
func NewWriter(writer io.Writer, prefixBytes int64) *SplitWriter {
	return &SplitWriter{writer, prefixBytes}
}

func (w *SplitWriter) ReadFrom(source io.Reader) (written int64, err error) {
	if w.prefixBytes > 0 {
		written, err = io.CopyN(w.writer, source, w.prefixBytes)
		w.prefixBytes -= written
		if errors.Is(err, io.EOF) {
			return written, nil
		}
		if err != nil {
			return written, err
		}
	}
	n, err := io.Copy(w.writer, source)
	written += n
	return written, err
}

func (w *SplitWriter) Write(data []byte) (written int, err error) {
	if 0 < w.prefixBytes && w.prefixBytes < int64(len(data)) {
		written, err = w.writer.Write(data[:w.prefixBytes])
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
