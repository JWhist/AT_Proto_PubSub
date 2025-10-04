package firehose

import (
	"testing"

	"atp-test/internal/models"
)

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Error("NewClient should not return nil")
	}

	// Check that filters are initialized to empty values
	filters := client.GetFilters()
	if filters.Repository != "" {
		t.Errorf("Expected empty repository filter, got %s", filters.Repository)
	}
	if filters.PathPrefix != "" {
		t.Errorf("Expected empty path prefix filter, got %s", filters.PathPrefix)
	}
	if filters.Keyword != "" {
		t.Errorf("Expected empty keyword filter, got %s", filters.Keyword)
	}
}

func TestUpdateFilters(t *testing.T) {
	client := NewClient()

	// Test updating filters
	newFilters := models.FilterOptions{
		Repository: "did:plc:test123",
		PathPrefix: "app.bsky.feed.post",
		Keyword:    "test",
	}

	client.UpdateFilters(newFilters)

	// Verify filters were updated
	filters := client.GetFilters()
	if filters.Repository != "did:plc:test123" {
		t.Errorf("Expected repository 'did:plc:test123', got %s", filters.Repository)
	}
	if filters.PathPrefix != "app.bsky.feed.post" {
		t.Errorf("Expected path prefix 'app.bsky.feed.post', got %s", filters.PathPrefix)
	}
	if filters.Keyword != "test" {
		t.Errorf("Expected keyword 'test', got %s", filters.Keyword)
	}
}

func TestGetFilters(t *testing.T) {
	client := NewClient()

	// Test initial empty filters
	filters := client.GetFilters()
	if filters.Repository != "" || filters.PathPrefix != "" || filters.Keyword != "" {
		t.Error("Expected all filters to be empty initially")
	}

	// Test after setting filters
	testFilters := models.FilterOptions{
		Repository: "test-repo",
		PathPrefix: "test-path",
		Keyword:    "test-keyword",
	}

	client.UpdateFilters(testFilters)
	retrievedFilters := client.GetFilters()

	if retrievedFilters.Repository != testFilters.Repository {
		t.Errorf("Repository mismatch: expected %s, got %s", testFilters.Repository, retrievedFilters.Repository)
	}
	if retrievedFilters.PathPrefix != testFilters.PathPrefix {
		t.Errorf("PathPrefix mismatch: expected %s, got %s", testFilters.PathPrefix, retrievedFilters.PathPrefix)
	}
	if retrievedFilters.Keyword != testFilters.Keyword {
		t.Errorf("Keyword mismatch: expected %s, got %s", testFilters.Keyword, retrievedFilters.Keyword)
	}
}

func TestConcurrentFilterAccess(t *testing.T) {
	client := NewClient()

	// Test concurrent reads and writes to ensure thread safety
	done := make(chan bool, 2)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			client.UpdateFilters(models.FilterOptions{
				Repository: "test-repo",
				PathPrefix: "test-path",
				Keyword:    "test-keyword",
			})
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			client.GetFilters()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify final state
	filters := client.GetFilters()
	if filters.Repository != "test-repo" {
		t.Error("Concurrent access test failed")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{
			name:     "a less than b",
			a:        5,
			b:        10,
			expected: 5,
		},
		{
			name:     "b less than a",
			a:        10,
			b:        5,
			expected: 5,
		},
		{
			name:     "a equals b",
			a:        7,
			b:        7,
			expected: 7,
		},
		{
			name:     "negative numbers",
			a:        -5,
			b:        -10,
			expected: -10,
		},
		{
			name:     "zero and positive",
			a:        0,
			b:        5,
			expected: 0,
		},
		{
			name:     "negative and positive",
			a:        -3,
			b:        2,
			expected: -3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

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

func TestMatchesFilter(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name     string
		op       models.ATOperation
		filters  models.FilterOptions
		expected bool
	}{
		{
			name: "Match with keyword in text",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"text": "This is a test message",
				},
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: true,
		},
		{
			name: "No match with keyword",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"text": "This is a message",
				},
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: false,
		},
		{
			name: "Case insensitive keyword match",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"text": "This is a TEST message",
				},
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: true,
		},
		{
			name: "No keyword filter matches any text",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"text": "Any message",
				},
			},
			filters: models.FilterOptions{
				Keyword: "",
			},
			expected: true,
		},
		{
			name: "No record returns false",
			op: models.ATOperation{
				Record: nil,
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: false,
		},
		{
			name: "Empty text with keyword filter returns false",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"text": "",
				},
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: false,
		},
		{
			name: "Match with message field",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"message": "This is a test message",
				},
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: true,
		},
		{
			name: "Match with content field",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"content": "This is a test message",
				},
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: true,
		},
		{
			name: "No text fields returns false",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"other": "data",
				},
			},
			filters: models.FilterOptions{
				Keyword: "",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.matchesFilter(tt.op, tt.filters)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHandleEvent(t *testing.T) {
	client := NewClient()

	t.Run("Filter by repository", func(t *testing.T) {
		filters := models.FilterOptions{
			Repository: "did:plc:specific",
		}
		client.UpdateFilters(filters)

		// This test checks that the function runs without panic
		// In a real scenario, we'd need to capture console output to verify filtering
		event := models.ATEvent{
			Did: "did:plc:different",
			Ops: []models.ATOperation{
				{
					Action: "create",
					Path:   "app.bsky.feed.post/123",
					Record: map[string]interface{}{
						"text": "test message",
					},
				},
			},
		}

		// This should not panic and should filter out the event
		client.handleEvent(event)
	})

	t.Run("Process matching event", func(t *testing.T) {
		filters := models.FilterOptions{
			Repository: "did:plc:test123",
			PathPrefix: "app.bsky.feed.post",
			Keyword:    "test",
		}
		client.UpdateFilters(filters)

		event := models.ATEvent{
			Did: "did:plc:test123",
			Ops: []models.ATOperation{
				{
					Action: "create",
					Path:   "app.bsky.feed.post/123",
					Record: map[string]interface{}{
						"text": "This is a test message",
					},
				},
			},
		}

		// This should not panic and should process the event
		client.handleEvent(event)
	})

	t.Run("Process update action", func(t *testing.T) {
		filters := models.FilterOptions{}
		client.UpdateFilters(filters)

		event := models.ATEvent{
			Did: "did:plc:test123",
			Ops: []models.ATOperation{
				{
					Action: "update",
					Path:   "app.bsky.feed.post/123",
					Record: map[string]interface{}{
						"text": "Updated message",
					},
				},
			},
		}

		// This should not panic and should process the update
		client.handleEvent(event)
	})

	t.Run("Skip delete action", func(t *testing.T) {
		filters := models.FilterOptions{}
		client.UpdateFilters(filters)

		event := models.ATEvent{
			Did: "did:plc:test123",
			Ops: []models.ATOperation{
				{
					Action: "delete",
					Path:   "app.bsky.feed.post/123",
					Record: nil,
				},
			},
		}

		// This should not panic and should skip the delete action
		client.handleEvent(event)
	})
}

func TestMatchesFilterWithInvalidJSON(t *testing.T) {
	client := NewClient()

	// Test with record that can't be marshaled to JSON properly
	op := models.ATOperation{
		Record: map[string]interface{}{
			"invalid": make(chan int), // channels can't be marshaled to JSON
		},
	}

	filters := models.FilterOptions{
		Keyword: "test",
	}

	result := client.matchesFilter(op, filters)
	if result != false {
		t.Error("Expected false for record that can't be marshaled")
	}
}

func TestMatchesFilterWithInvalidRecordStructure(t *testing.T) {
	client := NewClient()

	// Test with record that marshals to JSON but doesn't unmarshal to RecordContent properly
	op := models.ATOperation{
		Record: map[string]interface{}{
			"text":    123, // wrong type for text field
			"message": 456, // wrong type for message field
			"content": 789, // wrong type for content field
		},
	}

	filters := models.FilterOptions{
		Keyword: "test",
	}

	result := client.matchesFilter(op, filters)
	if result != false {
		t.Error("Expected false for record with invalid structure")
	}
}

func TestLogEvent(t *testing.T) {
	client := NewClient()

	event := models.ATEvent{
		Did:  "did:plc:test123",
		Time: "2025-10-04T12:00:00Z",
		Kind: "commit",
	}

	op := models.ATOperation{
		Action:     "create",
		Path:       "app.bsky.feed.post/12345",
		Collection: "app.bsky.feed.post",
		Rkey:       "12345",
		Record: map[string]interface{}{
			"text":      "This is a test message",
			"createdAt": "2025-10-04T12:00:00Z",
			"$type":     "app.bsky.feed.post",
			"langs":     []string{"en"},
			"reply": map[string]interface{}{
				"root": map[string]interface{}{
					"uri": "at://did:plc:other/app.bsky.feed.post/root",
				},
			},
		},
	}

	// This test verifies that logEvent doesn't panic and can handle various record structures
	client.logEvent(event, op)

	// Test with empty record
	opEmpty := models.ATOperation{
		Action:     "update",
		Path:       "app.bsky.feed.post/67890",
		Collection: "app.bsky.feed.post",
		Rkey:       "67890",
		Record:     nil,
	}

	client.logEvent(event, opEmpty)

	// Test with record that has only message field
	opMessage := models.ATOperation{
		Action:     "create",
		Path:       "app.bsky.feed.post/message",
		Collection: "app.bsky.feed.post",
		Rkey:       "message",
		Record: map[string]interface{}{
			"message": "This is a message field",
		},
	}

	client.logEvent(event, opMessage)

	// Test with record that has content field
	opContent := models.ATOperation{
		Action:     "create",
		Path:       "app.bsky.feed.post/content",
		Collection: "app.bsky.feed.post",
		Rkey:       "content",
		Record: map[string]interface{}{
			"content": "This is a content field",
		},
	}

	client.logEvent(event, opContent)
}

func TestHandleEventWithPathFiltering(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name      string
		filters   models.FilterOptions
		event     models.ATEvent
		shouldLog bool
	}{
		{
			name: "Repository filter match",
			filters: models.FilterOptions{
				Repository: "did:plc:test123",
			},
			event: models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Action: "create",
						Path:   "app.bsky.feed.post/123",
						Record: map[string]interface{}{"text": "test"},
					},
				},
			},
			shouldLog: true,
		},
		{
			name: "Repository filter no match",
			filters: models.FilterOptions{
				Repository: "did:plc:different",
			},
			event: models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Action: "create",
						Path:   "app.bsky.feed.post/123",
						Record: map[string]interface{}{"text": "test"},
					},
				},
			},
			shouldLog: false,
		},
		{
			name: "PathPrefix filter match",
			filters: models.FilterOptions{
				PathPrefix: "app.bsky.feed.post",
			},
			event: models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Action: "create",
						Path:   "app.bsky.feed.post/123",
						Record: map[string]interface{}{"text": "test"},
					},
				},
			},
			shouldLog: true,
		},
		{
			name: "PathPrefix filter no match",
			filters: models.FilterOptions{
				PathPrefix: "app.bsky.graph.follow",
			},
			event: models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Action: "create",
						Path:   "app.bsky.feed.post/123",
						Record: map[string]interface{}{"text": "test"},
					},
				},
			},
			shouldLog: false,
		},
		{
			name: "Delete action ignored",
			filters: models.FilterOptions{},
			event: models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Action: "delete",
						Path:   "app.bsky.feed.post/123",
						Record: nil,
					},
				},
			},
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.UpdateFilters(tt.filters)
			// We can't easily test the console output, but we can verify the function doesn't panic
			client.handleEvent(tt.event)
		})
	}
}

func TestMatchesFilterEdgeCases(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name     string
		op       models.ATOperation
		filters  models.FilterOptions
		expected bool
	}{
		{
			name: "Record with invalid JSON structure for marshaling",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"text": func() {}, // functions can't be marshaled
				},
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: false,
		},
		{
			name: "Record with complex nested structure",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"text": "This is a test message",
					"embed": map[string]interface{}{
						"$type": "app.bsky.embed.record",
						"record": map[string]interface{}{
							"uri": "at://did:plc:other/app.bsky.feed.post/abc",
						},
					},
				},
			},
			filters: models.FilterOptions{
				Keyword: "test",
			},
			expected: true,
		},
		{
			name: "Empty filters match any text",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"text": "Any text content",
				},
			},
			filters: models.FilterOptions{},
			expected: true,
		},
		{
			name: "No text fields with empty keyword filter",
			op: models.ATOperation{
				Record: map[string]interface{}{
					"nottext": "value",
					"other":   123,
				},
			},
			filters: models.FilterOptions{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.matchesFilter(tt.op, tt.filters)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPathParsing(t *testing.T) {
	client := NewClient()

	// Test that handleEvent correctly parses path into collection and rkey
	event := models.ATEvent{
		Did: "did:plc:test123",
		Ops: []models.ATOperation{
			{
				Action: "create",
				Path:   "app.bsky.feed.post/3jx6m3lpydk2p",
				Record: map[string]interface{}{
					"text": "Test message",
				},
			},
			{
				Action: "update",
				Path:   "app.bsky.graph.follow/3jx6m3lpydk2p",
				Record: map[string]interface{}{
					"subject": "did:plc:other",
				},
			},
			{
				Action: "create",
				Path:   "invalid", // Path without slash
				Record: map[string]interface{}{
					"text": "Test",
				},
			},
		},
	}

	// Test with no filters to process all operations
	client.UpdateFilters(models.FilterOptions{})
	client.handleEvent(event)

	// Test with path prefix filter
	client.UpdateFilters(models.FilterOptions{
		PathPrefix: "app.bsky.feed",
	})
	client.handleEvent(event)
}
