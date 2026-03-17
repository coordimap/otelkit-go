package main

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/coordimap/otelkit-go/httpotel"
	"github.com/coordimap/otelkit-go/otelkit"
)

func main() {
	tel, err := otelkit.New(context.Background())
	if err != nil {
		log.Fatalf("init telemetry: %v", err)
	}
	defer func() { _ = tel.Shutdown(context.Background()) }()

	client := httpotel.NewClient(nil)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com", nil)
	if err != nil {
		log.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	log.Printf("status=%d bytes=%d", resp.StatusCode, len(body))
}
