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
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// collectWrites is a [io.Writer] that appends each write to the writes slice.
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

func TestWrite_Split(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, 3)
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Req"), []byte("uest")}, innerWriter.writes)
}

func TestWrite_ShortWrite(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, 10)
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerWriter.writes)
}

func TestWrite_Zero(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, 0)
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerWriter.writes)
}

func TestWrite_NeedsTwoWrites(t *testing.T) {
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

func TestWrite_Compound(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(NewWriter(&innerWriter, 4), 1)
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("R"), []byte("equ"), []byte("est")}, innerWriter.writes)
}

// collectReader is a [io.Reader] that appends each Read from the Reader to the reads slice.
type collectReader struct {
	io.Reader
	reads [][]byte
}

func (r *collectReader) Read(buf []byte) (int, error) {
	n, err := r.Reader.Read(buf)
	if n > 0 {
		read := make([]byte, n)
		copy(read, buf[:n])
		r.reads = append(r.reads, read)
	}
	return n, err
}

func TestReadFrom(t *testing.T) {
	splitWriter := NewWriter(&bytes.Buffer{}, 3)
	rf, ok := splitWriter.(io.ReaderFrom)
	require.True(t, ok)

	cr := &collectReader{Reader: bytes.NewReader([]byte("Request1"))}
	n, err := rf.ReadFrom(cr)
	require.NoError(t, err)
	require.Equal(t, int64(8), n)
	require.Equal(t, [][]byte{[]byte("Req"), []byte("uest1")}, cr.reads)

	cr = &collectReader{Reader: bytes.NewReader([]byte("Request2"))}
	n, err = rf.ReadFrom(cr)
	require.NoError(t, err)
	require.Equal(t, int64(8), n)
	require.Equal(t, [][]byte{[]byte("Request2")}, cr.reads)
}

func TestReadFrom_ShortRead(t *testing.T) {
	splitWriter := NewWriter(&bytes.Buffer{}, 10)
	rf, ok := splitWriter.(io.ReaderFrom)
	require.True(t, ok)
	cr := &collectReader{Reader: bytes.NewReader([]byte("Request1"))}
	n, err := rf.ReadFrom(cr)
	require.NoError(t, err)
	require.Equal(t, int64(8), n)
	require.Equal(t, [][]byte{[]byte("Request1")}, cr.reads)

	cr = &collectReader{Reader: bytes.NewReader([]byte("Request2"))}
	n, err = rf.ReadFrom(cr)
	require.NoError(t, err)
	require.Equal(t, int64(8), n)
	require.Equal(t, [][]byte{[]byte("Re"), []byte("quest2")}, cr.reads)
}

func BenchmarkReadFrom(b *testing.B) {
	for n := 0; n < b.N; n++ {
		reader := bytes.NewReader(make([]byte, n))
		writer := NewWriter(io.Discard, 10)
		rf, ok := writer.(io.ReaderFrom)
		require.True(b, ok)
		_, err := rf.ReadFrom(reader)
		require.NoError(b, err)
	}
}
