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
	"bytes"
	"fmt"
	"io"
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
		in       [][]byte
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
		bufSize  int
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
		{
			name: "SplitFullClientHelloWithTinyConsumer",
			in: [][]byte{
				{0x16, 0x03, 0x01, 0x00, 0x10,
					0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
			},
			frag:    func(_ int) int { return 7 },
			bufSize: 1,
			expected: [][]byte{
				{0x16}, {0x03}, {0x01}, {0x00}, {0x07},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
				{0x16}, {0x03}, {0x01}, {0x00}, {0x09},
				{0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "SplitClientHelloSingleByteSeqWithTinyConsumer",
			in: [][]byte{
				{0x16}, {0x03}, {0x01}, {0x00}, {0x10},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99}, {0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
			},
			frag:    func(_ int) int { return 7 },
			bufSize: 1,
			expected: [][]byte{
				// the first payload will be combined with header
				{0x16}, {0x03}, {0x01}, {0x00}, {0x07},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
				{0x16}, {0x03}, {0x01}, {0x00}, {0x09},
				{0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
				{}, // empty Read due to MultiReader implementation
			},
		},
		{
			name: "NoSplitForNonClientHelloWithTinyConsumer",
			// invalid buffer size
			in:      [][]byte{{0x16, 0x03}, {0x01, 0x40, 0x01, 0xff, 0xee}, {0xdd, 0xcc, 0xbb, 0xaa}},
			frag:    func(_ int) int { return 1 },
			bufSize: 1,
			expected: [][]byte{
				{0x16}, {0x03}, {0x01}, {0x40}, {0x01},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa},
				{}, // empty Read due to MultiReader implementation
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inner := &collectReaderFrom{bufSize: tc.bufSize}
			w, err := NewFixedLenWriter(inner, tc.frag)
			require.NoError(t, err)
			require.NotNil(t, w)
			require.IsType(t, &fixedLenReaderFrom{}, w)
			rf := w.(io.ReaderFrom)

			src := newSourceReader(tc.in)
			totalN := src.TotalLen()
			n, err := rf.ReadFrom(src) // tc.in will be empty after this call
			require.NoError(t, err)
			require.Equal(t, int64(totalN), n)

			// All buf should come from ReadFrom calls
			require.Equal(t, tc.expected, inner.buf)
			require.Zero(t, inner.wrOps)
		})
	}
}

// Make sure FixedLenWriter's mixed Write and ReadFrom method can split TLS hello.
func TestFixedLenWriterMixedWriteAndReadFrom(t *testing.T) {
	type subCase struct {
		wrLenBeforeRF []int // Each int represents the number of bytes Write before ReadFrom
		expected      [][]byte
		expectWrCnt   int
	}

	cases := []struct {
		name     string
		in       [][]byte
		frag     FixedLenFragFunc
		subCases []subCase
	}{
		{
			name: "SplitFullClientHelloRecord",
			in: [][]byte{
				{0x16, 0x03, 0x02, 0x00, 0x10,
					0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
			},
			frag: func(_ int) int { return 7 },
			subCases: []subCase{
				{
					wrLenBeforeRF: []int{1, 2, 3, 4},
					expected: [][]byte{
						{0x16, 0x03, 0x02, 0x00, 0x07, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99},
						{0x16, 0x03, 0x02, 0x00, 0x09, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 0, // 0 direct Write to base because not enough header
				},
				{
					wrLenBeforeRF: []int{5},
					expected: [][]byte{
						{0x16, 0x03, 0x02, 0x00, 0x07}, {0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99},
						{0x16, 0x03, 0x02, 0x00, 0x09, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 1, // 1 direct Write to base of header1
				},
				{
					wrLenBeforeRF: []int{8},
					expected: [][]byte{
						{0x16, 0x03, 0x02, 0x00, 0x07}, {0xff, 0xee, 0xdd}, {0xcc, 0xbb, 0xaa, 0x99},
						{0x16, 0x03, 0x02, 0x00, 0x09, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 2, // 2 direct Write to base of header1 + payload1
				},
				{
					wrLenBeforeRF: []int{12},
					expected: [][]byte{
						{0x16, 0x03, 0x02, 0x00, 0x07}, {0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99},
						{0x16, 0x03, 0x02, 0x00, 0x09}, {0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
						{},
					},
					expectWrCnt: 3, // 3 direct Write to base of hdr1 + payload1 + hdr2
				},
				{
					wrLenBeforeRF: []int{13},
					expected: [][]byte{
						{0x16, 0x03, 0x02, 0x00, 0x07}, {0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99},
						{0x16, 0x03, 0x02, 0x00, 0x09}, {0x88}, {0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 4, // 4 direct Write to base of hdr1 + payload1 + hdr2 + payload2[0]
				},
				{
					wrLenBeforeRF: []int{20},
					expected: [][]byte{
						{0x16, 0x03, 0x02, 0x00, 0x07}, {0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99},
						{0x16, 0x03, 0x02, 0x00, 0x09}, {0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 4, // 4 direct Write to base of hdr1 + payload1 + hdr2 + payload2[0]
				},
				{
					wrLenBeforeRF: []int{21},
					expected: [][]byte{
						{0x16, 0x03, 0x02, 0x00, 0x07}, {0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99},
						{0x16, 0x03, 0x02, 0x00, 0x09}, {0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00},
						{},
					},
					expectWrCnt: 4, // 4 direct Write to base of hdr1 + payload1 + hdr2 + payload2[0]
				},
			},
		},
		{
			name: "SplitClientHelloSingleByteSeq",
			in: [][]byte{
				{0x16}, {0x03}, {0x03}, {0x00}, {0x10},
				{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99}, {0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
			},
			frag: func(_ int) int { return 7 },
			subCases: []subCase{
				{
					wrLenBeforeRF: []int{1, 2, 3, 4},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07, 0xff},
						{0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09, 0x88},
						{0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 0, // direct Write to base because not enough header
				},
				{
					wrLenBeforeRF: []int{5},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07}, {0xff},
						{0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09, 0x88},
						{0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 1, // direct Write to base of header1
				},
				{
					wrLenBeforeRF: []int{8},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07}, {0xff},
						{0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09, 0x88},
						{0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 4, // direct Write to base of hdr1 + 3 bytes of payload1
				},
				{
					wrLenBeforeRF: []int{12},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07}, {0xff},
						{0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09},
						{0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 9, // direct Write to base of hdr1 + payload1
				},
				{
					wrLenBeforeRF: []int{13},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09},
						{0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 10, // direct Write to base of hdr1 + payload1 + hdr2 + payload2[0]
				},
				{
					wrLenBeforeRF: []int{20},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09},
						{0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 17, // direct Write to base of hdr1 + payload1 + hdr2 + payload2[:-1]
				},
				{
					wrLenBeforeRF: []int{21},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{0xff}, {0xee}, {0xdd}, {0xcc}, {0xbb}, {0xaa}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09},
						{0x88}, {0x77}, {0x66}, {0x55}, {0x44}, {0x33}, {0x22}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 18, // direct Write to base of hdr1 + payload1 + hdr2 + payload2
				},
			},
		},
		{
			name: "SplitClientHelloWithEmptyWrites",
			in: [][]byte{
				{}, {0x16}, {0x03}, {}, {0x03}, {0x00}, {}, {0x10},
				{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99}, {0x88},
				{}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
			},
			frag: func(_ int) int { return 7 },
			subCases: []subCase{
				{
					wrLenBeforeRF: []int{1, 2, 3, 4},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09, 0x88},
						{}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 0, // direct Write to base because not enough header
				},
				{
					wrLenBeforeRF: []int{5},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09, 0x88},
						{}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 1, // direct Write to base of header1
				},
				{
					wrLenBeforeRF: []int{8},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09, 0x88},
						{}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 6, // direct Write to base of header1 + {}{0xff}{0xee}{}{0xdd}
				},
				{
					wrLenBeforeRF: []int{12},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09},
						{0x88}, {}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 13, // direct Write to base of header1 + payload1
				},
				{
					wrLenBeforeRF: []int{13},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09},
						{0x88}, {}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 14, // direct Write to base of hdr1 + payload1 + hdr2 + payload2[0]
				},
				{
					wrLenBeforeRF: []int{20},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09},
						{0x88}, {}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 25, // direct Write to base of header1 + payload1 + hdr2 + payload2[:-1]
				},
				{
					wrLenBeforeRF: []int{21},
					expected: [][]byte{
						{0x16, 0x03, 0x03, 0x00, 0x07},
						{}, {0xff}, {0xee}, {}, {0xdd}, {0xcc}, {}, {0xbb}, {0xaa}, {}, {0x99},
						{0x16, 0x03, 0x03, 0x00, 0x09},
						{0x88}, {}, {0x77}, {0x66}, {}, {0x55}, {0x44}, {}, {0x33}, {0x22}, {}, {0x11}, {0x00},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 26, // direct Write to base of hdr1 + payload1 + hdr2 + payload2
				},
			},
		},
		{
			name: "NoSplitForNonClientHello",
			// invalid record type
			in:   [][]byte{{0x17}, {0x03, 0x01}, {0x00, 0x06, 0xff}, {0xee, 0xdd, 0xcc}, {0xbb, 0xaa}},
			frag: func(_ int) int { return 1 },
			subCases: []subCase{
				{
					wrLenBeforeRF: []int{0, 1, 2, 3, 4},
					expected: [][]byte{
						{0x17, 0x03, 0x01, 0x00, 0x06, 0xff}, {0xee, 0xdd, 0xcc}, {0xbb, 0xaa},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 0, // direct Write to base because not enough header
				},
				{
					wrLenBeforeRF: []int{5},
					expected: [][]byte{
						{0x17, 0x03, 0x01, 0x00, 0x06}, {0xff}, {0xee, 0xdd, 0xcc}, {0xbb, 0xaa},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 1, // direct Write to base of header
				},
				{
					wrLenBeforeRF: []int{6},
					expected: [][]byte{
						{0x17, 0x03, 0x01, 0x00, 0x06}, {0xff}, {0xee, 0xdd, 0xcc}, {0xbb, 0xaa},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 2, // direct Write to base of header + payload[0]
				},
				{
					wrLenBeforeRF: []int{8},
					expected: [][]byte{
						{0x17, 0x03, 0x01, 0x00, 0x06}, {0xff}, {0xee, 0xdd}, {0xcc}, {0xbb, 0xaa},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 3, // direct Write to base of header + payload[0:2]
				},
				{
					wrLenBeforeRF: []int{10},
					expected: [][]byte{
						{0x17, 0x03, 0x01, 0x00, 0x06}, {0xff}, {0xee, 0xdd, 0xcc}, {0xbb}, {0xaa},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 4, // direct Write to base of header + payload[0:3]
				},
				{
					wrLenBeforeRF: []int{11},
					expected: [][]byte{
						{0x17, 0x03, 0x01, 0x00, 0x06}, {0xff}, {0xee, 0xdd, 0xcc}, {0xbb, 0xaa},
						{}, // empty Read due to MultiReader implementation
					},
					expectWrCnt: 4, // direct Write to base of header + payload
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, subtc := range tc.subCases {
				for _, wrLen := range subtc.wrLenBeforeRF {
					subCaseName := fmt.Sprintf("Write_%d_BytesThenReadFrom", wrLen)
					t.Run(subCaseName, func(t *testing.T) {
						inner := &collectReaderFrom{}
						w, err := NewFixedLenWriter(inner, tc.frag)
						require.NoError(t, err)
						require.NotNil(t, w)
						require.IsType(t, &fixedLenReaderFrom{}, w)
						rf := w.(io.ReaderFrom)

						src := newSourceReader(tc.in)
						totalN := src.TotalLen()
						n, err := src.WriteToLimit(w, wrLen)
						require.NoError(t, err)
						require.Equal(t, wrLen, n)

						nn, err := rf.ReadFrom(src)
						require.NoError(t, err)
						require.Equal(t, int64(totalN-wrLen), nn)

						// All buf should come from ReadFrom calls
						require.Equal(t, subtc.expected, inner.buf)
						require.Equal(t, subtc.expectWrCnt, inner.wrOps)

						assertWriteAndReadFromDirectlyGoesToBase(t, &inner.collectWriter, w)
					})
				}
			}
		})
	}
}

// Test assertions

func assertWriteAndReadFromDirectlyGoesToBase(t *testing.T, inner *collectWriter, w io.Writer) {
	prevWrOps := inner.wrOps
	prevBufLen := len(inner.buf)

	// All Write should be transmitted directly to base
	n, err := w.Write([]byte{0xaa, 0xcc})
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, prevWrOps+1, inner.wrOps)     // 1 additional Write
	require.Equal(t, prevBufLen+1, len(inner.buf)) // 1 additional buffer
	require.Equal(t, []byte{0xaa, 0xcc}, inner.buf[len(inner.buf)-1])

	// All following ReadFrom should be transmitted directly to base
	if rf, ok := w.(io.ReaderFrom); ok {
		nn, err := rf.ReadFrom(bytes.NewBuffer([]byte{0xdd, 0xbb}))
		require.NoError(t, err)
		require.Equal(t, int64(2), nn)
		require.Equal(t, prevWrOps+1, inner.wrOps)     // no additional Writes
		require.Equal(t, prevBufLen+3, len(inner.buf)) // 2 additional buffers
		require.Equal(t, [][]byte{{0xdd, 0xbb}, {}}, inner.buf[len(inner.buf)-2:])
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
	bufSize int
}

func (w *collectReaderFrom) ReadFrom(r io.Reader) (n int64, err error) {
	if w.bufSize <= 0 {
		w.bufSize = 512
	}
	tmp := make([]byte, w.bufSize)
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

func newSourceReader(b [][]byte) *slicesReader {
	r := &slicesReader{
		buf: make([][]byte, len(b)),
	}
	copy(r.buf, b)
	return r
}

func (r *slicesReader) TotalLen() (n int) {
	for _, b := range r.buf {
		n += len(b)
	}
	return
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

func (r *slicesReader) WriteToLimit(dst io.Writer, limit int) (n int, err error) {
	var m int
	for {
		if len(r.buf) == 0 || limit <= 0 {
			return
		}
		b := r.buf[0]
		if limit < len(b) {
			b = b[:limit]
		}
		m, err = dst.Write(b)
		n += m
		limit -= m
		if r.buf[0] = r.buf[0][m:]; len(r.buf[0]) == 0 {
			r.buf = r.buf[1:]
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
	}
}
