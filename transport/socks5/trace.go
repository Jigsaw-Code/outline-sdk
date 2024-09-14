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

package socks5

import (
	"context"
)

type contextKey struct{}

type SOCKS5ClientTrace struct {
	RequestStarted func(cmd byte, addr string)
	RequestDone    func(network string, bindAddr string, err error)
}

var socksClientTraceKey = contextKey{}

// WithTLSClientTrace adds TLS trace information to the context.
func WithSOCKS5ClientTrace(ctx context.Context, trace *SOCKS5ClientTrace) context.Context {
	return context.WithValue(ctx, socksClientTraceKey, trace)
}

// GetTLSClientTrace retrieves the TLS trace information from the context, if available.
func GetSOCKS5ClientTrace(ctx context.Context) *SOCKS5ClientTrace {
	if trace, ok := ctx.Value(socksClientTraceKey).(*SOCKS5ClientTrace); ok {
		return trace
	}
	return nil
}
