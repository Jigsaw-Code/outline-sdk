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

import "io"

// splitReader is a [io.Reader] that splits a read that spans the byte at prefixBytes position.
// For example, if you have a read of [0123456789] and prefixBytes = 3, you will get reads [012] and [3456789].
type splitReader struct {
	reader      io.Reader
	prefixBytes int64
}

var _ io.Reader = (*splitReader)(nil)

func (r *splitReader) Read(data []byte) (int, error) {
	if 0 < r.prefixBytes && r.prefixBytes < int64(len(data)) {
		data = data[:r.prefixBytes]
	}
	n, err := r.reader.Read(data)
	r.prefixBytes -= int64(n)
	return n, err
}
