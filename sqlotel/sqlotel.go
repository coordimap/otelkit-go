package sqlotel

import (
	"database/sql"
	"database/sql/driver"

	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Option = otelsql.Option

// Open opens an instrumented database connection.
func Open(driverName, dsn string, opts ...Option) (*sql.DB, error) {
	return otelsql.Open(driverName, dsn, opts...)
}

// OpenDB opens an instrumented database handle from a connector.
func OpenDB(connector driver.Connector, opts ...Option) *sql.DB {
	return otelsql.OpenDB(connector, opts...)
}

// Register registers an instrumented driver wrapper and returns its name.
func Register(driverName string, opts ...Option) (string, error) {
	return otelsql.Register(driverName, opts...)
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
