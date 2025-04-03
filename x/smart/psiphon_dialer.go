// +build psiphon
// If the build tag `psiphon` is set, allow importing and calling psiphon

package smart

import (
	"context"
	"errors"
	"fmt"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/Jigsaw-Code/outline-sdk/x/psiphon"
)

func getPsiphonDialer(ctx context.Context, psiphonJSON []byte) (transport.StreamDialer, error) {
	// Test calling psiphon package to prove it works
	dialer := psiphon.GetSingletonDialer()
	fmt.Printf("Initializing psiphon dialer: %+v\n", dialer)

	//TODO(laplante): set up and use psiphon dialer

	return nil, fmt.Errorf("psiphon is not yet supported, skipping: %w", errors.ErrUnsupported) 
}