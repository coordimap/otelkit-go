package httpotel

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// NewHandler instruments an inbound HTTP handler.
func NewHandler(handler http.Handler, operation string, opts ...otelhttp.Option) http.Handler {
	return otelhttp.NewHandler(handler, operation, opts...)
}

// NewTransport instruments an outbound HTTP round tripper.
func NewTransport(base http.RoundTripper, opts ...otelhttp.Option) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base, opts...)
}

// NewClient returns an HTTP client with an instrumented transport.
func NewClient(base *http.Client, opts ...otelhttp.Option) *http.Client {
	if base == nil {
		base = &http.Client{}
	}
	clone := *base
	clone.Transport = NewTransport(base.Transport, opts...)
	return &clone
}

// Inject injects the current context into outbound HTTP headers.
func Inject(ctx context.Context, header http.Header) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(header))
}

// Extract extracts context from inbound HTTP headers.
func Extract(ctx context.Context, header http.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(header))
}
