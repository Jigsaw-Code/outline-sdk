// Copyright 2024 Jigsaw Operations LLC
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

package tlsfrag

import (
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// Make sure NewFixedLenWriter checks parameters.
func TestNewFixedLenWriterCheckParameters(t *testing.T) {
	w, err := NewFixedLenWriter(nil, func(_ int) int { return 1 })
	require.Error(t, err)
	require.Nil(t, w)

	w, err = NewFixedLenWriter(io.Discard, nil)
	require.Error(t, err)
	require.Nil(t, w)

	w, err = NewFixedLenWriter(io.Discard, func(_ int) int { return 1 })
	require.NoError(t, err)
	require.NotNil(t, w)
}

// Make sure FixedLenWriter's Write method can split TLS hello.
func TestFixedLenWriterWrite(t *testing.T) {
	cases := []struct {
		name     string
		in       net.Buffers
		frag     FixedLenFragFunc
		expected [][]byte
	}{
		{
			name: "SplitFullClientHello",
			in: [][]byte{
				{0x16, 0x03, 0x01, 0x00, 0x10,
					0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
			},
			frag: func(_ int) int { return 7 },
			expected: [][]byte{
				// frag1.header + frag1.payload + frag2.header + frag2.payload
				{0x16, 0x03, 0x01, 0x00, 0x07},
				{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99},
				{0x16, 0x03, 0x01, 0x00, 0x09},
				{0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
			},
		},
		{
			name: "SplitClientHelloSingleByteSeq",
			in: [][]byte{
				{0x16}, {0x03}, {0x01}, {0x00}, {0x10},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99}, {0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
			},
			frag: func(_ int) int { return 7 },
			expected: [][]byte{
				// the header will be combined
				{0x16, 0x03, 0x01, 0x00, 0x07},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
				{0x16, 0x03, 0x01, 0x00, 0x09},
				{0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
			},
		},
		{
			name: "SplitClientHelloWithEmptyWrites",
			in: [][]byte{
				{}, {0x16}, {0x03}, {}, {0x01}, {0x00}, {}, {0x10},
				{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99}, {0x88},
				{}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
			},
			frag: func(_ int) int { return 7 },
			expected: [][]byte{
				// the header will be combined, empty writes within header will be ignored
				{0x16, 0x03, 0x01, 0x00, 0x07},
				{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
				{0x16, 0x03, 0x01, 0x00, 0x09},
				{0x88}, {}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
			},
		},
		{
			name: "SplitClientHelloJustOneByte",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 1 },
			expected: [][]byte{
				{0x16, 0x03, 0x01, 0x00, 0x01},
				{0xff},
				{0x16, 0x03, 0x01, 0x00, 0x05},
				{0xee, 0xdd, 0xcc, 0xbb, 0xaa},
			},
		},
		{
			name: "SplitClientHelloJustOneByteLast",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 5 },
			expected: [][]byte{
				{0x16, 0x03, 0x01, 0x00, 0x05},
				{0xff, 0xee, 0xdd, 0xcc, 0xbb},
				{0x16, 0x03, 0x01, 0x00, 0x01},
				{0xaa},
			},
		},
		{
			name: "NotSplitForZeroByte",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 0 },
			expected: [][]byte{
				// the 5-byte header will always be separate packets due to the peek behavior
				{0x16, 0x03, 0x01, 0x00, 0x06},
				{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa},
			},
		},
		{
			name: "NoSplitForZeroByteLast",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 6 },
			expected: [][]byte{
				// the 5-byte header will always be separate packets due to the peek behavior
				{0x16, 0x03, 0x01, 0x00, 0x06},
				{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa},
			},
		},
		{
			name: "NoSplitForNonClientHello",
			// invalid record type
			in:   [][]byte{{0x15, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 1 },
			expected: [][]byte{
				// the 5-byte header will always be separate packets due to the peek behavior
				{0x15, 0x03, 0x01, 0x00, 0x06},
				{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa},
			},
		},
		{
			name: "NoSplitForNonClientHelloSingleByteSeq",
			// invalid record length
			in:   [][]byte{{0x16}, {0x03}, {0x01}, {0x00}, {0x00}, {0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}},
			frag: func(_ int) int { return 1 },
			expected: [][]byte{
				// the 5-byte header will always be separate packets due to the peek behavior
				{0x16, 0x03, 0x01, 0x00, 0x00},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa},
			},
		},
		{
			name:     "NoDataOutForIncompleteHeader",
			in:       [][]byte{{0x16}, {0x03}},
			frag:     func(_ int) int { return 5 },
			expected: nil, // no write should ever been issued
		},
		{
			name:     "SplitPartialContentForFirstIncompleteRecord",
			in:       [][]byte{{0x16, 0x03, 0x01, 0x00, 0x10, 0xff, 0xee, 0xdd}},
			frag:     func(_ int) int { return 5 },
			expected: [][]byte{{0x16, 0x03, 0x01, 0x00, 0x05}, {0xff, 0xee, 0xdd}},
		},
		{
			name: "SplitPartialContentForFirstCompleteRecord",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x10, 0xff, 0xee, 0xdd, 0xcc, 0xbb}},
			frag: func(_ int) int { return 5 },
			expected: [][]byte{
				// the 1st record is done, Writer will write the second header as well
				{0x16, 0x03, 0x01, 0x00, 0x05},
				{0xff, 0xee, 0xdd, 0xcc, 0xbb},
				{0x16, 0x03, 0x01, 0x00, 0x0b},
			},
		},
		{
			name: "SplitPartialContentForSecondIncompleteRecord",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x10, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99}},
			frag: func(_ int) int { return 5 },
			expected: [][]byte{
				// the 1st record is done, Writer will write the second header and payload
				{0x16, 0x03, 0x01, 0x00, 0x05},
				{0xff, 0xee, 0xdd, 0xcc, 0xbb},
				{0x16, 0x03, 0x01, 0x00, 0x0b},
				{0xaa, 0x99},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inner := &collectWriter{}
			w, err := NewFixedLenWriter(inner, tc.frag)
			require.NoError(t, err)
			require.NotNil(t, w)
			require.IsType(t, &fixedLenWriter{}, w)

			for _, p := range tc.in {
				n, err := w.Write(p)
				require.NoError(t, err)
				require.Equal(t, len(p), n)
			}

			// All buf should come from Write calls
			require.Equal(t, tc.expected, inner.buf)
			require.Equal(t, len(inner.buf), inner.wrOps)
		})
	}
}

// Make sure FixedLenWriter's ReadFrom method can split TLS hello.
func TestFixedLenWriterReadFrom(t *testing.T) {
	cases := []struct {
		name     string
		in       [][]byte
		frag     FixedLenFragFunc
		expected [][]byte
	}{
		{
			name: "SplitFullClientHelloRecord",
			in: [][]byte{
				{0x16, 0x03, 0x01, 0x00, 0x10,
					0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
			},
			frag: func(_ int) int { return 7 },
			expected: [][]byte{
				{0x16, 0x03, 0x01, 0x00, 0x07, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99},
				{0x16, 0x03, 0x01, 0x00, 0x09, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "SplitClientHelloSingleByteSeq",
			in: [][]byte{
				{0x16}, {0x03}, {0x01}, {0x00}, {0x10},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99}, {0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
			},
			frag: func(_ int) int { return 7 },
			expected: [][]byte{
				// the first payload will be combined with header
				{0x16, 0x03, 0x01, 0x00, 0x07, 0xff},
				{0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
				{0x16, 0x03, 0x01, 0x00, 0x09, 0x88},
				{0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "SplitClientHelloWithEmptyWrites",
			in: [][]byte{
				{}, {0x16}, {0x03}, {}, {0x01}, {0x00}, {}, {0x10},
				{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99}, {0x88},
				{}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
			},
			frag: func(_ int) int { return 7 },
			expected: [][]byte{
				// the first payload byte `{} & {0x88}` will be combined with header
				{0x16, 0x03, 0x01, 0x00, 0x07},
				{0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
				{0x16, 0x03, 0x01, 0x00, 0x09, 0x88},
				{}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "SplitClientHelloJustOneByte",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 1 },
			expected: [][]byte{
				// payload will be combined with header
				{0x16, 0x03, 0x01, 0x00, 0x01, 0xff},
				{0x16, 0x03, 0x01, 0x00, 0x05, 0xee, 0xdd, 0xcc, 0xbb, 0xaa},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "SplitClientHelloJustOneByteLast",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 5 },
			expected: [][]byte{
				// payload will be combined with header
				{0x16, 0x03, 0x01, 0x00, 0x05, 0xff, 0xee, 0xdd, 0xcc, 0xbb},
				{0x16, 0x03, 0x01, 0x00, 0x01, 0xaa},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "NotSplitForZeroByte",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 0 },
			expected: [][]byte{
				{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "NoSplitForZeroByteLast",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 6 },
			expected: [][]byte{
				{0x16, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "NoSplitForNonClientHello",
			// invalid record type
			in:   [][]byte{{0x17, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa}},
			frag: func(_ int) int { return 1 },
			expected: [][]byte{
				{0x17, 0x03, 0x01, 0x00, 0x06, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "NoSplitForNonClientHelloSingleByteSeq",
			// invalid version
			in:   [][]byte{{0x16}, {0x01}, {0x03}, {0x00}, {0x06}, {0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}},
			frag: func(_ int) int { return 1 },
			expected: [][]byte{
				// header and the first payload will always be combined
				{0x16, 0x01, 0x03, 0x00, 0x06, 0xff},
				{0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name:     "NoDataOutForIncompleteHeader",
			in:       [][]byte{{0x16}, {0x03}},
			frag:     func(_ int) int { return 5 },
			expected: [][]byte{{}}, // reads 0 byte and io.EOF
		},
		{
			name:     "SplitPartialContentForFirstIncompleteRecord",
			in:       [][]byte{{0x16, 0x03, 0x01, 0x00, 0x10, 0xff, 0xee, 0xdd}},
			frag:     func(_ int) int { return 5 },
			expected: [][]byte{{0x16, 0x03, 0x01, 0x00, 0x05, 0xff, 0xee, 0xdd}, {}},
		},
		{
			name: "SplitPartialContentForFirstCompleteRecord",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x10, 0xff, 0xee, 0xdd, 0xcc, 0xbb}},
			frag: func(_ int) int { return 5 },
			expected: [][]byte{
				// the 1st record is done, ReadFrom will flush the second header as well
				{0x16, 0x03, 0x01, 0x00, 0x05, 0xff, 0xee, 0xdd, 0xcc, 0xbb},
				{0x16, 0x03, 0x01, 0x00, 0x0b},
				// No `{}` at the end because ReadFrom returns EOF together with the second header
			},
		},
		{
			name: "SplitPartialContentForSecondIncompleteRecord",
			in:   [][]byte{{0x16, 0x03, 0x01, 0x00, 0x10, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99}},
			frag: func(_ int) int { return 5 },
			expected: [][]byte{
				// the 1st record is done, ReadFrom will flush the second header and payload
				{0x16, 0x03, 0x01, 0x00, 0x05, 0xff, 0xee, 0xdd, 0xcc, 0xbb},
				{0x16, 0x03, 0x01, 0x00, 0x0b, 0xaa, 0x99},
				{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inner := &collectReaderFrom{}
			w, err := NewFixedLenWriter(inner, tc.frag)
			require.NoError(t, err)
			require.NotNil(t, w)
			require.IsType(t, &fixedLenReaderFrom{}, w)
			rf := w.(io.ReaderFrom)

			expectN := int64(0)
			for _, p := range tc.in {
				expectN += int64(len(p))
			}
			n, err := rf.ReadFrom(&slicesReader{tc.in}) // tc.in will be empty after this call
			require.NoError(t, err)
			require.Equal(t, expectN, n)

			// All buf should come from ReadFrom calls
			require.Equal(t, tc.expected, inner.buf)
			require.Zero(t, inner.wrOps)
		})
	}
}

// Private test helpers

type collectWriter struct {
	buf   [][]byte
	wrOps int
}

func (w *collectWriter) append(p []byte) int {
	q := make([]byte, len(p))
	w.buf = append(w.buf, q)
	return copy(q, p)
}

func (w *collectWriter) Write(p []byte) (n int, err error) {
	w.wrOps++
	return w.append(p), nil
}

type collectReaderFrom struct {
	collectWriter
}

func (w *collectReaderFrom) ReadFrom(r io.Reader) (n int64, err error) {
	tmp := make([]byte, 512)
	for {
		m, e := r.Read(tmp)
		n += int64(w.append(tmp[:m]))
		if err = e; err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
	}
}

// slicesReader is different from net.Buffers, it won't combine the data in buf.
type slicesReader struct {
	buf [][]byte
}

func (r *slicesReader) Read(p []byte) (n int, err error) {
	if len(r.buf) == 0 {
		return 0, io.EOF
	}
	n = copy(p, r.buf[0])
	if r.buf[0] = r.buf[0][n:]; len(r.buf[0]) == 0 {
		r.buf = r.buf[1:]
	}
	return
}
