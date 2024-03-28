// go:build (linux && !android) || windows || darwin
package sysproxy

import (
	"math/rand"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetWebProxt(t *testing.T) {
	host := net.IPv4(byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)))
	port := strconv.Itoa(rand.Intn(65536))

	SetWebProxy(host.String(), port)
	// generate a random hostname

	h, p, e, err := getWebProxy()
	require.NoError(t, err)
	require.Equal(t, host.String(), h)
	require.Equal(t, port, p)
	require.Equal(t, e, true)

	err = DisableWebProxy()
	require.NoError(t, err)
}

func TestSetWebProxywithDomain(t *testing.T) {
	// generate a random hostname
	host := generateRandomDomain()
	port := strconv.Itoa(rand.Intn(65536))

	err := SetWebProxy(host, port)
	require.NoError(t, err)

	h, p, e, err := getWebProxy()
	require.NoError(t, err)
	require.Equal(t, host, h)
	require.Equal(t, port, p)
	require.Equal(t, e, true)


	err = DisableWebProxy()
	require.NoError(t, err)

}
func TestClearWebProxy(t *testing.T) {
	err := DisableWebProxy()
	require.NoError(t, err)

	_, _, enabled, err := getWebProxy()
	require.NoError(t, err)
	require.Equal(t, false, enabled)

}

func TestSetSocksProxy(t *testing.T) {
	host := net.IPv4(byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)))
	port := strconv.Itoa(rand.Intn(65536))

	err := SetSOCKSProxy(host.String(), port)
	require.NoError(t, err)

	h, p, e, err := getSOCKSProxy()
	require.NoError(t, err)
	require.Equal(t, host.String(), h)
	require.Equal(t, port, p)
	require.Equal(t, e, true)

	err = DisableSOCKSProxy()
	require.NoError(t, err)
}

func TestClearSocksProxy(t *testing.T) {
	err := DisableSOCKSProxy()
	require.NoError(t, err)

	_, _, enabled, err := getSOCKSProxy()
	require.NoError(t, err)
	require.Equal(t, false, enabled)
}

func generateRandomDomain() string {

	// Define the characters allowed in the domain name
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"

	// Generate a random length for the domain name (between 5 and 15 characters)
	length := rand.Intn(11) + 5

	// Create a builder to efficiently build the domain name string
	var builder strings.Builder

	// Generate random characters for the domain name
	for i := 0; i < length; i++ {
		index := rand.Intn(len(chars))
		builder.WriteByte(chars[index])
	}

	// Append the ".com" suffix to the domain name
	builder.WriteString(".com")

	return builder.String()
}
