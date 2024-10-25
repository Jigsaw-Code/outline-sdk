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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertCipher(t *testing.T, cipher string, saltSize, tagSize int) {
	key, err := NewEncryptionKey(cipher, "")
	require.NoError(t, err)
	require.Equal(t, saltSize, key.SaltSize())

	dummyAead, err := key.NewAEAD(make([]byte, key.SaltSize()))
	require.NoError(t, err)
	require.Equal(t, tagSize, key.TagSize())
	require.Equal(t, key.TagSize(), dummyAead.Overhead())
}

func TestSizes(t *testing.T) {
	// Values from https://shadowsocks.org/en/spec/AEAD-Ciphers.html
	assertCipher(t, CHACHA20IETFPOLY1305, 32, 16)
	assertCipher(t, AES256GCM, 32, 16)
	assertCipher(t, AES192GCM, 24, 16)
	assertCipher(t, AES128GCM, 16, 16)
}

func TestShadowsocksCipherNames(t *testing.T) {
	key, err := NewEncryptionKey("chacha20-ietf-poly1305", "")
	require.NoError(t, err)
	require.Equal(t, chacha20IETFPOLY1305Cipher, key.cipher)

	key, err = NewEncryptionKey("aes-256-gcm", "")
	require.NoError(t, err)
	require.Equal(t, aes256GCMCipher, key.cipher)

	key, err = NewEncryptionKey("aes-192-gcm", "")
	require.NoError(t, err)
	require.Equal(t, aes192GCMCipher, key.cipher)

	key, err = NewEncryptionKey("aes-128-gcm", "")
	require.NoError(t, err)
	require.Equal(t, aes128GCMCipher, key.cipher)
}

func TestUnsupportedCipher(t *testing.T) {
	_, err := NewEncryptionKey("aes-256-cfb", "")
	var unsupportedErr ErrUnsupportedCipher
	if assert.ErrorAs(t, err, &unsupportedErr) {
		assert.Equal(t, "aes-256-cfb", unsupportedErr.Name)
		assert.Equal(t, "unsupported cipher aes-256-cfb", unsupportedErr.Error())
	}
}

func TestMaxNonceSize(t *testing.T) {
	for _, aeadName := range supportedCiphers {
		key, err := NewEncryptionKey(aeadName, "")
		if err != nil {
			t.Errorf("Failed to create Cipher %v: %v", aeadName, err)
		}
		aead, err := key.NewAEAD(make([]byte, key.SaltSize()))
		if err != nil {
			t.Errorf("Failed to create AEAD %v: %v", aeadName, err)
		}
		if aead.NonceSize() > len(zeroNonce) {
			t.Errorf("Cipher %v has nonce size %v > zeroNonce (%v)", aeadName, aead.NonceSize(), len(zeroNonce))
		}
	}
}

func TestMaxTagSize(t *testing.T) {
	var calculatedMax int
	for _, cipher := range supportedCiphers {
		key, err := NewEncryptionKey(cipher, "")
		if !assert.NoError(t, err, "Failed to create cipher %v", cipher) {
			continue
		}
		assert.LessOrEqualf(t, key.TagSize(), maxTagSize, "Tag size for cipher %v (%v) is greater than the max (%v)", cipher, key.TagSize(), maxTagSize)
		if key.TagSize() > calculatedMax {
			calculatedMax = key.TagSize()
		}
	}
	require.Equal(t, maxTagSize, calculatedMax)
}
