// Copyright 2024 Jigsaw Operations LLC
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

package psiphon

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseConfig_ParseCorrectly(t *testing.T) {
	config, err := ParseConfig([]byte(`{
		"PropagationChannelId": "ID1",
		"SponsorId": "ID2"
	}`))
	require.NoError(t, err)
	require.Equal(t, "ID1", config.PropagationChannelId)
	require.Equal(t, "ID2", config.SponsorId)
}

func TestParseConfig_DefaultClientPlatform(t *testing.T) {
	config, err := ParseConfig([]byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("OutlineSDK/%v/%v", runtime.GOOS, runtime.GOARCH), config.ClientPlatform)
}

func TestParseConfig_OverrideClientPlatform(t *testing.T) {
	config, err := ParseConfig([]byte(`{"ClientPlatform": "win"}`))
	require.NoError(t, err)
	require.Equal(t, "win", config.ClientPlatform)
}

func TestParseConfig_AcceptOkOptions(t *testing.T) {
	_, err := ParseConfig([]byte(`{
		"DisableLocalHTTPProxy": true,
		"DisableLocalSocksProxy": true,
		"TargetApiProtocol": "ssh"
	}`))
	require.NoError(t, err)
}

func TestParseConfig_RejectBadOptions(t *testing.T) {
	_, err := ParseConfig([]byte(`{"DisableLocalHTTPProxy": false}`))
	require.Error(t, err)

	_, err = ParseConfig([]byte(`{"DisableLocalSocksProxy": false}`))
	require.Error(t, err)

	_, err = ParseConfig([]byte(`{"TargetApiProtocol": "web"}`))
	require.Error(t, err)
}

func TestParseConfig_RejectUnknownFields(t *testing.T) {
	_, err := ParseConfig([]byte(`{
		"PropagationChannelId": "ID",
		"UknownField": false
	}`))
	require.Error(t, err)
}
