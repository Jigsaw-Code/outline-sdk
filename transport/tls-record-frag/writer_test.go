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

package tlsrecordfrag

import (
	"bytes"
	"io"
	"testing"

	"encoding/binary"
	"github.com/stretchr/testify/require"
)

type collectWrites struct {
	writes [][]byte
}

var _ io.Writer = (*collectWrites)(nil)

var header = []byte{0x16, 0x03, 0x01}

func (w *collectWrites) Write(data []byte) (int, error) {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	w.writes = append(w.writes, dataCopy)
	return len(data), nil
}

func TestWrite(t *testing.T) {
	data := []byte{0x16, 0x03, 0x01, 0, 10, 0x01, 0, 0, 6, 0x03, 0x03, 1, 2, 3, 4}
	var innerWriter collectWrites
	trfWriter := NewWriter(&innerWriter, 1)
	n, err := trfWriter.Write(data)
	require.NoError(t, err)
	require.Equal(t, n, len(data)+5)
	require.Equal(t, [][]byte{[]byte{0x16, 0x03, 0x01, 0, 1, 0x1, 0x16, 0x03, 0x01, 0, 9, 0, 0, 6, 0x03, 0x03, 1, 2, 3, 4}}, innerWriter.writes)
}

func TestReadFrom(t *testing.T) {
	data := []byte{0x16, 0x03, 0x01, 0, 10, 0x01, 0, 0, 6, 0x03, 0x03, 1, 2, 3, 4, 0xff}
	var innerWriter collectWrites
	trfWriter := NewWriter(&innerWriter, 2)
	n, err := trfWriter.ReadFrom(bytes.NewReader(data))
	require.NoError(t, err)
	require.Equal(t, n, int64(len(data))+5)
	require.Equal(t, [][]byte{[]byte{0x16, 0x03, 0x01, 0, 2, 0x1, 0, 0x16, 0x03, 0x01, 0, 8, 0, 6, 0x03, 0x03, 1, 2, 3, 4}, []byte{0xff}}, innerWriter.writes)
}

func TestWrite_MultipleRecords(t *testing.T) {
	var innerWriter collectWrites
	trfWriter := NewWriter(&innerWriter, 3)
	data := make([]byte, 15)
	copy(data, header)
	binary.BigEndian.PutUint16(data[3:], 4)
	copy(data[5:], []byte{4, 3, 2, 1})
	copy(data[9:], header)
	binary.BigEndian.PutUint16(data[12:], 1)
	data[14] = 0x7f

	_, err := trfWriter.Write(data)
	require.NoError(t, err)

	result := make([]byte, 20)
	copy(result, header)
	binary.BigEndian.PutUint16(result[3:], 3)
	copy(result[5:], []byte{4, 3, 2})

	copy(result[8:], header)
	binary.BigEndian.PutUint16(result[11:], 1)
	result[13] = 1

	copy(result[14:], []byte{0x16, 0x03, 0x01, 0, 0x01, 0x7f})

	require.Equal(t, [][]byte{result}, innerWriter.writes)
}
