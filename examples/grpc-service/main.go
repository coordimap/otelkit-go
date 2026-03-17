package main

import (
	"context"
	"log"
	"net"
	"os/signal"
	"syscall"

	"github.com/coordimap/otelkit-go/grpcotel"
	"github.com/coordimap/otelkit-go/otelkit"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

type healthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (healthServer) Check(context.Context, *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tel, err := otelkit.New(ctx)
	if err != nil {
		log.Fatalf("init telemetry: %v", err)
	}
	defer func() { _ = tel.Shutdown(context.Background()) }()

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(grpcotel.UnaryServerInterceptor()))
	grpc_health_v1.RegisterHealthServer(server, healthServer{})
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()
	log.Fatal(server.Serve(lis))
}
