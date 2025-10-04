package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// FilterOptions represents the command-line filter options
type FilterOptions struct {
	Repository string
	Keyword    string
}

// ATEvent represents an AT Protocol event from the firehose
type ATEvent struct {
	Event string `json:"event"`
	Did   string `json:"did"`
	Time  string `json:"time"`
	Kind  string `json:"kind"`
	Ops   []struct {
		Action     string      `json:"action"`
		Path       string      `json:"path"`
		Collection string      `json:"collection"`
		Rkey       string      `json:"rkey"`
		Record     interface{} `json:"record,omitempty"`
		Cid        string      `json:"cid,omitempty"`
	} `json:"ops"`
}

// RecordContent represents the content of an AT Protocol record
type RecordContent struct {
	Text    string                 `json:"text"`
	Message string                 `json:"message"`
	Content string                 `json:"content"`
	Reply   map[string]interface{} `json:"reply,omitempty"`
	Langs   []string               `json:"langs,omitempty"`
	Type    string                 `json:"$type"`
	Created string                 `json:"createdAt"`
}

// FirehoseFilterServer handles the AT Protocol firehose connection and filtering
type FirehoseFilterServer struct {
	filters FilterOptions
	conn    *websocket.Conn
}

// NewFirehoseFilterServer creates a new server instance
func NewFirehoseFilterServer(filters FilterOptions) *FirehoseFilterServer {
	return &FirehoseFilterServer{
		filters: filters,
	}
}

// Start begins the firehose connection and event processing
func (s *FirehoseFilterServer) Start(ctx context.Context) error {
	fmt.Println("Starting AT Protocol Firehose Filter Server...")
	fmt.Println("Filters:")
	fmt.Printf("  Repository: %s\n", getFilterString(s.filters.Repository))
	fmt.Printf("  Keyword: %s\n", getFilterString(s.filters.Keyword))
	fmt.Println("Connecting to firehose...")

	// Connect to the AT Protocol firehose
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial("wss://bsky.network/xrpc/com.atproto.sync.subscribeRepos", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to firehose: %w", err)
	}
	s.conn = conn

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		fmt.Println("\nShutting down...")
		s.conn.Close()
	}()

	// Read messages from the WebSocket
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return err
			}

			// Handle the message
			s.handleMessage(message)
		}
	}
}

// handleMessage processes incoming messages from the firehose
func (s *FirehoseFilterServer) handleMessage(message []byte) {
	// Try to parse as JSON first (some messages might be JSON)
	var event ATEvent
	if err := json.Unmarshal(message, &event); err != nil {
		// If JSON parsing fails, this might be a CAR file or binary data
		// For now, we'll skip non-JSON messages
		return
	}

	// Process the event
	s.handleEvent(event)
}

// handleEvent processes an AT Protocol event
func (s *FirehoseFilterServer) handleEvent(event ATEvent) {
	// Filter by repository if specified
	if s.filters.Repository != "" && event.Did != s.filters.Repository {
		return
	}

	// Process operations in the event
	for _, op := range event.Ops {
		if op.Action == "create" || op.Action == "update" {
			if s.matchesFilter(op) {
				s.logEvent(event, op)
			}
		}
	}
}

// matchesFilter checks if an operation matches the filter criteria
func (s *FirehoseFilterServer) matchesFilter(op struct {
	Action     string      `json:"action"`
	Path       string      `json:"path"`
	Collection string      `json:"collection"`
	Rkey       string      `json:"rkey"`
	Record     interface{} `json:"record,omitempty"`
	Cid        string      `json:"cid,omitempty"`
}) bool {
	if op.Record == nil {
		return false
	}

	// Convert record to JSON and then to RecordContent
	recordBytes, err := json.Marshal(op.Record)
	if err != nil {
		return false
	}

	var record RecordContent
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
	if s.filters.Keyword == "" {
		return text != ""
	}

	// Check if text contains keyword (case-insensitive)
	if text != "" {
		return strings.Contains(strings.ToLower(text), strings.ToLower(s.filters.Keyword))
	}

	return false
}

// logEvent logs a matching event to the console
func (s *FirehoseFilterServer) logEvent(event ATEvent, op struct {
	Action     string      `json:"action"`
	Path       string      `json:"path"`
	Collection string      `json:"collection"`
	Rkey       string      `json:"rkey"`
	Record     interface{} `json:"record,omitempty"`
	Cid        string      `json:"cid,omitempty"`
}) {
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
			var record RecordContent
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

// parseArgs parses command-line arguments and returns FilterOptions
func parseArgs() FilterOptions {
	var filters FilterOptions
	var showHelp bool

	flag.StringVar(&filters.Repository, "repository", "", "Filter by repository DID")
	flag.StringVar(&filters.Repository, "r", "", "Filter by repository DID (shorthand)")
	flag.StringVar(&filters.Keyword, "keyword", "", "Filter by keyword in text")
	flag.StringVar(&filters.Keyword, "k", "", "Filter by keyword in text (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showHelp, "h", false, "Show help message (shorthand)")

	flag.Parse()

	if showHelp {
		fmt.Println("AT Protocol Firehose Filter Server")
		fmt.Println()
		fmt.Println("Usage: go run main.go [options]")
		fmt.Println("   or: ./atp-test [options]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  -r, -repository <repo>    Filter by repository DID")
		fmt.Println("  -k, -keyword <keyword>    Filter by keyword in text")
		fmt.Println("  -h, -help                 Show this help message")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  go run main.go -keyword hello")
		fmt.Println("  go run main.go -repository did:plc:abc123 -keyword test")
		fmt.Println("  ./atp-test -k \"hello world\"")
		os.Exit(0)
	}

	return filters
}

func main() {
	// Parse command-line arguments
	filters := parseArgs()

	// Create server instance
	server := NewFirehoseFilterServer(filters)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	// Start the server
	if err := server.Start(ctx); err != nil {
		if err == context.Canceled {
			// Expected shutdown
			return
		}
		log.Fatalf("Server error: %v", err)
	}
}
