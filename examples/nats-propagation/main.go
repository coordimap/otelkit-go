package main

import (
	"context"
	"log"

	"github.com/coordimap/otelkit-go/natsotel"
	"github.com/coordimap/otelkit-go/otelkit"
	"github.com/nats-io/nats.go"
)

func main() {
	tel, err := otelkit.New(context.Background())
	if err != nil {
		log.Fatalf("init telemetry: %v", err)
	}
	defer func() { _ = tel.Shutdown(context.Background()) }()

	msg := &nats.Msg{Subject: "coordimap.events", Header: nats.Header{}}
	msg.Header = natsotel.Inject(context.Background(), msg.Header)

	consumerCtx := natsotel.Extract(context.Background(), msg.Header)
	log.Printf("received remote span=%t", consumerCtx != nil)
}
