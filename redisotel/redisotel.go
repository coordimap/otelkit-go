package redisotel

import (
	"strings"

	redisuptrace "github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Option = redisuptrace.Option
type TracingOption = redisuptrace.TracingOption
type MetricsOption = redisuptrace.MetricsOption

// InstrumentTracing instruments a go-redis client with OpenTelemetry tracing.
func InstrumentTracing(client redis.UniversalClient, opts ...TracingOption) error {
	return redisuptrace.InstrumentTracing(client, opts...)
}

// InstrumentTracingReducedNoise instruments a go-redis client with lower-noise tracing defaults.
func InstrumentTracingReducedNoise(client redis.UniversalClient, opts ...TracingOption) error {
	defaults := []TracingOption{
		WithDBStatement(false),
		WithCallerEnabled(false),
		WithDialFilter(true),
		WithCommandFilter(ReducedCommandFilter),
		WithCommandsFilter(ReducedCommandsFilter),
	}
	return redisuptrace.InstrumentTracing(client, append(defaults, opts...)...)
}

// InstrumentMetrics instruments a go-redis client with OpenTelemetry metrics.
func InstrumentMetrics(client redis.UniversalClient, opts ...MetricsOption) error {
	return redisuptrace.InstrumentMetrics(client, opts...)
}

// WithAttributes adds attributes to Redis spans and metrics.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return redisuptrace.WithAttributes(attrs...)
}

// WithDBSystem overrides the db.system attribute.
func WithDBSystem(dbSystem string) Option {
	return redisuptrace.WithDBSystem(dbSystem)
}

// WithTracerProvider configures the tracer provider.
func WithTracerProvider(provider trace.TracerProvider) TracingOption {
	return redisuptrace.WithTracerProvider(provider)
}

// WithDBStatement enables or disables raw Redis command capture on spans.
func WithDBStatement(on bool) TracingOption {
	return redisuptrace.WithDBStatement(on)
}

// WithCallerEnabled enables or disables caller metadata on spans.
func WithCallerEnabled(on bool) TracingOption {
	return redisuptrace.WithCallerEnabled(on)
}

// WithCommandFilter filters individual commands from tracing.
func WithCommandFilter(filter func(cmd redis.Cmder) bool) TracingOption {
	return redisuptrace.WithCommandFilter(filter)
}

// WithCommandsFilter filters pipelines from tracing.
func WithCommandsFilter(filter func(cmds []redis.Cmder) bool) TracingOption {
	return redisuptrace.WithCommandsFilter(filter)
}

// WithDialFilter enables or disables dial tracing.
func WithDialFilter(on bool) TracingOption {
	return redisuptrace.WithDialFilter(on)
}

// WithMeterProvider configures the meter provider.
func WithMeterProvider(provider metric.MeterProvider) MetricsOption {
	return redisuptrace.WithMeterProvider(provider)
}

// WithCloseChan configures the shutdown signal for Redis metrics collection.
func WithCloseChan(closeChan chan struct{}) MetricsOption {
	return redisuptrace.WithCloseChan(closeChan)
}

// DefaultCommandFilter filters AUTH-like commands from tracing.
func DefaultCommandFilter(cmd redis.Cmder) bool {
	return redisuptrace.DefaultCommandFilter(cmd)
}

// ReducedCommandFilter filters auth-like and low-value housekeeping commands from tracing.
func ReducedCommandFilter(cmd redis.Cmder) bool {
	if DefaultCommandFilter(cmd) {
		return true
	}

	switch strings.ToLower(cmd.Name()) {
	case "ping", "client", "cluster", "command":
		return true
	default:
		return false
	}
}

// ReducedCommandsFilter skips pipeline spans only when every command is low-value noise.
func ReducedCommandsFilter(cmds []redis.Cmder) bool {
	if len(cmds) == 0 {
		return false
	}
	for _, cmd := range cmds {
		if !ReducedCommandFilter(cmd) {
			return false
		}
	}
	return true
}

// BasicCommandFilter filters AUTH-like commands from tracing.
// Deprecated: use DefaultCommandFilter instead.
func BasicCommandFilter(cmd redis.Cmder) bool {
	return redisuptrace.BasicCommandFilter(cmd)
}
