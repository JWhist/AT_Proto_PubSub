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

	"github.com/JWhist/AT_Proto_PubSub/internal/api"
	"github.com/JWhist/AT_Proto_PubSub/internal/firehose"
)

func main() {
	fmt.Println("AT Protocol Firehose Filter Server with WebSocket Subscriptions")
	fmt.Println("Use the API endpoints to create filter subscriptions:")
	fmt.Println("  GET  http://localhost:8080/api/status")
	fmt.Println("  GET  http://localhost:8080/api/subscriptions")
	fmt.Println("  POST http://localhost:8080/api/filters/create")
	fmt.Println("  GET  http://localhost:8080/api/subscriptions/{filterKey}")
	fmt.Println("  GET  http://localhost:8080/api/stats")
	fmt.Println("")
	fmt.Println("WebSocket connection:")
	fmt.Println("  ws://localhost:8080/ws/{filterKey}")
	fmt.Println("")
	fmt.Println("API Documentation:")
	fmt.Println("  http://localhost:8080/swagger/")
	fmt.Println()

	// Create firehose client instance (starts with no filters)
	firehoseClient := firehose.NewClient()

	// Create API server
	apiServer := api.NewServer(firehoseClient, "8080")

	// Connect firehose events to subscription manager
	firehoseClient.SetEventCallback(apiServer.GetSubscriptionManager().BroadcastEvent)

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
