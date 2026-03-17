package grpcotel

import (
	"context"
	"net"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type testHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	seenTraceparent bool
}

func (s *testHealthServer) Check(ctx context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	s.seenTraceparent = len(md.Get("traceparent")) > 0
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func TestUnaryInterceptorsPropagateContext(t *testing.T) {
	t.Parallel()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	lis := bufconn.Listen(1024 * 1024)
	serverImpl := &testHealthServer{}
	server := grpc.NewServer(grpc.UnaryInterceptor(UnaryServerInterceptor()))
	grpc_health_v1.RegisterHealthServer(server, serverImpl)
	go func() {
		_ = server.Serve(lis)
	}()
	defer server.Stop()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(UnaryClientInterceptor()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	ctx, span := sdktrace.NewTracerProvider().Tracer("test").Start(context.Background(), "client")
	defer span.End()
	if _, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{}); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !serverImpl.seenTraceparent {
		t.Fatal("expected traceparent metadata")
	}
}
