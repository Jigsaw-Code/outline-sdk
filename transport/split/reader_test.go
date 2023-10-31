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
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitReader(t *testing.T) {
	splitReader := &splitReader{bytes.NewReader([]byte("Request")), 3}
	buf := make([]byte, 10)
	n, err := splitReader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, int(3), n)
	require.Equal(t, []byte("Req"), buf[:n])

	n, err = splitReader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, int(4), n)
	require.Equal(t, []byte("uest"), buf[:n])
}
