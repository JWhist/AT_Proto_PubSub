package firehose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/events/schedulers/sequential"
	"github.com/fxamacker/cbor/v2"
	"github.com/gorilla/websocket"
	carv2 "github.com/ipld/go-car/v2"

	"github.com/JWhist/AT_Proto_PubSub/internal/config"
	"github.com/JWhist/AT_Proto_PubSub/internal/models"
)

// Client handles the AT Protocol firehose connection and filtering
type Client struct {
	filters       models.FilterOptions
	mutex         sync.RWMutex
	conn          *websocket.Conn
	eventCallback func(*models.ATEvent)
	callbackMu    sync.RWMutex
	config        *config.Config
}

// NewClient creates a new firehose client instance
func NewClient() *Client {
	return &Client{
		filters: models.FilterOptions{},
	}
}

// NewClientWithConfig creates a new firehose client instance with configuration
func NewClientWithConfig(cfg *config.Config) *Client {
	return &Client{
		filters: models.FilterOptions{},
		config:  cfg,
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

	// Get firehose URL from config or use default
	firehoseURL := "wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos"
	if c.config != nil && c.config.Firehose.URL != "" {
		firehoseURL = c.config.Firehose.URL
	}

	// Connect to the AT Protocol firehose using the proper indigo library
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(firehoseURL, nil)
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

	// Process CAR blocks to extract records
	if len(evt.Blocks) > 0 {
		// Decode CAR blocks to extract records
		records, err := c.decodeCarBlocks(evt.Blocks)
		if err != nil {
			// Silently continue on CAR decode errors
			records = make(map[string]interface{})
		}

		// Convert operations with decoded records
		for _, op := range evt.Ops {
			atOp := models.ATOperation{
				Action: op.Action,
				Path:   op.Path,
			}
			if op.Cid != nil {
				atOp.Cid = op.Cid.String()

				// Try to find the corresponding record for this CID
				if record, exists := records[op.Cid.String()]; exists {
					atOp.Record = record
				}
			}

			// Extract collection from path (e.g., "app.bsky.feed.post/abc123" -> "app.bsky.feed.post")
			pathParts := strings.Split(op.Path, "/")
			if len(pathParts) > 0 {
				atOp.Collection = pathParts[0]
				if len(pathParts) > 1 {
					atOp.Rkey = pathParts[1]
				}
			}

			atEvent.Ops = append(atEvent.Ops, atOp)
		}
	} else {
		// Fallback for operations without blocks
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
	}

	// Send event to callback (subscription manager) if set
	if callback := c.getEventCallback(); callback != nil {
		callback(&atEvent)
	}

	// Process the event with legacy filtering (for backward compatibility)
	c.handleEvent(atEvent)
	return nil
}

// decodeCarBlocks decodes CAR (Content Addressable Archive) blocks and extracts records
func (c *Client) decodeCarBlocks(carData []byte) (map[string]interface{}, error) {
	records := make(map[string]interface{})

	// Create a reader from the CAR data
	carReader := bytes.NewReader(carData)

	// Read the CAR file
	blockReader, err := carv2.NewBlockReader(carReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create CAR block reader: %w", err)
	}

	// Iterate through all blocks in the CAR file
	for {
		block, err := blockReader.Next()
		if err != nil {
			// End of blocks
			break
		}

		// Try to decode the block data as CBOR
		var record interface{}
		if err := cbor.Unmarshal(block.RawData(), &record); err != nil {
			// Skip blocks that aren't valid CBOR records
			continue
		}

		// Convert CBOR map to string-keyed map for easier handling
		convertedRecord := c.convertCBORToStringMap(record)

		// Store the record using the CID as the key
		cidStr := block.Cid().String()
		records[cidStr] = convertedRecord
	}

	return records, nil
}

// convertCBORToStringMap converts CBOR interface{} maps to string-keyed maps
func (c *Client) convertCBORToStringMap(data interface{}) interface{} {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			if keyStr, ok := key.(string); ok {
				result[keyStr] = c.convertCBORToStringMap(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = c.convertCBORToStringMap(item)
		}
		return result
	default:
		return v
	}
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

			// Check if matches filter (for legacy compatibility)
			c.matchesFilter(op, currentFilters)
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

	// Check if text contains any of the keywords (comma-separated, case-insensitive)
	if text != "" {
		// Split keywords by comma and check for any match
		keywordList := strings.Split(filters.Keyword, ",")
		textLower := strings.ToLower(text)

		for _, keyword := range keywordList {
			keyword = strings.TrimSpace(keyword) // Remove any surrounding whitespace
			if keyword != "" && strings.Contains(textLower, strings.ToLower(keyword)) {
				return true // Return true if any keyword matches
			}
		}
	}

	return false
}

// getFilterString returns "ALL" if filter is empty, otherwise returns the filter value
func getFilterString(filter string) string {
	if filter == "" {
		return "ALL"
	}
	return filter
}
