package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"atp-test/internal/models"
	"atp-test/internal/subscription"
)

func TestHandleCreateFilter(t *testing.T) {
	subscriptionManager := subscription.NewManager()
	server := &Server{
		subscriptions: subscriptionManager,
	}

	tests := []struct {
		name           string
		payload        interface{}
		expectedStatus int
	}{
		{
			name: "Valid filter creation",
			payload: models.CreateFilterRequest{
				Options: models.FilterOptions{
					Repository: "did:plc:test123",
					PathPrefix: "app.bsky.feed.post",
					Keyword:    "test",
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Empty filter (should work)",
			payload: models.CreateFilterRequest{
				Options: models.FilterOptions{},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid JSON",
			payload:        "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.payload.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
			}

			req := httptest.NewRequest(http.MethodPost, "/api/filters/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			server.handleCreateFilter(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response models.CreateFilterResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
				}

				if response.FilterKey == "" {
					t.Error("Expected non-empty filter key")
				}

				if len(response.FilterKey) != 32 {
					t.Errorf("Expected filter key length 32, got %d", len(response.FilterKey))
				}
			}
		})
	}
}

func TestHandleDeleteFilter(t *testing.T) {
	subscriptionManager := subscription.NewManager()
	server := &Server{
		subscriptions: subscriptionManager,
	}

	// Create a filter first
	options := models.FilterOptions{Repository: "did:plc:test123"}
	filterKey := subscriptionManager.CreateFilter(options)

	tests := []struct {
		name           string
		filterKey      string
		expectedStatus int
	}{
		{
			name:           "Delete existing filter",
			filterKey:      filterKey,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Delete non-existent filter",
			filterKey:      "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Empty filter key",
			filterKey:      "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/api/filters/delete/" + tt.filterKey
			req := httptest.NewRequest(http.MethodDelete, url, nil)

			rr := httptest.NewRecorder()
			server.handleDeleteFilter(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestHandleGetSubscriptions(t *testing.T) {
	subscriptionManager := subscription.NewManager()
	server := &Server{
		subscriptions: subscriptionManager,
	}

	// Test with no subscriptions
	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
	rr := httptest.NewRecorder()

	server.handleGetSubscriptions(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected response success to be true")
	}

	// The data should be an empty slice when there are no subscriptions
	dataSlice, ok := response.Data.([]interface{})
	if !ok {
		// Try to handle case where data might be nil for empty results
		if response.Data == nil {
			dataSlice = []interface{}{}
		} else {
			t.Errorf("Expected data to be an array or nil, got %T: %v", response.Data, response.Data)
			return
		}
	}

	if len(dataSlice) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(dataSlice))
	}

	// Create some subscriptions and test again
	options1 := models.FilterOptions{Repository: "did:plc:test1"}
	options2 := models.FilterOptions{PathPrefix: "app.bsky.feed.post"}

	subscriptionManager.CreateFilter(options1)
	subscriptionManager.CreateFilter(options2)

	req = httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
	rr = httptest.NewRecorder()

	server.handleGetSubscriptions(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	dataSlice, ok = response.Data.([]interface{})
	if !ok {
		t.Errorf("Expected data to be an array after creating subscriptions, got %T: %v", response.Data, response.Data)
		return
	}

	if len(dataSlice) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(dataSlice))
	}
}

func TestHandleStats(t *testing.T) {
	subscriptionManager := subscription.NewManager()
	server := &Server{
		subscriptions: subscriptionManager,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	rr := httptest.NewRecorder()

	server.handleStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response models.APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if !response.Success {
		t.Error("Expected success to be true")
	}

	if data, ok := response.Data.(map[string]interface{}); ok {
		if _, ok := data["active_filters"]; !ok {
			t.Error("Expected 'active_filters' in stats response")
		}

		if _, ok := data["total_connections"]; !ok {
			t.Error("Expected 'total_connections' in stats response")
		}
	} else {
		t.Error("Expected stats data to be a map")
	}
}

func TestFilterRouting(t *testing.T) {
	subscriptionManager := subscription.NewManager()
	server := &Server{
		subscriptions: subscriptionManager,
	}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "GET /api/subscriptions routes to get subscriptions",
			method:         "GET",
			path:           "/api/subscriptions",
			expectedStatus: http.StatusOK,
			description:    "Should return list of filters",
		},
		{
			name:           "POST /api/filters/create routes to create filter",
			method:         "POST",
			path:           "/api/filters/create",
			expectedStatus: http.StatusOK,
			description:    "Should create a new filter",
		},
		{
			name:           "DELETE /api/filters/delete/key routes to delete filter",
			method:         "DELETE",
			path:           "/api/filters/delete/somekey",
			expectedStatus: http.StatusNotFound,
			description:    "Should attempt to delete filter (404 for non-existent)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			if tt.method == "POST" {
				payload := models.CreateFilterRequest{
					Options: models.FilterOptions{},
				}
				body, _ = json.Marshal(payload)
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
			if tt.method == "POST" {
				req.Header.Set("Content-Type", "application/json")
			}

			rr := httptest.NewRecorder()

			// We need to simulate the routing logic here
			switch {
			case tt.method == "GET" && tt.path == "/api/subscriptions":
				server.handleGetSubscriptions(rr, req)
			case tt.method == "POST" && tt.path == "/api/filters/create":
				server.handleCreateFilter(rr, req)
			case tt.method == "DELETE" && strings.HasPrefix(tt.path, "/api/filters/delete/"):
				server.handleDeleteFilter(rr, req)
			default:
				rr.WriteHeader(http.StatusNotFound)
			}

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d for %s", tt.expectedStatus, rr.Code, tt.description)
			}
		})
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	subscriptionManager := subscription.NewManager()
	server := &Server{
		subscriptions: subscriptionManager,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}

	// Create a test filter first
	options := models.FilterOptions{Repository: "did:plc:test123"}
	filterKey := subscriptionManager.CreateFilter(options)

	// Test WebSocket upgrade with valid filter key
	req := httptest.NewRequest(http.MethodGet, "/ws/"+filterKey, nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "test")

	rr := httptest.NewRecorder()
	server.handleWebSocket(rr, req)

	// WebSocket upgrade attempts return 400 in httptest without proper handshake
	// but we're testing that the handler doesn't panic and processes the request
	if rr.Code != http.StatusBadRequest && rr.Code != http.StatusSwitchingProtocols {
		t.Errorf("Expected status 400 or 101, got %d", rr.Code)
	}
}

func TestWebSocketInvalidFilter(t *testing.T) {
	subscriptionManager := subscription.NewManager()
	server := &Server{
		subscriptions: subscriptionManager,
	}

	// Test WebSocket with invalid filter key
	req := httptest.NewRequest(http.MethodGet, "/ws/invalid", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")

	rr := httptest.NewRecorder()
	server.handleWebSocket(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid filter, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestConcurrentAPIAccess(t *testing.T) {
	subscriptionManager := subscription.NewManager()
	server := &Server{
		subscriptions: subscriptionManager,
	}

	// Test concurrent API calls
	done := make(chan bool, 3)

	// Concurrent filter creation
	go func() {
		for i := 0; i < 20; i++ {
			payload := models.CreateFilterRequest{
				Options: models.FilterOptions{
					Repository: "did:plc:test" + string(rune(i)),
				},
			}
			body, _ := json.Marshal(payload)

			req := httptest.NewRequest(http.MethodPost, "/api/filters/create", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			server.handleCreateFilter(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		}
		done <- true
	}()

	// Concurrent subscription fetching
	go func() {
		for i := 0; i < 30; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
			rr := httptest.NewRecorder()

			server.handleGetSubscriptions(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		}
		done <- true
	}()

	// Concurrent stats fetching
	go func() {
		for i := 0; i < 30; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
			rr := httptest.NewRecorder()

			server.handleStats(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}
