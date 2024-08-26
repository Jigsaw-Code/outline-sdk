// Copyright 2023 Jigsaw Operations LLC
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

package tls

import (
	"context"
	"crypto/tls"
)

type contextKey struct{}

type TLSClientTrace struct {
	TLSHandshakeStart func()
	TLSHandshakeDone  func(state tls.ConnectionState, err error)
}

var tlsClientTraceKey = contextKey{}

// WithTLSClientTrace adds TLS trace information to the context.
func WithTLSClientTrace(ctx context.Context, trace *TLSClientTrace) context.Context {
	return context.WithValue(ctx, tlsClientTraceKey, trace)
}

// GetTLSClientTrace retrieves the TLS trace information from the context, if available.
func GetTLSClientTrace(ctx context.Context) *TLSClientTrace {
	if trace, ok := ctx.Value(tlsClientTraceKey).(*TLSClientTrace); ok {
		return trace
	}
	return nil
}
