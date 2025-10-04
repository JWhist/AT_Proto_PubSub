package api

import (
	"encoding/json"
	"fmt"
	"net/http"

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
