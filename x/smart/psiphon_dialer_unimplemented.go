//go:build !psiphon

// If the build tag `psiphon` is not set, create a stub function to avoid pulling in GPL'd code

package smart

import (
	"context"
	"errors"
	"fmt"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

func (f *StrategyFinder) newPsiphonDialer(ctx context.Context, psiphonJSON []byte, psiphonSignature string) (transport.StreamDialer, error) {
	f.logCtx(ctx, "❌ Attempted to start psiphon tunnel %v but library was built without psiphon support. Please build using -tag psiphon.\n", psiphonSignature)

	return nil, fmt.Errorf("To use psiphon configuration in x/smart the package must be built with the build tag `psiphon`: %w", errors.ErrUnsupported)
}
