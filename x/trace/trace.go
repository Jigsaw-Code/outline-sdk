package trace

import (
	"context"
	"crypto/tls"

	"golang.org/x/net/dns/dnsmessage"
)

type DNSClientTrace struct {
	QuestionSent func(question dnsmessage.Question)
	ResponsDone  func(question dnsmessage.Question, response *dnsmessage.Message, err error)
	ConnectDone  func(network, addr string, err error)
	WroteDone    func(err error)
	ReadDone     func(err error)
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
