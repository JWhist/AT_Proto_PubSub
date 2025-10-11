package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/JWhist/AT_Proto_PubSub/internal/api"
	"github.com/JWhist/AT_Proto_PubSub/internal/config"
	"github.com/JWhist/AT_Proto_PubSub/internal/firehose"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfigWithDefaults(*configFile)
	if err != nil {
		log.Printf("Failed to load config from %s, using defaults: %v", *configFile, err)
		cfg = config.GetDefaultConfig()
	}

	// Print startup information with config values
	fmt.Println("AT Protocol Firehose Filter Server with WebSocket Subscriptions")
	fmt.Printf("Configuration loaded from: %s\n", *configFile)
	fmt.Printf("Server will start on: %s\n", cfg.GetBaseURL())
	fmt.Println("Use the API endpoints to create filter subscriptions:")
	fmt.Printf("  GET  %s/api/status\n", cfg.GetBaseURL())
	fmt.Printf("  GET  %s/api/subscriptions\n", cfg.GetBaseURL())
	fmt.Printf("  POST %s/api/filters/create\n", cfg.GetBaseURL())
	fmt.Printf("  GET  %s/api/subscriptions/{filterKey}\n", cfg.GetBaseURL())
	fmt.Printf("  GET  %s/api/stats\n", cfg.GetBaseURL())
	fmt.Println("")
	fmt.Println("WebSocket connection:")
	fmt.Printf("  ws://%s:%s/ws/{filterKey}\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println("")
	fmt.Println("API Documentation:")
	fmt.Printf("  %s/swagger/\n", cfg.GetBaseURL())
	fmt.Println()

	// Create firehose client instance with configuration
	firehoseClient := firehose.NewClientWithConfig(cfg)

	// Create API server with configuration
	apiServer := api.NewServerWithConfig(firehoseClient, cfg)

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

	// Start metrics server in a goroutine
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Printf("Starting metrics server on %s:%s\n", cfg.Server.MetricsHost, cfg.Server.MetricsPort)
		if err := http.ListenAndServe(fmt.Sprintf("%s:%s", cfg.Server.MetricsHost, cfg.Server.MetricsPort), nil); err != nil {
			log.Printf("Metrics server error: %v", err)
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

	// Graceful shutdown with configured timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}

	fmt.Println("Server stopped")
}
