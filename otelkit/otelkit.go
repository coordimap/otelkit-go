package otelkit

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	otellog "go.opentelemetry.io/otel/log"
	globalog "go.opentelemetry.io/otel/log/global"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
)

// Option customizes telemetry bootstrap.
type Option func(*options)

type options struct {
	config          *Config
	resourceOptions []resource.Option
	tracerOptions   []sdktrace.TracerProviderOption
	meterOptions    []sdkmetric.Option
	loggerOptions   []sdklog.LoggerProviderOption
	propagator      propagation.TextMapPropagator
	installGlobals  bool
}

// Telemetry holds configured telemetry providers and accessors.
type Telemetry struct {
	config         Config
	resource       *resource.Resource
	propagator     propagation.TextMapPropagator
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	loggerProvider *sdklog.LoggerProvider
	shutdowns      []func(context.Context) error
	flushers       []func(context.Context) error
}

// New initializes telemetry providers from environment-backed configuration.
func New(ctx context.Context, opts ...Option) (*Telemetry, error) {
	loadedConfig, err := LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}

	options := options{config: &loadedConfig, installGlobals: true}
	for _, opt := range opts {
		opt(&options)
	}
	if options.config == nil {
		options.config = &loadedConfig
	}

	res, err := newResource(*options.config, options.resourceOptions...)
	if err != nil {
		return nil, err
	}

	propagator := options.propagator
	if propagator == nil {
		propagator, err = options.config.Propagator()
		if err != nil {
			return nil, err
		}
	}

	tel := &Telemetry{config: *options.config, resource: res, propagator: propagator}

	if tel.tracerProvider, err = buildTracerProvider(ctx, *options.config, res, options.tracerOptions...); err != nil {
		return nil, err
	}
	if tel.tracerProvider != nil {
		tel.shutdowns = append(tel.shutdowns, tel.tracerProvider.Shutdown)
		tel.flushers = append(tel.flushers, tel.tracerProvider.ForceFlush)
	}

	if tel.meterProvider, err = buildMeterProvider(ctx, *options.config, res, options.meterOptions...); err != nil {
		return nil, err
	}
	if tel.meterProvider != nil {
		tel.shutdowns = append(tel.shutdowns, tel.meterProvider.Shutdown)
		tel.flushers = append(tel.flushers, tel.meterProvider.ForceFlush)
	}

	if tel.loggerProvider, err = buildLoggerProvider(ctx, *options.config, res, options.loggerOptions...); err != nil {
		return nil, err
	}
	if tel.loggerProvider != nil {
		tel.shutdowns = append(tel.shutdowns, tel.loggerProvider.Shutdown)
		tel.flushers = append(tel.flushers, tel.loggerProvider.ForceFlush)
	}

	if options.installGlobals {
		if tel.tracerProvider != nil {
			otel.SetTracerProvider(tel.tracerProvider)
		}
		if tel.meterProvider != nil {
			otel.SetMeterProvider(tel.meterProvider)
		}
		otel.SetTextMapPropagator(tel.propagator)
		if tel.loggerProvider != nil {
			globalog.SetLoggerProvider(tel.loggerProvider)
		}
	}

	return tel, nil
}

// WithConfig overrides environment-loaded configuration.
func WithConfig(cfg Config) Option {
	return func(o *options) {
		o.config = &cfg
	}
}

// WithResourceOptions appends resource options merged after environment attributes.
func WithResourceOptions(opts ...resource.Option) Option {
	return func(o *options) {
		o.resourceOptions = append(o.resourceOptions, opts...)
	}
}

// WithTracerProviderOptions appends tracer provider options.
func WithTracerProviderOptions(opts ...sdktrace.TracerProviderOption) Option {
	return func(o *options) {
		o.tracerOptions = append(o.tracerOptions, opts...)
	}
}

// WithMeterProviderOptions appends meter provider options.
func WithMeterProviderOptions(opts ...sdkmetric.Option) Option {
	return func(o *options) {
		o.meterOptions = append(o.meterOptions, opts...)
	}
}

// WithLoggerProviderOptions appends logger provider options.
func WithLoggerProviderOptions(opts ...sdklog.LoggerProviderOption) Option {
	return func(o *options) {
		o.loggerOptions = append(o.loggerOptions, opts...)
	}
}

// WithPropagator injects a custom propagator.
func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(o *options) {
		o.propagator = propagator
	}
}

// WithoutGlobals skips global provider installation.
func WithoutGlobals() Option {
	return func(o *options) {
		o.installGlobals = false
	}
}

// Config returns the loaded configuration.
func (t *Telemetry) Config() Config {
	return t.config
}

// Resource returns the merged resource.
func (t *Telemetry) Resource() *resource.Resource {
	return t.resource
}

// Propagator returns the configured text map propagator.
func (t *Telemetry) Propagator() propagation.TextMapPropagator {
	return t.propagator
}

// TracerProvider returns the configured tracer provider.
func (t *Telemetry) TracerProvider() trace.TracerProvider {
	if t.tracerProvider == nil {
		return otel.GetTracerProvider()
	}
	return t.tracerProvider
}

// MeterProvider returns the configured meter provider.
func (t *Telemetry) MeterProvider() metricapi.MeterProvider {
	if t.meterProvider == nil {
		return otel.GetMeterProvider()
	}
	return t.meterProvider
}

// LoggerProvider returns the configured logger provider.
func (t *Telemetry) LoggerProvider() otellog.LoggerProvider {
	if t.loggerProvider == nil {
		return globalog.GetLoggerProvider()
	}
	return t.loggerProvider
}

// Tracer returns a named tracer using the configured provider.
func (t *Telemetry) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return t.TracerProvider().Tracer(name, opts...)
}

// Meter returns a named meter using the configured provider.
func (t *Telemetry) Meter(name string, opts ...metricapi.MeterOption) metricapi.Meter {
	return t.MeterProvider().Meter(name, opts...)
}

// Logger returns a named logger using the configured provider.
func (t *Telemetry) Logger(name string, opts ...otellog.LoggerOption) otellog.Logger {
	return t.LoggerProvider().Logger(name, opts...)
}

// ForceFlush flushes all configured providers.
func (t *Telemetry) ForceFlush(ctx context.Context) error {
	var errs []error
	for _, flush := range t.flushers {
		if err := flush(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Shutdown flushes and stops all configured providers.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	var errs []error
	for i := len(t.shutdowns) - 1; i >= 0; i-- {
		if err := t.shutdowns[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func buildTracerProvider(ctx context.Context, cfg Config, res *resource.Resource, options ...sdktrace.TracerProviderOption) (*sdktrace.TracerProvider, error) {
	if disabled(cfg.TracesExporter) {
		return sdktrace.NewTracerProvider(append([]sdktrace.TracerProviderOption{sdktrace.WithResource(res)}, options...)...), nil
	}
	if !strings.EqualFold(cfg.TracesExporter, defaultSignal) {
		return nil, fmt.Errorf("unsupported OTEL_TRACES_EXPORTER %q", cfg.TracesExporter)
	}

	sampler, err := cfg.SamplerConfig()
	if err != nil {
		return nil, err
	}

	spanExporter, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	providerOptions := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(spanExporter),
	}
	providerOptions = append(providerOptions, options...)
	return sdktrace.NewTracerProvider(providerOptions...), nil
}

func buildMeterProvider(ctx context.Context, cfg Config, res *resource.Resource, options ...sdkmetric.Option) (*sdkmetric.MeterProvider, error) {
	providerOptions := []sdkmetric.Option{sdkmetric.WithResource(res)}
	if !disabled(cfg.MetricsExporter) {
		if !strings.EqualFold(cfg.MetricsExporter, defaultSignal) {
			return nil, fmt.Errorf("unsupported OTEL_METRICS_EXPORTER %q", cfg.MetricsExporter)
		}
		metricExporter, err := newMetricExporter(ctx, cfg)
		if err != nil {
			return nil, err
		}
		providerOptions = append(providerOptions, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)))
	}
	providerOptions = append(providerOptions, options...)
	return sdkmetric.NewMeterProvider(providerOptions...), nil
}

func buildLoggerProvider(ctx context.Context, cfg Config, res *resource.Resource, options ...sdklog.LoggerProviderOption) (*sdklog.LoggerProvider, error) {
	providerOptions := []sdklog.LoggerProviderOption{sdklog.WithResource(res)}
	if !disabled(cfg.LogsExporter) {
		if !strings.EqualFold(cfg.LogsExporter, defaultSignal) {
			return nil, fmt.Errorf("unsupported OTEL_LOGS_EXPORTER %q", cfg.LogsExporter)
		}
		logExporter, err := newLogExporter(ctx, cfg)
		if err != nil {
			return nil, err
		}
		providerOptions = append(providerOptions, sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)))
	}
	providerOptions = append(providerOptions, options...)
	return sdklog.NewLoggerProvider(providerOptions...), nil
}

func newTraceExporter(ctx context.Context, cfg Config) (sdktrace.SpanExporter, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.OTLPProtocol)) {
	case "grpc", "":
		return otlptracegrpc.New(ctx)
	case "http/protobuf":
		return otlptracehttp.New(ctx)
	default:
		return nil, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL %q", cfg.OTLPProtocol)
	}
}

func newMetricExporter(ctx context.Context, cfg Config) (sdkmetric.Exporter, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.OTLPProtocol)) {
	case "grpc", "":
		return otlpmetricgrpc.New(ctx)
	case "http/protobuf":
		return otlpmetrichttp.New(ctx)
	default:
		return nil, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL %q", cfg.OTLPProtocol)
	}
}

func newLogExporter(ctx context.Context, cfg Config) (sdklog.Exporter, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.OTLPProtocol)) {
	case "grpc", "":
		return otlploggrpc.New(ctx)
	case "http/protobuf":
		return otlploghttp.New(ctx)
	default:
		return nil, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL %q", cfg.OTLPProtocol)
	}
}

func disabled(exporter string) bool {
	return strings.EqualFold(strings.TrimSpace(exporter), "none")
}
