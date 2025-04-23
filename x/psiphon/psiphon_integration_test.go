//go:build psiphon

package psiphon

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func readPsiphonConfigFromFile(tb testing.TB) (string, error) {
	configBytes, err := os.ReadFile("integration_test_config.yaml")
	if err != nil {
		return "", err
	}
	return string(configBytes), nil
}

func newValidTestConfig(tb testing.TB) (*DialerConfig, func()) {
	// It's useful to test actually starting psiphon connections,
	// but doing so requires supplying a valid psiphon config with private information.
	// To run these tests please supply your own config in integration_test_config.yaml
	privatePsiphonConfig, err := readPsiphonConfigFromFile(tb)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			tb.Skip("Integration testing for Psiphon requires adding a user-supplied config in integration_test_config.yaml")
		}
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
	dialer.Start(startCtx, cfg)
	startCancel()
	require.NotNil(t, dialer.tunnel)
	dialer.Stop()
}
