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

func newPsiphonDialer(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error) {
	config := &psiphon.DialerConfig{ProviderConfig: psiphonJSON}

	cacheBaseDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("Failed to get the user cache directory: %w", err)
	}

	config.DataRootDirectory = path.Join(cacheBaseDir, "psiphon")
	if err := os.MkdirAll(config.DataRootDirectory, 0700); err != nil {
		return nil, fmt.Errorf("Failed to create storage directory: %w", err)
	}
	fmt.Printf("Using data store in %v\n", config.DataRootDirectory)

	dialer := psiphon.GetSingletonDialer()
	// The psiphon singleton persists in the background across connections
	ctx = context.WithoutCancel(ctx)
	fmt.Printf("🏃 Attempting to start psiphon tunnel: %v\n", psiphonSignature)
	if err := dialer.Start(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to start psiphon dialer: %w", err)
	}

	return dialer, nil
}