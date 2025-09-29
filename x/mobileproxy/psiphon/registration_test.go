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

//go:build psiphon

package psiphon

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func Test_getPsiphonConfigSignature_ValidFields(t *testing.T) {
	var yamlNode any
	require.NoError(t, yaml.Unmarshal([]byte(`{
		"PropagationChannelId": "FFFFFFFFFFFFFFFF",
		"SponsorId": "FFFFFFFFFFFFFFFF",
		"ClientPlatform": "outline",
		"ClientVersion": "1"
	}`), &yamlNode))
	expected := "{PropagationChannelId: FFFFFFFFFFFFFFFF, SponsorId: FFFFFFFFFFFFFFFF, [...]}"
	actual := getPsiphonConfigSignature(yamlNode)
	require.Equal(t, expected, actual)
}

func Test_getPsiphonConfigSignature_InvalidFields(t *testing.T) {
	// If we don't understand the psiphon config we received for any reason
	// then just output it as an opaque string
	var yamlNode any
	require.NoError(t, yaml.Unmarshal([]byte(`{"ClientPlatform": "outline", "ClientVersion": "1"}`), &yamlNode))
	expected := `{"ClientPlatform":"outline","ClientVersion":"1"}`
	actual := getPsiphonConfigSignature(yamlNode)
	require.Equal(t, expected, actual)
}

func Test_getPsiphonConfigSignature_InvalidJson(t *testing.T) {
	key := "key"
	// Create a yamlNode that is not representable in JSON.
	yamlNode := map[*string]int{&key: 4}
	expected := `invalid config: json: unsupported type: map[*string]int`
	actual := getPsiphonConfigSignature(yamlNode)
	require.Equal(t, expected, actual)
}
