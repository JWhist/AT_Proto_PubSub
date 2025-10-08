package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

// corsMiddleware adds CORS headers to HTTP responses
func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if s.config.Server.CORS.AllowAllOrigins {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" {
			for _, allowedOrigin := range s.config.Server.CORS.AllowedOrigins {
				if origin == allowedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		// Set other CORS headers
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.config.Server.CORS.AllowedMethods, ", "))
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.config.Server.CORS.AllowedHeaders, ", "))
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
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

		// Check if the origin is in the allowed origins list
		origin := r.Header.Get("Origin")
		for _, allowedOrigin := range cfg.Server.CORS.AllowedOrigins {
			if origin == allowedOrigin {
				return true
			}
		}

		// If no origin header or not in allowed list, deny
		return false
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

	// Register API routes with CORS middleware
	mux.HandleFunc("/api/filters", apiServer.corsMiddleware(apiServer.handleFilters))
	mux.HandleFunc("/api/filters/update", apiServer.corsMiddleware(apiServer.handleUpdateFilters))
	mux.HandleFunc("/api/filters/create", apiServer.corsMiddleware(apiServer.handleCreateFilter))
	mux.HandleFunc("/api/subscriptions", apiServer.corsMiddleware(apiServer.handleGetSubscriptions))
	mux.HandleFunc("/api/subscriptions/", apiServer.corsMiddleware(apiServer.handleGetSubscription))
	mux.HandleFunc("/api/stats", apiServer.corsMiddleware(apiServer.handleStats))
	mux.HandleFunc("/api/status", apiServer.corsMiddleware(apiServer.handleStatus))
	mux.HandleFunc("/ws/", apiServer.handleWebSocket)
	mux.HandleFunc("/", apiServer.corsMiddleware(apiServer.handleRoot))

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
