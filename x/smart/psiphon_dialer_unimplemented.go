//go:build !psiphon

// If the build tag `psiphon` is not set, create a stub function to avoid pulling in GPL'd code

package smart

import (
	"context"
	"errors"
	"fmt"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

func newPsiphonDialer(_ *StrategyFinder, _ context.Context, _ []byte) (transport.StreamDialer, error) {
	return nil, fmt.Errorf("attempted to start psiphon tunnel but library was built without psiphon support. Please build using -tag psiphon: %w", errors.ErrUnsupported)
}
