package subscription

import (
	"testing"
	"time"

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

	// Create a few filters with required keywords
	options1 := models.FilterOptions{
		Repository: "did:plc:test1",
		Keyword:    "test",
	}
	options2 := models.FilterOptions{
		PathPrefix: "app.bsky.feed.post",
		Keyword:    "hello",
	}

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
		{
			name: "Multiple keywords - first matches",
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
				Keyword: "test,hello,world",
			},
			expected: true,
		},
		{
			name: "Multiple keywords - second matches",
			event: &models.ATEvent{
				Did: "did:plc:test123",
				Ops: []models.ATOperation{
					{
						Path: "app.bsky.feed.post/123",
						Record: map[string]interface{}{
							"text": "Hello there, how are you?",
						},
					},
				},
			},
			options: models.FilterOptions{
				Keyword: "test,hello,world",
			},
			expected: true,
		},
		{
			name: "Multiple keywords - none match",
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
				Keyword: "test,hello,world",
			},
			expected: false,
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

func TestRecordContainsKeywords(t *testing.T) {
	manager := NewManager()

	tests := []struct {
		name     string
		record   interface{}
		keywords string
		expected bool
	}{
		{
			name: "Single keyword match",
			record: map[string]interface{}{
				"text": "This is a test message",
			},
			keywords: "test",
			expected: true,
		},
		{
			name: "Multiple keywords - first matches",
			record: map[string]interface{}{
				"text": "This is a test message",
			},
			keywords: "test,hello,world",
			expected: true,
		},
		{
			name: "Multiple keywords - second matches",
			record: map[string]interface{}{
				"text": "Hello there, how are you?",
			},
			keywords: "test,hello,world",
			expected: true,
		},
		{
			name: "Multiple keywords - last matches",
			record: map[string]interface{}{
				"text": "World peace is important",
			},
			keywords: "test,hello,world",
			expected: true,
		},
		{
			name: "Multiple keywords - none match",
			record: map[string]interface{}{
				"text": "This is a different message",
			},
			keywords: "test,hello,world",
			expected: false,
		},
		{
			name: "Keywords with spaces",
			record: map[string]interface{}{
				"text": "Hello there, how are you?",
			},
			keywords: "test, hello , world",
			expected: true,
		},
		{
			name: "Empty keywords string",
			record: map[string]interface{}{
				"text": "This is a test message",
			},
			keywords: "",
			expected: false,
		},
		{
			name: "Case insensitive match",
			record: map[string]interface{}{
				"text": "This is a TEST message",
			},
			keywords: "test,hello,world",
			expected: true,
		},
		{
			name: "Partial word match",
			record: map[string]interface{}{
				"text": "This is testing something",
			},
			keywords: "test,hello,world",
			expected: true,
		},
		{
			name:     "Empty record",
			record:   map[string]interface{}{},
			keywords: "test,hello,world",
			expected: false,
		},
		{
			name:     "Nil record",
			record:   nil,
			keywords: "test,hello,world",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.recordContainsKeywords(tt.record, tt.keywords)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for keywords: %s", tt.expected, result, tt.keywords)
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

	// Create some filters with required keywords
	options1 := models.FilterOptions{
		Repository: "did:plc:test1",
		Keyword:    "test",
	}
	options2 := models.FilterOptions{
		Repository: "did:plc:test2",
		Keyword:    "hello",
	}

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
				Keyword:    "test",
			}
			manager.CreateFilter(options)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			options := models.FilterOptions{
				PathPrefix: "app.bsky.feed.post",
				Keyword:    "hello",
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

func TestEmptyFilterCleanup(t *testing.T) {
	manager := NewManager()

	// Create some filters
	options1 := models.FilterOptions{Repository: "did:plc:test1", Keyword: "test"}
	options2 := models.FilterOptions{Repository: "did:plc:test2", Keyword: "hello"}
	options3 := models.FilterOptions{Repository: "did:plc:test3", Keyword: "world"}

	filterKey1 := manager.CreateFilter(options1)
	filterKey2 := manager.CreateFilter(options2)
	filterKey3 := manager.CreateFilter(options3)

	// Verify all filters exist
	if len(manager.GetSubscriptions()) != 3 {
		t.Errorf("Expected 3 filters initially, got %d", len(manager.GetSubscriptions()))
	}

	// Add a connection to filterKey2 only
	manager.AddConnection(filterKey2, nil) // Using nil for simplicity in test

	// Create a mock event
	event := &models.ATEvent{
		Did: "did:plc:test123",
		Ops: []models.ATOperation{
			{
				Action: "create",
				Path:   "app.bsky.feed.post/test",
				Record: map[string]interface{}{
					"text": "test message",
				},
			},
		},
	}

	// Broadcast event - this should NOT trigger cleanup of empty filters anymore
	manager.BroadcastEvent(event)

	// Verify that all filters still exist (no cleanup during broadcast)
	remainingFilters := manager.GetSubscriptions()
	if len(remainingFilters) != 3 {
		t.Errorf("Expected 3 filters after broadcast (no cleanup), got %d", len(remainingFilters))
	}

	// Now remove the connection from filterKey2 - this should trigger cleanup
	manager.RemoveConnection(filterKey2, nil)

	// Verify filterKey2 was cleaned up when its last connection was removed
	afterRemoveFilters := manager.GetSubscriptions()
	if len(afterRemoveFilters) != 2 {
		t.Errorf("Expected 2 filters after connection removal, got %d", len(afterRemoveFilters))
	}

	// Verify filterKey2 no longer exists but others do
	_, exists2 := manager.GetSubscription(filterKey2)
	_, exists1 := manager.GetSubscription(filterKey1)
	_, exists3 := manager.GetSubscription(filterKey3)

	if exists2 {
		t.Error("Filter with removed connection should have been cleaned up")
	}
	if !exists1 || !exists3 {
		t.Error("Filters without connections should still exist (only cleaned up when last connection is removed)")
	}
}

func TestPeriodicCleanup(t *testing.T) {
	// Create a manager but we'll manually control the cleanup for testing
	manager := &Manager{
		subscriptions:  make(map[string]*Subscription),
		maxConnections: 1000,
		cleanupStop:    make(chan bool, 1),
	}

	// Create some filters
	options1 := models.FilterOptions{Repository: "did:plc:test1", Keyword: "test"}
	options2 := models.FilterOptions{Repository: "did:plc:test2", Keyword: "hello"}

	filterKey1 := manager.CreateFilter(options1)
	filterKey2 := manager.CreateFilter(options2)

	// Verify both filters exist
	if len(manager.GetSubscriptions()) != 2 {
		t.Errorf("Expected 2 filters initially, got %d", len(manager.GetSubscriptions()))
	}

	// Add and immediately remove a connection from filterKey2 to simulate it having had activity
	manager.AddConnection(filterKey2, nil)
	manager.RemoveConnection(filterKey2, nil)

	// Artificially age the filters by modifying their timestamps
	manager.mu.Lock()
	now := time.Now()
	oldTime := now.Add(-15 * time.Minute) // 15 minutes ago (past grace period)

	// Age filterKey1 (never had connections)
	if sub1, exists := manager.subscriptions[filterKey1]; exists {
		sub1.CreatedAt = oldTime
	}

	// Age filterKey2 (had connections but empty now)
	if sub2, exists := manager.subscriptions[filterKey2]; exists {
		pastTime := oldTime
		sub2.LastConnectionAt = &pastTime
	}
	manager.mu.Unlock()

	// Run periodic cleanup manually
	manager.performPeriodicCleanup()

	// Verify both old filters were cleaned up
	remainingFilters := manager.GetSubscriptions()
	if len(remainingFilters) != 0 {
		t.Errorf("Expected 0 filters after periodic cleanup, got %d", len(remainingFilters))
	}

	// Verify the filters no longer exist
	_, exists1 := manager.GetSubscription(filterKey1)
	_, exists2 := manager.GetSubscription(filterKey2)
	if exists1 || exists2 {
		t.Error("Old filters should have been cleaned up by periodic cleanup")
	}
}

func TestGetMatchingKeywords(t *testing.T) {
	manager := NewManager()
	defer manager.Shutdown()

	// Create test event with content that matches some keywords
	event := &models.ATEvent{
		Did: "did:plc:test123",
		Ops: []models.ATOperation{
			{
				Action: "create",
				Path:   "app.bsky.feed.post/test",
				Record: map[string]interface{}{
					"text": "This post is about cats and dogs",
				},
			},
		},
	}

	tests := []struct {
		name             string
		keywords         string
		expectedMatches  []string
	}{
		{
			name:             "Single matching keyword",
			keywords:         "cats",
			expectedMatches:  []string{"cats"},
		},
		{
			name:             "Multiple keywords, some match",
			keywords:         "cats,birds,dogs",
			expectedMatches:  []string{"cats", "dogs"},
		},
		{
			name:             "Multiple keywords, none match",
			keywords:         "fish,birds,hamsters",
			expectedMatches:  []string{},
		},
		{
			name:             "Keywords with spaces",
			keywords:         " cats , birds , dogs ",
			expectedMatches:  []string{"cats", "dogs"},
		},
		{
			name:             "Empty keywords",
			keywords:         "",
			expectedMatches:  nil,
		},
		{
			name:             "Case insensitive matching",
			keywords:         "CATS,DOGS,BIRDS",
			expectedMatches:  []string{"CATS", "DOGS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := manager.getMatchingKeywords(event, tt.keywords)
			
			if len(matches) != len(tt.expectedMatches) {
				t.Errorf("Expected %d matches, got %d. Expected: %v, Got: %v", 
					len(tt.expectedMatches), len(matches), tt.expectedMatches, matches)
				return
			}

			// Check that all expected matches are present
			expectedMap := make(map[string]bool)
			for _, expected := range tt.expectedMatches {
				expectedMap[expected] = true
			}

			for _, match := range matches {
				if !expectedMap[match] {
					t.Errorf("Unexpected match: %s", match)
				}
				delete(expectedMap, match)
			}

			if len(expectedMap) > 0 {
				t.Errorf("Missing expected matches: %v", expectedMap)
			}
		})
	}
}
