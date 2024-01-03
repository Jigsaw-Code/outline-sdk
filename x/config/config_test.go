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
	require.Equal(t, "split:5|ss://[redacted]@example.com:1234?prefix=HTTP1.1", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("split:5|ss://jhvdsjkfhvkhsadvf@example.com:1234")
	fmt.Println(sanitizedConfig)
	require.NoError(t, err)
	require.Equal(t, "split:5|ss://[redacted]@example.com:1234", sanitizedConfig)

	// Test that a valid config is accepted.
	sanitizedConfig, err = SanitizeConfig("split:5|tlsfrag:5")
	fmt.Println(sanitizedConfig)
	require.NoError(t, err)
	require.Equal(t, "split:5|tlsfrag:5", sanitizedConfig)

	// Test that an invalid config is rejected.
	_, err = SanitizeConfig("ss://[redacted]@")
	require.Error(t, err)
}

func TestGetHostnamesFromConfig(t *testing.T) {
	// Test that empty config is accepted.
	_, err := GetHostnamesFromConfig("")
	require.NoError(t, err)

	// Test that a valid config is accepted.
	hostname, err := GetHostnamesFromConfig("ss://example.com:1234?prefix=HTTP1.1|socks5://example2.com:1234")
	fmt.Println(hostname)
	require.NoError(t, err)
	require.Equal(t, []string{"example.com", "example2.com"}, hostname)

	//Test that an invalid config is rejected.
	_, err = GetHostnamesFromConfig("|fadsf|")
	require.Error(t, err)

	//Test that an invalid config is rejected.
	_, err = GetHostnamesFromConfig("|")
	require.Error(t, err)
}
