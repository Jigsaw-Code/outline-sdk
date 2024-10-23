// Copyright 2020 The Outline Authors
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

package shadowsocks

import (
	"bytes"
	"testing"
)

func TestRandomSaltGenerator(t *testing.T) {
	if err := RandomSaltGenerator.GetSalt(nil); err != nil {
		t.Error(err)
	}
	salt := make([]byte, 16)
	if err := RandomSaltGenerator.GetSalt(salt); err != nil {
		t.Error(err)
	}
	if bytes.Equal(salt, make([]byte, 16)) {
		t.Error("Salt is all zeros")
	}
}

func BenchmarkRandomSaltGenerator(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		salt := make([]byte, 32)
		for pb.Next() {
			if err := RandomSaltGenerator.GetSalt(salt); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// setRandomBitsToOne replaces any random bits in the output with 1.
func setRandomBitsToOne(salter SaltGenerator, output []byte) error {
	salt := make([]byte, len(output))
	// OR together 128 salts.  The probability that any random bit is
	// 0 for all 128 random salts is 2^-128, which is close enough to zero.
	for i := 0; i < 128; i++ {
		if err := salter.GetSalt(salt); err != nil {
			return err
		}
		for i := range salt {
			output[i] |= salt[i]
		}
	}
	return nil
}

// Test that the prefix bytes are respected, and the remainder are random.
func TestTypicalPrefix(t *testing.T) {
	prefix := []byte("twelve bytes")
	salter := NewPrefixSaltGenerator(prefix)

	output := make([]byte, 32)
	if err := setRandomBitsToOne(salter, output); err != nil {
		t.Error(err)
	}

	for i := 0; i < 12; i++ {
		if output[i] != prefix[i] {
			t.Error("prefix mismatch")
		}
	}

	for _, b := range output[12:] {
		if b != 0xFF {
			t.Error("unexpected zero bit")
		}
	}
}

// Test that all bytes are random when the prefix is nil
func TestNilPrefix(t *testing.T) {
	salter := NewPrefixSaltGenerator(nil)

	output := make([]byte, 64)
	if err := setRandomBitsToOne(salter, output); err != nil {
		t.Error(err)
	}
	for _, b := range output {
		if b != 0xFF {
			t.Error("unexpected zero bit")
		}
	}
}
