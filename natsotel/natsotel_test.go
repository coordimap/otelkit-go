package natsotel

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestInjectExtract(t *testing.T) {
	t.Parallel()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    [16]byte{1},
		SpanID:     [8]byte{2},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}))
	headers := Inject(ctx, nil)
	if headers.Get("traceparent") == "" {
		t.Fatal("expected traceparent header")
	}
	extracted := Extract(context.Background(), headers)
	if !trace.SpanContextFromContext(extracted).IsValid() {
		t.Fatal("expected extracted span context")
	}

	carrier := NewHeaderCarrier(nats.Header{})
	carrier.Set("x-test", "value")
	if carrier.Get("x-test") != "value" {
		t.Fatal("expected header value")
	}
}
