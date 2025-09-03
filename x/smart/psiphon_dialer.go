//go:build psiphon
// +build psiphon

// If the build tag `psiphon` is set, allow importing and calling psiphon

package smart

import (
	"context"
	"fmt"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/psiphon"
)

func newPsiphonDialer(finder *StrategyFinder, ctx context.Context, psiphonJSON []byte) (transport.StreamDialer, error) {
	config := &psiphon.DialerConfig{ProviderConfig: psiphonJSON}

	userCacheDir, err := getUserCacheDir()
	if err != nil {
		return nil, err
	}
	finder.logCtx(ctx, "Using data store in %v\n", userCacheDir)
	config.DataRootDirectory = userCacheDir

	dialer := psiphon.GetSingletonDialer()
	if err := dialer.Start(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to start psiphon dialer: %w", err)
	}

	return dialer, nil
}
