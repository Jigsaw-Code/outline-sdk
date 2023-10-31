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

type splitReader struct {
	reader      io.Reader
	prefixBytes int
}

var _ io.Reader = (*splitReader)(nil)

func (r *splitReader) Read(data []byte) (int, error) {
	if r.prefixBytes > 0 && len(data) > int(r.prefixBytes) {
		data = data[:r.prefixBytes]
	}
	n, err := r.reader.Read(data)
	r.prefixBytes -= n
	return n, err
}
