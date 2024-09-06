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

package config

import (
	"encoding/base64"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_sanitizeShadowsocksURL(t *testing.T) {
	ssURL, err := url.Parse("ss://YWVzLTEyOC1nY206dGVzdA@192.168.100.1:8888")
	require.NoError(t, err)
	sanitized, err := sanitizeShadowsocksURL(ssURL)
	require.NoError(t, err)
	require.Equal(t, "ss://REDACTED@192.168.100.1:8888", sanitized)
}

func Test_sanitizeShadowsocksURL_withPrefix(t *testing.T) {
	ssURL, err := url.Parse("ss://YWVzLTEyOC1nY206dGVzdA@192.168.100.1:8888?prefix=foo")
	require.NoError(t, err)
	sanitized, err := sanitizeShadowsocksURL(ssURL)
	require.NoError(t, err)
	require.Equal(t, "ss://REDACTED@192.168.100.1:8888?prefix=foo", sanitized)
}

func TestParseShadowsocksURLFullyEncoded(t *testing.T) {
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:1234567@example.com:1234?prefix=HTTP%2F1.1%20"))
	urls, err := parseConfig("ss://" + string(encoded) + "#outline-123")
	require.NoError(t, err)
	require.Equal(t, 1, len(urls))

	config, err := parseShadowsocksURL(urls[0])

	require.NoError(t, err)
	require.Equal(t, "example.com:1234", config.serverAddress)
	require.Equal(t, "HTTP/1.1 ", string(config.prefix))
}

func TestParseShadowsocksURLUserInfoEncoded(t *testing.T) {
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:1234567"))
	urls, err := parseConfig("ss://" + string(encoded) + "@example.com:1234?prefix=HTTP%2F1.1%20" + "#outline-123")
	require.NoError(t, err)
	require.Equal(t, 1, len(urls))

	config, err := parseShadowsocksURL(urls[0])

	require.NoError(t, err)
	require.Equal(t, "example.com:1234", config.serverAddress)
	require.Equal(t, "HTTP/1.1 ", string(config.prefix))
}

func TestParseShadowsocksURLNoEncoding(t *testing.T) {
	configString := "ss://aes-256-gcm:1234567@example.com:1234"
	urls, err := parseConfig(configString)
	require.NoError(t, err)
	require.Equal(t, 1, len(urls))

	config, err := parseShadowsocksURL(urls[0])

	require.NoError(t, err)
	require.Equal(t, "example.com:1234", config.serverAddress)
}

func TestParseShadowsocksURLInvalidCipherInfoFails(t *testing.T) {
	configString := "ss://aes-256-gcm1234567@example.com:1234"
	urls, err := parseConfig(configString)
	require.NoError(t, err)
	require.Equal(t, 1, len(urls))

	_, err = parseShadowsocksURL(urls[0])

	require.Error(t, err)
}

func TestParseShadowsocksURLUnsupportedCypherFails(t *testing.T) {
	configString := "ss://Y2hhY2hhMjAtaWV0Zi1wb2x5MTMwnTpLeTUyN2duU3FEVFB3R0JpQ1RxUnlT@example.com:1234"
	urls, err := parseConfig(configString)
	require.NoError(t, err)
	require.Equal(t, 1, len(urls))

	_, err = parseShadowsocksURL(urls[0])

	require.Error(t, err)
}

func TestParseShadowsocksLegacyBase64URL(t *testing.T) {
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:1234567@example.com:1234?prefix=HTTP%2F1.1%20"))
	urls, err := parseConfig("ss://" + string(encoded) + "#outline-123")
	require.NoError(t, err)
	require.Equal(t, 1, len(urls))

	config, err := parseShadowsocksLegacyBase64URL(urls[0])

	require.NoError(t, err)
	require.Equal(t, "example.com:1234", config.serverAddress)
	require.Equal(t, "HTTP/1.1 ", string(config.prefix))
}

func TestParseShadowsocksSIP002URLUnsuccessful(t *testing.T) {
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("aes-256-gcm:1234567@example.com:1234?prefix=HTTP%2F1.1%20"))
	urls, err := parseConfig("ss://" + string(encoded) + "#outline-123")
	require.NoError(t, err)
	require.Equal(t, 1, len(urls))

	_, err = parseShadowsocksSIP002URL(urls[0])

	require.Error(t, err)
}
