package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFilterOptions_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		filter   FilterOptions
		expected string
	}{
		{
			name: "all fields populated",
			filter: FilterOptions{
				Repository: "did:plc:test123",
				PathPrefix: "app.bsky.feed.post",
				Keyword:    "golang",
			},
			expected: `{"repository":"did:plc:test123","pathPrefix":"app.bsky.feed.post","keyword":"golang"}`,
		},
		{
			name:     "empty filter",
			filter:   FilterOptions{},
			expected: `{"repository":"","pathPrefix":"","keyword":""}`,
		},
		{
			name: "partial filter",
			filter: FilterOptions{
				PathPrefix: "app.bsky.feed",
			},
			expected: `{"repository":"","pathPrefix":"app.bsky.feed","keyword":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.filter)
			if err != nil {
				t.Fatalf("Failed to marshal FilterOptions: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("Marshal result = %s, want %s", string(data), tt.expected)
			}

			// Test unmarshaling
			var filter FilterOptions
			err = json.Unmarshal(data, &filter)
			if err != nil {
				t.Fatalf("Failed to unmarshal FilterOptions: %v", err)
			}
			if filter != tt.filter {
				t.Errorf("Unmarshal result = %+v, want %+v", filter, tt.filter)
			}
		})
	}
}

func TestAPIResponse_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		response APIResponse
		wantErr  bool
	}{
		{
			name: "successful response with data",
			response: APIResponse{
				Success: true,
				Message: "Operation successful",
				Data:    map[string]string{"key": "value"},
			},
		},
		{
			name: "error response without data",
			response: APIResponse{
				Success: false,
				Message: "Operation failed",
			},
		},
		{
			name: "response with complex data",
			response: APIResponse{
				Success: true,
				Message: "Complex data",
				Data: FilterOptions{
					Repository: "did:plc:test",
					PathPrefix: "app.bsky",
					Keyword:    "test",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.response)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Marshal error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			// Test unmarshaling
			var response APIResponse
			err = json.Unmarshal(data, &response)
			if err != nil {
				t.Fatalf("Failed to unmarshal APIResponse: %v", err)
			}

			if response.Success != tt.response.Success {
				t.Errorf("Success = %v, want %v", response.Success, tt.response.Success)
			}
			if response.Message != tt.response.Message {
				t.Errorf("Message = %s, want %s", response.Message, tt.response.Message)
			}
		})
	}
}

func TestFilterUpdateRequest_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		request FilterUpdateRequest
	}{
		{
			name:    "empty request",
			request: FilterUpdateRequest{},
		},
		{
			name: "repository only",
			request: FilterUpdateRequest{
				Repository: stringPtr("did:plc:test123"),
			},
		},
		{
			name: "pathPrefix only",
			request: FilterUpdateRequest{
				PathPrefix: stringPtr("app.bsky.feed"),
			},
		},
		{
			name: "keyword only",
			request: FilterUpdateRequest{
				Keyword: stringPtr("golang"),
			},
		},
		{
			name: "all fields",
			request: FilterUpdateRequest{
				Repository: stringPtr("did:plc:test123"),
				PathPrefix: stringPtr("app.bsky.feed.post"),
				Keyword:    stringPtr("golang"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test marshaling
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal FilterUpdateRequest: %v", err)
			}

			// Test unmarshaling
			var request FilterUpdateRequest
			err = json.Unmarshal(data, &request)
			if err != nil {
				t.Fatalf("Failed to unmarshal FilterUpdateRequest: %v", err)
			}

			// Compare pointer values
			if !equalStringPtr(request.Repository, tt.request.Repository) {
				t.Errorf("Repository = %v, want %v", ptrValueOrNil(request.Repository), ptrValueOrNil(tt.request.Repository))
			}
			if !equalStringPtr(request.PathPrefix, tt.request.PathPrefix) {
				t.Errorf("PathPrefix = %v, want %v", ptrValueOrNil(request.PathPrefix), ptrValueOrNil(tt.request.PathPrefix))
			}
			if !equalStringPtr(request.Keyword, tt.request.Keyword) {
				t.Errorf("Keyword = %v, want %v", ptrValueOrNil(request.Keyword), ptrValueOrNil(tt.request.Keyword))
			}
		})
	}
}

func TestATEvent_JSONMarshaling(t *testing.T) {
	event := ATEvent{
		Event: "commit",
		Did:   "did:plc:test123",
		Time:  "2025-10-04T12:00:00Z",
		Kind:  "commit",
		Ops: []ATOperation{
			{
				Action:     "create",
				Path:       "app.bsky.feed.post/12345",
				Collection: "app.bsky.feed.post",
				Rkey:       "12345",
				Cid:        "bafyreiabc123",
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal ATEvent: %v", err)
	}

	// Test unmarshaling
	var unmarshaled ATEvent
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ATEvent: %v", err)
	}

	if unmarshaled.Event != event.Event {
		t.Errorf("Event = %s, want %s", unmarshaled.Event, event.Event)
	}
	if unmarshaled.Did != event.Did {
		t.Errorf("Did = %s, want %s", unmarshaled.Did, event.Did)
	}
	if len(unmarshaled.Ops) != len(event.Ops) {
		t.Errorf("Ops length = %d, want %d", len(unmarshaled.Ops), len(event.Ops))
	}
	if len(unmarshaled.Ops) > 0 {
		if unmarshaled.Ops[0].Action != event.Ops[0].Action {
			t.Errorf("First op action = %s, want %s", unmarshaled.Ops[0].Action, event.Ops[0].Action)
		}
	}
}

func TestATOperation_JSONMarshaling(t *testing.T) {
	operation := ATOperation{
		Action:     "create",
		Path:       "app.bsky.feed.post/12345",
		Collection: "app.bsky.feed.post",
		Rkey:       "12345",
		Record: map[string]interface{}{
			"text":      "Hello world",
			"createdAt": "2025-10-04T12:00:00Z",
		},
		Cid: "bafyreiabc123",
	}

	// Test marshaling
	data, err := json.Marshal(operation)
	if err != nil {
		t.Fatalf("Failed to marshal ATOperation: %v", err)
	}

	// Test unmarshaling
	var unmarshaled ATOperation
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ATOperation: %v", err)
	}

	if unmarshaled.Action != operation.Action {
		t.Errorf("Action = %s, want %s", unmarshaled.Action, operation.Action)
	}
	if unmarshaled.Path != operation.Path {
		t.Errorf("Path = %s, want %s", unmarshaled.Path, operation.Path)
	}
	if unmarshaled.Collection != operation.Collection {
		t.Errorf("Collection = %s, want %s", unmarshaled.Collection, operation.Collection)
	}
	if unmarshaled.Rkey != operation.Rkey {
		t.Errorf("Rkey = %s, want %s", unmarshaled.Rkey, operation.Rkey)
	}
	if unmarshaled.Cid != operation.Cid {
		t.Errorf("Cid = %s, want %s", unmarshaled.Cid, operation.Cid)
	}
}

func TestRecordContent_JSONMarshaling(t *testing.T) {
	record := RecordContent{
		Text:    "Hello world",
		Message: "Test message",
		Content: "Test content",
		Reply: map[string]interface{}{
			"root": map[string]string{
				"uri": "at://test/post/123",
				"cid": "bafyreiabc123",
			},
		},
		Langs:   []string{"en", "es"},
		Type:    "app.bsky.feed.post",
		Created: "2025-10-04T12:00:00Z",
	}

	// Test marshaling
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Failed to marshal RecordContent: %v", err)
	}

	// Test unmarshaling
	var unmarshaled RecordContent
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal RecordContent: %v", err)
	}

	if unmarshaled.Text != record.Text {
		t.Errorf("Text = %s, want %s", unmarshaled.Text, record.Text)
	}
	if unmarshaled.Type != record.Type {
		t.Errorf("Type = %s, want %s", unmarshaled.Type, record.Type)
	}
	if len(unmarshaled.Langs) != len(record.Langs) {
		t.Errorf("Langs length = %d, want %d", len(unmarshaled.Langs), len(record.Langs))
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrValueOrNil(ptr *string) interface{} {
	if ptr == nil {
		return nil
	}
	return *ptr
}

func TestEnrichedATEvent_JSONMarshaling(t *testing.T) {
	now := time.Now()

	enrichedEvent := EnrichedATEvent{
		Event: "commit",
		Did:   "did:plc:test123",
		Time:  "2025-10-04T21:15:32.123Z",
		Kind:  "commit",
		Ops: []ATOperation{
			{
				Action: "create",
				Path:   "app.bsky.feed.post/test123",
				Record: map[string]interface{}{
					"text": "Test message",
				},
			},
		},
		Timestamps: EventTimestamps{
			Original:  "2025-10-04T21:15:32.123Z",
			Received:  now.Format(time.RFC3339Nano),
			Forwarded: now.Add(time.Millisecond).Format(time.RFC3339Nano),
			FilterKey: "abc123def456",
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(enrichedEvent)
	if err != nil {
		t.Fatalf("Failed to marshal EnrichedATEvent: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled EnrichedATEvent
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal EnrichedATEvent: %v", err)
	}

	// Verify key fields
	if unmarshaled.Event != enrichedEvent.Event {
		t.Errorf("Expected event %s, got %s", enrichedEvent.Event, unmarshaled.Event)
	}

	if unmarshaled.Did != enrichedEvent.Did {
		t.Errorf("Expected did %s, got %s", enrichedEvent.Did, unmarshaled.Did)
	}

	if unmarshaled.Timestamps.FilterKey != enrichedEvent.Timestamps.FilterKey {
		t.Errorf("Expected filter key %s, got %s", enrichedEvent.Timestamps.FilterKey, unmarshaled.Timestamps.FilterKey)
	}

	if unmarshaled.Timestamps.Original != enrichedEvent.Timestamps.Original {
		t.Errorf("Expected original timestamp %s, got %s", enrichedEvent.Timestamps.Original, unmarshaled.Timestamps.Original)
	}
}

func TestEventTimestamps_JSONMarshaling(t *testing.T) {
	now := time.Now()

	timestamps := EventTimestamps{
		Original:  "2025-10-04T21:15:32.123Z",
		Received:  now.Format(time.RFC3339Nano),
		Forwarded: now.Add(time.Millisecond).Format(time.RFC3339Nano),
		FilterKey: "filter123",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(timestamps)
	if err != nil {
		t.Fatalf("Failed to marshal EventTimestamps: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled EventTimestamps
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal EventTimestamps: %v", err)
	}

	if unmarshaled.FilterKey != timestamps.FilterKey {
		t.Errorf("Expected filter key %s, got %s", timestamps.FilterKey, unmarshaled.FilterKey)
	}

	if unmarshaled.Original != timestamps.Original {
		t.Errorf("Expected original %s, got %s", timestamps.Original, unmarshaled.Original)
	}
}
