//go:build psiphon

package psiphon

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// It's useful to test actually starting psiphon connections,
// but doing so requires supplying a valid psiphon config with private information.
// To run these tests please supply your own config here.
var privatePsiphonConfig = `{<YOUR_CONFIG_HERE>}`

func newValidTestConfig(tb testing.TB) (*DialerConfig, func()) {
	tempDir, err := os.MkdirTemp("", "psiphon")
	require.NoError(tb, err)
	return &DialerConfig{
		DataRootDirectory: tempDir,
		ProviderConfig:    json.RawMessage(privatePsiphonConfig),
	}, func() { os.RemoveAll(tempDir) }
}

func TestDialer_CancelledAfterStart_DoesntCloseTunnel(t *testing.T) {
	if privatePsiphonConfig == "{<YOUR_CONFIG_HERE>}" {
		t.Skip("Integration testing for Psiphon requires adding a user-supplied config")
	}

	cfg, delete := newValidTestConfig(t)
	defer delete()
	startCtx, startCancel := context.WithCancel(context.Background())
	dialer := GetSingletonDialer()
	dialer.Start(startCtx, cfg)
	startCancel()
	require.NotNil(t, dialer.tunnel)
	dialer.Stop()
}
