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
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/cloudflare/circl/hpke"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/cryptobyte"
)

// TestGenerateGreaseECHConfigListSuccess tests if the function executes without error.
func TestGenerateGreaseECHConfigListSuccess(t *testing.T) {
	publicName := "grease.example.com"
	_, err := GenerateGreaseECHConfigList(rand.Reader, publicName)
	require.NoError(t, err)
}

// TestParseGreaseECHConfigList tests if the generated list can be parsed and has the expected structure.
func TestParseGreaseECHConfigList(t *testing.T) {
	publicName := "grease.example.com"
	echConfigListBytes, err := GenerateGreaseECHConfigList(rand.Reader, publicName)
	require.NoError(t, err)

	parser := cryptobyte.String(echConfigListBytes)

	var echConfigList cryptobyte.String
	require.True(t, parser.ReadUint16LengthPrefixed(&echConfigList))
	require.True(t, parser.Empty())

	// The list contains one ECHConfig. We parse it directly.
	var version uint16
	require.True(t, echConfigList.ReadUint16(&version))
	require.Equal(t, uint16(0xfe0d), version)

	var contents cryptobyte.String
	require.True(t, echConfigList.ReadUint16LengthPrefixed(&contents))
	require.True(t, echConfigList.Empty(), "ECHConfigList should contain only one ECHConfig for GREASE")

	// Parse ECHConfigContents
	var configID uint8
	require.True(t, contents.ReadUint8(&configID))

	var kemIDUint16 uint16
	require.True(t, contents.ReadUint16(&kemIDUint16))
	require.Equal(t, hpke.KEM_X25519_HKDF_SHA256, hpke.KEM(kemIDUint16))

	var publicKey cryptobyte.String
	require.True(t, contents.ReadUint16LengthPrefixed(&publicKey))
	require.False(t, publicKey.Empty())

	var cipherSuites cryptobyte.String
	require.True(t, contents.ReadUint16LengthPrefixed(&cipherSuites))

	var kdfIDUint16 uint16
	require.True(t, cipherSuites.ReadUint16(&kdfIDUint16))
	require.Equal(t, hpke.KDF_HKDF_SHA256, hpke.KDF(kdfIDUint16))

	var aeadIDUint16 uint16
	require.True(t, cipherSuites.ReadUint16(&aeadIDUint16))
	require.Equal(t, hpke.AEAD_AES128GCM, hpke.AEAD(aeadIDUint16))
	require.True(t, cipherSuites.Empty(), "Unexpected bytes after cipher suite")

	var maxNameLength uint8
	require.True(t, contents.ReadUint8(&maxNameLength))

	var publicNameBytes cryptobyte.String
	require.True(t, contents.ReadUint8LengthPrefixed(&publicNameBytes))
	require.Equal(t, publicName, string(publicNameBytes))

	var extensions cryptobyte.String
	require.True(t, contents.ReadUint16LengthPrefixed(&extensions))
	require.True(t, extensions.Empty(), "Extensions block should be empty for this GREASE config")
	require.True(t, contents.Empty(), "Unexpected bytes at end of ECHConfigContents")
}

// TestRandomness checks if consecutive calls produce different random elements.
func TestRandomness(t *testing.T) {
	publicName := "grease.example.com"
	list1, err1 := GenerateGreaseECHConfigList(rand.Reader, publicName)
	require.NoError(t, err1)
	list2, err2 := GenerateGreaseECHConfigList(rand.Reader, publicName)
	require.NoError(t, err2)

	require.NotEqual(t, list1, list2, "Generated ECHConfigLists are identical, randomness failed")

	// Quick parse to check config_id and public_key
	parseConfigIDAndKey := func(b []byte) (uint8, []byte, error) {
		parser := cryptobyte.String(b)
		var list, contents cryptobyte.String
		var version uint16
		if !parser.ReadUint16LengthPrefixed(&list) ||
			!list.ReadUint16(&version) || // version
			!list.ReadUint16LengthPrefixed(&contents) {
			return 0, nil, fmt.Errorf("failed to parse basic structure")
		}
		var configID uint8
		var kemID uint16
		var publicKey cryptobyte.String
		if !contents.ReadUint8(&configID) ||
			!contents.ReadUint16(&kemID) ||
			!contents.ReadUint16LengthPrefixed(&publicKey) {
			return 0, nil, fmt.Errorf("failed to parse contents structure")
		}
		return configID, publicKey, nil
	}

	configID1, key1, err1 := parseConfigIDAndKey(list1)
	require.NoError(t, err1)
	configID2, key2, err2 := parseConfigIDAndKey(list2)
	require.NoError(t, err2)

	if configID1 == configID2 {
		t.Logf("Warning: config_ids are the same, less ideal for randomness but possible.")
	}
	require.NotEqual(t, key1, key2, "Public keys are identical, randomness failed")
}
