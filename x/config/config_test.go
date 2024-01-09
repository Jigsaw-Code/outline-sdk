package config

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeConfig(t *testing.T) {
	// Test that empty config is accepted.
	_, err := SanitizeConfig("")
	require.NoError(t, err)

	// Test that a valid config is accepted.
	sanitizedConfig, err := SanitizeConfig("split:5|ss://jhvdsjkfhvkhsadvf@example.com:1234?prefix=HTTP1.1")
	require.NoError(t, err)
	require.Equal(t, "split:5|ss://REDACTED@example.com:1234?prefix=HTTP1.1", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("split:5|ss://jhvdsjkfhvkhsadvf@example.com:1234")
	require.NoError(t, err)
	require.Equal(t, "split:5|ss://REDACTED@example.com:1234", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("split:5|vless://ac08785d-203d-4db4-915c-eb4e23435fd62@example.com:443?path=%2Fvless&security=tls&encryption=none&alpn=h2&host=sub.hello.com&fp=chrome&type=ws&sni=sub.hello.com#vless-ws-tls-cdn")
	require.NoError(t, err)
	require.Equal(t, "split:5|vless://REDACTED@example.com:443?path=%2Fvless&security=tls&encryption=none&alpn=h2&host=sub.hello.com&fp=chrome&type=ws&sni=sub.hello.com#vless-ws-tls-cdn", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("split:5|tlsfrag:5")
	require.NoError(t, err)
	require.Equal(t, "split:5|tlsfrag:5", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("transport://hjdbfjhbqfjheqrf")
	require.NoError(t, err)
	require.Equal(t, "transport://UNKNOWN", sanitizedConfig)

	// Test that an invalid config is rejected.
	_, err = SanitizeConfig("::hghg")
	require.Error(t, err)
}

func TestSanitizeURL(t *testing.T) {
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte("jhvdsjkfhvkhsadvf@example.com:1234?prefix=HTTP1.1"))
	//decoded, _ := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(encoded)
	u, err := parseConfigPart("ss://" + string(encoded))
	require.NoError(t, err)
	sanitizedURL, err := sanitizeURL(u)
	require.NoError(t, err)
	require.Equal(t, "ss://REDACTED@example.com:1234?prefix=HTTP1.1", sanitizedURL)
}
