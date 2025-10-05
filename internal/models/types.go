package models

import "time"

// FilterOptions represents the filter options that can be set via API
type FilterOptions struct {
	Repository string `json:"repository"`
	PathPrefix string `json:"pathPrefix"`
	Keyword    string `json:"keyword"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// FilterUpdateRequest represents the request body for updating filters
type FilterUpdateRequest struct {
	Repository *string `json:"repository,omitempty"`
	PathPrefix *string `json:"pathPrefix,omitempty"`
	Keyword    *string `json:"keyword,omitempty"`
}

// ATEvent represents an AT Protocol event from the firehose
type ATEvent struct {
	Event string        `json:"event"`
	Did   string        `json:"did"`
	Time  string        `json:"time"`
	Kind  string        `json:"kind"`
	Ops   []ATOperation `json:"ops"`
}

// EnrichedATEvent represents an AT Protocol event with additional timestamp metadata
type EnrichedATEvent struct {
	// Original AT Protocol event data
	Event string        `json:"event"`
	Did   string        `json:"did"`
	Time  string        `json:"time"` // Original firehose timestamp
	Kind  string        `json:"kind"`
	Ops   []ATOperation `json:"ops"`

	// Additional timestamp metadata
	Timestamps EventTimestamps `json:"timestamps"`
}

// EventTimestamps contains various timestamps for event lifecycle tracking
type EventTimestamps struct {
	Original  string `json:"original"`  // Original timestamp from AT Protocol firehose
	Received  string `json:"received"`  // When we received the event from firehose
	Forwarded string `json:"forwarded"` // When we forward to WebSocket clients
	FilterKey string `json:"filterKey"` // Which filter matched this event
}

// ATOperation represents an operation within an AT Protocol event
type ATOperation struct {
	Action     string      `json:"action"`
	Path       string      `json:"path"`
	Collection string      `json:"collection"`
	Rkey       string      `json:"rkey"`
	Record     interface{} `json:"record,omitempty"`
	Cid        string      `json:"cid,omitempty"`
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

// WebSocket subscription models

// FilterSubscription represents a filter subscription with connection info
type FilterSubscription struct {
	FilterKey   string        `json:"filterKey"`
	Options     FilterOptions `json:"options"`
	CreatedAt   time.Time     `json:"createdAt"`
	Connections int           `json:"connections"`
}

// CreateFilterRequest represents the request body for creating a new filter subscription
type CreateFilterRequest struct {
	Options FilterOptions `json:"options"`
}

// CreateFilterResponse represents the response when creating a filter subscription
type CreateFilterResponse struct {
	FilterKey string        `json:"filterKey"`
	Options   FilterOptions `json:"options"`
	CreatedAt time.Time     `json:"createdAt"`
}

// WSMessage represents a WebSocket message sent to clients
type WSMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}
