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

type SplitReaderFrom struct {
	rf          io.ReaderFrom
	prefixBytes int64
}

var _ io.ReaderFrom = (*SplitReaderFrom)(nil)

// NewReaderFrom creates a [io.Writer] that ensures the byte sequence is split at prefixBytes.
// A write will end right after byte index prefixBytes - 1, before a write starting at byte index prefixBytes.
// For example, if you have a write of [0123456789] and prefixBytes = 3, you will get writes [012] and [3456789].
func NewReaderFrom(rf io.ReaderFrom, prefixBytes int64) *SplitReaderFrom {
	return &SplitReaderFrom{rf, prefixBytes}
}

func (w *SplitReaderFrom) ReadFrom(source io.Reader) (int64, error) {
	var n int64
	var err error
	if w.prefixBytes > 0 {
		n, err = w.rf.ReadFrom(io.LimitReader(source, w.prefixBytes))
		w.prefixBytes -= n
		if err != nil || w.prefixBytes > 0 {
			return n, err
		}
		// At this point, we read a full prefix and prefixBytes must be zero.
		// We only want to trigger a new ReadFrom if there's more data to read
		// to a avoid an empty ReadFrom.
		if _, err := source.Read([]byte{}); errors.Is(err, io.EOF) {
			return n, nil
		}
	}
	n0, err := w.rf.ReadFrom(source)
	n += n0
	return n, err
}
