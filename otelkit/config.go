package otelkit

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	defaultPropagators = "tracecontext,baggage"
	defaultSignal      = "otlp"
	defaultProtocol    = "grpc"
)

// Config holds telemetry configuration loaded from environment variables.
type Config struct {
	ServiceName        string
	ResourceAttributes []attribute.KeyValue
	OTLPEndpoint       string
	OTLPProtocol       string
	OTLPHeaders        map[string]string
	TracesExporter     string
	MetricsExporter    string
	LogsExporter       string
	Propagators        []string
	Sampler            string
	SamplerArg         string
}

// LoadConfigFromEnv loads OpenTelemetry configuration from standard environment variables.
func LoadConfigFromEnv() (Config, error) {
	return LoadConfig(os.LookupEnv)
}

// LoadConfig loads OpenTelemetry configuration using the provided environment lookup function.
func LoadConfig(lookup func(string) (string, bool)) (Config, error) {
	var cfg Config

	cfg.ServiceName = getEnv(lookup, "OTEL_SERVICE_NAME")
	resourceAttributes, err := parseResourceAttributes(getEnv(lookup, "OTEL_RESOURCE_ATTRIBUTES"))
	if err != nil {
		return Config{}, fmt.Errorf("parse OTEL_RESOURCE_ATTRIBUTES: %w", err)
	}
	if cfg.ServiceName != "" {
		resourceAttributes = upsertAttribute(resourceAttributes, semconv.ServiceName(cfg.ServiceName))
	}
	cfg.ResourceAttributes = resourceAttributes
	cfg.OTLPEndpoint = getEnv(lookup, "OTEL_EXPORTER_OTLP_ENDPOINT")
	cfg.OTLPProtocol = firstNonEmpty(getEnv(lookup, "OTEL_EXPORTER_OTLP_PROTOCOL"), defaultProtocol)
	headers, err := parseHeaders(getEnv(lookup, "OTEL_EXPORTER_OTLP_HEADERS"))
	if err != nil {
		return Config{}, fmt.Errorf("parse OTEL_EXPORTER_OTLP_HEADERS: %w", err)
	}
	cfg.OTLPHeaders = headers
	cfg.TracesExporter = firstNonEmpty(getEnv(lookup, "OTEL_TRACES_EXPORTER"), defaultSignal)
	cfg.MetricsExporter = firstNonEmpty(getEnv(lookup, "OTEL_METRICS_EXPORTER"), defaultSignal)
	cfg.LogsExporter = firstNonEmpty(getEnv(lookup, "OTEL_LOGS_EXPORTER"), defaultSignal)
	cfg.Propagators = splitCSV(firstNonEmpty(getEnv(lookup, "OTEL_PROPAGATORS"), defaultPropagators))
	cfg.Sampler = firstNonEmpty(getEnv(lookup, "OTEL_TRACES_SAMPLER"), "parentbased_always_on")
	cfg.SamplerArg = getEnv(lookup, "OTEL_TRACES_SAMPLER_ARG")

	return cfg, nil
}

// Resource returns the configured resource merged with standard resource detection.
func (c Config) Resource() (*resource.Resource, error) {
	return newResource(c)
}

// Propagator returns the configured text map propagator.
func (c Config) Propagator() (propagation.TextMapPropagator, error) {
	return newPropagator(c.Propagators)
}

// SamplerConfig returns the configured trace sampler.
func (c Config) SamplerConfig() (trace.Sampler, error) {
	return newSampler(c.Sampler, c.SamplerArg)
}

func parseHeaders(value string) (map[string]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	headers := make(map[string]string)
	for _, part := range splitCSV(value) {
		key, rawValue, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("invalid header %q", part)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid header %q", part)
		}
		headers[key] = strings.TrimSpace(rawValue)
	}
	return headers, nil
}

func parseResourceAttributes(value string) ([]attribute.KeyValue, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}

	parts, err := splitEscapedList(value)
	if err != nil {
		return nil, err
	}
	attributes := make([]attribute.KeyValue, 0, len(parts))
	for _, part := range parts {
		key, rawValue, ok := splitEscapedPair(part)
		if !ok {
			return nil, fmt.Errorf("invalid resource attribute %q", part)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("invalid resource attribute %q", part)
		}
		attributes = upsertAttribute(attributes, attribute.String(key, rawValue))
	}
	return attributes, nil
}

func splitEscapedList(value string) ([]string, error) {
	var parts []string
	var current strings.Builder
	escaped := false
	for _, r := range value {
		switch {
		case escaped:
			if r == ',' || r == '=' || r == '\\' {
				current.WriteRune(r)
			} else {
				current.WriteRune('\\')
				current.WriteRune(r)
			}
			escaped = false
		case r == '\\':
			escaped = true
		case r == ',':
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		return nil, fmt.Errorf("unterminated escape sequence")
	}
	parts = append(parts, current.String())
	return parts, nil
}

func splitEscapedPair(value string) (string, string, bool) {
	var key strings.Builder
	var rawValue strings.Builder
	escaped := false
	seenSeparator := false
	writeRune := func(r rune) {
		if seenSeparator {
			rawValue.WriteRune(r)
		} else {
			key.WriteRune(r)
		}
	}
	for _, r := range value {
		switch {
		case escaped:
			if r == ',' || r == '=' || r == '\\' {
				writeRune(r)
			} else {
				writeRune('\\')
				writeRune(r)
			}
			escaped = false
		case r == '\\':
			escaped = true
		case r == '=' && !seenSeparator:
			seenSeparator = true
		default:
			writeRune(r)
		}
	}
	if escaped {
		return "", "", false
	}
	if !seenSeparator {
		return "", "", false
	}
	return key.String(), rawValue.String(), true
}

func newSampler(name, arg string) (trace.Sampler, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "always_on":
		return trace.AlwaysSample(), nil
	case "always_off":
		return trace.NeverSample(), nil
	case "traceidratio":
		ratio, err := parseSamplerArg(arg)
		if err != nil {
			return nil, err
		}
		return trace.TraceIDRatioBased(ratio), nil
	case "parentbased_always_on", "":
		return trace.ParentBased(trace.AlwaysSample()), nil
	case "parentbased_always_off":
		return trace.ParentBased(trace.NeverSample()), nil
	case "parentbased_traceidratio":
		ratio, err := parseSamplerArg(arg)
		if err != nil {
			return nil, err
		}
		return trace.ParentBased(trace.TraceIDRatioBased(ratio)), nil
	default:
		return nil, fmt.Errorf("unsupported OTEL_TRACES_SAMPLER %q", name)
	}
}

func parseSamplerArg(value string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 1, nil
	}
	ratio, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("invalid sampler arg %q: %w", value, err)
	}
	if ratio < 0 || ratio > 1 {
		return 0, fmt.Errorf("sampler arg must be between 0 and 1")
	}
	return ratio, nil
}

func getEnv(lookup func(string) (string, bool), key string) string {
	value, ok := lookup(key)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func upsertAttribute(attributes []attribute.KeyValue, kv attribute.KeyValue) []attribute.KeyValue {
	for i := range attributes {
		if attributes[i].Key == kv.Key {
			attributes[i] = kv
			return attributes
		}
	}
	return append(attributes, kv)
}
