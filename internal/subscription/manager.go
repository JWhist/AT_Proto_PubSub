package subscription

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/JWhist/AT_Proto_PubSub/internal/models"
)

// Manager handles filter subscriptions and WebSocket connections
type Manager struct {
	mu               sync.RWMutex
	subscriptions    map[string]*Subscription
	maxConnections   int
	totalConnections int
}

// Subscription represents a filter with its associated WebSocket connections
type Subscription struct {
	FilterKey   string
	Options     models.FilterOptions
	CreatedAt   time.Time
	Connections map[*websocket.Conn]bool
	mu          sync.RWMutex
}

// NewManager creates a new subscription manager
func NewManager() *Manager {
	return &Manager{
		subscriptions:  make(map[string]*Subscription),
		maxConnections: 1000, // Default limit
	}
}

// NewManagerWithConfig creates a new subscription manager with configuration
func NewManagerWithConfig(maxConnections int) *Manager {
	return &Manager{
		subscriptions:  make(map[string]*Subscription),
		maxConnections: maxConnections,
	}
}

// CreateFilter creates a new filter subscription and returns a unique key
func (m *Manager) CreateFilter(options models.FilterOptions) string {
	// Validate that at least one filter criteria is provided
	if options.Repository == "" && options.PathPrefix == "" && options.Keyword == "" {
		log.Printf("❌ Rejected filter creation: no filter criteria provided")
		return "" // Return empty string to indicate failure
	}

	filterKey := generateFilterKey()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscriptions[filterKey] = &Subscription{
		FilterKey:   filterKey,
		Options:     options,
		CreatedAt:   time.Now(),
		Connections: make(map[*websocket.Conn]bool),
	}

	log.Printf("📝 Created filter %s with options: Repository=%s, PathPrefix=%s, Keyword=%s",
		filterKey[:8]+"...",
		getFilterDisplayValue(options.Repository),
		getFilterDisplayValue(options.PathPrefix),
		getFilterDisplayValue(options.Keyword))

	return filterKey
}

// GetSubscription returns a specific subscription by filter key
func (m *Manager) GetSubscription(filterKey string) (*models.FilterSubscription, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sub, exists := m.subscriptions[filterKey]
	if !exists {
		return nil, false
	}

	sub.mu.RLock()
	defer sub.mu.RUnlock()

	return &models.FilterSubscription{
		FilterKey:   sub.FilterKey,
		Options:     sub.Options,
		CreatedAt:   sub.CreatedAt,
		Connections: len(sub.Connections),
	}, true
}

// GetSubscriptions returns all current filter subscriptions
func (m *Manager) GetSubscriptions() []models.FilterSubscription {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var subs []models.FilterSubscription
	for _, sub := range m.subscriptions {
		sub.mu.RLock()
		subs = append(subs, models.FilterSubscription{
			FilterKey:   sub.FilterKey,
			Options:     sub.Options,
			CreatedAt:   sub.CreatedAt,
			Connections: len(sub.Connections),
		})
		sub.mu.RUnlock()
	}
	return subs
}

// ConnectionResult represents the result of trying to add a connection
type ConnectionResult struct {
	Success      bool
	ErrorMessage string
	ErrorCode    string
}

// AddConnection adds a WebSocket connection to a filter subscription
func (m *Manager) AddConnection(filterKey string, conn *websocket.Conn) bool {
	result := m.AddConnectionWithResult(filterKey, conn)
	return result.Success
}

// AddConnectionWithResult adds a WebSocket connection and returns detailed result
func (m *Manager) AddConnectionWithResult(filterKey string, conn *websocket.Conn) ConnectionResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if we've reached the maximum connection limit
	if m.totalConnections >= m.maxConnections {
		log.Printf("❌ Connection rejected: maximum connections (%d) reached", m.maxConnections)
		return ConnectionResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Maximum connections limit reached (%d/%d)", m.totalConnections, m.maxConnections),
			ErrorCode:    "MAX_CONNECTIONS_REACHED",
		}
	}

	sub, exists := m.subscriptions[filterKey]
	if !exists {
		log.Printf("❌ Attempted to connect to non-existent filter: %s", filterKey[:8]+"...")
		return ConnectionResult{
			Success:      false,
			ErrorMessage: "Invalid filter key",
			ErrorCode:    "INVALID_FILTER_KEY",
		}
	}

	sub.mu.Lock()
	sub.Connections[conn] = true
	connectionCount := len(sub.Connections)
	sub.mu.Unlock()

	m.totalConnections++

	log.Printf("🔌 Added connection to filter %s (filter connections: %d, total connections: %d/%d)",
		filterKey[:8]+"...", connectionCount, m.totalConnections, m.maxConnections)

	return ConnectionResult{
		Success: true,
	}
}

// RemoveConnection removes a WebSocket connection from a filter subscription
func (m *Manager) RemoveConnection(filterKey string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sub, exists := m.subscriptions[filterKey]
	if !exists {
		return
	}

	sub.mu.Lock()
	_, wasConnected := sub.Connections[conn]
	if wasConnected {
		delete(sub.Connections, conn)
		m.totalConnections--
	}
	connectionCount := len(sub.Connections)
	sub.mu.Unlock()

	if wasConnected {
		log.Printf("🔌 Removed connection from filter %s (filter connections: %d, total connections: %d/%d)",
			filterKey[:8]+"...", connectionCount, m.totalConnections, m.maxConnections)
	}
}

// BroadcastEvent sends an event to all matching filter subscriptions
func (m *Manager) BroadcastEvent(event *models.ATEvent) {
	receivedAt := time.Now() // Track when we received this event

	m.mu.RLock()
	defer m.mu.RUnlock()

	matchCount := 0
	for _, sub := range m.subscriptions {
		if m.matchesFilter(event, sub.Options) {
			m.broadcastToSubscription(sub, event, receivedAt)
			matchCount++
		}
	}

	if matchCount > 0 {
		didPreview := event.Did
		if len(didPreview) > 20 {
			didPreview = didPreview[:20] + "..."
		}
		log.Printf("📡 Broadcasted event to %d matching filter(s) (did: %s)", matchCount, didPreview)
	}
}

// matchesFilter checks if an event matches the filter criteria
func (m *Manager) matchesFilter(event *models.ATEvent, options models.FilterOptions) bool {
	// Safety check: if no filter criteria are set, reject all events
	// This prevents accidentally forwarding the entire firehose
	if options.Repository == "" && options.PathPrefix == "" && options.Keyword == "" {
		log.Printf("⚠️  Blocking event for filter with no criteria (safety check)")
		return false
	}

	// Repository filter (exact match on DID)
	if options.Repository != "" && event.Did != options.Repository {
		return false
	}

	// Path prefix filter
	if options.PathPrefix != "" {
		hasMatchingPath := false
		for _, op := range event.Ops {
			if strings.HasPrefix(op.Path, options.PathPrefix) {
				hasMatchingPath = true
				break
			}
		}
		if !hasMatchingPath {
			return false
		}
	}

	// Keyword filter - check in record content
	if options.Keyword != "" {
		hasMatchingKeyword := false
		for _, op := range event.Ops {
			if m.recordContainsKeywords(op.Record, options.Keyword) {
				hasMatchingKeyword = true
				break
			}
		}
		if !hasMatchingKeyword {
			return false
		}
	}

	return true
}

// recordContainsKeywords checks if a record contains any of the specified keywords (comma-separated)
func (m *Manager) recordContainsKeywords(record interface{}, keywords string) bool {
	if record == nil || keywords == "" {
		return false
	}

	// Convert record to JSON and parse text fields
	recordBytes, err := json.Marshal(record)
	if err != nil {
		return false
	}

	var recordContent models.RecordContent
	if err := json.Unmarshal(recordBytes, &recordContent); err != nil {
		return false
	}

	// Check various text fields
	text := recordContent.Text
	if text == "" {
		text = recordContent.Message
	}
	if text == "" {
		text = recordContent.Content
	}

	if text == "" {
		return false
	}

	// Split keywords by comma and check for any match
	keywordList := strings.Split(keywords, ",")
	textLower := strings.ToLower(text)

	for _, keyword := range keywordList {
		keyword = strings.TrimSpace(keyword) // Remove any surrounding whitespace
		if keyword != "" && strings.Contains(textLower, strings.ToLower(keyword)) {
			return true // Return true if any keyword matches
		}
	}

	return false
}

// recordContainsKeyword checks if a record contains the specified keyword (kept for compatibility)
func (m *Manager) recordContainsKeyword(record interface{}, keyword string) bool {
	return m.recordContainsKeywords(record, keyword)
}

// broadcastToSubscription sends an event to all connections in a subscription
func (m *Manager) broadcastToSubscription(sub *Subscription, event *models.ATEvent, receivedAt time.Time) {
	sub.mu.RLock()
	connections := make([]*websocket.Conn, 0, len(sub.Connections))
	for conn := range sub.Connections {
		connections = append(connections, conn)
	}
	sub.mu.RUnlock()

	if len(connections) == 0 {
		return
	}

	// Create enriched event with timestamp metadata
	forwardedAt := time.Now()
	enrichedEvent := models.EnrichedATEvent{
		Event: event.Event,
		Did:   event.Did,
		Time:  event.Time,
		Kind:  event.Kind,
		Ops:   event.Ops,
		Timestamps: models.EventTimestamps{
			Original:  event.Time,                           // Original firehose timestamp
			Received:  receivedAt.Format(time.RFC3339Nano),  // When we received from firehose
			Forwarded: forwardedAt.Format(time.RFC3339Nano), // When we forward to clients
			FilterKey: sub.FilterKey,                        // Which filter matched
		},
	}

	message := models.WSMessage{
		Type:      "event",
		Timestamp: forwardedAt,
		Data:      enrichedEvent,
	}

	deadConnections := make([]*websocket.Conn, 0)

	for _, conn := range connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("⚠️  Failed to send message to connection: %v", err)
			deadConnections = append(deadConnections, conn)
		} else {
			// Log successful forwarding to WebSocket with timing info
			didPreview := event.Did
			if len(didPreview) > 20 {
				didPreview = didPreview[8:20] + "..."
			} else if len(didPreview) > 8 {
				didPreview = didPreview[8:] + "..."
			}

			filterPreview := sub.FilterKey
			if len(filterPreview) > 8 {
				filterPreview = filterPreview[:8] + "..."
			}

			if len(event.Ops) > 0 {
				op := event.Ops[0] // Log first operation
				log.Printf("📤 Forwarded event to WebSocket: action=%s, path=%s (repo: %s) [filter: %s, forwarded: %s]",
					op.Action, op.Path, didPreview, filterPreview, forwardedAt.Format("15:04:05.000"))
			} else {
				log.Printf("📤 Forwarded event to WebSocket (repo: %s) [filter: %s, forwarded: %s]",
					didPreview, filterPreview, forwardedAt.Format("15:04:05.000"))
			}
		}
	}

	// Clean up dead connections
	if len(deadConnections) > 0 {
		sub.mu.Lock()
		removedCount := 0
		for _, conn := range deadConnections {
			if _, exists := sub.Connections[conn]; exists {
				delete(sub.Connections, conn)
				removedCount++
			}
			if err := conn.Close(); err != nil {
				log.Printf("Failed to close dead connection: %v", err)
			}
		}
		sub.mu.Unlock()

		// Update total connections count (need to get manager lock)
		m.mu.Lock()
		m.totalConnections -= removedCount
		m.mu.Unlock()

		log.Printf("🧹 Cleaned up %d dead connections from filter %s (total connections: %d/%d)",
			removedCount, sub.FilterKey[:8]+"...", m.totalConnections, m.maxConnections)
	}
}

// GetStats returns statistics about the subscription manager
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	activeFilters := len(m.subscriptions)
	connectionUtilization := float64(m.totalConnections) / float64(max(m.maxConnections, 1)) * 100

	return map[string]interface{}{
		"active_filters":         activeFilters,
		"total_connections":      m.totalConnections,
		"max_connections":        m.maxConnections,
		"connection_utilization": fmt.Sprintf("%.1f%%", connectionUtilization),
		"available_connections":  m.maxConnections - m.totalConnections,
		"uptime":                 time.Since(time.Now()).String(), // This would be better tracked at startup
		"avg_connections":        float64(m.totalConnections) / float64(max(activeFilters, 1)),
	}
}

// generateFilterKey creates a unique filter key
func generateFilterKey() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Failed to generate random bytes: %v", err)
		// Fallback to time-based key if random fails
		return hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return hex.EncodeToString(bytes)
}

// getFilterDisplayValue returns "ALL" for empty filters or the actual value
func getFilterDisplayValue(filter string) string {
	if filter == "" {
		return "ALL"
	}
	return filter
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
