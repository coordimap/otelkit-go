package main

import (
	"context"
	"log"

	"github.com/coordimap/otelkit-go/otelkit"
	"github.com/coordimap/otelkit-go/redisotel"
	"github.com/redis/go-redis/v9"
)

func main() {
	tel, err := otelkit.New(context.Background())
	if err != nil {
		log.Fatalf("init telemetry: %v", err)
	}
	defer func() { _ = tel.Shutdown(context.Background()) }()

	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	if err := redisotel.InstrumentTracingReducedNoise(client); err != nil {
		log.Fatalf("instrument redis tracing: %v", err)
	}

	if err := client.Set(context.Background(), "coordimap", "otelkit", 0).Err(); err != nil {
		log.Fatalf("set key: %v", err)
	}

	value, err := client.Get(context.Background(), "coordimap").Result()
	if err != nil {
		log.Fatalf("get key: %v", err)
	}
	log.Printf("value=%q", value)
}
