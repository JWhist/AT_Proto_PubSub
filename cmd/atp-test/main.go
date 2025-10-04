package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atp-test/internal/api"
	"atp-test/internal/firehose"
)

func main() {
	fmt.Println("AT Protocol Firehose Filter Server with API")
	fmt.Println("Use the API endpoints to set filters:")
	fmt.Println("  GET  http://localhost:8080/api/status")
	fmt.Println("  GET  http://localhost:8080/api/filters")
	fmt.Println("  POST http://localhost:8080/api/filters/update")
	fmt.Println()

	// Create firehose client instance (starts with no filters)
	firehoseClient := firehose.NewClient()

	// Create API server
	apiServer := api.NewServer(firehoseClient, "8080")

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start API server in a goroutine
	go func() {
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
			cancel()
		}
	}()

	// Start firehose client in a goroutine
	go func() {
		if err := firehoseClient.Start(ctx); err != nil {
			if err == context.Canceled {
				// Expected shutdown
				return
			}
			log.Printf("Firehose client error: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nReceived shutdown signal...")
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}

	fmt.Println("Server stopped")
}
