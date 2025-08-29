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

package ech

import (
	"fmt"
	"io"

	"github.com/cloudflare/circl/hpke"
	"golang.org/x/crypto/cryptobyte"
)

// addHpkeKeyConfig adds the HpkeKeyConfig
func addHpkeKeyConfig(b *cryptobyte.Builder, rand io.Reader) error {
	randConfigID := make([]byte, 1)
	if _, err := io.ReadFull(rand, randConfigID); err != nil {
		return fmt.Errorf("failed to read random config ID: %w", err)
	}
	b.AddUint8(randConfigID[0]) // uint8 config_id
	kem_id := uint16(hpke.KEM_X25519_HKDF_SHA256)
	b.AddUint16(kem_id) // HpkeKemId (uint16) kem_id

	kem := hpke.KEM(kem_id)
	publicKey, _, err := kem.Scheme().GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate KEM key pair: %w", err)
	}
	publicKeyBytes, err := publicKey.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	// opaque public_key<1..2^16-1> (HpkePublicKey)
	b.AddUint16LengthPrefixed(func(child *cryptobyte.Builder) {
		child.AddBytes(publicKeyBytes)
	})

	// HpkeSymmetricCipherSuite cipher_suites<4..2^16-4>
	b.AddUint16LengthPrefixed(func(child *cryptobyte.Builder) {
		child.AddUint16(uint16(hpke.KDF_HKDF_SHA256)) // HpkeKdfId(uint16) kdf_id
		// Note: BoringSSL chooses between AES128GCM and CHACHA20_PLOY1305 based on whether
		// hardware acceleration is configured.
		child.AddUint16(uint16(hpke.AEAD_AES128GCM)) // HpkeAeadId(uint16) aead_id
	})
	return nil
}

// addECHConfigContents appends the serialized ECHConfigContents to the given builder.
func addECHConfigContents(b *cryptobyte.Builder, rand io.Reader, publicName string) error {
	// HpkeKeyConfig key_config
	if err := addHpkeKeyConfig(b, rand); err != nil {
		return fmt.Errorf("failed to add HPKE key config: %w", err)
	}

	// uint8 maximum_name_length
	b.AddUint8(uint8(42))

	// opaque public_name<1..255>
	publicNameBytes := []byte(publicName)
	b.AddUint8LengthPrefixed(func(child *cryptobyte.Builder) {
		child.AddBytes(publicNameBytes)
	})

	// ECHConfigExtension extensions<0..2^16-1>
	b.AddUint16LengthPrefixed(func(child *cryptobyte.Builder) {
		// No extensions
	})

	return nil
}

// addECHConfig appends a serialized ECHConfig to the given builder.
func addECHConfig(b *cryptobyte.Builder, rand io.Reader, publicName string) {
	// uint16 version
	b.AddUint16(0xfe0d)
	// uint16 length
	b.AddUint16LengthPrefixed(func(child *cryptobyte.Builder) {
		// ECHConfigContents contents
		if err := addECHConfigContents(child, rand, publicName); err != nil {
			b.SetError(fmt.Errorf("failed to add ECHConfigContents: %w", err))
			return
		}
	})
}

// GenerateGreaseECHConfigList creates a serialized ECHConfigList containing one
// GREASE ECHConfig.
// Client behavior: https://www.ietf.org/archive/id/draft-ietf-tls-esni-25.html#name-grease-ech
// ECHConfigList format https://www.ietf.org/archive/id/draft-ietf-tls-esni-25.html#name-encrypted-clienthello-confi
func GenerateGreaseECHConfigList(rand io.Reader, publicName string) ([]byte, error) {
	var b cryptobyte.Builder
	// ECHConfig ECHConfigList<4..2^16-1>
	b.AddUint16LengthPrefixed(func(child *cryptobyte.Builder) {
		addECHConfig(child, rand, publicName)
	})
	return b.Bytes()
}
