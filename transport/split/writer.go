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

package split

import (
	"io"
)

type splitWriter struct {
	writer            io.Writer
	prefixBytes       int64
	repeatsNumberLeft int64 // How many times left to split
	skipBytes         int64 // When splitting multiple times, how many bytes to skip in between different splittings
}

var _ io.Writer = (*splitWriter)(nil)

type splitWriterReaderFrom struct {
	*splitWriter
	rf io.ReaderFrom
}

var _ io.ReaderFrom = (*splitWriterReaderFrom)(nil)

// NewWriter creates a [io.Writer] that ensures the byte sequence is split at prefixBytes.
// A write will end right after byte index prefixBytes - 1, before a write starting at byte index prefixBytes.
// For example, if you have a write of [0123456789] and prefixBytes = 3, you will get writes [012] and [3456789].
// If the input writer is a [io.ReaderFrom], the output writer will be too.
// If repeatsNumber > 1, then packets will be split multiple times, skipping skipBytes in between splits.
// Example:
// prefixBytes = 1
// repeatsNumber = 3
// skipBytes = 6
// Array of [0132456789 10 11 12 13 14 15 16 ...] will become
// [0] [123456] [789 10 11 12] [13 14 15 16 ...]
func NewWriter(writer io.Writer, prefixBytes int64, repeatsNumber int64, skipBytes int64) io.Writer {
	sw := &splitWriter{writer, prefixBytes, repeatsNumber, skipBytes}
	if repeatsNumber == 0 && skipBytes == 0 {
		if rf, ok := writer.(io.ReaderFrom); ok {
			return &splitWriterReaderFrom{sw, rf}
		}
	}
	return sw
}

func (w *splitWriterReaderFrom) ReadFrom(source io.Reader) (int64, error) {
	reader := io.MultiReader(io.LimitReader(source, w.prefixBytes), source)
	written, err := w.rf.ReadFrom(reader)
	w.prefixBytes -= written
	return written, err
}

func (w *splitWriter) Write(data []byte) (written int, err error) {
	for 0 < w.prefixBytes && w.prefixBytes < int64(len(data)) {
		dataToSend := data[:w.prefixBytes]
		n, err := w.writer.Write(dataToSend)
		written += n
		w.prefixBytes -= int64(n)
		if err != nil {
			return written, err
		}
		data = data[n:]

		w.repeatsNumberLeft -= 1
		if w.repeatsNumberLeft > 0 && w.prefixBytes == 0 {
			w.prefixBytes = w.skipBytes
		}
	}
	n, err := w.writer.Write(data)
	written += n
	w.prefixBytes -= int64(n)
	return written, err
}
