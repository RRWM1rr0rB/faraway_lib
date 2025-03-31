package logging

import (
	"context"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/RRWM1rr0rB/faraway_lib/tracing"
)

const (
	requestIDLogKey = "request_id"
	traceIDLogKey   = "trace_id"
	spanIDLogKey    = "span_id"
)

func Middleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		mLogger := L(ctx).With(slog.String("endpoint", r.URL.RequestURI()))

		if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
			mLogger = mLogger.With(slog.String(traceIDLogKey, span.TraceID().String()))
			tracing.TraceValue(ctx, traceIDLogKey, span.TraceID().String())
			mLogger = mLogger.With(slog.String(spanIDLogKey, span.TraceID().String()))
		}

		ctx = ContextWithLogger(ctx, mLogger)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// WithTraceIDInLogger is a gRPC interceptor that enriches the logger with method name, trace ID, and span ID.
func WithTraceIDInLogger() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		// Add gRPC method name to the logger
		mLogger := L(ctx).With(slog.String("method", info.FullMethod))

		// Extract SpanContext from the context
		if span := trace.SpanContextFromContext(ctx); span.IsValid() {
			// Add TraceID and SpanID if available
			traceID := span.TraceID().String()
			spanID := span.SpanID().String()

			mLogger = mLogger.With(
				slog.String(traceIDLogKey, traceID),
				slog.String(spanIDLogKey, spanID),
			)

			// Optional: propagate trace ID to tracing system (e.g., for metrics)
			tracing.TraceValue(ctx, traceIDLogKey, traceID)
		}

		// Update the context with the new logger
		ctx = ContextWithLogger(ctx, mLogger)

		return handler(ctx, req)
	}
}
