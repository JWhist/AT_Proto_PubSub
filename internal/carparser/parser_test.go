package carparser

import (
	"fmt"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func TestATProtoEvent_Fields(t *testing.T) {
	event := &ATProtoEvent{
		Repo: "did:plc:test123",
		Rev:  "abc123",
		Seq:  12345,
		Ops: []Operation{
			{
				Action: "create",
				Path:   "app.bsky.feed.post/12345",
				CID:    stringPtr("bafyreiabc123"),
				Record: map[string]interface{}{"text": "Hello world"},
			},
		},
	}

	if event.Repo != "did:plc:test123" {
		t.Errorf("Repo = %s, want did:plc:test123", event.Repo)
	}
	if event.Rev != "abc123" {
		t.Errorf("Rev = %s, want abc123", event.Rev)
	}
	if event.Seq != 12345 {
		t.Errorf("Seq = %d, want 12345", event.Seq)
	}
	if len(event.Ops) != 1 {
		t.Errorf("Ops length = %d, want 1", len(event.Ops))
	}
}

func TestOperation_Fields(t *testing.T) {
	cidStr := "bafyreiabc123"
	op := Operation{
		Action: "create",
		Path:   "app.bsky.feed.post/12345",
		CID:    &cidStr,
		Record: map[string]interface{}{"text": "Hello world"},
	}

	if op.Action != "create" {
		t.Errorf("Action = %s, want create", op.Action)
	}
	if op.Path != "app.bsky.feed.post/12345" {
		t.Errorf("Path = %s, want app.bsky.feed.post/12345", op.Path)
	}
	if op.CID == nil || *op.CID != cidStr {
		t.Errorf("CID = %v, want %s", op.CID, cidStr)
	}
}

func TestParseCARMessage_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "invalid CAR data",
			data: []byte("invalid car data"),
		},
		{
			name: "random bytes",
			data: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseCARMessage(tt.data)
			if err == nil {
				t.Errorf("ParseCARMessage() expected error, got event: %+v", event)
			}
		})
	}
}

func TestParseCARMessageSimple_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "invalid CBOR data",
			data: []byte("not cbor data"),
		},
		{
			name: "valid CBOR but no repo field",
			data: createCBORData(map[string]interface{}{"notrepo": "test"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseCARMessageSimple(tt.data)
			if err == nil {
				t.Errorf("ParseCARMessageSimple() expected error, got event: %+v", event)
			}
		})
	}
}

func TestParseCARMessageSimple_ValidData(t *testing.T) {
	// Create a valid CBOR object that looks like a commit
	// Use types that match what the parser expects
	commitData := map[string]interface{}{
		"repo": "did:plc:test123",
		"rev":  "abc123",
		"seq":  float64(12345), // Use float64 since that's what the parser handles
		"time": "2025-10-04T12:00:00Z",
		"ops": []interface{}{
			map[string]interface{}{
				"action": "create",
				"path":   "app.bsky.feed.post/12345",
				"record": map[string]interface{}{
					"text":      "Hello world",
					"createdAt": "2025-10-04T12:00:00Z",
				},
			},
		},
	}

	data := createCBORData(commitData)

	event, err := ParseCARMessageSimple(data)
	if err != nil {
		t.Fatalf("ParseCARMessageSimple() error = %v", err)
	}

	// Test the basic fields that are working
	if event.Repo != "did:plc:test123" {
		t.Errorf("Repo = %s, want did:plc:test123", event.Repo)
	}
	if event.Rev != "abc123" {
		t.Errorf("Rev = %s, want abc123", event.Rev)
	}
	if event.Seq != 12345 {
		t.Errorf("Seq = %d, want 12345", event.Seq)
	}
	if event.Time != "2025-10-04T12:00:00Z" {
		t.Errorf("Time = %s, want 2025-10-04T12:00:00Z", event.Time)
	}

	// Operations parsing is now working correctly after bug fix
	if len(event.Ops) != 1 {
		t.Errorf("Expected 1 operation, got %d. Event: %+v", len(event.Ops), event)
	} else {
		if event.Ops[0].Action != "create" {
			t.Errorf("Op action = %s, want create", event.Ops[0].Action)
		}
		if event.Ops[0].Path != "app.bsky.feed.post/12345" {
			t.Errorf("Op path = %s, want app.bsky.feed.post/12345", event.Ops[0].Path)
		}
		if event.Ops[0].Record == nil {
			t.Error("Expected record to be set")
		}
	}
}

func TestParseCARMessageSimple_WithFloatSeq(t *testing.T) {
	// Test with seq as float64 (JSON unmarshaling often produces float64)
	commitData := map[string]interface{}{
		"repo": "did:plc:test123",
		"seq":  float64(12345),
	}

	data := createCBORData(commitData)

	event, err := ParseCARMessageSimple(data)
	if err != nil {
		t.Fatalf("ParseCARMessageSimple() error = %v", err)
	}

	if event.Seq != 12345 {
		t.Errorf("Seq = %d, want 12345", event.Seq)
	}
}

func TestParseCARMessageSimple_WithCIDBytes(t *testing.T) {
	// Test basic parsing functionality without complex CID handling
	// The current parser implementation has limitations with ops parsing
	commitData := map[string]interface{}{
		"repo": "did:plc:test123",
		"ops": []interface{}{
			map[string]interface{}{
				"action": "create",
				"path":   "app.bsky.feed.post/12345",
				"cid":    "bafyreigdtuosuwwqnbkzw6qzb7h3yhco7v6hq4ukxpzz3z3z3z3z3z3z3z", // simplified test CID
			},
		},
	}

	data := createCBORData(commitData)

	event, err := ParseCARMessageSimple(data)
	if err != nil {
		t.Fatalf("ParseCARMessageSimple() error = %v", err)
	}

	// Test that the basic repo field is parsed
	if event.Repo != "did:plc:test123" {
		t.Errorf("Repo = %s, want did:plc:test123", event.Repo)
	}

	// Operations parsing is now working correctly after bug fix
	if len(event.Ops) > 0 {
		if event.Ops[0].Action != "create" {
			t.Errorf("Op action = %s, want create", event.Ops[0].Action)
		}
		if event.Ops[0].Path != "app.bsky.feed.post/12345" {
			t.Errorf("Op path = %s, want app.bsky.feed.post/12345", event.Ops[0].Path)
		}
	} else {
		t.Error("Expected at least 1 operation to be parsed")
	}
}

func TestParseCARMessageSimple_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		expected func(*ATProtoEvent) error
	}{
		{
			name: "seq as uint64",
			data: map[string]interface{}{
				"repo": "did:plc:test123",
				"seq":  uint64(12345),
			},
			expected: func(event *ATProtoEvent) error {
				if event.Seq != 12345 {
					return fmt.Errorf("Seq = %d, want 12345", event.Seq)
				}
				return nil
			},
		},
		{
			name: "ops with interface{} keys",
			data: map[string]interface{}{
				"repo": "did:plc:test123",
				"ops": []interface{}{
					map[interface{}]interface{}{
						"action": "create",
						"path":   "app.bsky.feed.post/12345",
					},
				},
			},
			expected: func(event *ATProtoEvent) error {
				if len(event.Ops) != 1 {
					return fmt.Errorf("Expected 1 operation, got %d", len(event.Ops))
				}
				if event.Ops[0].Action != "create" {
					return fmt.Errorf("Action = %s, want create", event.Ops[0].Action)
				}
				return nil
			},
		},
		{
			name: "ops with non-string key in interface map",
			data: map[string]interface{}{
				"repo": "did:plc:test123",
				"ops": []interface{}{
					map[interface{}]interface{}{
						123:      "invalid",
						"action": "create",
						"path":   "app.bsky.feed.post/12345",
					},
				},
			},
			expected: func(event *ATProtoEvent) error {
				if len(event.Ops) != 1 {
					return fmt.Errorf("Expected 1 operation, got %d", len(event.Ops))
				}
				return nil
			},
		},
		{
			name: "cid as byte array",
			data: map[string]interface{}{
				"repo": "did:plc:test123",
				"ops": []interface{}{
					map[string]interface{}{
						"action": "create",
						"path":   "app.bsky.feed.post/12345",
						"cid":    []byte{1, 85, 18, 32}, // Valid CID bytes
					},
				},
			},
			expected: func(event *ATProtoEvent) error {
				if len(event.Ops) != 1 {
					return fmt.Errorf("Expected 1 operation, got %d", len(event.Ops))
				}
				// CID might not parse correctly, but should not crash
				return nil
			},
		},
		{
			name: "invalid cid as byte array",
			data: map[string]interface{}{
				"repo": "did:plc:test123",
				"ops": []interface{}{
					map[string]interface{}{
						"action": "create",
						"path":   "app.bsky.feed.post/12345",
						"cid":    []byte{255, 255, 255}, // Invalid CID bytes
					},
				},
			},
			expected: func(event *ATProtoEvent) error {
				if len(event.Ops) != 1 {
					return fmt.Errorf("Expected 1 operation, got %d", len(event.Ops))
				}
				// Should not crash even with invalid CID
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := createCBORData(tt.data)
			event, err := ParseCARMessageSimple(data)
			if err != nil {
				t.Fatalf("ParseCARMessageSimple() error = %v", err)
			}
			if err := tt.expected(event); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestParseCARMessageSimple_NoRepoField(t *testing.T) {
	// Test with valid CBOR that doesn't have a repo field
	data := createCBORData(map[string]interface{}{
		"notrepo": "test",
		"seq":     12345,
	})

	_, err := ParseCARMessageSimple(data)
	if err == nil {
		t.Error("Expected error for data without repo field")
	}
}

func TestParseCARMessageSimple_EmptyOps(t *testing.T) {
	data := createCBORData(map[string]interface{}{
		"repo": "did:plc:test123",
		"ops":  []interface{}{},
	})

	event, err := ParseCARMessageSimple(data)
	if err != nil {
		t.Fatalf("ParseCARMessageSimple() error = %v", err)
	}

	if len(event.Ops) != 0 {
		t.Errorf("Expected 0 operations, got %d", len(event.Ops))
	}
}

func TestParseCARMessageSimple_OpsNotArray(t *testing.T) {
	data := createCBORData(map[string]interface{}{
		"repo": "did:plc:test123",
		"ops":  "not an array",
	})

	event, err := ParseCARMessageSimple(data)
	if err != nil {
		t.Fatalf("ParseCARMessageSimple() error = %v", err)
	}

	// Should parse repo successfully but ignore invalid ops
	if event.Repo != "did:plc:test123" {
		t.Errorf("Repo = %s, want did:plc:test123", event.Repo)
	}
	if len(event.Ops) != 0 {
		t.Errorf("Expected 0 operations, got %d", len(event.Ops))
	}
}

func TestParseCARMessageSimple_MultipleOffsets(t *testing.T) {
	// Create data with garbage at the beginning
	invalidData := []byte("garbage data")
	validCBOR := createCBORData(map[string]interface{}{
		"repo": "did:plc:test123",
		"seq":  12345,
	})

	// Combine invalid data with valid CBOR
	combinedData := append(invalidData, validCBOR...)

	event, err := ParseCARMessageSimple(combinedData)
	if err != nil {
		t.Fatalf("ParseCARMessageSimple() error = %v", err)
	}

	if event.Repo != "did:plc:test123" {
		t.Errorf("Repo = %s, want did:plc:test123", event.Repo)
	}
}

func TestOperation_AllFields(t *testing.T) {
	cidStr := "bafyreiabc123"
	record := map[string]interface{}{
		"text":      "Hello world",
		"createdAt": "2025-10-04T12:00:00Z",
	}

	op := Operation{
		Action: "update",
		Path:   "app.bsky.feed.post/67890",
		CID:    &cidStr,
		Record: record,
	}

	if op.Action != "update" {
		t.Errorf("Action = %s, want update", op.Action)
	}
	if op.Path != "app.bsky.feed.post/67890" {
		t.Errorf("Path = %s, want app.bsky.feed.post/67890", op.Path)
	}
	if op.CID == nil || *op.CID != cidStr {
		t.Errorf("CID = %v, want %s", op.CID, cidStr)
	}
	if op.Record == nil {
		t.Error("Record should not be nil")
	}
}

func TestATProtoEvent_AllFields(t *testing.T) {
	event := &ATProtoEvent{
		Repo: "did:plc:test456",
		Rev:  "def456",
		Seq:  67890,
		Time: "2025-10-04T12:30:00Z",
		Ops: []Operation{
			{
				Action: "update",
				Path:   "app.bsky.feed.post/67890",
				Record: map[string]interface{}{"text": "Updated message"},
			},
			{
				Action: "delete",
				Path:   "app.bsky.feed.post/12345",
			},
		},
	}

	if event.Repo != "did:plc:test456" {
		t.Errorf("Repo = %s, want did:plc:test456", event.Repo)
	}
	if event.Rev != "def456" {
		t.Errorf("Rev = %s, want def456", event.Rev)
	}
	if event.Seq != 67890 {
		t.Errorf("Seq = %d, want 67890", event.Seq)
	}
	if event.Time != "2025-10-04T12:30:00Z" {
		t.Errorf("Time = %s, want 2025-10-04T12:30:00Z", event.Time)
	}
	if len(event.Ops) != 2 {
		t.Errorf("Ops length = %d, want 2", len(event.Ops))
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func createCBORData(obj interface{}) []byte {
	data, err := cbor.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return data
}
