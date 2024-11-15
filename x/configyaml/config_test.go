// Copyright 2024 The Outline Authors
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

package configyaml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseConfig_Direct(t *testing.T) {
	type TestConfig struct {
		Endpoint struct {
			Host string
			Port int
		}
		Proto string
	}
	var testConfig TestConfig
	configText := `
  endpoint:
    host: example.com
    port: 1234
  proto: ss`

	decoder := yaml.NewDecoder(strings.NewReader(configText))
	decoder.KnownFields(true)
	require.NoError(t, decoder.Decode(&testConfig))
	assert.Equal(t, "example.com", testConfig.Endpoint.Host)
	assert.Equal(t, 1234, testConfig.Endpoint.Port)
	assert.Equal(t, "ss", testConfig.Proto)
}

func TestParseConfig_Node(t *testing.T) {
	type TestConfig struct {
		Endpoint struct {
			Host string
			Port int
		}
		Proto string
	}
	configText := `
  endpoint:
    host: example.com
    port: 1234
  proto: ss
  extra: foo`

	parsedConfig, err := ParseConfig(configText)
	require.NoError(t, err)
	var testConfig TestConfig
	// TODO: Make it fail on extra field.
	require.NoError(t, parsedConfig.Decode(&testConfig))
	assert.Equal(t, "example.com", testConfig.Endpoint.Host)
	assert.Equal(t, 1234, testConfig.Endpoint.Port)
	assert.Equal(t, "ss", testConfig.Proto)
}

func TestParseConfig_Nest(t *testing.T) {
	type TestConfig struct {
		Type string
		// TODO: Implement Unmarshaler wrapper instead.
		Config yaml.Node
	}
	configText := `
  type: dial
  config:
    host: example.com
    port: 1234`

	parsedConfig, err := ParseConfig(configText)
	require.NoError(t, err)
	var testConfig TestConfig
	// TODO: Make it fail on extra field.
	require.NoError(t, parsedConfig.Decode(&testConfig))
	require.Equal(t, "dial", testConfig.Type)
	require.NotNil(t, testConfig.Config)

	type EndpointConfig struct {
		Host string
		Port int
	}
	var endpointCfg EndpointConfig
	require.NoError(t, testConfig.Config.Decode(&endpointCfg))
	assert.Equal(t, "example.com", endpointCfg.Host)
	assert.Equal(t, 1234, endpointCfg.Port)
}
