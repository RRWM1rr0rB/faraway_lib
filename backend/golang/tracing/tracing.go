package tracing

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdk_trace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrNewExporter = errors.New("failed to create OTLP exporter")
)

// New initializes OpenTelemetry tracing with OTLP exporter.
func New(params ...ConfigParam) (trace.TracerProvider, error) {
	cfg := &config{
		host: defaultHost,
		port: defaultPort,
	}
	for _, param := range params {
		param(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(cfg.host+":"+cfg.port),
		otlptracehttp.WithInsecure(),
	)

	exporter, err := otlptrace.New(context.Background(), client)
	if err != nil {
		return nil, errors.Join(ErrNewExporter, err)
	}

	res, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceInstanceID(cfg.serviceID),
			semconv.ServiceName(cfg.serviceName),
			semconv.ServiceVersion(cfg.serviceVersion),
			semconv.DeploymentEnvironment(cfg.envName),
		),
	)

	provider := sdk_trace.NewTracerProvider(
		sdk_trace.WithBatcher(exporter),
		sdk_trace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return provider, nil
}

// Start creates a new span.
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("").Start(ctx, name, opts...)
}

// Continue creates a new span that continues the given span.
//
// It works by fetching the parent span from the context and creating a new span
// with the same parent. If the parent span is not recording, it returns the
// context and span unchanged.
func Continue(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		// If the span is not recording, return the context and span unchanged.
		return ctx, span
	}

	// Otherwise, create a new span with the same parent.
	return Start(ctx, name, opts...)
}
