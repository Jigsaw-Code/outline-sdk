//go:build psiphon

package psiphon

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func readPsiphonConfigFromFile(tb testing.TB) string {
	// It's useful to test actually starting psiphon connections,
	// but doing so requires supplying a valid psiphon config with private information.
	// To run these tests please supply your own config in integration_test_config.yaml
	configBytes, err := os.ReadFile("./integration_test_config.yaml")
	if err != nil {
		require.NoError(tb, err)
	}
	return string(configBytes)
}

func newValidTestConfig(tb testing.TB) (*DialerConfig, func()) {
	privatePsiphonConfig := readPsiphonConfigFromFile(tb)
	if strings.Contains(privatePsiphonConfig, "{<YOUR_CONFIG_HERE>}") {
		tb.Skip("Integration testing for Psiphon requires adding a user-supplied config in integration_test_config.yaml")
	}
	tempDir, err := os.MkdirTemp("", "psiphon")
	require.NoError(tb, err)
	return &DialerConfig{
		DataRootDirectory: tempDir,
		ProviderConfig:    json.RawMessage(privatePsiphonConfig),
	}, func() { os.RemoveAll(tempDir) }
}

func TestDialer_CancelledAfterStart_DoesntCloseTunnel(t *testing.T) {
	cfg, delete := newValidTestConfig(t)
	defer delete()
	startCtx, startCancel := context.WithCancel(context.Background())
	dialer := GetSingletonDialer()

	startDone := make(chan error)
	go func() { startDone <- dialer.Start(startCtx, cfg) }()
	<-startDone
	startCancel() // Cancel only after start is done.

	// Cancelling the start does not nix the tunnel.
	require.NotNil(t, dialer.tunnel)

	dialer.Stop()
}

func TestDialer_FetchExample(t *testing.T) {
	cfg, delete := newValidTestConfig(t)
	defer delete()
	startCtx, startCancel := context.WithCancel(context.Background())
	defer startCancel()
	dialer := GetSingletonDialer()

	startDone := make(chan error)
	go func() { startDone <- dialer.Start(startCtx, cfg) }()
	require.NoError(t, <-startDone)

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialStream(ctx, addr)
	}
	httpClient := &http.Client{Transport: &http.Transport{DialContext: dialContext}, Timeout: 5 * time.Second}

	req, err := http.NewRequest("GET", "https://example.com", nil)
	require.NoError(t, err)
	resp, err := httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)

	dialer.Stop()
}
