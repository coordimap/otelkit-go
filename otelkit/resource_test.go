package otelkit

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestNewResourceMergesWithServiceNamePrecedence(t *testing.T) {
	t.Parallel()
	cfg := Config{
		ServiceName: "coordimap-worker",
		ResourceAttributes: []attribute.KeyValue{
			attribute.String("service.name", "wrong-name"),
			attribute.String("deployment.environment", "test"),
		},
	}

	res, err := newResource(cfg, resource.WithAttributes(attribute.String("team", "platform")))
	if err != nil {
		t.Fatalf("newResource() error = %v", err)
	}

	if value, ok := res.Set().Value(semconv.ServiceNameKey); !ok || value.AsString() != "coordimap-worker" {
		t.Fatalf("service.name = %q", value.AsString())
	}
	if value, ok := res.Set().Value("deployment.environment"); !ok || value.AsString() != "test" {
		t.Fatalf("deployment.environment = %q", value.AsString())
	}
	if value, ok := res.Set().Value("team"); !ok || value.AsString() != "platform" {
		t.Fatalf("team = %q", value.AsString())
	}
}
