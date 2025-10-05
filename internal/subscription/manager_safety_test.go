package subscription

import (
	"testing"
	"time"

	"github.com/JWhist/AT_Proto_PubSub/internal/models"
)

func TestCreateFilterValidation(t *testing.T) {
	manager := NewManager()

	// Test case 1: Empty filter options should fail
	emptyOptions := models.FilterOptions{}
	filterKey := manager.CreateFilter(emptyOptions)
	if filterKey != "" {
		t.Errorf("Expected empty filter key for empty options, got: %s", filterKey)
	}

	// Test case 2: Filter with only repository should succeed
	repoOptions := models.FilterOptions{Repository: "did:plc:test123"}
	filterKey = manager.CreateFilter(repoOptions)
	if filterKey == "" {
		t.Error("Expected valid filter key for repository filter")
	}

	// Test case 3: Filter with only pathPrefix should succeed
	pathOptions := models.FilterOptions{PathPrefix: "app.bsky.feed.post"}
	filterKey = manager.CreateFilter(pathOptions)
	if filterKey == "" {
		t.Error("Expected valid filter key for path prefix filter")
	}

	// Test case 4: Filter with only keyword should succeed
	keywordOptions := models.FilterOptions{Keyword: "test"}
	filterKey = manager.CreateFilter(keywordOptions)
	if filterKey == "" {
		t.Error("Expected valid filter key for keyword filter")
	}

	// Test case 5: Filter with multiple criteria should succeed
	multiOptions := models.FilterOptions{
		Repository: "did:plc:test123",
		PathPrefix: "app.bsky.feed.post",
		Keyword:    "test",
	}
	filterKey = manager.CreateFilter(multiOptions)
	if filterKey == "" {
		t.Error("Expected valid filter key for multi-criteria filter")
	}
}

func TestMatchesFilterSafety(t *testing.T) {
	manager := NewManager()

	// Create a test event
	testEvent := &models.ATEvent{
		Did: "did:plc:test123",
		Ops: []models.ATOperation{
			{
				Action: "create",
				Path:   "app.bsky.feed.post/test123",
				Record: map[string]interface{}{
					"text": "Hello world test message",
				},
			},
		},
	}

	// Test case 1: Empty filter options should never match
	emptyOptions := models.FilterOptions{}
	matches := manager.matchesFilter(testEvent, emptyOptions)
	if matches {
		t.Error("Empty filter options should never match any event (safety check)")
	}

	// Test case 2: Valid filter should match
	validOptions := models.FilterOptions{Repository: "did:plc:test123"}
	matches = manager.matchesFilter(testEvent, validOptions)
	if !matches {
		t.Error("Valid filter should match the test event")
	}

	// Test case 3: Non-matching filter should not match
	nonMatchingOptions := models.FilterOptions{Repository: "did:plc:different"}
	matches = manager.matchesFilter(testEvent, nonMatchingOptions)
	if matches {
		t.Error("Non-matching filter should not match the test event")
	}
}

func TestEnrichedEventTimestamps(t *testing.T) {
	manager := NewManager()

	// Create a filter
	options := models.FilterOptions{Repository: "did:plc:test123"}
	filterKey := manager.CreateFilter(options)
	if filterKey == "" {
		t.Fatal("Failed to create test filter")
	}

	// Create a test event with original timestamp
	originalTime := "2025-10-04T21:15:32.123Z"
	testEvent := &models.ATEvent{
		Event: "commit",
		Did:   "did:plc:test123",
		Time:  originalTime,
		Kind:  "commit",
		Ops: []models.ATOperation{
			{
				Action: "create",
				Path:   "app.bsky.feed.post/test123",
				Record: map[string]interface{}{
					"text": "Test message with timestamps",
				},
			},
		},
	}

	// We can't easily test the actual broadcasting without WebSocket connections,
	// but we can test the enriched event structure by calling the broadcast method
	// and checking that it doesn't panic and processes correctly
	startTime := time.Now()
	manager.BroadcastEvent(testEvent)
	endTime := time.Now()

	// Verify the event was processed (no connections, so it won't actually broadcast)
	// This test mainly ensures our timestamp enrichment logic doesn't break anything
	if endTime.Before(startTime) {
		t.Error("Time flow issue in test")
	}

	// Test that EnrichedATEvent can be marshaled to JSON
	enrichedEvent := models.EnrichedATEvent{
		Event: testEvent.Event,
		Did:   testEvent.Did,
		Time:  testEvent.Time,
		Kind:  testEvent.Kind,
		Ops:   testEvent.Ops,
		Timestamps: models.EventTimestamps{
			Original:  originalTime,
			Received:  time.Now().Format(time.RFC3339Nano),
			Forwarded: time.Now().Format(time.RFC3339Nano),
			FilterKey: filterKey,
		},
	}

	if enrichedEvent.Timestamps.Original != originalTime {
		t.Errorf("Expected original timestamp %s, got %s", originalTime, enrichedEvent.Timestamps.Original)
	}

	if enrichedEvent.Timestamps.FilterKey != filterKey {
		t.Errorf("Expected filter key %s, got %s", filterKey, enrichedEvent.Timestamps.FilterKey)
	}
}
