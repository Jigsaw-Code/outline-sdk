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
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

type Cipher struct {
	name        string
	newInstance func(key []byte) (cipher.AEAD, error)
	keySize     int
	saltSize    int
	tagSize     int
}

// List of supported AEAD ciphers, as specified at https://shadowsocks.org/guide/aead.html
var (
	CHACHA20IETFPOLY1305 = &Cipher{"AEAD_CHACHA20_POLY1305", chacha20poly1305.New, chacha20poly1305.KeySize, 32, 16}
	AES256GCM            = &Cipher{"AEAD_AES_256_GCM", newAesGCM, 32, 32, 16}
	AES192GCM            = &Cipher{"AEAD_AES_192_GCM", newAesGCM, 24, 24, 16}
	AES128GCM            = &Cipher{"AEAD_AES_128_GCM", newAesGCM, 16, 16, 16}
)

var supportedCiphers = [](*Cipher){CHACHA20IETFPOLY1305, AES256GCM, AES192GCM, AES128GCM}

// CipherByName returns a [*Cipher] with the given name, or an error if the cipher is not supported.
// The name must be the IETF name (as per https://www.iana.org/assignments/aead-parameters/aead-parameters.xhtml) or the
// Shadowsocks alias from https://shadowsocks.org/guide/aead.html.
func CipherByName(name string) (*Cipher, error) {
	switch strings.ToUpper(name) {
	case "AEAD_CHACHA20_POLY1305", "CHACHA20-IETF-POLY1305":
		return CHACHA20IETFPOLY1305, nil
	case "AEAD_AES_256_GCM", "AES-256-GCM":
		return AES256GCM, nil
	case "AEAD_AES_192_GCM", "AES-192-GCM":
		return AES192GCM, nil
	case "AEAD_AES_128_GCM", "AES-128-GCM":
		return AES128GCM, nil
	default:
		return nil, fmt.Errorf("unsupported cipher %v", name)
	}
}

func newAesGCM(key []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

func maxTagSize() int {
	max := 0
	for _, spec := range supportedCiphers {
		if spec.tagSize > max {
			max = spec.tagSize
		}
	}
	return max
}

// EncryptionKey encapsulates a Shadowsocks AEAD spec and a secret
type EncryptionKey struct {
	aead   *Cipher
	secret []byte
}

// SaltSize is the size of the salt for this Cipher
func (c *EncryptionKey) SaltSize() int {
	return c.aead.saltSize
}

// TagSize is the size of the AEAD tag for this Cipher
func (c *EncryptionKey) TagSize() int {
	return c.aead.tagSize
}

var subkeyInfo = []byte("ss-subkey")

// NewAEAD creates the AEAD for this cipher
func (c *EncryptionKey) NewAEAD(salt []byte) (cipher.AEAD, error) {
	sessionKey := make([]byte, c.aead.keySize)
	r := hkdf.New(sha1.New, c.secret, salt, subkeyInfo)
	if _, err := io.ReadFull(r, sessionKey); err != nil {
		return nil, err
	}
	return c.aead.newInstance(sessionKey)
}

// Function definition at https://www.openssl.org/docs/manmaster/man3/EVP_BytesToKey.html
func simpleEVPBytesToKey(data []byte, keyLen int) ([]byte, error) {
	var derived, di []byte
	h := md5.New()
	for len(derived) < keyLen {
		_, err := h.Write(di)
		if err != nil {
			return nil, err
		}
		_, err = h.Write(data)
		if err != nil {
			return nil, err
		}
		derived = h.Sum(derived)
		di = derived[len(derived)-h.Size():]
		h.Reset()
	}
	return derived[:keyLen], nil
}

// NewEncryptionKey creates a Cipher given a cipher name and a secret
func NewEncryptionKey(cipher *Cipher, secretText string) (*EncryptionKey, error) {
	// Key derivation as per https://shadowsocks.org/en/spec/AEAD-Ciphers.html
	secret, err := simpleEVPBytesToKey([]byte(secretText), cipher.keySize)
	if err != nil {
		return nil, err
	}
	return &EncryptionKey{cipher, secret}, nil
}

// Assumes all ciphers have NonceSize() <= 12.
var zeroNonce [12]byte

// DecryptOnce will decrypt the cipherText using the cipher and salt, appending the output to plainText.
func DecryptOnce(key *EncryptionKey, salt []byte, plainText, cipherText []byte) ([]byte, error) {
	aead, err := key.NewAEAD(salt)
	if err != nil {
		return nil, err
	}
	if len(cipherText) < aead.Overhead() {
		return nil, io.ErrUnexpectedEOF
	}
	if cap(plainText)-len(plainText) < len(cipherText)-aead.Overhead() {
		return nil, io.ErrShortBuffer
	}
	return aead.Open(plainText, zeroNonce[:aead.NonceSize()], cipherText, nil)
}
