package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/JWhist/AT_Proto_PubSub/internal/models"
)

// @title AT Protocol PubSub API
// @version 1.0.0
// @description A real-time AT Protocol firehose filtering and subscription service.
// @description
// @description ## Overview
// @description This API provides filtering and subscription capabilities for the AT Protocol firehose, allowing clients to:
// @description - Create filtered subscriptions for specific repositories, content types, or keywords (comma-separated)
// @description - Subscribe to real-time events via WebSocket connections
// @description - Monitor subscription statistics and health
// @description
// @description ## Safety Features
// @description - **Filter Validation**: All filters must specify at least one criteria to prevent forwarding the entire firehose
// @description - **Enhanced Timestamps**: All forwarded events include detailed timing metadata for observability
// @description - **Thread Safety**: All operations are thread-safe and tested with race condition detection
// @description
// @description ## WebSocket Protocol
// @description Connect to `/ws/{filterKey}` to receive real-time filtered events with ping/pong support.

// @contact.name AT Protocol PubSub
// @contact.url https://github.com/JWhist/AT_Proto_PubSub

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

// @tag.name Health
// @tag.description Server health and status endpoints

// @tag.name Filters
// @tag.description Filter configuration and management

// @tag.name Subscriptions
// @tag.description Subscription management and statistics

// @tag.name WebSocket
// @tag.description Real-time WebSocket connections

// handleRoot provides basic information about the API
// @Summary API Information
// @Description Get basic information about the API and available endpoints
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "API information retrieved successfully"
// @Router / [get]
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := models.APIResponse{
		Success: true,
		Message: "AT Protocol Firehose Filter Server API",
		Data: map[string]interface{}{
			"endpoints": []string{
				"GET /api/status - Get server status",
				"GET /api/filters - Get current filters",
				"POST /api/filters/create - Create new filter subscription",
				"GET /api/subscriptions/{filterKey} - Get subscription details",
				"GET /api/stats - Get subscription statistics",
			},
			"filters": map[string]string{
				"repository": "Filter by repository DID (e.g., 'did:plc:abc123')",
				"pathPrefix": "Filter by operation path prefix (e.g., 'app.bsky.feed.post')",
				"keyword":    "Filter by keywords in text content (comma-separated, e.g., 'hello,world,test')",
			},
			"requirements": []string{
				"Keyword filter is required for all subscriptions",
				"Each filter field (repository, pathPrefix, keyword) must contain at least 3 letters",
				"Keywords are comma-separated and each must have at least 3 letters",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleStatus returns the current server status
// @Summary Server Status
// @Description Get the current server status and active filters
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Server status retrieved successfully"
// @Router /api/status [get]
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filters := s.firehoseClient.GetFilters()

	response := models.APIResponse{
		Success: true,
		Message: "Server is running",
		Data: map[string]interface{}{
			"status":  "active",
			"filters": filters,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleFilters returns the current filter settings
// @Summary Get Current Filters
// @Description Retrieve the current global filter settings
// @Tags Filters
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Current filters retrieved successfully"
// @Router /api/filters [get]
func (s *Server) handleFilters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filters := s.firehoseClient.GetFilters()

	response := models.APIResponse{
		Success: true,
		Message: "Current filter settings",
		Data:    filters,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleUpdateFilters updates the filter settings
// @Summary Update Global Filters
// @Description Update the global filter settings (legacy endpoint)
// @Tags Filters
// @Accept json
// @Produce json
// @Param request body models.FilterUpdateRequest true "Filter update request"
// @Success 200 {object} models.APIResponse "Filters updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request body"
// @Router /api/filters/update [post]
func (s *Server) handleUpdateFilters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.FilterUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := models.APIResponse{
			Success: false,
			Message: "Invalid JSON in request body: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
		}
		return
	}

	// Get current filters
	currentFilters := s.firehoseClient.GetFilters()

	// Update only the provided fields
	if req.Repository != nil {
		currentFilters.Repository = *req.Repository
	}
	if req.PathPrefix != nil {
		currentFilters.PathPrefix = *req.PathPrefix
	}
	if req.Keyword != nil {
		currentFilters.Keyword = *req.Keyword
	}

	// Apply the updated filters
	s.firehoseClient.UpdateFilters(currentFilters)

	fmt.Printf("Filters updated via API: Repository=%s, PathPrefix=%s, Keyword=%s\n",
		getFilterString(currentFilters.Repository),
		getFilterString(currentFilters.PathPrefix),
		getFilterString(currentFilters.Keyword))

	response := models.APIResponse{
		Success: true,
		Message: "Filters updated successfully",
		Data:    currentFilters,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// getFilterString returns "ALL" if filter is empty, otherwise returns the filter value
func getFilterString(filter string) string {
	if filter == "" {
		return "ALL"
	}
	return filter
}

// handleCreateFilter creates a new filter subscription and returns a filter key
// @Summary Create Filter Subscription
// @Description Create a new filter subscription for receiving real-time events. Keyword filter is required and must contain at least 3 letters to prevent forwarding the entire firehose.
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param request body models.CreateFilterRequest true "Filter creation request"
// @Success 200 {object} models.CreateFilterResponse "Filter subscription created successfully"
// @Failure 400 {object} models.APIResponse "Invalid request - keyword filter required or insufficient letters"
// @Router /api/filters/create [post]
func (s *Server) handleCreateFilter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateFilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := models.APIResponse{
			Success: false,
			Message: "Invalid JSON in request body: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
		}
		return
	}

	// Validate that keyword filter is always provided
	if req.Options.Keyword == "" {
		response := models.APIResponse{
			Success: false,
			Message: "Keyword filter is required. Filters must include keywords to prevent forwarding the entire firehose.",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
		}
		return
	}

	// Validate filter content - each non-empty field must contain at least 3 letters
	if validationErr := validateFilterContent(req.Options); validationErr != "" {
		response := models.APIResponse{
			Success: false,
			Message: validationErr,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
		}
		return
	}

	filterKey := s.subscriptions.CreateFilter(req.Options)
	if filterKey == "" {
		response := models.APIResponse{
			Success: false,
			Message: "Failed to create filter - no criteria provided",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(response); encErr != nil {
			http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
		}
		return
	}

	response := models.CreateFilterResponse{
		FilterKey: filterKey,
		Options:   req.Options,
		CreatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleGetSubscriptions returns all filter subscriptions
// @Summary Get All Subscriptions
// @Description Retrieve all active filter subscriptions
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Subscriptions retrieved successfully"
// @Router /api/subscriptions [get]
func (s *Server) handleGetSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subscriptions := s.subscriptions.GetSubscriptions()

	response := models.APIResponse{
		Success: true,
		Message: "Filter subscriptions retrieved successfully",
		Data:    subscriptions,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleGetSubscription returns a specific filter subscription
// @Summary Get Subscription Details
// @Description Get detailed information about a specific filter subscription
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Param filterKey path string true "The unique filter key for the subscription"
// @Success 200 {object} models.APIResponse "Subscription details retrieved successfully"
// @Failure 404 {object} models.APIResponse "Subscription not found"
// @Router /api/subscriptions/{filterKey} [get]
func (s *Server) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract filter key from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/subscriptions/")
	if path == "" {
		http.Error(w, "Filter key required", http.StatusBadRequest)
		return
	}

	subscription, exists := s.subscriptions.GetSubscription(path)

	var response models.APIResponse
	if exists {
		response = models.APIResponse{
			Success: true,
			Message: "Filter subscription retrieved successfully",
			Data:    subscription,
		}
	} else {
		response = models.APIResponse{
			Success: false,
			Message: "Filter subscription not found",
		}
		w.WriteHeader(http.StatusNotFound)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleStats returns subscription manager statistics
// @Summary Get Statistics
// @Description Get subscription manager statistics and metrics
// @Tags Subscriptions
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Statistics retrieved successfully"
// @Router /api/stats [get]
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.subscriptions.GetStats()

	response := models.APIResponse{
		Success: true,
		Message: "Statistics retrieved successfully",
		Data:    stats,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// handleWebSocket handles WebSocket upgrade and message routing
// @Summary WebSocket Connection
// @Description Establish a WebSocket connection to receive real-time filtered events. Connect to /ws/{filterKey} with the filter key obtained from creating a subscription.
// @Tags WebSocket
// @Param filterKey path string true "The unique filter key obtained from creating a subscription"
// @Success 101 "WebSocket connection established"
// @Failure 400 "Filter key required or invalid"
// @Failure 404 "Invalid filter key"
// @Router /ws/{filterKey} [get]
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract filter key from URL path
	path := strings.TrimPrefix(r.URL.Path, "/ws/")
	if path == "" {
		http.Error(w, "Filter key required", http.StatusBadRequest)
		return
	}

	// Upgrade the HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Set connection timeouts and limits
	const (
		writeWait      = 30 * time.Second    // Time allowed to write a message (increased for better reliability)
		pongWait       = 60 * time.Second    // Time allowed to read the next pong message
		pingPeriod     = (pongWait * 9) / 10 // Send pings to peer with this period (must be less than pongWait)
		maxMessageSize = 512                 // Maximum message size allowed
	)

	// Configure connection
	conn.SetReadLimit(maxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
	}
	conn.SetPongHandler(func(string) error {
		if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			log.Printf("Failed to set read deadline in pong handler: %v", err)
		}
		return nil
	})

	// Add connection to the subscription
	result := s.subscriptions.AddConnectionWithResult(path, conn)
	if !result.Success {
		errorData := map[string]string{
			"error":     result.ErrorMessage,
			"errorCode": result.ErrorCode,
			"filterKey": path,
		}

		errorMsg := models.WSMessage{
			Type:      "error",
			Timestamp: time.Now(),
			Data:      errorData,
		}
		if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
			log.Printf("Failed to set write deadline for error message: %v", err)
		}
		if err := conn.WriteJSON(errorMsg); err != nil {
			log.Printf("Failed to write error message: %v", err)
		}
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
		return
	}

	// Send welcome message
	welcomeMsg := models.WSMessage{
		Type:      "connected",
		Timestamp: time.Now(),
		Data: map[string]string{
			"filterKey": path,
			"status":    "connected",
			"message":   "Successfully connected to filter subscription",
		},
	}
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		log.Printf("Failed to set write deadline for welcome message: %v", err)
	}
	if err := conn.WriteJSON(welcomeMsg); err != nil {
		log.Printf("Failed to send welcome message: %v", err)
	}

	log.Printf("🔌 WebSocket connected for filter %s", path[:8]+"...")

	// Handle connection lifecycle with proper cleanup
	defer func() {
		s.subscriptions.RemoveConnection(path, conn)
		if err := conn.Close(); err != nil && !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
			log.Printf("Error closing connection: %v", err)
		}
		log.Printf("🔌 WebSocket disconnected for filter %s", path[:8]+"...")
	}()

	// Start ping ticker to keep connection alive
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	// Channel to signal when read goroutine should stop
	done := make(chan struct{})
	defer close(done)

	// Start a goroutine to handle reading messages
	go func() {
		defer func() {
			select {
			case done <- struct{}{}:
			default:
			}
		}()

		for {
			var msg map[string]interface{}
			if err := conn.ReadJSON(&msg); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("WebSocket unexpected close: %v", err)
				}
				return
			}

			// Handle client messages
			if msgType, ok := msg["type"].(string); ok {
				switch msgType {
				case "ping":
					pongMsg := models.WSMessage{
						Type:      "pong",
						Timestamp: time.Now(),
						Data:      map[string]string{"status": "alive"},
					}
					if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
						log.Printf("Failed to set write deadline for pong: %v", err)
					}
					if err := conn.WriteJSON(pongMsg); err != nil {
						log.Printf("Failed to send pong: %v", err)
						return
					}
				case "get_filter":
					// Send current filter configuration
					subscription, exists := s.subscriptions.GetSubscription(path)
					if exists {
						filterMsg := models.WSMessage{
							Type:      "filter_info",
							Timestamp: time.Now(),
							Data:      subscription,
						}
						if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
							log.Printf("Failed to set write deadline for filter info: %v", err)
						}
						if err := conn.WriteJSON(filterMsg); err != nil {
							log.Printf("Failed to send filter info: %v", err)
							return
						}
					}
				default:
					// Echo unknown messages back
					echoMsg := models.WSMessage{
						Type:      "echo",
						Timestamp: time.Now(),
						Data:      msg,
					}
					if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
						log.Printf("Failed to set write deadline for echo message: %v", err)
					}
					if err := conn.WriteJSON(echoMsg); err != nil {
						log.Printf("Failed to echo message: %v", err)
						return
					}
				}
			}
		}
	}()

	// Handle pings and connection management
	for {
		select {
		case <-done:
			// Read goroutine has finished
			return
		case <-ticker.C:
			// Send ping to client
			if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				log.Printf("Failed to set write deadline for ping: %v", err)
			}
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}
		}
	}
}

// validateFilterContent validates that non-empty filter fields contain at least 3 letters
func validateFilterContent(options models.FilterOptions) string {
	letterRegex := regexp.MustCompile(`[a-zA-Z]`)

	// Validate repository field
	if options.Repository != "" {
		if countLetters(options.Repository, letterRegex) < 3 {
			return "Repository filter must contain at least 3 letters"
		}
	}

	// Validate pathPrefix field
	if options.PathPrefix != "" {
		if countLetters(options.PathPrefix, letterRegex) < 3 {
			return "Path prefix filter must contain at least 3 letters"
		}
	}

	// Validate keyword field - check each keyword individually
	if options.Keyword != "" {
		keywords := strings.Split(options.Keyword, ",")
		for _, keyword := range keywords {
			keyword = strings.TrimSpace(keyword)
			if keyword != "" && countLetters(keyword, letterRegex) < 3 {
				return fmt.Sprintf("Keyword '%s' must contain at least 3 letters", keyword)
			}
		}
	}

	return "" // No validation errors
}

// countLetters counts the number of letters in a string
func countLetters(s string, letterRegex *regexp.Regexp) int {
	matches := letterRegex.FindAllString(s, -1)
	return len(matches)
}
