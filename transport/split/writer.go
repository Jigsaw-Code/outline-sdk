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

type repeatedSplit struct {
	count int
	bytes int64
}

type splitWriter struct {
	writer          io.Writer
	nextSplitBytes  int64
	remainingSplits []repeatedSplit
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
// It's possible to enable multiple splits with the [AddSplitSequence] option, which adds count splits every skipBytes bytes.
// Example:
// prefixBytes = 1, AddSplitSequence(count=2, bytes=6)
// Array of [0 1 3 2 4 5 6 7 8 9 10 11 12 13 14 15 16 ...] will become
// [0] [1 2 3 4 5 6] [7 8 9 10 11 12] [13 14 15 16 ...]
func NewWriter(writer io.Writer, prefixBytes int64, options ...Option) io.Writer {
	sw := &splitWriter{writer: writer, nextSplitBytes: prefixBytes, remainingSplits: []repeatedSplit{}}
	for _, option := range options {
		option(sw)
	}
	if len(sw.remainingSplits) == 0 {
		// TODO(fortuna): Support ReaderFrom for repeat split.
		if rf, ok := writer.(io.ReaderFrom); ok {
			return &splitWriterReaderFrom{sw, rf}
		}
	}
	return sw
}

type Option func(w *splitWriter)

// AddSplitSequence will add count splits, each of skipBytes length.
func AddSplitSequence(count int, skipBytes int64) Option {
	return func(w *splitWriter) {
		if count > 0 {
			w.remainingSplits = append(w.remainingSplits, repeatedSplit{count: count, bytes: skipBytes})
		}
	}
}

func (w *splitWriterReaderFrom) ReadFrom(source io.Reader) (int64, error) {
	reader := io.MultiReader(io.LimitReader(source, w.nextSplitBytes), source)
	written, err := w.rf.ReadFrom(reader)
	w.nextSplitBytes -= written
	return written, err
}

func (w *splitWriter) Write(data []byte) (written int, err error) {
	for 0 < w.nextSplitBytes && w.nextSplitBytes < int64(len(data)) {
		dataToSend := data[:w.nextSplitBytes]
		n, err := w.writer.Write(dataToSend)
		written += n
		w.nextSplitBytes -= int64(n)
		if err != nil {
			return written, err
		}
		data = data[n:]

		// Split done. Update nextSplitBytes.
		if len(w.remainingSplits) > 0 {
			w.nextSplitBytes = w.remainingSplits[0].bytes
			w.remainingSplits[0].count -= 1
			if w.remainingSplits[0].count == 0 {
				w.remainingSplits = w.remainingSplits[1:]
			}
		}
	}
	n, err := w.writer.Write(data)
	written += n
	w.nextSplitBytes -= int64(n)
	return written, err
}
