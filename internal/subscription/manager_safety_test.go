package subscription

import (
	"testing"

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