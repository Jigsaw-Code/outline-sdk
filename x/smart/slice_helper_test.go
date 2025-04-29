// Copyright 2025 The Outline Authors
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

package smart

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMoveToFront(t *testing.T) {
	cases := []struct {
		name     string
		input    []int
		idx      int
		expected []int
	}{{
		name:     "negative",
		input:    []int{1, 2, 3, 4},
		idx:      -1,
		expected: []int{1, 2, 3, 4},
	}, {
		name:     "zero",
		input:    []int{1, 2, 3, 4},
		idx:      0,
		expected: []int{1, 2, 3, 4},
	}, {
		name:     "middle element",
		input:    []int{1, 2, 3, 4, 5},
		idx:      2, // move 3 to front
		expected: []int{3, 1, 2, 4, 5},
	}, {
		name:     "last element",
		input:    []int{1, 2, 3, 4},
		idx:      3, // move 4 to front
		expected: []int{4, 1, 2, 3},
	}, {
		name:     "overflow",
		input:    []int{1, 2, 3, 4},
		idx:      4,
		expected: []int{1, 2, 3, 4},
	}, {
		name:     "more overflow",
		input:    []int{1, 2, 3, 4},
		idx:      5,
		expected: []int{1, 2, 3, 4},
	}, {
		name:     "empty slice",
		input:    []int{},
		idx:      0,
		expected: []int{},
	}, {
		name:     "single element slice ",
		input:    []int{1},
		idx:      0,
		expected: []int{1},
	}, {
		name:     "single element slice overflow",
		input:    []int{1},
		idx:      1,
		expected: []int{1},
	}, {
		name:     "two elements",
		input:    []int{1, 2},
		idx:      1,
		expected: []int{2, 1},
	}, {
		name:     "slice with duplicates",
		input:    []int{1, 2, 1, 3},
		idx:      2,
		expected: []int{1, 1, 2, 3},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := make([]int, len(tc.input))
			copy(actual, tc.input)

			moveToFront(actual, tc.idx)
			require.Equal(t, tc.expected, actual)
		})
	}
}
