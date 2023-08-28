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
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type collectWrites struct {
	writes [][]byte
}

var _ io.Writer = (*collectWrites)(nil)

func (w *collectWrites) Write(data []byte) (int, error) {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	w.writes = append(w.writes, dataCopy)
	return len(data), nil
}

func TestWriter_Split(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, 3)
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Req"), []byte("uest")}, innerWriter.writes)
}

func TestWriter_ShortWrite(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, 10)
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerWriter.writes)
}

func TestWriter_Zero(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, 0)
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerWriter.writes)
}

func TestWriter_NeedsTwoWrites(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, 5)
	n, err := splitWriter.Write([]byte("Re"))
	require.NoError(t, err)
	require.Equal(t, 2, n)
	n, err = splitWriter.Write([]byte("quest"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, [][]byte{[]byte("Re"), []byte("que"), []byte("st")}, innerWriter.writes)
}

func TestWriter_Compound(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(NewWriter(&innerWriter, 4), 1)
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("R"), []byte("equ"), []byte("est")}, innerWriter.writes)
}
