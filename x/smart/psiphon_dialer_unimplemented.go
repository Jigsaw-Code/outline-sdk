//go:build !psiphon
// If the build tag `psiphon` is not set, create a stub function to avoid pulling in GPL'd code

package smart

import (
	"context"
	"errors"
	"fmt"

	"github.com/Jigsaw-Code/outline-sdk/transport"
)

func newPsiphonDialer(ctx context.Context, psiphonJSON []byte) (transport.StreamDialer, error) {
	return nil, fmt.Errorf("To use psiphon configuration in x/smart the package must be built with the build tag `psiphon`: %w", errors.ErrUnsupported) 
}