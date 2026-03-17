package otelkit

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func newResource(cfg Config, options ...resource.Option) (*resource.Resource, error) {
	envAttributes := append([]attribute.KeyValue{}, cfg.ResourceAttributes...)
	for _, kv := range serviceNameAttribute(cfg.ServiceName) {
		envAttributes = upsertAttribute(envAttributes, kv)
	}

	detected, err := resource.New(
		context.Background(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("detect resource: %w", err)
	}

	envResource := resource.NewSchemaless(envAttributes...)
	merged, err := resource.Merge(detected, envResource)
	if err != nil {
		return nil, fmt.Errorf("merge environment resource: %w", err)
	}

	if len(options) == 0 {
		return merged, nil
	}

	additional, err := resource.New(context.Background(), options...)
	if err != nil {
		return nil, fmt.Errorf("build resource options: %w", err)
	}

	merged, err = resource.Merge(merged, additional)
	if err != nil {
		return nil, fmt.Errorf("merge resource options: %w", err)
	}
	return merged, nil
}

func serviceNameAttribute(serviceName string) []attribute.KeyValue {
	if serviceName == "" {
		return nil
	}
	return []attribute.KeyValue{semconv.ServiceName(serviceName)}
}
