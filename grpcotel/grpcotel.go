package grpcotel

import (
	"context"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	grpcstatus "google.golang.org/grpc/status"
)

const instrumentationScope = "github.com/coordimap/otelkit-go/grpcotel"

// UnaryServerInterceptor returns an OpenTelemetry unary server interceptor.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		ctx = extractIncomingContext(ctx)
		ctx, span := otel.Tracer(instrumentationScope).Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		resp, err := handler(ctx, req)
		recordStatus(span, err)
		return resp, err
	}
}

// StreamServerInterceptor returns an OpenTelemetry stream server interceptor.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := extractIncomingContext(stream.Context())
		ctx, span := otel.Tracer(instrumentationScope).Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		err := handler(srv, &serverStream{ServerStream: stream, ctx: ctx})
		recordStatus(span, err)
		return err
	}
}

// UnaryClientInterceptor returns an OpenTelemetry unary client interceptor.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, span := otel.Tracer(instrumentationScope).Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()

		ctx = injectOutgoingContext(ctx)
		err := invoker(ctx, method, req, reply, cc, opts...)
		recordStatus(span, err)
		return err
	}
}

// StreamClientInterceptor returns an OpenTelemetry stream client interceptor.
func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, span := otel.Tracer(instrumentationScope).Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient))
		ctx = injectOutgoingContext(ctx)

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			recordStatus(span, err)
			span.End()
			return nil, err
		}
		return &clientStream{ClientStream: stream, span: span}, nil
	}
}

// NewServerStatsHandler returns the standard otelgrpc server stats handler.
func NewServerStatsHandler(opts ...otelgrpc.Option) stats.Handler {
	return otelgrpc.NewServerHandler(opts...)
}

// NewClientStatsHandler returns the standard otelgrpc client stats handler.
func NewClientStatsHandler(opts ...otelgrpc.Option) stats.Handler {
	return otelgrpc.NewClientHandler(opts...)
}

type metadataCarrier struct {
	metadata.MD
}

func (c metadataCarrier) Get(key string) string {
	values := c.MD.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c metadataCarrier) Set(key, value string) {
	c.MD.Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.MD))
	for key := range c.MD {
		keys = append(keys, key)
	}
	return keys
}

func extractIncomingContext(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, metadataCarrier{MD: md})
}

func injectOutgoingContext(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.MD{}
	}
	otel.GetTextMapPropagator().Inject(ctx, metadataCarrier{MD: md})
	return metadata.NewOutgoingContext(ctx, md)
}

func recordStatus(span trace.Span, err error) {
	if err == nil {
		span.SetStatus(codes.Ok, "")
		return
	}
	span.RecordError(err)
	status := grpcstatus.Convert(err)
	span.SetAttributes(attribute.String("rpc.grpc.status_code", status.Code().String()))
	span.SetStatus(codes.Error, status.Message())
}

type serverStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *serverStream) Context() context.Context {
	return s.ctx
}

type clientStream struct {
	grpc.ClientStream
	span trace.Span
}

func (s *clientStream) CloseSend() error {
	err := s.ClientStream.CloseSend()
	recordStatus(s.span, err)
	s.span.End()
	return err
}

var _ propagation.TextMapCarrier = metadataCarrier{}
