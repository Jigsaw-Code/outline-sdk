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

type collectReads struct {
	tb    testing.TB
	reads [][]byte
}

var _ io.ReaderFrom = (*collectReads)(nil)

func (c *collectReads) ReadFrom(reader io.Reader) (int64, error) {
	var b bytes.Buffer
	n, err := b.ReadFrom(reader)
	c.tb.Logf("ReadFrom: n=%v, err=%v", n, err)
	if err != nil {
		return n, err
	}
	dataCopy := make([]byte, n)
	copy(dataCopy, b.Bytes())
	c.reads = append(c.reads, dataCopy)
	return n, nil
}

func TestWrite_Split(t *testing.T) {
	innerReaderFrom := collectReads{tb: t}
	splitReaderFrom := NewReaderFrom(&innerReaderFrom, 3)
	n, err := splitReaderFrom.ReadFrom(bytes.NewReader([]byte("Request")))
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	require.Equal(t, [][]byte{[]byte("Req"), []byte("uest")}, innerReaderFrom.reads)
}

func TestWrite_OnEOF(t *testing.T) {
	innerReaderFrom := collectReads{tb: t}
	splitReaderFrom := NewReaderFrom(&innerReaderFrom, 7)
	n, err := splitReaderFrom.ReadFrom(bytes.NewReader([]byte("Request")))
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerReaderFrom.reads)
}

func TestWrite_ShortWrite(t *testing.T) {
	var innerReaderFrom collectReads
	splitReaderFrom := NewReaderFrom(&innerReaderFrom, 10)
	n, err := splitReaderFrom.ReadFrom(bytes.NewReader([]byte("Request")))
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerReaderFrom.reads)
}

func TestWrite_Zero(t *testing.T) {
	var innerReaderFrom collectReads
	splitReaderFrom := NewReaderFrom(&innerReaderFrom, 0)
	n, err := splitReaderFrom.ReadFrom(bytes.NewReader([]byte("Request")))
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerReaderFrom.reads)
}

func TestWrite_NeedsTwoWrites(t *testing.T) {
	var innerReaderFrom collectReads
	splitReaderFrom := NewReaderFrom(&innerReaderFrom, 5)
	n, err := splitReaderFrom.ReadFrom(bytes.NewReader([]byte("Re")))
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	n, err = splitReaderFrom.ReadFrom(bytes.NewReader([]byte("quest")))
	require.NoError(t, err)
	require.Equal(t, int64(5), n)
	require.Equal(t, [][]byte{[]byte("Re"), []byte("que"), []byte("st")}, innerReaderFrom.reads)
}

func TestWrite_Compound(t *testing.T) {
	var innerReaderFrom collectReads
	splitReaderFrom := NewReaderFrom(NewReaderFrom(&innerReaderFrom, 4), 1)
	n, err := splitReaderFrom.ReadFrom(bytes.NewReader([]byte("Request")))
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	require.Equal(t, [][]byte{[]byte("R"), []byte("equ"), []byte("est")}, innerReaderFrom.reads)
}

func TestReadFrom(t *testing.T) {
	var innerReaderFrom collectReads
	splitReaderFrom := NewReaderFrom(&innerReaderFrom, 3)
	n, err := splitReaderFrom.ReadFrom(bytes.NewReader([]byte("Request")))
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	require.Equal(t, [][]byte{[]byte("Req"), []byte("uest")}, innerReaderFrom.reads)
}

func TestReadFrom_ShortRead(t *testing.T) {
	var innerReaderFrom collectReads
	splitReaderFrom := NewReaderFrom(&innerReaderFrom, 10)
	n, err := splitReaderFrom.ReadFrom(bytes.NewReader([]byte("Request")))
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	require.Equal(t, [][]byte{[]byte("Request")}, innerReaderFrom.reads)
}
