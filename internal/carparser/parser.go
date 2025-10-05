package carparser

import (
	"bytes"
	"fmt"
	"io"

	"github.com/fxamacker/cbor/v2"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car/v2"
)

// ATProtoEvent represents a parsed AT Protocol event from the firehose
type ATProtoEvent struct {
	Repo string      `json:"repo"`
	Rev  string      `json:"rev"`
	Seq  int64       `json:"seq"`
	Time string      `json:"time"`
	Ops  []Operation `json:"ops"`
}

// Operation represents an AT Protocol operation
type Operation struct {
	Action string      `json:"action"`
	Path   string      `json:"path"`
	CID    *string     `json:"cid,omitempty"`
	Record interface{} `json:"record,omitempty"`
}

// ParseCARMessage parses a CAR file message from the AT Protocol firehose
func ParseCARMessage(data []byte) (*ATProtoEvent, error) {
	reader := bytes.NewReader(data)

	// Parse the CAR file using BlockReader
	blockReader, err := car.NewBlockReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create CAR block reader: %w", err)
	}

	// Read blocks and look for the commit data
	var event *ATProtoEvent
	for {
		block, err := blockReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CAR block: %w", err)
		}

		// Try to decode the block as CBOR
		var cborData map[string]interface{}
		if err := cbor.Unmarshal(block.RawData(), &cborData); err != nil {
			// Skip blocks that aren't CBOR or aren't the format we expect
			continue
		}

		// Check if this looks like a commit block
		if repo, hasRepo := cborData["repo"].(string); hasRepo {
			event = &ATProtoEvent{
				Repo: repo,
			}

			// Extract other fields
			if rev, ok := cborData["rev"].(string); ok {
				event.Rev = rev
			}
			if seq, ok := cborData["seq"].(int64); ok {
				event.Seq = seq
			}
			if time, ok := cborData["time"].(string); ok {
				event.Time = time
			}

			// Parse operations - with detailed debugging
			if ops, ok := cborData["ops"].([]interface{}); ok {
				fmt.Printf("🔍 Found ops field with %d operations\n", len(ops))
				for i, op := range ops {
					if opMap, ok := op.(map[string]interface{}); ok {
						operation := Operation{}

						if action, ok := opMap["action"].(string); ok {
							operation.Action = action
							fmt.Printf("🔍 Operation %d action: %s\n", i, action)
						}
						if path, ok := opMap["path"].(string); ok {
							operation.Path = path
							fmt.Printf("🔍 Operation %d path: %s\n", i, path)
						}
						if cidStr, ok := opMap["cid"].(string); ok {
							operation.CID = &cidStr
						}
						if record, ok := opMap["record"]; ok {
							operation.Record = record
						}

						event.Ops = append(event.Ops, operation)
					}
				}
			} else {
				fmt.Printf("🔍 No 'ops' field found or not an array\n")
			}
			break
		}
	}

	if event == nil {
		return nil, fmt.Errorf("no valid AT Protocol event found in CAR file")
	}

	return event, nil
}

// ParseCARMessageSimple provides a simpler approach that looks for the main commit object
func ParseCARMessageSimple(data []byte) (*ATProtoEvent, error) {
	// Try to find CBOR data that looks like a commit
	// The firehose sends messages that contain multiple CBOR objects

	// Look for the commit object pattern in the data
	// AT Protocol commits usually start with specific CBOR patterns

	// For now, let's try a more direct approach - scan for CBOR maps
	var offset int
	for offset < len(data) {
		// Try to decode CBOR from this position
		decoder := cbor.NewDecoder(bytes.NewReader(data[offset:]))

		var obj map[string]interface{}
		if err := decoder.Decode(&obj); err != nil {
			offset++
			continue
		}

		// Check if this looks like a commit object
		if repo, hasRepo := obj["repo"].(string); hasRepo {
			event := &ATProtoEvent{
				Repo: repo,
			}

			// Extract fields safely
			if rev, ok := obj["rev"].(string); ok {
				event.Rev = rev
			}
			if seq, ok := obj["seq"].(int64); ok {
				event.Seq = seq
			} else if seq, ok := obj["seq"].(uint64); ok {
				event.Seq = int64(seq)
			} else if seq, ok := obj["seq"].(float64); ok {
				event.Seq = int64(seq)
			}
			if time, ok := obj["time"].(string); ok {
				event.Time = time
			}

			// Parse operations
			if ops, ok := obj["ops"].([]interface{}); ok {
				for _, op := range ops {
					var operation Operation
					var opMap map[string]interface{}

					// Handle both map[string]interface{} and map[interface{}]interface{}
					if stringMap, ok := op.(map[string]interface{}); ok {
						opMap = stringMap
					} else if interfaceMap, ok := op.(map[interface{}]interface{}); ok {
						// Convert map[interface{}]interface{} to map[string]interface{}
						opMap = make(map[string]interface{})
						for k, v := range interfaceMap {
							if keyStr, ok := k.(string); ok {
								opMap[keyStr] = v
							}
						}
					}

					if opMap != nil {
						if action, ok := opMap["action"].(string); ok {
							operation.Action = action
						}
						if path, ok := opMap["path"].(string); ok {
							operation.Path = path
						}
						if cidBytes, ok := opMap["cid"].([]byte); ok {
							// CID is often encoded as bytes, try to decode
							if c, err := cid.Cast(cidBytes); err == nil {
								cidStr := c.String()
								operation.CID = &cidStr
							}
						} else if cidStr, ok := opMap["cid"].(string); ok {
							// CID might also come as a string
							operation.CID = &cidStr
						}
						if record, ok := opMap["record"]; ok {
							operation.Record = record
						}

						event.Ops = append(event.Ops, operation)
					}
				}
			}

			return event, nil
		}

		// Move to next byte and try again
		offset++
	}

	return nil, fmt.Errorf("no AT Protocol commit found in message")
}
