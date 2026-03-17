package natsotel

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// HeaderCarrier adapts NATS headers to the OpenTelemetry text map carrier interface.
type HeaderCarrier struct {
	Header nats.Header
}

// Get returns a header value.
func (c HeaderCarrier) Get(key string) string {
	return c.Header.Get(key)
}

// Set sets a header value.
func (c HeaderCarrier) Set(key, value string) {
	if c.Header == nil {
		return
	}
	c.Header.Set(key, value)
}

// Keys returns available header keys.
func (c HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.Header))
	for key := range c.Header {
		keys = append(keys, key)
	}
	return keys
}

// NewHeaderCarrier creates a carrier backed by the provided header map.
func NewHeaderCarrier(header nats.Header) HeaderCarrier {
	if header == nil {
		header = nats.Header{}
	}
	return HeaderCarrier{Header: header}
}

// Inject injects trace context into NATS headers.
func Inject(ctx context.Context, header nats.Header) nats.Header {
	carrier := NewHeaderCarrier(header)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Header
}

// Extract extracts trace context from NATS headers.
func Extract(ctx context.Context, header nats.Header) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, NewHeaderCarrier(header))
}

var _ propagation.TextMapCarrier = HeaderCarrier{}
