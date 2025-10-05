package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/JWhist/AT_Proto_PubSub/internal/firehose"
	"github.com/JWhist/AT_Proto_PubSub/internal/subscription"
)

// Server handles HTTP API requests for filter management
type Server struct {
	firehoseClient *firehose.Client
	subscriptions  *subscription.Manager
	server         *http.Server
	upgrader       websocket.Upgrader
}

// NewServer creates a new API server instance
func NewServer(firehoseClient *firehose.Client, port string) *Server {
	mux := http.NewServeMux()

	apiServer := &Server{
		firehoseClient: firehoseClient,
		subscriptions:  subscription.NewManager(),
		server: &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	// Register API routes
	mux.HandleFunc("/api/filters", apiServer.handleFilters)
	mux.HandleFunc("/api/filters/update", apiServer.handleUpdateFilters)
	mux.HandleFunc("/api/filters/create", apiServer.handleCreateFilter)
	mux.HandleFunc("/api/subscriptions", apiServer.handleGetSubscriptions)
	mux.HandleFunc("/api/subscriptions/", apiServer.handleGetSubscription)
	mux.HandleFunc("/api/stats", apiServer.handleStats)
	mux.HandleFunc("/api/status", apiServer.handleStatus)
	mux.HandleFunc("/ws/", apiServer.handleWebSocket)
	mux.HandleFunc("/", apiServer.handleRoot)

	return apiServer
}

// GetSubscriptionManager returns the subscription manager for external access
func (s *Server) GetSubscriptionManager() *subscription.Manager {
	return s.subscriptions
}

// Start starts the API server
func (s *Server) Start() error {
	fmt.Printf("Starting API server on %s\n", s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
