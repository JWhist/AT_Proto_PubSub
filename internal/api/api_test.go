package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JWhist/AT_Proto_PubSub/internal/firehose"
	"github.com/JWhist/AT_Proto_PubSub/internal/models"
)

func TestGetFilterString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string returns ALL",
			input:    "",
			expected: "ALL",
		},
		{
			name:     "Non-empty string returns input",
			input:    "test-filter",
			expected: "test-filter",
		},
		{
			name:     "Space returns space",
			input:    " ",
			expected: " ",
		},
		{
			name:     "Special characters",
			input:    "app.bsky.feed.post",
			expected: "app.bsky.feed.post",
		},
		{
			name:     "DID format",
			input:    "did:plc:abc123",
			expected: "did:plc:abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFilterString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNewServer(t *testing.T) {
	// Test that NewServer can be called and creates a server
	// Using nil client should not cause immediate panic in constructor
	server := NewServer(nil, "8080")

	if server == nil {
		t.Error("NewServer should not return nil")
		return
	}

	if server.server == nil {
		t.Error("Expected server.server to be initialized")
		return
	}

	if server.server.Addr != ":8080" {
		t.Errorf("Expected server address :8080, got %s", server.server.Addr)
	}
}

func TestNewServerWithDifferentPort(t *testing.T) {
	tests := []struct {
		name         string
		port         string
		expectedAddr string
	}{
		{
			name:         "Port 3000",
			port:         "3000",
			expectedAddr: ":3000",
		},
		{
			name:         "Port 8081",
			port:         "8081",
			expectedAddr: ":8081",
		},
		{
			name:         "Empty port",
			port:         "",
			expectedAddr: ":",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(nil, tt.port)

			if server == nil {
				t.Error("NewServer should not return nil")
				return
			}

			if server.server == nil {
				t.Error("Expected server.server to be initialized")
				return
			}

			if server.server.Addr != tt.expectedAddr {
				t.Errorf("Expected server address %s, got %s", tt.expectedAddr, server.server.Addr)
			}
		})
	}
}

func TestHandleRoot(t *testing.T) {
	client := firehose.NewClient()
	server := NewServer(client, "8080")

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:           "GET request success",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name:           "POST request method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
		},
		{
			name:           "PUT request method not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			w := httptest.NewRecorder()

			server.handleRoot(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse {
				var response models.APIResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}

				if !strings.Contains(response.Message, "AT Protocol") {
					t.Error("Expected message to contain 'AT Protocol'")
				}

				if response.Data == nil {
					t.Error("Expected data to be present")
				}
			}
		})
	}
}

func TestHandleStatus(t *testing.T) {
	client := firehose.NewClient()
	server := NewServer(client, "8080")

	// Set some filters first
	testFilters := models.FilterOptions{
		Repository: "did:plc:test123",
		PathPrefix: "app.bsky.feed.post",
		Keyword:    "test",
	}
	client.UpdateFilters(testFilters)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:           "GET request success",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name:           "POST request method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/status", nil)
			w := httptest.NewRecorder()

			server.handleStatus(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse {
				var response models.APIResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}

				if response.Message != "Server is running" {
					t.Errorf("Expected message 'Server is running', got %s", response.Message)
				}

				// Check that filters are included in response
				data, ok := response.Data.(map[string]interface{})
				if !ok {
					t.Error("Expected data to be a map")
				}

				if data["status"] != "active" {
					t.Error("Expected status to be 'active'")
				}
			}
		})
	}
}

func TestHandleFilters(t *testing.T) {
	client := firehose.NewClient()
	server := NewServer(client, "8080")

	// Set some filters first
	testFilters := models.FilterOptions{
		Repository: "did:plc:test123",
		PathPrefix: "app.bsky.feed.post",
		Keyword:    "test",
	}
	client.UpdateFilters(testFilters)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:           "GET request success",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name:           "POST request method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/filters", nil)
			w := httptest.NewRecorder()

			server.handleFilters(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse {
				var response models.APIResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if !response.Success {
					t.Error("Expected success to be true")
				}

				if response.Message != "Current filter settings" {
					t.Errorf("Expected message 'Current filter settings', got %s", response.Message)
				}

				// Parse the filters from response data
				filtersData, err := json.Marshal(response.Data)
				if err != nil {
					t.Fatalf("Failed to marshal filter data: %v", err)
				}

				var filters models.FilterOptions
				if err := json.Unmarshal(filtersData, &filters); err != nil {
					t.Fatalf("Failed to unmarshal filter data: %v", err)
				}

				if filters.Repository != testFilters.Repository {
					t.Errorf("Expected repository %s, got %s", testFilters.Repository, filters.Repository)
				}
			}
		})
	}
}

func TestHandleUpdateFilters(t *testing.T) {
	client := firehose.NewClient()
	server := NewServer(client, "8080")

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		checkResponse  bool
		expectedError  bool
	}{
		{
			name:           "POST valid JSON - partial update",
			method:         http.MethodPost,
			body:           `{"repository": "did:plc:newrepo"}`,
			expectedStatus: http.StatusOK,
			checkResponse:  true,
			expectedError:  false,
		},
		{
			name:           "POST valid JSON - full update",
			method:         http.MethodPost,
			body:           `{"repository": "did:plc:full", "pathPrefix": "app.bsky.feed.post", "keyword": "test"}`,
			expectedStatus: http.StatusOK,
			checkResponse:  true,
			expectedError:  false,
		},
		{
			name:           "POST empty JSON",
			method:         http.MethodPost,
			body:           `{}`,
			expectedStatus: http.StatusOK,
			checkResponse:  true,
			expectedError:  false,
		},
		{
			name:           "POST invalid JSON",
			method:         http.MethodPost,
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse:  true,
			expectedError:  true,
		},
		{
			name:           "GET request method not allowed",
			method:         http.MethodGet,
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
			expectedError:  false,
		},
		{
			name:           "PUT request method not allowed",
			method:         http.MethodPut,
			body:           `{"repository": "test"}`,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/api/filters/update", strings.NewReader(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/api/filters/update", nil)
			}
			w := httptest.NewRecorder()

			server.handleUpdateFilters(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse {
				var response models.APIResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if tt.expectedError {
					if response.Success {
						t.Error("Expected success to be false for error case")
					}
				} else {
					if !response.Success {
						t.Error("Expected success to be true")
					}
				}
			}
		})
	}
}

func TestServerStartStop(t *testing.T) {
	client := firehose.NewClient()
	server := NewServer(client, "0") // Port 0 for automatic port assignment

	// Test that Start and Stop methods exist and can be called
	// We can't easily test the actual server lifecycle without complex setup

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test Stop with context (even though server isn't started)
	err := server.Stop(ctx)
	if err != nil {
		t.Logf("Stop returned error (expected): %v", err)
	}

	// Test that server has the expected structure
	if server.server == nil {
		t.Error("Expected server.server to be initialized")
	}

	if server.firehoseClient != client {
		t.Error("Expected firehoseClient to match")
	}
}
