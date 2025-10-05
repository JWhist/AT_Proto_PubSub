package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/JWhist/AT_Proto_PubSub/internal/config"
	"github.com/JWhist/AT_Proto_PubSub/internal/firehose"
	"github.com/JWhist/AT_Proto_PubSub/internal/subscription"

	_ "github.com/JWhist/AT_Proto_PubSub/docs" // Import generated docs
)

// Server handles HTTP API requests for filter management
type Server struct {
	firehoseClient *firehose.Client
	subscriptions  *subscription.Manager
	server         *http.Server
	upgrader       websocket.Upgrader
	config         *config.Config
}

// NewServer creates a new API server instance
func NewServer(firehoseClient *firehose.Client, port string) *Server {
	return NewServerWithConfig(firehoseClient, &config.Config{
		Server: config.ServerConfig{
			Port: port,
			// Host left empty to default to binding all interfaces (:port)
		},
	})
}

// NewServerWithConfig creates a new API server instance with configuration
func NewServerWithConfig(firehoseClient *firehose.Client, cfg *config.Config) *Server {
	mux := http.NewServeMux()

	// Configure CORS based on config
	checkOrigin := func(r *http.Request) bool {
		if cfg.Server.CORS.AllowAllOrigins {
			return true
		}
		// TODO: Implement specific origin checking based on cfg.Server.CORS.AllowedOrigins
		return true
	}

	apiServer := &Server{
		firehoseClient: firehoseClient,
		subscriptions:  subscription.NewManagerWithConfig(cfg.Server.MaxConnections),
		server: &http.Server{
			Addr:    cfg.GetListenAddress(),
			Handler: mux,
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: checkOrigin,
		},
		config: cfg,
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

	// Register Swagger UI
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

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
