// Copyright 2022 The Outline Authors
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
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Microbenchmark for the performance of Shadowsocks UDP encryption.
func BenchmarkPack(b *testing.B) {
	b.StopTimer()
	b.ResetTimer()

	key, err := NewEncryptionKey(CHACHA20IETFPOLY1305, "test secret")
	require.NoError(b, err)
	MTU := 1500
	pkt := make([]byte, MTU)
	plaintextBuf := pkt[key.SaltSize() : len(pkt)-key.TagSize()]

	start := time.Now()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		Pack(pkt, plaintextBuf, key)
	}
	b.StopTimer()
	elapsed := time.Since(start)

	megabits := float64(8*len(plaintextBuf)*b.N) * 1e-6
	b.ReportMetric(megabits/(elapsed.Seconds()), "mbps")
}

type fixedSaltGenerator struct {
	Salt []byte
}

func (sg *fixedSaltGenerator) GetSalt(salt []byte) error {
	n := copy(salt, sg.Salt)
	if n < len(salt) {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func TestPack(t *testing.T) {
	key := makeTestKey(t)
	payload := makeTestPayload(100)
	encrypted := make([]byte, len(payload)+key.SaltSize()+key.cipher.tagSize)
	salt := makeTestPayload(key.SaltSize())
	sg := &fixedSaltGenerator{salt}
	encrypted, err := PackSalt(encrypted, payload, key, sg)
	require.NoError(t, err)
	// Ensure the selected salt is used.
	require.Equal(t, salt, encrypted[:len(salt)])

	// Ensure it decrypts correctly.
	decrypted, err := Unpack(nil, encrypted, key)
	require.NoError(t, err)
	require.Equal(t, payload, decrypted)
}
