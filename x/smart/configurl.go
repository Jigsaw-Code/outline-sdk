// Copyright 2026 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package smart

import (
	"context"

	"golang.getoutline.org/sdk/transport"
	"golang.getoutline.org/sdk/x/configurl"
)

type streamBaseParserKey struct{}

func withStreamBaseParser(ctx context.Context, parser configurl.BuildFunc[transport.StreamDialer]) context.Context {
	return context.WithValue(ctx, streamBaseParserKey{}, parser)
}

// registerContextStreamBase runs during setup, before any searches. Capture the
// original parser itself: calling New after registration would look up the
// replacement and recurse. Even a missing/disabled original base can be overridden for TLS.
func registerContextStreamBase(registry configurl.MutableRegistry) {
	originalBase, originalErr := configurl.Lookup[transport.StreamDialer](registry, "")
	newBase := func(ctx context.Context, config *configurl.Config) (transport.StreamDialer, error) {
		if override, ok := ctx.Value(streamBaseParserKey{}).(configurl.BuildFunc[transport.StreamDialer]); ok && override != nil {
			return override(ctx, config)
		}
		if originalErr != nil {
			return nil, originalErr
		}
		return originalBase(ctx, config)
	}
	configurl.RegisterBase(registry, newBase)
}
