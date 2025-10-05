package subscription

import (
	"testing"

	"github.com/JWhist/AT_Proto_PubSub/internal/models"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()
	if manager == nil {
		t.Error("NewManager should not return nil")
		return
	}

	if manager.subscriptions == nil {
		t.Error("Manager subscriptions map should be initialized")
	}
}

func TestCreateFilter(t *testing.T) {
	manager := NewManager()

	options := models.FilterOptions{
		Repository: "did:plc:test123",
		PathPrefix: "app.bsky.feed.post",
		Keyword:    "test",
	}

	filterKey := manager.CreateFilter(options)

	if filterKey == "" {
		t.Error("Filter key should not be empty")
	}

	if len(filterKey) != 32 { // 16 bytes hex encoded = 32 characters
		t.Errorf("Expected filter key length 32, got %d", len(filterKey))
	}

	// Verify the filter was stored
	subscription, exists := manager.GetSubscription(filterKey)
	if !exists {
		t.Error("Filter should exist after creation")
	}

	if subscription.FilterKey != filterKey {
		t.Errorf("Expected filter key %s, got %s", filterKey, subscription.FilterKey)
	}

	if subscription.Options.Repository != options.Repository {
		t.Errorf("Expected repository %s, got %s", options.Repository, subscription.Options.Repository)
	}
}

func TestGetSubscriptions(t *testing.T) {
	manager := NewManager()

	// Initially should be empty
	subs := manager.GetSubscriptions()
	if len(subs) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(subs))
	}

	// Create a few filters
	options1 := models.FilterOptions{Repository: "did:plc:test1"}
	options2 := models.FilterOptions{PathPrefix: "app.bsky.feed.post"}

	key1 := manager.CreateFilter(options1)
	key2 := manager.CreateFilter(options2)

	subs = manager.GetSubscriptions()
	if len(subs) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subs))
	}

	// Check that both filters are present
	found1, found2 := false, false
	for _, sub := range subs {
		if sub.FilterKey == key1 {
			found1 = true
		}
		if sub.FilterKey == key2 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Both created filters should be in the subscriptions list")
	}
}

func TestMatchesFilter(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name     string
		event    *models.ATEvent
		options  models.FilterOptions
		expected bool
	}{
		{
			name: "Repository filter match",
			event: &models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{Path: "app.bsky.feed.post/123"},
				},
			},
			options: models.FilterOptions{
				Repository: "did:plc:test123",
			},
			expected: true,
		},
		{
			name: "Repository filter no match",
			event: &models.ATEvent{
				Did: "did:plc:different",
				Ops: []models.ATOperation{
					{Path: "app.bsky.feed.post/123"},
				},
			},
			options: models.FilterOptions{
				Repository: "did:plc:test123",
			},
			expected: false,
		},
		{
			name: "PathPrefix filter match",
			event: &models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{Path: "app.bsky.feed.post/123"},
				},
			},
			options: models.FilterOptions{
				PathPrefix: "app.bsky.feed.post",
			},
			expected: true,
		},
		{
			name: "PathPrefix filter no match",
			event: &models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{Path: "app.bsky.graph.follow/123"},
				},
			},
			options: models.FilterOptions{
				PathPrefix: "app.bsky.feed.post",
			},
			expected: false,
		},
		{
			name: "Keyword filter match",
			event: &models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Path: "app.bsky.feed.post/123",
						Record: map[string]interface{}{
							"text": "This is a test message",
						},
					},
				},
			},
			options: models.FilterOptions{
				Keyword: "test",
			},
			expected: true,
		},
		{
			name: "Keyword filter no match",
			event: &models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Path: "app.bsky.feed.post/123",
						Record: map[string]interface{}{
							"text": "This is a different message",
						},
					},
				},
			},
			options: models.FilterOptions{
				Keyword: "test",
			},
			expected: false,
		},
		{
			name: "Multiple filters all match",
			event: &models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Path: "app.bsky.feed.post/123",
						Record: map[string]interface{}{
							"text": "This is a test message",
						},
					},
				},
			},
			options: models.FilterOptions{
				Repository: "did:plc:test123",
				PathPrefix: "app.bsky.feed.post",
				Keyword:    "test",
			},
			expected: true,
		},
		{
			name: "Empty filters blocked by safety check",
			event: &models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{Path: "app.bsky.feed.post/123"},
				},
			},
			options:  models.FilterOptions{},
			expected: false, // Changed from true to false due to safety check
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.matchesFilter(tt.event, tt.options)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRecordContainsKeyword(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name     string
		record   interface{}
		keyword  string
		expected bool
	}{
		{
			name: "Text field contains keyword",
			record: map[string]interface{}{
				"text": "This is a test message",
			},
			keyword:  "test",
			expected: true,
		},
		{
			name: "Message field contains keyword",
			record: map[string]interface{}{
				"message": "This is a test message",
			},
			keyword:  "test",
			expected: true,
		},
		{
			name: "Content field contains keyword",
			record: map[string]interface{}{
				"content": "This is a test message",
			},
			keyword:  "test",
			expected: true,
		},
		{
			name: "Case insensitive match",
			record: map[string]interface{}{
				"text": "This is a TEST message",
			},
			keyword:  "test",
			expected: true,
		},
		{
			name: "No text fields",
			record: map[string]interface{}{
				"other": "some value",
			},
			keyword:  "test",
			expected: false,
		},
		{
			name: "Empty record",
			record: map[string]interface{}{
				"text": "",
			},
			keyword:  "test",
			expected: false,
		},
		{
			name:     "Nil record",
			record:   nil,
			keyword:  "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.recordContainsKeyword(tt.record, tt.keyword)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	manager := NewManager()

	// Test with no filters
	stats := manager.GetStats()
	if stats["active_filters"] != 0 {
		t.Errorf("Expected 0 active filters, got %v", stats["active_filters"])
	}
	if stats["total_connections"] != 0 {
		t.Errorf("Expected 0 total connections, got %v", stats["total_connections"])
	}

	// Create some filters
	options1 := models.FilterOptions{Repository: "did:plc:test1"}
	options2 := models.FilterOptions{Repository: "did:plc:test2"}

	manager.CreateFilter(options1)
	manager.CreateFilter(options2)

	stats = manager.GetStats()
	if stats["active_filters"] != 2 {
		t.Errorf("Expected 2 active filters, got %v", stats["active_filters"])
	}
}

func TestGenerateFilterKey(t *testing.T) {
	// Test that keys are unique
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key := generateFilterKey()
		if keys[key] {
			t.Errorf("Duplicate filter key generated: %s", key)
		}
		keys[key] = true

		if len(key) != 32 {
			t.Errorf("Expected filter key length 32, got %d", len(key))
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager()

	// Test concurrent filter creation and access
	done := make(chan bool, 4)

	// Multiple creators
	go func() {
		for i := 0; i < 50; i++ {
			options := models.FilterOptions{
				Repository: "did:plc:test",
			}
			manager.CreateFilter(options)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			options := models.FilterOptions{
				PathPrefix: "app.bsky.feed.post",
			}
			manager.CreateFilter(options)
		}
		done <- true
	}()

	// Multiple readers
	go func() {
		for i := 0; i < 100; i++ {
			manager.GetSubscriptions()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			manager.GetStats()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify final state
	subs := manager.GetSubscriptions()
	if len(subs) != 100 {
		t.Errorf("Expected 100 subscriptions after concurrent creation, got %d", len(subs))
	}
}
