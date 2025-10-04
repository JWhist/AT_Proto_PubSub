// Package main demonstrates the firehose filter logic
// This can be run independently to test filtering without connecting to the live firehose
// Run with: go run examples/filter-demo.go
package main

import (
	"fmt"
	"strings"
)

// ExampleEvent represents a simplified AT Protocol event for demonstration
type ExampleEvent struct {
	Did        string `json:"did"`
	Collection string `json:"collection"`
	Text       string `json:"text"`
}

// Example events that might come from the firehose
var exampleEvents = []ExampleEvent{
	{
		Did:        "did:plc:abc123",
		Collection: "app.bsky.feed.post",
		Text:       "Hello world! This is a test post.",
	},
	{
		Did:        "did:plc:xyz789",
		Collection: "app.bsky.feed.post",
		Text:       "Another post without the keyword.",
	},
	{
		Did:        "did:plc:abc123",
		Collection: "app.bsky.feed.post",
		Text:       "Testing the firehose filter.",
	},
	{
		Did:        "did:plc:def456",
		Collection: "app.bsky.feed.like",
		Text:       "I love this test!",
	},
}

// filterEvents filters events based on repository and keyword criteria
func filterEvents(events []ExampleEvent, repositoryFilter, keywordFilter string) []ExampleEvent {
	var filtered []ExampleEvent

	for _, event := range events {
		// Filter by repository if specified
		if repositoryFilter != "" && event.Did != repositoryFilter {
			continue
		}

		// Filter by keyword if specified
		if keywordFilter != "" && event.Text != "" {
			if !strings.Contains(strings.ToLower(event.Text), strings.ToLower(keywordFilter)) {
				continue
			}
		} else if keywordFilter == "" && event.Text == "" {
			// If no keyword filter, skip events without text
			continue
		}

		filtered = append(filtered, event)
	}

	return filtered
}

// printEvents prints a slice of events in a readable format
func printEvents(events []ExampleEvent) {
	for i, event := range events {
		fmt.Printf("  [%d] DID: %s\n", i+1, event.Did)
		fmt.Printf("      Collection: %s\n", event.Collection)
		fmt.Printf("      Text: %s\n", event.Text)
		if i < len(events)-1 {
			fmt.Println()
		}
	}
	if len(events) == 0 {
		fmt.Println("  (no matching events)")
	}
}

func main() {
	fmt.Println("Example of how the firehose filter server works")
	fmt.Println("This demonstrates the filtering logic without connecting to the live firehose")
	fmt.Println()

	fmt.Println("Example 1: No filters (all events with text)")
	result1 := filterEvents(exampleEvents, "", "")
	printEvents(result1)
	fmt.Println()

	fmt.Println("Example 2: Filter by keyword \"test\"")
	result2 := filterEvents(exampleEvents, "", "test")
	printEvents(result2)
	fmt.Println()

	fmt.Println("Example 3: Filter by repository \"did:plc:abc123\"")
	result3 := filterEvents(exampleEvents, "did:plc:abc123", "")
	printEvents(result3)
	fmt.Println()

	fmt.Println("Example 4: Filter by repository \"did:plc:abc123\" and keyword \"test\"")
	result4 := filterEvents(exampleEvents, "did:plc:abc123", "test")
	printEvents(result4)
	fmt.Println()

	fmt.Println("Run 'go run main.go' to start the actual firehose filter server!")
}
