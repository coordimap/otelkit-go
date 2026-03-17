package otelkit

import (
	"testing"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
	env := map[string]string{
		"OTEL_SERVICE_NAME":           "coordimap-api",
		"OTEL_RESOURCE_ATTRIBUTES":    `deployment.environment=dev,service.name=wrong,team=core`,
		"OTEL_EXPORTER_OTLP_ENDPOINT": "https://collector:4318",
		"OTEL_EXPORTER_OTLP_PROTOCOL": "http/protobuf",
		"OTEL_EXPORTER_OTLP_HEADERS":  "x-api-key=secret,tenant=coordimap",
		"OTEL_TRACES_EXPORTER":        "otlp",
		"OTEL_METRICS_EXPORTER":       "none",
		"OTEL_LOGS_EXPORTER":          "otlp",
		"OTEL_PROPAGATORS":            "tracecontext,baggage,b3",
		"OTEL_TRACES_SAMPLER":         "traceidratio",
		"OTEL_TRACES_SAMPLER_ARG":     "0.5",
	}

	cfg, err := LoadConfig(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.ServiceName != "coordimap-api" {
		t.Fatalf("ServiceName = %q", cfg.ServiceName)
	}
	if got := findAttribute(cfg.ResourceAttributes, string(semconv.ServiceNameKey)); got != "coordimap-api" {
		t.Fatalf("service.name = %q", got)
	}
	if got := findAttribute(cfg.ResourceAttributes, "deployment.environment"); got != "dev" {
		t.Fatalf("deployment.environment = %q", got)
	}
	if cfg.OTLPHeaders["x-api-key"] != "secret" || cfg.OTLPHeaders["tenant"] != "coordimap" {
		t.Fatalf("headers = %#v", cfg.OTLPHeaders)
	}
	if cfg.OTLPProtocol != "http/protobuf" {
		t.Fatalf("OTLPProtocol = %q", cfg.OTLPProtocol)
	}
	if len(cfg.Propagators) != 3 {
		t.Fatalf("Propagators = %#v", cfg.Propagators)
	}
}

func TestLoadConfigInvalidResourceAttributes(t *testing.T) {
	t.Parallel()
	_, err := LoadConfig(func(key string) (string, bool) {
		if key == "OTEL_RESOURCE_ATTRIBUTES" {
			return "broken", true
		}
		return "", false
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseResourceAttributesEscaped(t *testing.T) {
	t.Parallel()
	attributes, err := parseResourceAttributes(`key=value\,with\,commas,eq=has\=equals,path=C:\\tmp`)
	if err != nil {
		t.Fatalf("parseResourceAttributes() error = %v", err)
	}
	got := map[string]string{}
	for _, kv := range attributes {
		got[string(kv.Key)] = kv.Value.AsString()
	}
	if got["key"] != "value,with,commas" {
		t.Fatalf("key = %q", got["key"])
	}
	if got["eq"] != "has=equals" {
		t.Fatalf("eq = %q", got["eq"])
	}
	if got["path"] != `C:\tmp` {
		t.Fatalf("path = %q", got["path"])
	}
}

func findAttribute(attributes []attribute.KeyValue, key string) string {
	for _, kv := range attributes {
		if string(kv.Key) == key {
			return kv.Value.AsString()
		}
	}
	return ""
}
