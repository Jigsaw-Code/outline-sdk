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

package connectivity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	"github.com/Jigsaw-Code/outline-sdk/transport"
	"github.com/miekg/dns"
)

type JSONStdoutExporter struct{}

// func (e *JSONStdoutExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
// 	for _, span := range spans {
// 		fmt.Printf("Span: %s, Duration: %v\n", span.Name(), span.EndTime().Sub(span.StartTime()))
// 	}
// 	return nil
// }

func (e *JSONStdoutExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	for _, span := range spans {
		fmt.Printf("Span: %s, Duration: %v\n", span.Name(), span.EndTime().Sub(span.StartTime()))
		jsonSpan, err := json.Marshal(span)
		if err != nil {
			return err
		}
		fmt.Println(span)
		fmt.Println(string(jsonSpan))
	}
	return nil
}

func (e *JSONStdoutExporter) Shutdown(ctx context.Context) error {
	// Perform any cleanup if necessary
	return nil
}

// exporter := &JSONStdoutExporter{}
// exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
// 	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint("localhost:4317"))

func initTracing() *trace.TracerProvider {
	collectorURL := "localhost:4318" // Default URL if not specified

	ctx := context.Background()
	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(collectorURL),
		otlptracehttp.WithInsecure(), // Use WithTLSCredentials for a secure connection
	)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("Outline Connectivity Tester"), // Explicitly set service name
			// Add other attributes as needed
		)),
		// Additional configurations like resources, sampler, etc.
	)
	otel.SetTracerProvider(tp)

	return tp
}

// TestError captures the observed error of the connectivity test.
type TestError struct {
	// Which operation in the test that failed: "dial", "write" or "read"
	Op string
	// The POSIX error, when available
	PosixError string
	// The error observed for the action
	Err error
}

var _ error = (*TestError)(nil)

func (err *TestError) Error() string {
	return fmt.Sprintf("%v: %v", err.Op, err.Err)
}

func (err *TestError) Unwrap() error {
	return err.Err
}

// TestResolverStreamConnectivity uses the given [transport.StreamEndpoint] to connect to a DNS resolver and resolve the test domain.
// The context can be used to set a timeout or deadline, or to pass values to the dialer.
func TestResolverStreamConnectivity(ctx context.Context, resolver transport.StreamEndpoint, testDomain string) (time.Duration, error) {
	tracer := otel.Tracer("TestResolverStreamConnectivity")
	ctx, span := tracer.Start(ctx, "TestResolverStreamConnectivity")
	defer span.End()
	fmt.Println("TestResolverStreamConnectivity")
	duration, err := testResolver(ctx, resolver.Connect, testDomain)
	if err != nil {
		fmt.Println("TestResolverStreamConnectivity error")
		span.RecordError(err)
	}
	return duration, err
}

// TestResolverPacketConnectivity uses the given [transport.PacketEndpoint] to connect to a DNS resolver and resolve the test domain.
// The context can be used to set a timeout or deadline, or to pass values to the listener.
func TestResolverPacketConnectivity(ctx context.Context, resolver transport.PacketEndpoint, testDomain string) (time.Duration, error) {
	tracer := otel.Tracer("TestResolverPacketConnectivity")
	ctx, span := tracer.Start(ctx, "TestResolverPacketConnectivity")
	defer span.End()
	duration, err := testResolver(ctx, resolver.Connect, testDomain)
	if err != nil {
		span.RecordError(err)
	}
	return duration, err
}

func isTimeout(err error) bool {
	var timeErr interface{ Timeout() bool }
	return errors.As(err, &timeErr) && timeErr.Timeout()
}

func makeTestError(op string, err error) error {
	var code string
	var errno syscall.Errno
	if errors.As(err, &errno) {
		code = errnoName(errno)
	} else if isTimeout(err) {
		code = "ETIMEDOUT"
	}
	return &TestError{Op: op, PosixError: code, Err: err}
}

func testResolver[C net.Conn](ctx context.Context, connect func(context.Context) (C, error), testDomain string) (time.Duration, error) {
	tracer := otel.Tracer("testResolver")
	ctx, parentSpan := tracer.Start(ctx, "testResolver")
	defer parentSpan.End()

	deadline, ok := ctx.Deadline()
	if !ok {
		// Default deadline is 5 seconds.
		deadline = time.Now().Add(5 * time.Second)
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		// Releases the timer.
		defer cancel()
	}
	testTime := time.Now()
	testErr := func() error {
		ctx, dialSpan := tracer.Start(ctx, "dial")
		conn, dialErr := connect(ctx)
		defer dialSpan.End()
		if dialErr != nil {
			dialSpan.RecordError(dialErr)
			return makeTestError("dial", dialErr)
		}
		defer conn.Close()
		conn.SetDeadline(deadline)
		dnsConn := dns.Conn{Conn: conn}

		var dnsRequest dns.Msg
		dnsRequest.SetQuestion(dns.Fqdn(testDomain), dns.TypeA)
		ctx, writeSpan := tracer.Start(ctx, "write")
		writeErr := dnsConn.WriteMsg(&dnsRequest)
		defer writeSpan.End()
		if writeErr != nil {
			writeSpan.RecordError(writeErr)
			return makeTestError("write", writeErr)
		}

		_, readSpan := tracer.Start(ctx, "read")
		_, readErr := dnsConn.ReadMsg()
		defer readSpan.End()
		if readErr != nil {
			readSpan.RecordError(readErr)
			// An early close on the connection may cause a "unexpected EOF" error. That's an application-layer error,
			// not triggered by a syscall error so we don't capture an error code.
			// TODO: figure out how to standardize on those errors.
			return makeTestError("read", readErr)
		}
		return nil
	}()
	duration := time.Since(testTime)
	return duration, testErr
}
