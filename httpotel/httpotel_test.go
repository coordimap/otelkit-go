package httpotel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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
	headers := http.Header{}
	Inject(ctx, headers)
	if headers.Get("traceparent") == "" {
		t.Fatal("expected traceparent header")
	}
	extracted := Extract(context.Background(), headers)
	if !trace.SpanContextFromContext(extracted).IsValid() {
		t.Fatal("expected extracted span context")
	}
}

func TestNewTransportInjectsHeaders(t *testing.T) {
	t.Parallel()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("traceparent") == "" {
			t.Error("expected traceparent header")
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	client := NewClient(nil)
	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    [16]byte{3},
		SpanID:     [8]byte{4},
		TraceFlags: trace.FlagsSampled,
	}))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext() error = %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d", resp.StatusCode)
	}
}
