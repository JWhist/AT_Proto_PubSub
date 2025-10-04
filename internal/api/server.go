package api

import (
	"context"
	"fmt"
	"net/http"

	"atp-test/internal/firehose"
)

// Server handles HTTP API requests for filter management
type Server struct {
	firehoseClient *firehose.Client
	server         *http.Server
}

// NewServer creates a new API server instance
func NewServer(firehoseClient *firehose.Client, port string) *Server {
	mux := http.NewServeMux()

	apiServer := &Server{
		firehoseClient: firehoseClient,
		server: &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		},
	}

	// Register API routes
	mux.HandleFunc("/api/filters", apiServer.handleFilters)
	mux.HandleFunc("/api/filters/update", apiServer.handleUpdateFilters)
	mux.HandleFunc("/api/status", apiServer.handleStatus)
	mux.HandleFunc("/", apiServer.handleRoot)

	return apiServer
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
