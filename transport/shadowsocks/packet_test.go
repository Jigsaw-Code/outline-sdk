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
	"bytes"
	"crypto/rand"
	"errors"
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

func TestPackSalt(t *testing.T) {
	key := makeTestKey(t)
	payload := makeTestPayload(100)
	encrypted := make([]byte, len(payload)+key.SaltSize()+key.TagSize())
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

func TestPack(t *testing.T) {
	key, err := NewEncryptionKey("aes-256-gcm", "password")
	require.NoError(t, err)
	plaintext := []byte("test message")
	dst := make([]byte, 1024)

	encrypted, err := Pack(dst, plaintext, key)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)
	require.Greater(t, len(encrypted), len(plaintext))

	// Decrypt the message and compare it to the original plaintext.
	decrypted, err := Unpack(nil, encrypted, key)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

func TestPack_ShortBuffer(t *testing.T) {
	key, err := NewEncryptionKey("aes-256-gcm", "password")
	require.NoError(t, err)
	plaintext := []byte("test message")
	dst := make([]byte, 1) // Too short

	_, err = Pack(dst, plaintext, key)
	require.ErrorIs(t, err, io.ErrShortBuffer)
}

func TestPack_Random(t *testing.T) {
	// Test with various sizes of plaintext
	for i := 0; i < 10; i++ {
		key, err := NewEncryptionKey("aes-256-gcm", "password")
		require.NoError(t, err)

		plaintextSize := i * 100
		plaintext := make([]byte, plaintextSize)
		_, err = rand.Read(plaintext)
		require.NoError(t, err)

		dst := make([]byte, plaintextSize+key.SaltSize()+key.TagSize()+100)

		encrypted, err := Pack(dst, plaintext, key)
		require.NoError(t, err)
		require.NotEmpty(t, encrypted)
		require.Greater(t, len(encrypted), len(plaintext))

		// Decrypt the message and compare it to the original plaintext.
		decrypted, err := Unpack(nil, encrypted, key)
		require.NoError(t, err)
		require.Equal(t, plaintext, decrypted)
	}
}

func TestPackSalt_ShortBuffer(t *testing.T) {
	key, err := NewEncryptionKey("aes-256-gcm", "password")
	require.NoError(t, err)
	plaintext := []byte("test message")
	dst := make([]byte, 1) // Too short for salt

	_, err = PackSalt(dst, plaintext, key, RandomSaltGenerator)
	require.ErrorIs(t, err, io.ErrShortBuffer)
}

func TestPackSalt_CustomSaltGenerator(t *testing.T) {
	key, err := NewEncryptionKey("aes-256-gcm", "password")
	require.NoError(t, err)
	plaintext := []byte("test message")
	dst := make([]byte, 1024)

	// Create a custom salt generator that always returns the same salt.
	salt := make([]byte, key.SaltSize())
	_, err = rand.Read(salt)
	require.NoError(t, err)
	sg := &fixedSaltGenerator{Salt: salt}

	encrypted1, err := PackSalt(dst, plaintext, key, sg)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted1)

	encrypted2, err := PackSalt(dst, plaintext, key, sg)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted2)

	// Verify that the encrypted messages are the same because the salt is the same.
	require.True(t, bytes.Equal(encrypted1, encrypted2))

	// Decrypt the message and compare it to the original plaintext.
	decrypted, err := Unpack(nil, encrypted1, key)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)
}

type errorSaltGenerator struct {
	Error error
}

func (sg *errorSaltGenerator) GetSalt(salt []byte) error {
	return sg.Error
}

func TestPackSalt_BadSaltGenerator(t *testing.T) {
	key, err := NewEncryptionKey("aes-256-gcm", "password")
	require.NoError(t, err)
	plaintext := []byte("test message")
	dst := make([]byte, 1024)
	sg := &errorSaltGenerator{Error: errors.New("failed to generate salt")}
	_, err = PackSalt(dst, plaintext, key, sg)
	require.ErrorContains(t, err, "failed to generate salt")
}
