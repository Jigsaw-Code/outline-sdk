//go:build psiphon
// +build psiphon

// If the build tag `psiphon` is set, allow importing and calling psiphon

package smart

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/psiphon"
)

func newPsiphonDialer(finder *StrategyFinder, ctx context.Context, psiphonJSON []byte) (transport.StreamDialer, error) {
	config := &psiphon.DialerConfig{ProviderConfig: psiphonJSON}

	cacheBaseDir, err := os.UserCacheDir()
	if err != nil {
		// This call fails on android and ios, in which case we use the psiphon default

		// https://github.com/Psiphon-Labs/psiphon-tunnel-core/blob/6560abad2812fa438b76ee6c0f74ee27a4aba82e/ClientLibrary/clientlib/clientlib.go#L40
		// "Empty string means the default will be used (current working directory)."
		finder.logCtx(ctx, "Failed to get user cache directory, using the psiphon default (current dir)\n")
	}
	if err == nil {
		config.DataRootDirectory = path.Join(cacheBaseDir, "psiphon")

		if err := os.MkdirAll(config.DataRootDirectory, 0700); err != nil {
			return nil, fmt.Errorf("Failed to create storage directory: %w", err)
		}
		finder.logCtx(ctx, "Using data store in %v\n", config.DataRootDirectory)
	}

	dialer := psiphon.GetSingletonDialer()
	if err := dialer.Start(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to start psiphon dialer: %w", err)
	}

	return dialer, nil
}
