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
	splitWriter := NewWriter(&innerWriter, NewFixedSplitIterator(3))
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Req"), []byte("uest")}, innerWriter.writes)
}

func TestWrite_SplitZero(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewRepeatedSplitIterator(RepeatedSplit{1, 0}, RepeatedSplit{0, 1}, RepeatedSplit{10, 0}, RepeatedSplit{0, 2}))
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerWriter.writes)
}

func TestWrite_SplitZeroLong(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewRepeatedSplitIterator(RepeatedSplit{1, 0}, RepeatedSplit{1_000_000_000_000_000_000, 0}))
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerWriter.writes)
}

func TestWrite_SplitZeroPrefix(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewRepeatedSplitIterator(RepeatedSplit{1, 0}, RepeatedSplit{3, 2}))
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Re"), []byte("qu"), []byte("es"), []byte("t")}, innerWriter.writes)
}

func TestWrite_SplitMulti(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewRepeatedSplitIterator(RepeatedSplit{1, 1}, RepeatedSplit{3, 2}, RepeatedSplit{2, 3}))
	n, err := splitWriter.Write([]byte("RequestRequestRequest"))
	require.NoError(t, err)
	require.Equal(t, 21, n)
	require.Equal(t, [][]byte{[]byte("R"), []byte("eq"), []byte("ue"), []byte("st"), []byte("Req"), []byte("ues"), []byte("tRequest")}, innerWriter.writes)
}

func TestWrite_ShortWrite(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewFixedSplitIterator(10))
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerWriter.writes)
}

func TestWrite_Zero(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewFixedSplitIterator(0))
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerWriter.writes)
}

func TestWrite_NeedsTwoWrites(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewFixedSplitIterator(5))
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
	splitWriter := NewWriter(NewWriter(&innerWriter, NewFixedSplitIterator(4)), NewFixedSplitIterator(1))
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("R"), []byte("equ"), []byte("est")}, innerWriter.writes)
}

func TestWrite_RepeatNumber3_SkipBytes5(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewRepeatedSplitIterator(RepeatedSplit{1, 1}, RepeatedSplit{3, 5}))
	n, err := splitWriter.Write([]byte("RequestRequestRequest."))
	require.NoError(t, err)
	require.Equal(t, 7*3+1, n)
	require.Equal(t, [][]byte{
		[]byte("R"),      // prefix
		[]byte("eques"),  // split 1
		[]byte("tRequ"),  // split 2
		[]byte("estRe"),  // split 3
		[]byte("quest."), // tail
	}, innerWriter.writes)
}

func TestWrite_RepeatNumber3_SkipBytes0(t *testing.T) {
	var innerWriter collectWrites
	splitWriter := NewWriter(&innerWriter, NewRepeatedSplitIterator(RepeatedSplit{1, 1}, RepeatedSplit{0, 3}))
	n, err := splitWriter.Write([]byte("Request"))
	require.NoError(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, [][]byte{[]byte("R"), []byte("equest")}, innerWriter.writes)
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
	splitWriter := NewWriter(&bytes.Buffer{}, NewFixedSplitIterator(3))
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

func TestReadFrom_Multi(t *testing.T) {
	splitWriter := NewWriter(&bytes.Buffer{}, NewRepeatedSplitIterator(RepeatedSplit{1, 1}, RepeatedSplit{3, 2}, RepeatedSplit{2, 3}))
	rf, ok := splitWriter.(io.ReaderFrom)
	require.True(t, ok)

	cr := &collectReader{Reader: bytes.NewReader([]byte("RequestRequestRequest"))}
	n, err := rf.ReadFrom(cr)
	require.NoError(t, err)
	require.Equal(t, int64(21), n)
	require.Equal(t, [][]byte{[]byte("R"), []byte("eq"), []byte("ue"), []byte("st"), []byte("Req"), []byte("ues"), []byte("tRequest")}, cr.reads)
}

func TestReadFrom_ShortRead(t *testing.T) {
	splitWriter := NewWriter(&bytes.Buffer{}, NewFixedSplitIterator(10))
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
		writer := NewWriter(io.Discard, NewFixedSplitIterator(10))
		rf, ok := writer.(io.ReaderFrom)
		require.True(b, ok)
		_, err := rf.ReadFrom(reader)
		require.NoError(b, err)
	}
}
