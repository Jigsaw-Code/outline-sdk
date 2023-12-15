// tracing_utils.go in the connectivity package

package connectivity

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	// necessary imports
)

// JSONStdoutExporter, initTracing, and other related functions go here
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
