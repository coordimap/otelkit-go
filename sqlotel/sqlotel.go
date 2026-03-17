package sqlotel

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"os"
	"strings"

	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Method = otelsql.Method
type SpanOptions = otelsql.SpanOptions
type SpanNameFormatter = otelsql.SpanNameFormatter
type AttributesGetter = otelsql.AttributesGetter
type InstrumentAttributesGetter = otelsql.InstrumentAttributesGetter
type InstrumentErrorAttributesGetter = otelsql.InstrumentErrorAttributesGetter
type SpanFilter = otelsql.SpanFilter
type Option = otelsql.Option

const envSpanPreset = "OTELKIT_SQLOTEL_SPAN_PRESET"

// Open opens an instrumented database connection.
func Open(driverName, dsn string, opts ...Option) (*sql.DB, error) {
	return otelsql.Open(driverName, dsn, withDefaultOptions(opts)...)
}

// OpenDB opens an instrumented database handle from a connector.
func OpenDB(connector driver.Connector, opts ...Option) *sql.DB {
	return otelsql.OpenDB(connector, withDefaultOptions(opts)...)
}

// Register registers an instrumented driver wrapper and returns its name.
func Register(driverName string, opts ...Option) (string, error) {
	return otelsql.Register(driverName, withRegisterOptions(opts)...)
}

// WithAttributes adds attributes to spans and measurements.
func WithAttributes(attrs ...attribute.KeyValue) Option {
	return otelsql.WithAttributes(attrs...)
}

// WithDBSystem adds the db.system attribute.
func WithDBSystem(system string) Option {
	return otelsql.WithAttributes(attribute.String("db.system", system))
}

// WithDBName adds the db.name attribute.
func WithDBName(name string) Option {
	return otelsql.WithAttributes(attribute.String("db.name", name))
}

// WithServerAddress adds the server.address attribute.
func WithServerAddress(addr string) Option {
	return otelsql.WithAttributes(attribute.String("server.address", addr))
}

// WithTracerProvider configures the tracer provider.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return otelsql.WithTracerProvider(tp)
}

// WithMeterProvider configures the meter provider.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return otelsql.WithMeterProvider(mp)
}

// WithSpanNameFormatter configures span naming for SQL operations.
func WithSpanNameFormatter(formatter SpanNameFormatter) Option {
	return otelsql.WithSpanNameFormatter(formatter)
}

// WithSpanOptions configures which SQL spans and events are emitted.
func WithSpanOptions(opts SpanOptions) Option {
	return otelsql.WithSpanOptions(opts)
}

// WithSQLCommenter enables or disables SQL commenter propagation.
func WithSQLCommenter(enabled bool) Option {
	return otelsql.WithSQLCommenter(enabled)
}

// WithAttributesGetter configures dynamic span attributes.
func WithAttributesGetter(getter AttributesGetter) Option {
	return otelsql.WithAttributesGetter(getter)
}

// WithInstrumentAttributesGetter configures dynamic metric attributes.
func WithInstrumentAttributesGetter(getter InstrumentAttributesGetter) Option {
	return otelsql.WithInstrumentAttributesGetter(getter)
}

// WithInstrumentErrorAttributesGetter configures metric attributes derived from errors.
func WithInstrumentErrorAttributesGetter(getter InstrumentErrorAttributesGetter) Option {
	return otelsql.WithInstrumentErrorAttributesGetter(getter)
}

// WithDisableSkipErrMeasurement controls whether driver.ErrSkip is treated as an error in measurements.
func WithDisableSkipErrMeasurement(disable bool) Option {
	return otelsql.WithDisableSkipErrMeasurement(disable)
}

// WithReducedSpanNoise disables the noisiest low-value SQL spans while preserving query and transaction spans.
func WithReducedSpanNoise() Option {
	return otelsql.WithSpanOptions(SpanOptions{
		OmitRows:             true,
		OmitConnPrepare:      true,
		OmitConnResetSession: true,
		OmitConnectorConnect: true,
		RowsNext:             false,
		Ping:                 false,
		DisableErrSkip:       true,
	})
}

// WithSpanFilter skips span creation when the filter returns false.
func WithSpanFilter(filter SpanFilter) Option {
	return otelsql.WithSpanOptions(SpanOptions{SpanFilter: filter})
}

// WithQuerySpanFilter skips query-bearing spans when the filter returns false.
func WithQuerySpanFilter(filter func(ctx context.Context, method Method, query string) bool) Option {
	return WithSpanFilter(func(ctx context.Context, method Method, query string, _ []driver.NamedValue) bool {
		return filter(ctx, method, query)
	})
}

func withDefaultOptions(opts []Option) []Option {
	defaults := defaultOptionsFromEnv(os.LookupEnv)
	if len(defaults) == 0 {
		return opts
	}
	merged := make([]Option, 0, len(defaults)+len(opts))
	merged = append(merged, defaults...)
	merged = append(merged, opts...)
	return merged
}

func withRegisterOptions(opts []Option) []Option {
	defaults := defaultOptionsFromEnv(os.LookupEnv)
	merged := make([]Option, 0, len(defaults)+len(opts))
	merged = append(merged, defaults...)
	merged = append(merged, opts...)
	return merged
}

func defaultOptionsFromEnv(lookup func(string) (string, bool)) []Option {
	preset := strings.ToLower(strings.TrimSpace(getEnv(lookup, envSpanPreset)))
	switch preset {
	case "", "default", "reduced":
		return []Option{WithReducedSpanNoise()}
	case "none":
		return nil
	default:
		return []Option{WithReducedSpanNoise()}
	}
}

func getEnv(lookup func(string) (string, bool), key string) string {
	value, ok := lookup(key)
	if !ok {
		return ""
	}
	return value
}
