package firehose

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	"github.com/gorilla/websocket"

	"github.com/JWhist/AT_Proto_PubSub/internal/models"
)

// Client handles the AT Protocol firehose connection and filtering
type Client struct {
	filters       models.FilterOptions
	mutex         sync.RWMutex
	conn          *websocket.Conn
	eventCallback func(*models.ATEvent)
	callbackMu    sync.RWMutex
}

// NewClient creates a new firehose client instance
func NewClient() *Client {
	return &Client{
		filters: models.FilterOptions{},
	}
}

// UpdateFilters updates the filter options in a thread-safe manner
func (c *Client) UpdateFilters(newFilters models.FilterOptions) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.filters = newFilters
}

// GetFilters returns the current filter options in a thread-safe manner
func (c *Client) GetFilters() models.FilterOptions {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.filters
}

// SetEventCallback sets a callback function to be called for each received event
func (c *Client) SetEventCallback(callback func(*models.ATEvent)) {
	c.callbackMu.Lock()
	defer c.callbackMu.Unlock()
	c.eventCallback = callback
}

// getEventCallback safely gets the current event callback
func (c *Client) getEventCallback() func(*models.ATEvent) {
	c.callbackMu.RLock()
	defer c.callbackMu.RUnlock()
	return c.eventCallback
}

// Start begins the firehose connection and event processing
func (c *Client) Start(ctx context.Context) error {
	filters := c.GetFilters()
	fmt.Println("Starting AT Protocol Firehose Filter Server...")
	fmt.Println("Initial Filters:")
	fmt.Printf("  Repository: %s\n", getFilterString(filters.Repository))
	fmt.Printf("  Path Prefix: %s\n", getFilterString(filters.PathPrefix))
	fmt.Printf("  Keyword: %s\n", getFilterString(filters.Keyword))
	fmt.Println("Connecting to firehose...")

	// Connect to the AT Protocol firehose using the proper indigo library
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial("wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to firehose: %w", err)
	}
	c.conn = conn
	fmt.Println("✅ Successfully connected to firehose!")

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		fmt.Println("\nShutting down firehose connection...")
		if err := c.conn.Close(); err != nil {
			fmt.Printf("Error closing connection: %v\n", err)
		}
	}()

	fmt.Println("📡 Listening for firehose messages...")

	// Set up AT Protocol event callbacks
	rsc := &events.RepoStreamCallbacks{
		RepoCommit: func(evt *atproto.SyncSubscribeRepos_Commit) error {
			return c.handleRepoCommit(evt)
		},
	}

	// Create scheduler and handle the repo stream
	sched := sequential.NewScheduler("atp-filter", rsc.EventHandler)
	logger := slog.Default()
	return events.HandleRepoStream(ctx, conn, sched, logger)
}

// handleRepoCommit processes repo commit events from the firehose
func (c *Client) handleRepoCommit(evt *atproto.SyncSubscribeRepos_Commit) error {
	// Convert to our internal event format
	atEvent := models.ATEvent{
		Did:  evt.Repo,
		Time: evt.Time,
		Kind: "commit",
	}

	// Convert operations
	for _, op := range evt.Ops {
		atOp := models.ATOperation{
			Action: op.Action,
			Path:   op.Path,
		}
		if op.Cid != nil {
			atOp.Cid = op.Cid.String()
		}
		atEvent.Ops = append(atEvent.Ops, atOp)
	}

	// Send event to callback (subscription manager) if set
	if callback := c.getEventCallback(); callback != nil {
		callback(&atEvent)
	}

	// Process the event with legacy filtering (for backward compatibility)
	c.handleEvent(atEvent)
	return nil
}

// handleEvent processes an AT Protocol event
func (c *Client) handleEvent(event models.ATEvent) {
	currentFilters := c.GetFilters()

	// Filter by repository if specified (repository filter should match DID)
	if currentFilters.Repository != "" && event.Did != currentFilters.Repository {
		return
	}

	// Process operations in the event
	for _, op := range event.Ops {
		if op.Action == "create" || op.Action == "update" {
			// Extract collection from path (e.g., "app.bsky.feed.post/abc123" -> "app.bsky.feed.post")
			pathParts := strings.Split(op.Path, "/")
			if len(pathParts) > 0 {
				op.Collection = pathParts[0]
				if len(pathParts) > 1 {
					op.Rkey = pathParts[1]
				}
			}

			// Filter by path prefix if specified
			if currentFilters.PathPrefix != "" && !strings.HasPrefix(op.Collection, currentFilters.PathPrefix) {
				continue
			}

			if c.matchesFilter(op, currentFilters) {
				c.logEvent(event, op)
			}
		}
	}
}

// matchesFilter checks if an operation matches the filter criteria
func (c *Client) matchesFilter(op models.ATOperation, filters models.FilterOptions) bool {
	if op.Record == nil {
		return false
	}

	// Convert record to JSON and then to RecordContent
	recordBytes, err := json.Marshal(op.Record)
	if err != nil {
		return false
	}

	var record models.RecordContent
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		return false
	}

	// Get text content from various possible fields
	text := record.Text
	if text == "" {
		text = record.Message
	}
	if text == "" {
		text = record.Content
	}

	// If no keyword filter, match all records with text
	if filters.Keyword == "" {
		return text != ""
	}

	// Check if text contains keyword (case-insensitive)
	if text != "" {
		return strings.Contains(strings.ToLower(text), strings.ToLower(filters.Keyword))
	}

	return false
}

// logEvent logs a matching event to the console
func (c *Client) logEvent(event models.ATEvent, op models.ATOperation) {
	timestamp := time.Now().Format(time.RFC3339)

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("[%s] %s event\n", timestamp, strings.ToUpper(op.Action))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Repository: %s\n", event.Did)
	fmt.Printf("Collection: %s\n", op.Collection)
	fmt.Printf("Record Key: %s\n", op.Rkey)
	fmt.Printf("URI: at://%s/%s/%s\n", event.Did, op.Collection, op.Rkey)

	if op.Record != nil {
		// Convert record to JSON and then to RecordContent for better parsing
		recordBytes, err := json.Marshal(op.Record)
		if err == nil {
			var record models.RecordContent
			if err := json.Unmarshal(recordBytes, &record); err == nil {
				// Log text content
				text := record.Text
				if text == "" {
					text = record.Message
				}
				if text == "" {
					text = record.Content
				}
				if text != "" {
					fmt.Printf("Text: %s\n", text)
				}

				// Log other relevant fields
				if record.Reply != nil {
					replyJSON, _ := json.Marshal(record.Reply)
					fmt.Printf("Reply to: %s\n", string(replyJSON))
				}

				if len(record.Langs) > 0 {
					langsJSON, _ := json.Marshal(record.Langs)
					fmt.Printf("Languages: %s\n", string(langsJSON))
				}
			}
		}
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
}

// getFilterString returns "ALL" if filter is empty, otherwise returns the filter value
func getFilterString(filter string) string {
	if filter == "" {
		return "ALL"
	}
	return filter
}
