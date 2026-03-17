package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/coordimap/otelkit-go/httpotel"
	"github.com/coordimap/otelkit-go/otelkit"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tel, err := otelkit.New(ctx)
	if err != nil {
		log.Fatalf("init telemetry: %v", err)
	}
	defer func() {
		_ = tel.Shutdown(context.Background())
	}()

	mux := http.NewServeMux()
	mux.Handle("/hello", httpotel.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello from coordimap"))
	}), "hello"))

	server := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()
	log.Fatal(server.ListenAndServe())
}
