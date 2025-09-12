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
	"errors"
	"fmt"

	"golang.org/x/crypto/cryptobyte"
)

// From https://cs.opensource.google/go/go/+/master:src/crypto/tls/common.go
// TLS extension numbers
const (
	extensionEncryptedClientHello uint16 = 0xfe0d
)

// From https://cs.opensource.google/go/go/+/master:src/crypto/tls/handshake_messages.go

// readUint16LengthPrefixed acts like s.ReadUint16LengthPrefixed, but targets a
// []byte instead of a cryptobyte.String.
func readUint16LengthPrefixed(s *cryptobyte.String, out *[]byte) bool {
	return s.ReadUint16LengthPrefixed((*cryptobyte.String)(out))
}

// From https://cs.opensource.google/go/go/+/master:src/crypto/tls/ech.go

type echCipher struct {
	KDFID  uint16
	AEADID uint16
}

type echExtension struct {
	Type uint16
	Data []byte
}

type echConfig struct {
	raw []byte

	Version uint16
	Length  uint16

	ConfigID             uint8
	KemID                uint16
	PublicKey            []byte
	SymmetricCipherSuite []echCipher

	MaxNameLength uint8
	PublicName    []byte
	Extensions    []echExtension
}

var errMalformedECHConfigList = errors.New("tls: malformed ECHConfigList")

type echConfigErr struct {
	field string
}

func (e *echConfigErr) Error() string {
	if e.field == "" {
		return "tls: malformed ECHConfig"
	}
	return fmt.Sprintf("tls: malformed ECHConfig, invalid %s field", e.field)
}

func parseECHConfig(enc []byte) (skip bool, ec echConfig, err error) {
	s := cryptobyte.String(enc)
	ec.raw = []byte(enc)
	if !s.ReadUint16(&ec.Version) {
		return false, echConfig{}, &echConfigErr{"version"}
	}
	if !s.ReadUint16(&ec.Length) {
		return false, echConfig{}, &echConfigErr{"length"}
	}
	if len(ec.raw) < int(ec.Length)+4 {
		return false, echConfig{}, &echConfigErr{"length"}
	}
	ec.raw = ec.raw[:ec.Length+4]
	if ec.Version != extensionEncryptedClientHello {
		s.Skip(int(ec.Length))
		return true, echConfig{}, nil
	}
	if !s.ReadUint8(&ec.ConfigID) {
		return false, echConfig{}, &echConfigErr{"config_id"}
	}
	if !s.ReadUint16(&ec.KemID) {
		return false, echConfig{}, &echConfigErr{"kem_id"}
	}
	if !readUint16LengthPrefixed(&s, &ec.PublicKey) {
		return false, echConfig{}, &echConfigErr{"public_key"}
	}
	var cipherSuites cryptobyte.String
	if !s.ReadUint16LengthPrefixed(&cipherSuites) {
		return false, echConfig{}, &echConfigErr{"cipher_suites"}
	}
	for !cipherSuites.Empty() {
		var c echCipher
		if !cipherSuites.ReadUint16(&c.KDFID) {
			return false, echConfig{}, &echConfigErr{"cipher_suites kdf_id"}
		}
		if !cipherSuites.ReadUint16(&c.AEADID) {
			return false, echConfig{}, &echConfigErr{"cipher_suites aead_id"}
		}
		ec.SymmetricCipherSuite = append(ec.SymmetricCipherSuite, c)
	}
	if !s.ReadUint8(&ec.MaxNameLength) {
		return false, echConfig{}, &echConfigErr{"maximum_name_length"}
	}
	var publicName cryptobyte.String
	if !s.ReadUint8LengthPrefixed(&publicName) {
		return false, echConfig{}, &echConfigErr{"public_name"}
	}
	ec.PublicName = publicName
	var extensions cryptobyte.String
	if !s.ReadUint16LengthPrefixed(&extensions) {
		return false, echConfig{}, &echConfigErr{"extensions"}
	}
	for !extensions.Empty() {
		var e echExtension
		if !extensions.ReadUint16(&e.Type) {
			return false, echConfig{}, &echConfigErr{"extensions type"}
		}
		if !extensions.ReadUint16LengthPrefixed((*cryptobyte.String)(&e.Data)) {
			return false, echConfig{}, &echConfigErr{"extensions data"}
		}
		ec.Extensions = append(ec.Extensions, e)
	}

	return false, ec, nil
}

// parseECHConfigList parses a draft-ietf-tls-esni-18 ECHConfigList, returning a
// slice of parsed ECHConfigs, in the same order they were parsed, or an error
// if the list is malformed.
func parseECHConfigList(data []byte) ([]echConfig, error) {
	s := cryptobyte.String(data)
	var length uint16
	if !s.ReadUint16(&length) {
		return nil, errMalformedECHConfigList
	}
	if length != uint16(len(data)-2) {
		return nil, errMalformedECHConfigList
	}
	var configs []echConfig
	for len(s) > 0 {
		if len(s) < 4 {
			return nil, errors.New("tls: malformed ECHConfig")
		}
		configLen := uint16(s[2])<<8 | uint16(s[3])
		skip, ec, err := parseECHConfig(s)
		if err != nil {
			return nil, err
		}
		s = s[configLen+4:]
		if !skip {
			configs = append(configs, ec)
		}
	}
	return configs, nil
}
