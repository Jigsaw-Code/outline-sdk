package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeConfig(t *testing.T) {
	// Test that empty config is accepted.
	_, err := SanitizeConfig("")
	require.NoError(t, err)

	// Test that a valid config is accepted.
	sanitizedConfig, err := SanitizeConfig("split:5|ss://jhvdsjkfhvkhsadvf@example.com:1234?prefix=HTTP1.1")
	fmt.Println(sanitizedConfig)
	require.NoError(t, err)
	require.Equal(t, "split:5|ss://redacted@example.com:1234?prefix=HTTP1.1", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("split:5|ss://jhvdsjkfhvkhsadvf@example.com:1234")
	fmt.Println(sanitizedConfig)
	require.NoError(t, err)
	require.Equal(t, "split:5|ss://redacted@example.com:1234", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("split:5|vless://ac08785d-203d-4db4-915c-eb4e23435fd62@example.com:443?path=%2Fvless&security=tls&encryption=none&alpn=h2&host=sub.hello.com&fp=chrome&type=ws&sni=sub.hello.com#vless-ws-tls-cdn")
	fmt.Println(sanitizedConfig)
	require.NoError(t, err)
	require.Equal(t, "split:5|vless://redacted@example.com:443?path=%2Fvless&security=tls&encryption=none&alpn=h2&host=sub.hello.com&fp=chrome&type=ws&sni=sub.hello.com#vless-ws-tls-cdn", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("split:5|tlsfrag:5")
	fmt.Println(sanitizedConfig)
	require.NoError(t, err)
	require.Equal(t, "split:5|tlsfrag:5", sanitizedConfig)

	// Test that an invalid config is rejected.
	_, err = SanitizeConfig("::hghg")
	require.Error(t, err)
}
