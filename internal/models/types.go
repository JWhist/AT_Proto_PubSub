package models

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
