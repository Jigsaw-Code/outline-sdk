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
		return nil, fmt.Errorf("Failed to get the user cache directory: %w", err)
	}

	config.DataRootDirectory = path.Join(cacheBaseDir, "psiphon")
	if err := os.MkdirAll(config.DataRootDirectory, 0700); err != nil {
		return nil, fmt.Errorf("Failed to create storage directory: %w", err)
	}
	finder.logCtx(ctx, "Using data store in %v\n", config.DataRootDirectory)

	// Create a new context that propagates the original context cancellation, but only during the construction.
	startCtx, cancel := context.WithCancel(ctx)
	context.AfterFunc(startCtx, cancel)

	dialer := psiphon.GetSingletonDialer()
	if err := dialer.Start(startCtx, config); err != nil {
		return nil, fmt.Errorf("failed to start psiphon dialer: %w", err)
	}

	return dialer, nil
}
