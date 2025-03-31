package tracing

import (
	"net/http"
	"sync"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
)

type traceHandlers struct {
	mu   sync.RWMutex
	data map[string]http.Handler
}

func (h *traceHandlers) Get(path string) (http.Handler, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.data[path], h.data[path] != nil
}

func (h *traceHandlers) Set(path string, handler http.Handler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data[path] = handler
}

// Middleware provides OpenTelemetry tracing for HTTP and gRPC.
func Middleware(next http.Handler) http.Handler {
	pathHandlers := &traceHandlers{data: make(map[string]http.Handler)}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var uri string
		if r.URL != nil {
			uri = r.URL.Path
		} else {
			uri = r.RequestURI
		}
		pathKey := r.Method + " " + uri

		// Check existing handler
		if h, ok := pathHandlers.Get(pathKey); ok {
			h.ServeHTTP(w, r)
			return
		}

		// Create new handler
		newHandler := otelhttp.NewHandler(
			next,
			pathKey,
			otelhttp.WithPropagators(otel.GetTextMapPropagator()),
		)
		pathHandlers.Set(pathKey, newHandler)
		newHandler.ServeHTTP(w, r)
	})
}

// --- gRPC Interceptors ---

// WithAllTracing returns gRPC server options with tracing.
func WithAllTracing() []grpc.ServerOption {
	return []grpc.ServerOption{
		UnaryServerInterceptor(),
		StreamServerInterceptor(),
	}
}

func UnaryServerInterceptor() grpc.ServerOption {
	return grpc.UnaryInterceptor(
		otelgrpc.UnaryServerInterceptor(
			otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		),
	)
}

func StreamServerInterceptor() grpc.ServerOption {
	return grpc.StreamInterceptor(
		otelgrpc.StreamServerInterceptor(
			otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		),
	)
}

// Client-side interceptors

func WithUnaryInterceptor() grpc.DialOption {
	return grpc.WithUnaryInterceptor(
		otelgrpc.UnaryClientInterceptor(
			otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		),
	)
}

func WithStreamInterceptor() grpc.DialOption {
	return grpc.WithStreamInterceptor(
		otelgrpc.StreamClientInterceptor(
			otelgrpc.WithPropagators(otel.GetTextMapPropagator()),
		),
	)
}
