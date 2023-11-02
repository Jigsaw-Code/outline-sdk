// Copyright 2019 Jigsaw Operations LLC
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

package transport

import (
	"bytes"
	"io"
	"sync"
)

type writerAdaptor struct {
	// The underlying io.ReaderFrom.
	rf io.ReaderFrom
	// Coverts []byte to io.Reader.
	r bytes.Reader
	// Sequences the writes.
	mu sync.Mutex
}

func (w *writerAdaptor) Write(buf []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.r.Reset(buf)
	n, err := w.rf.ReadFrom(&w.r)
	return int(n), err
}

func AsWriter(rf io.ReaderFrom) io.Writer {
	return &writerAdaptor{rf: rf}
}
