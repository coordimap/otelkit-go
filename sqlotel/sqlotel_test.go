package sqlotel

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/noop"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var driverSeq atomic.Uint64

type testDriver struct{}

func (d *testDriver) Open(name string) (driver.Conn, error) {
	return &testConn{}, nil
}

type testConn struct{}

func (c *testConn) Prepare(query string) (driver.Stmt, error) {
	return &testStmt{}, nil
}

func (c *testConn) Close() error {
	return nil
}

func (c *testConn) Begin() (driver.Tx, error) {
	return &testTx{}, nil
}

func (c *testConn) Ping(ctx context.Context) error {
	return nil
}

type testStmt struct{}

func (s *testStmt) Close() error {
	return nil
}

func (s *testStmt) NumInput() int {
	return -1
}

func (s *testStmt) Exec(args []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (s *testStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &testRows{}, nil
}

type testTx struct{}

func (tx *testTx) Commit() error {
	return nil
}

func (tx *testTx) Rollback() error {
	return nil
}

type testRows struct{}

func (r *testRows) Columns() []string {
	return []string{"value"}
}

func (r *testRows) Close() error {
	return nil
}

func (r *testRows) Next(dest []driver.Value) error {
	return io.EOF
}

func registerTestDriver(t *testing.T) string {
	t.Helper()

	name := fmt.Sprintf("sqlotel-test-%d", driverSeq.Add(1))
	sql.Register(name, &testDriver{})
	return name
}

func TestRegister(t *testing.T) {
	t.Parallel()

	base := registerTestDriver(t)
	wrapped, err := Register(base)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if wrapped == "" {
		t.Fatal("Register() returned empty driver name")
	}
	if wrapped == base {
		t.Fatal("Register() returned original driver name")
	}

	db, err := sql.Open(wrapped, "test-dsn")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() error = %v", err)
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()

	base := registerTestDriver(t)
	db, err := Open(base, "test-dsn")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() error = %v", err)
	}
}

func TestHelperOptions(t *testing.T) {
	t.Parallel()

	base := registerTestDriver(t)
	db, err := Open(
		base,
		"test-dsn",
		WithAttributes(),
		WithDBSystem("sqlite"),
		WithDBName("asset_repository"),
		WithServerAddress("localhost:5432"),
		WithTracerProvider(sdktrace.NewTracerProvider()),
		WithMeterProvider(noop.NewMeterProvider()),
		WithSpanNameFormatter(func(_ context.Context, method Method, _ string) string { return string(method) }),
		WithSpanOptions(SpanOptions{OmitRows: true}),
		WithSQLCommenter(false),
		WithAttributesGetter(func(context.Context, Method, string, []driver.NamedValue) []attribute.KeyValue { return nil }),
		WithInstrumentAttributesGetter(func(context.Context, Method, string, []driver.NamedValue) []attribute.KeyValue { return nil }),
		WithInstrumentErrorAttributesGetter(func(error) []attribute.KeyValue { return nil }),
		WithDisableSkipErrMeasurement(true),
		WithReducedSpanNoise(),
		WithSpanFilter(func(context.Context, Method, string, []driver.NamedValue) bool { return true }),
		WithQuerySpanFilter(func(_ context.Context, _ Method, query string) bool { return !strings.Contains(query, "healthcheck") }),
	)
	if err != nil {
		t.Fatalf("Open() with helper options error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() error = %v", err)
	}
}

func TestDefaultOptionsFromEnv(t *testing.T) {
	t.Parallel()

	if got := defaultOptionsFromEnv(func(string) (string, bool) { return "", false }); len(got) != 1 {
		t.Fatalf("defaultOptionsFromEnv() len = %d, want 1", len(got))
	}
	if got := defaultOptionsFromEnv(func(key string) (string, bool) {
		if key == envSpanPreset {
			return "reduced", true
		}
		return "", false
	}); len(got) != 1 {
		t.Fatalf("defaultOptionsFromEnv() len = %d, want 1", len(got))
	}
	if got := defaultOptionsFromEnv(func(key string) (string, bool) {
		if key == envSpanPreset {
			return "none", true
		}
		return "", false
	}); got != nil {
		t.Fatalf("defaultOptionsFromEnv() = %#v, want nil", got)
	}
}

func TestRegisterUsesReducedSpanNoiseByDefault(t *testing.T) {
	t.Parallel()

	got := withRegisterOptions(nil)
	if len(got) == 0 {
		t.Fatal("withRegisterOptions() returned no defaults")
	}
}

func TestDefaultOptionsUnknownPresetFallsBackToReduced(t *testing.T) {
	t.Parallel()

	got := defaultOptionsFromEnv(func(key string) (string, bool) {
		if key == envSpanPreset {
			return "unexpected", true
		}
		return "", false
	})
	if len(got) != 1 {
		t.Fatalf("defaultOptionsFromEnv() len = %d, want 1", len(got))
	}
}

func TestOpenUsesEnvPreset(t *testing.T) {
	t.Setenv(envSpanPreset, "reduced")

	base := registerTestDriver(t)
	db, err := Open(base, "test-dsn")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db.Ping() error = %v", err)
	}
}
