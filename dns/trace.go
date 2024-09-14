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

package dns

import (
	"context"

	"golang.org/x/net/dns/dnsmessage"
)

type DNSClientTrace struct {
	ResolverSetup func(resolverType string, network string, address string)
	QuestionReady func(question dnsmessage.Question)
	ResponseDone  func(question dnsmessage.Question, response *dnsmessage.Message, err error)
	WroteDone     func(question dnsmessage.Question, err error)
	ReadDone      func(question dnsmessage.Question, err error)
}
type contextKey struct{}

var dnsClientTraceKey = contextKey{}

// WithDNSClientTrace adds DNS trace information to the context.
func WithDNSClientTrace(ctx context.Context, trace *DNSClientTrace) context.Context {
	return context.WithValue(ctx, dnsClientTraceKey, trace)
}

// GetDNSClientTrace retrieves the DNS trace information from the context, if available.
func GetDNSClientTrace(ctx context.Context) *DNSClientTrace {
	if trace, ok := ctx.Value(dnsClientTraceKey).(*DNSClientTrace); ok {
		return trace
	}
	return nil
}
