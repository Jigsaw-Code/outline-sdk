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
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func Test_parsePsiphon(t *testing.T) {
	var yamlNode any
	require.NoError(t, yaml.Unmarshal([]byte(`{
		"PropagationChannelId": "FFFFFFFFFFFFFFFF",
		"SponsorId": "FFFFFFFFFFFFFFFF",
		"ClientPlatform": "outline",
		"ClientVersion": "1"
	}`), &yamlNode))
	dialer, err := parsePsiphon(context.Background(), yamlNode)
	require.NotNil(t, dialer)
	require.NoError(t, err)
}
