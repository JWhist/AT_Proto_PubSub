package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"atp-test/internal/models"
)

// handleRoot provides basic information about the API
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
				"POST /api/filters/update - Update filters (repository, pathPrefix, keyword)",
			},
			"filters": map[string]string{
				"repository": "Filter by repository DID (e.g., 'did:plc:abc123')",
				"pathPrefix": "Filter by operation path prefix (e.g., 'app.bsky.feed.post')",
				"keyword":    "Filter by keyword in text content",
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

	filterKey := s.subscriptions.CreateFilter(req.Options)

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

	// Add connection to the subscription
	if !s.subscriptions.AddConnection(path, conn) {
		errorMsg := models.WSMessage{
			Type:      "error",
			Timestamp: time.Now(),
			Data:      map[string]string{"error": "Invalid filter key", "filterKey": path},
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
	if err := conn.WriteJSON(welcomeMsg); err != nil {
		log.Printf("Failed to send welcome message: %v", err)
	}

	log.Printf("🔌 WebSocket connected for filter %s", path[:8]+"...")

	// Handle connection lifecycle
	defer func() {
		s.subscriptions.RemoveConnection(path, conn)
		if err := conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
		log.Printf("🔌 WebSocket disconnected for filter %s", path[:8]+"...")
	}()

	// Keep connection alive and handle client messages
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle ping/pong or other client messages
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "ping":
				pongMsg := models.WSMessage{
					Type:      "pong",
					Timestamp: time.Now(),
					Data:      map[string]string{"status": "alive"},
				}
				if err := conn.WriteJSON(pongMsg); err != nil {
					log.Printf("Failed to send pong: %v", err)
					break
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
					if err := conn.WriteJSON(filterMsg); err != nil {
						log.Printf("Failed to send filter info: %v", err)
						break
					}
				}
			default:
				// Echo unknown messages back
				echoMsg := models.WSMessage{
					Type:      "echo",
					Timestamp: time.Now(),
					Data:      msg,
				}
				if err := conn.WriteJSON(echoMsg); err != nil {
					log.Printf("Failed to echo message: %v", err)
					break
				}
			}
		}
	}
}
