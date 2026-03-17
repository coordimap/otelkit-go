package redisotel

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestInstrumentTracing(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() {
		_ = client.Close()
	})

	err := InstrumentTracing(
		client,
		WithTracerProvider(trace.NewTracerProvider()),
		WithDBStatement(false),
		WithCallerEnabled(false),
		WithDialFilter(true),
		WithCommandFilter(func(redis.Cmder) bool { return false }),
		WithCommandsFilter(func([]redis.Cmder) bool { return false }),
	)
	if err != nil {
		t.Fatalf("InstrumentTracing() error = %v", err)
	}
}

func TestInstrumentTracingReducedNoise(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() {
		_ = client.Close()
	})

	err := InstrumentTracingReducedNoise(client, WithTracerProvider(trace.NewTracerProvider()))
	if err != nil {
		t.Fatalf("InstrumentTracingReducedNoise() error = %v", err)
	}
}

func TestInstrumentMetrics(t *testing.T) {
	t.Parallel()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	t.Cleanup(func() {
		_ = client.Close()
	})

	closeCh := make(chan struct{})
	t.Cleanup(func() {
		close(closeCh)
	})

	err := InstrumentMetrics(
		client,
		WithMeterProvider(noop.NewMeterProvider()),
		WithCloseChan(closeCh),
	)
	if err != nil {
		t.Fatalf("InstrumentMetrics() error = %v", err)
	}
}

func TestDefaultCommandFilter(t *testing.T) {
	t.Parallel()

	auth := redis.NewCmd(context.Background(), "AUTH", "user", "secret")
	if !DefaultCommandFilter(auth) {
		t.Fatal("DefaultCommandFilter() = false, want true")
	}

	get := redis.NewCmd(context.Background(), "GET", "key")
	if DefaultCommandFilter(get) {
		t.Fatal("DefaultCommandFilter() = true, want false")
	}
}

func TestReducedCommandFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  redis.Cmder
		want bool
	}{
		{name: "auth", cmd: redis.NewCmd(context.Background(), "AUTH", "user", "secret"), want: true},
		{name: "ping", cmd: redis.NewCmd(context.Background(), "PING"), want: true},
		{name: "client", cmd: redis.NewCmd(context.Background(), "CLIENT", "SETINFO", "LIB-NAME", "app"), want: true},
		{name: "get", cmd: redis.NewCmd(context.Background(), "GET", "key"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReducedCommandFilter(tt.cmd); got != tt.want {
				t.Fatalf("ReducedCommandFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReducedCommandsFilter(t *testing.T) {
	t.Parallel()

	noise := []redis.Cmder{
		redis.NewCmd(context.Background(), "PING"),
		redis.NewCmd(context.Background(), "CLIENT", "SETINFO", "LIB-NAME", "app"),
	}
	if !ReducedCommandsFilter(noise) {
		t.Fatal("ReducedCommandsFilter() = false, want true")
	}

	mixed := []redis.Cmder{
		redis.NewCmd(context.Background(), "PING"),
		redis.NewCmd(context.Background(), "GET", "key"),
	}
	if ReducedCommandsFilter(mixed) {
		t.Fatal("ReducedCommandsFilter() = true, want false")
	}
}
