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
	mu            sync.RWMutex
	subscriptions map[string]*Subscription
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
		subscriptions: make(map[string]*Subscription),
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

// AddConnection adds a WebSocket connection to a filter subscription
func (m *Manager) AddConnection(filterKey string, conn *websocket.Conn) bool {
	m.mu.RLock()
	sub, exists := m.subscriptions[filterKey]
	m.mu.RUnlock()

	if !exists {
		log.Printf("❌ Attempted to connect to non-existent filter: %s", filterKey[:8]+"...")
		return false
	}

	sub.mu.Lock()
	sub.Connections[conn] = true
	connectionCount := len(sub.Connections)
	sub.mu.Unlock()

	log.Printf("🔌 Added connection to filter %s (total connections: %d)", filterKey[:8]+"...", connectionCount)
	return true
}

// RemoveConnection removes a WebSocket connection from a filter subscription
func (m *Manager) RemoveConnection(filterKey string, conn *websocket.Conn) {
	m.mu.RLock()
	sub, exists := m.subscriptions[filterKey]
	m.mu.RUnlock()

	if !exists {
		return
	}

	sub.mu.Lock()
	delete(sub.Connections, conn)
	connectionCount := len(sub.Connections)
	sub.mu.Unlock()

	log.Printf("🔌 Removed connection from filter %s (remaining connections: %d)", filterKey[:8]+"...", connectionCount)
}

// BroadcastEvent sends an event to all matching filter subscriptions
func (m *Manager) BroadcastEvent(event *models.ATEvent) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	matchCount := 0
	for _, sub := range m.subscriptions {
		if m.matchesFilter(event, sub.Options) {
			m.broadcastToSubscription(sub, event)
			matchCount++
		}
	}

	if matchCount > 0 {
		log.Printf("📡 Broadcasted event to %d matching filter(s) (did: %s...)", matchCount, event.Did[:20])
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
			if m.recordContainsKeyword(op.Record, options.Keyword) {
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

// recordContainsKeyword checks if a record contains the specified keyword
func (m *Manager) recordContainsKeyword(record interface{}, keyword string) bool {
	if record == nil {
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

	return strings.Contains(strings.ToLower(text), strings.ToLower(keyword))
}

// broadcastToSubscription sends an event to all connections in a subscription
func (m *Manager) broadcastToSubscription(sub *Subscription, event *models.ATEvent) {
	sub.mu.RLock()
	connections := make([]*websocket.Conn, 0, len(sub.Connections))
	for conn := range sub.Connections {
		connections = append(connections, conn)
	}
	sub.mu.RUnlock()

	if len(connections) == 0 {
		return
	}

	message := models.WSMessage{
		Type:      "event",
		Timestamp: time.Now(),
		Data:      event,
	}

	deadConnections := make([]*websocket.Conn, 0)

	for _, conn := range connections {
		if err := conn.WriteJSON(message); err != nil {
			log.Printf("⚠️  Failed to send message to connection: %v", err)
			deadConnections = append(deadConnections, conn)
		} else {
			// Log successful forwarding to WebSocket
			if len(event.Ops) > 0 {
				op := event.Ops[0] // Log first operation
				log.Printf("📤 Forwarded event to WebSocket: action=%s, path=%s (repo: %s...)",
					op.Action, op.Path, event.Did[8:20])
			} else {
				log.Printf("📤 Forwarded event to WebSocket (repo: %s...)", event.Did[8:20])
			}
		}
	}

	// Clean up dead connections
	if len(deadConnections) > 0 {
		sub.mu.Lock()
		for _, conn := range deadConnections {
			delete(sub.Connections, conn)
			if err := conn.Close(); err != nil {
				log.Printf("Failed to close dead connection: %v", err)
			}
		}
		sub.mu.Unlock()
		log.Printf("🧹 Cleaned up %d dead connections from filter %s", len(deadConnections), sub.FilterKey[:8]+"...")
	}
}

// GetStats returns statistics about the subscription manager
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalConnections := 0
	activeFilters := len(m.subscriptions)

	for _, sub := range m.subscriptions {
		sub.mu.RLock()
		totalConnections += len(sub.Connections)
		sub.mu.RUnlock()
	}

	return map[string]interface{}{
		"active_filters":    activeFilters,
		"total_connections": totalConnections,
		"uptime":            time.Since(time.Now()).String(), // This would be better tracked at startup
		"avg_connections":   float64(totalConnections) / float64(max(activeFilters, 1)),
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
