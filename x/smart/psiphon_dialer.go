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

func getUserCacheDir(finder *StrategyFinder, ctx context.Context) (string, error) {
	cacheBaseDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("Failed to get the user cache directory: %w", err)
	}

	userCacheDir := path.Join(cacheBaseDir, "psiphon")
	if err := os.MkdirAll(cacheBaseDir, 0700); err != nil {
		return "", fmt.Errorf("Failed to create storage directory: %w", err)
	}
	finder.logCtx(ctx, "Using data store in %v\n", userCacheDir)

	return userCacheDir, nil
}

func newPsiphonDialer(finder *StrategyFinder, ctx context.Context, psiphonJSON []byte) (transport.StreamDialer, error) {
	config := &psiphon.DialerConfig{ProviderConfig: psiphonJSON}

	userCacheDir, err := getUserCacheDir(finder, ctx)
	if err != nil {
		return nil, err
	}
	config.DataRootDirectory = userCacheDir

	dialer := psiphon.GetSingletonDialer()
	if err := dialer.Start(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to start psiphon dialer: %w", err)
	}

	return dialer, nil
}
