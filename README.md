# Coordimap OpenTelemetry Kit for Go

`otelkit-go` provides a small reusable OpenTelemetry bootstrap layer for Coordimap Go microservices. It keeps to standard OpenTelemetry Go packages, supports standard environment-variable configuration, and adds thin helpers for HTTP, gRPC, NATS, and `database/sql` instrumentation.

## Install

```bash
go get github.com/coordimap/otelkit-go
```

## Packages

- `otelkit`: env-based config, provider bootstrap, shutdown, tracer and meter accessors
- `httpotel`: inbound HTTP middleware and outbound HTTP client helpers
- `grpcotel`: gRPC client and server interceptors
- `natsotel`: NATS header propagation helpers
- `sqlotel`: thin `database/sql` wrappers backed by `github.com/XSAM/otelsql`

## Supported environment variables

- `OTEL_SERVICE_NAME`
- `OTEL_RESOURCE_ATTRIBUTES`
- `OTEL_EXPORTER_OTLP_ENDPOINT`
- `OTEL_EXPORTER_OTLP_PROTOCOL`
- `OTEL_EXPORTER_OTLP_HEADERS`
- `OTEL_TRACES_EXPORTER`
- `OTEL_METRICS_EXPORTER`
- `OTEL_LOGS_EXPORTER`
- `OTEL_PROPAGATORS`
- `OTEL_TRACES_SAMPLER`
- `OTEL_TRACES_SAMPLER_ARG`

Supported exporter values:

- `otlp`
- `none`

Supported OTLP protocols:

- `grpc`
- `http/protobuf`

Supported propagators:

- `tracecontext`
- `baggage`
- `b3`
- `b3multi`
- `jaeger`
- `xray`
- `none`

Supported samplers:

- `always_on`
- `always_off`
- `traceidratio`
- `parentbased_always_on`
- `parentbased_always_off`
- `parentbased_traceidratio`

## Bootstrap API

```go
ctx := context.Background()

tel, err := otelkit.New(ctx)
if err != nil {
    return err
}
defer tel.Shutdown(context.Background())

tracer := tel.Tracer("coordimap.api")
meter := tel.Meter("coordimap.api")

_ = tracer
_ = meter
```

`otelkit.New` loads config from standard OpenTelemetry env vars, initializes trace/metric/log providers, installs them as globals by default, and returns explicit `ForceFlush` and `Shutdown` hooks for `main()`.

## `OTEL_RESOURCE_ATTRIBUTES` behavior

`OTEL_RESOURCE_ATTRIBUTES` is parsed into resource attributes and merged with detected process, host, OS, and telemetry SDK metadata.

- detected resource attributes are loaded first
- env resource attributes override detected values
- `OTEL_SERVICE_NAME` overrides any `service.name` present in `OTEL_RESOURCE_ATTRIBUTES`
- extra resource options passed through `otelkit.WithResourceOptions(...)` are merged last

Example:

```bash
export OTEL_SERVICE_NAME=coordimap-api
export OTEL_RESOURCE_ATTRIBUTES=service.name=legacy-name,deployment.environment=staging,team=platform
```

The effective `service.name` becomes `coordimap-api`.

## Examples

Runnable examples live under `examples/`:

- `examples/http-service`
- `examples/grpc-service`
- `examples/http-client`
- `examples/nats-propagation`

Example OTLP HTTP collector configuration:

```bash
export OTEL_SERVICE_NAME=coordimap-api
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
export OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
go run ./examples/http-service
```

### HTTP service

```go
mux := http.NewServeMux()
mux.Handle("/hello", httpotel.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    _, _ = w.Write([]byte("hello"))
}), "hello"))
```

### gRPC service

```go
server := grpc.NewServer(
    grpc.UnaryInterceptor(grpcotel.UnaryServerInterceptor()),
)
```

### Outbound HTTP client

```go
client := httpotel.NewClient(nil)
req, _ := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL, nil)
resp, err := client.Do(req)
```

### NATS publish and consume propagation

```go
msg := &nats.Msg{Subject: "coordimap.events", Header: nats.Header{}}
msg.Header = natsotel.Inject(ctx, msg.Header)

consumerCtx := natsotel.Extract(context.Background(), msg.Header)
```

### `database/sql` connections

```go
db, err := sqlotel.Open(
	"postgres",
	dsn,
	sqlotel.WithDBSystem("postgresql"),
	sqlotel.WithDBName("asset_repository"),
	sqlotel.WithServerAddress("postgres:5432"),
)
if err != nil {
	return err
}
defer db.Close()
```

## Injecting Coordimap-specific behavior

The package does not hard-code Coordimap resource fields, exporters, or interceptors. Any Coordimap-specific behavior stays optional and injectable:

```go
tel, err := otelkit.New(
    ctx,
    otelkit.WithResourceOptions(resource.WithAttributes(
        attribute.String("coordimap.cluster", clusterName),
        attribute.String("coordimap.region", region),
    )),
)
```

You can also skip global installation with `otelkit.WithoutGlobals()` and wire providers explicitly.

## Migration guidance for existing services

1. Remove service-local tracer provider bootstrap code and OTLP exporter setup.
2. Add `otelkit.New(ctx)` during process startup.
3. Replace direct `otelhttp` or `otelgrpc` imports with `httpotel` and `grpcotel` helpers where convenient.
4. Replace `sql.Open(...)` with `sqlotel.Open(...)` for instrumented `database/sql` connections.
5. Use `tel.Tracer("service-name")` and `tel.Meter("service-name")` instead of building providers manually.
6. Defer `tel.Shutdown(context.Background())` in `main()`.
7. Move collector configuration to env vars.

Minimal startup snippet:

```go
func main() {
    ctx := context.Background()

    tel, err := otelkit.New(ctx)
    if err != nil {
        log.Fatalf("init telemetry: %v", err)
    }
    defer func() {
        _ = tel.Shutdown(context.Background())
    }()

    tracer := tel.Tracer("coordimap.api")
    _ = tracer
}
```

## Adoption guide for existing Coordimap services

1. Set `OTEL_SERVICE_NAME` in each service deployment.
2. Set collector env vars once per service or shared workload template.
3. Replace existing HTTP transport wrapping with `httpotel.NewClient(nil)` or `httpotel.NewTransport(...)`.
4. Replace gRPC interceptor wiring with `grpcotel.UnaryServerInterceptor()` and `grpcotel.UnaryClientInterceptor()`.
5. Inject or extract NATS headers with `natsotel.Inject` and `natsotel.Extract` around publish and consume code.
6. Swap `sql.Open(...)` calls to `sqlotel.Open(...)` anywhere services use `database/sql` directly.

## Tradeoffs and follow-up improvements

- Only standard OTLP exporters are enabled; per-signal protocol overrides are not yet added.
- Sampler support is intentionally limited to the common standard samplers.
- The package initializes log exporters but leaves application log bridge choice to each service.
- A future version could add richer metric reader tuning, per-signal OTLP env overrides, and optional semantic-convention presets for Coordimap services.
