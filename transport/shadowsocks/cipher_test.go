// Copyright 2020 Jigsaw Operations LLC
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

	"github.com/stretchr/testify/require"
)

func assertCipher(t *testing.T, cipher *Cipher, saltSize, tagSize int) {
	key, err := NewEncryptionKey(cipher, "")
	require.Nil(t, err)
	require.Equal(t, saltSize, key.SaltSize())

	dummyAead, err := key.aead.newInstance(make([]byte, key.aead.keySize))
	require.Nil(t, err)
	require.Equal(t, dummyAead.Overhead(), key.TagSize())
	require.Equal(t, tagSize, key.TagSize())
}

func TestSizes(t *testing.T) {
	// Values from https://shadowsocks.org/en/spec/AEAD-Ciphers.html
	assertCipher(t, CHACHA20IETFPOLY1305, 32, 16)
	assertCipher(t, AES256GCM, 32, 16)
	assertCipher(t, AES192GCM, 24, 16)
	assertCipher(t, AES128GCM, 16, 16)
}

func TestShadowsocksCipherNames(t *testing.T) {
	cipher, err := CipherByName("chacha20-ietf-poly1305")
	require.Nil(t, err)
	require.Equal(t, CHACHA20IETFPOLY1305, cipher)

	cipher, err = CipherByName("aes-256-gcm")
	require.Nil(t, err)
	require.Equal(t, AES256GCM, cipher)

	cipher, err = CipherByName("aes-192-gcm")
	require.Nil(t, err)
	require.Equal(t, AES192GCM, cipher)

	cipher, err = CipherByName("aes-128-gcm")
	require.Nil(t, err)
	require.Equal(t, AES128GCM, cipher)
}

func TestUnsupportedCipher(t *testing.T) {
	_, err := CipherByName("aes-256-cfb")
	if err == nil {
		t.Errorf("Should get an error for unsupported cipher")
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
