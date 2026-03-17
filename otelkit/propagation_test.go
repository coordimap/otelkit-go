package otelkit

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestConfigPropagator(t *testing.T) {
	t.Parallel()
	cfg := Config{Propagators: []string{"tracecontext", "baggage", "b3"}}
	prop, err := cfg.Propagator()
	if err != nil {
		t.Fatalf("Propagator() error = %v", err)
	}

	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    [16]byte{1},
		SpanID:     [8]byte{2},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}))
	headers := http.Header{}
	prop.Inject(ctx, propagation.HeaderCarrier(headers))
	if headers.Get("traceparent") == "" {
		t.Fatal("expected traceparent header")
	}
	if headers.Get("x-b3-traceid") == "" && headers.Get("b3") == "" {
		t.Fatal("expected B3 header")
	}

	otel.SetTextMapPropagator(prop)
	_ = otel.GetTextMapPropagator()
}
