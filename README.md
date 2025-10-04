# atp-test
Testing firehose read and filter of events

A server that reads the AT Protocol firehose and filters events based on repository and keyword criteria. **Filters are now configured via HTTP API endpoints instead of command-line flags.**

## Prerequisites

- Go 1.19 or later
- Internet connectivity to connect to the AT Protocol firehose

## Installation

```bash
# Clone the repository (if needed)
git clone <repository-url>
cd atp-test

# Download dependencies
go mod tidy
```

## Building

```bash
# Build the executable
make build

# Or using go directly
go build -o atp-test ./cmd/atp-test

# Or run directly
make run
# Or: go run ./cmd/atp-test
```

## Usage

The server now provides an HTTP API for managing filters instead of using command-line flags.

### Starting the Server

```bash
# Run from source
make run
# Or: go run ./cmd/atp-test

# Or run the compiled binary
make build && ./atp-test
```

The server will start two services:
- **Firehose Connection**: Connects to AT Protocol firehose at `wss://bsky.network`
- **HTTP API Server**: Listens on `http://localhost:8080` for filter management

### API Endpoints

#### Get Server Status
```bash
curl http://localhost:8080/api/status
```

#### Get Current Filters
```bash
curl http://localhost:8080/api/filters
```

#### Update Filters
```bash
# Set repository filter (filters by DID)
curl -X POST -H "Content-Type: application/json" \
  -d '{"repository":"did:plc:abc123xyz"}' \
  http://localhost:8080/api/filters/update

# Set path prefix filter (filters by operation type)
curl -X POST -H "Content-Type: application/json" \
  -d '{"pathPrefix":"app.bsky.feed.post"}' \
  http://localhost:8080/api/filters/update

# Set keyword filter
curl -X POST -H "Content-Type: application/json" \
  -d '{"keyword":"hello world"}' \
  http://localhost:8080/api/filters/update

# Set multiple filters at once
curl -X POST -H "Content-Type: application/json" \
  -d '{"repository":"did:plc:abc123xyz","pathPrefix":"app.bsky.feed","keyword":"test"}' \
  http://localhost:8080/api/filters/update

# Clear all filters (set to empty strings)
curl -X POST -H "Content-Type: application/json" \
  -d '{"repository":"","pathPrefix":"","keyword":""}' \
  http://localhost:8080/api/filters/update
```

### API Response Format

All API endpoints return JSON responses in this format:
```json
{
  "success": true,
  "message": "Description of the result",
  "data": {
    "repository": "did:plc:abc123",
    "pathPrefix": "app.bsky.feed.post",
    "keyword": "test"
  }
}
```

### Filter Types

#### Repository Filter
Filters events by repository DID (the source account):
- Example: `"repository": "did:plc:abc123xyz"`
- Matches events from a specific AT Protocol account

#### Path Prefix Filter  
Filters events by operation path/collection prefix:
- Example: `"pathPrefix": "app.bsky.feed.post"` (only posts)
- Example: `"pathPrefix": "app.bsky.feed"` (posts, likes, reposts, etc.)
- Example: `"pathPrefix": "app.bsky"` (all Bluesky operations)

#### Keyword Filter
Filters events by text content within the record:
- Example: `"keyword": "hello world"`
- Searches text, message, or content fields (case-insensitive)

## How it works

The server provides two main components:

1. **HTTP API Server** (Port 8080): Accepts filter configuration via REST API
2. **Firehose Connection**: Connects to `wss://bsky.network` and processes events

### Event Processing Flow:
1. Server connects to the AT Protocol firehose 
2. Filters can be set/updated via HTTP API calls (repository, path prefix, keyword)
3. Incoming events are filtered based on current settings:
   - **Repository**: Matches the DID of the event source
   - **Path Prefix**: Matches the beginning of the operation collection/path
   - **Keyword**: Searches within the text content of records
4. Matching events are logged to the console with detailed information

### Filter Combination:
All active filters work together (AND logic):
- If repository="did:plc:abc123" AND pathPrefix="app.bsky.feed.post" AND keyword="hello"
- Only posts from that specific account containing "hello" will be shown

### Dynamic Filter Updates
Filters can be updated in real-time while the server is running. Changes take effect immediately for new incoming events.

### Example Filter Logic

You can test the filtering logic locally without connecting to the firehose:

```bash
make example
# Or: go run ./examples/filter-demo.go
```

This will demonstrate how events are filtered based on repository and keyword criteria.

## Output Format

Each matching event is displayed with:
- Timestamp
- Event type (CREATE or UPDATE)
- Repository DID
- Collection name
- Record key
- URI
- Text content (if available)
- Other relevant metadata

### Example Output

```
Starting AT Protocol Firehose Filter Server...
Initial Filters:
  Repository: ALL
  Keyword: ALL
Starting API server on :8080
Connecting to firehose...

Filters updated via API: Repository=did:plc:abc123xyz456, Keyword=test

================================================================================
[2024-10-04T21:15:32.123Z] CREATE event
--------------------------------------------------------------------------------
Repository: did:plc:abc123xyz456
Collection: app.bsky.feed.post
Record Key: 3l4k5j6h7g8f
URI: at://did:plc:abc123xyz456/app.bsky.feed.post/3l4k5j6h7g8f
Text: This is a test post to see if the filter is working correctly!
Languages: ["en"]
================================================================================
```

Press `Ctrl+C` to stop the server.

## Development

### Running in development mode:
```bash
go run main.go
```

### Testing the API:
```bash
# Start the server
make run

# In another terminal, test the endpoints:
curl http://localhost:8080/api/status
curl http://localhost:8080/api/filters
curl -X POST -H "Content-Type: application/json" -d '{"keyword":"test"}' http://localhost:8080/api/filters/update
```

### Building for production:
```bash
make build-prod
```

### Cross-compilation examples:
```bash
# Build for all platforms
make build-all

# Or build for specific platforms
make build-linux    # For Linux
make build-windows  # For Windows  
make build-mac      # For macOS (Apple Silicon)
```

### Other useful commands:
```bash
make example        # Run the filter demo
make clean          # Clean build artifacts
make fmt            # Format code
make help           # Show all available commands
```

## API Reference

### GET /api/status
Returns the current server status and filter settings.

**Response:**
```json
{
  "success": true,
  "message": "Server is running",
  "data": {
    "status": "active",
    "filters": {
      "repository": "did:plc:abc123",
      "pathPrefix": "app.bsky.feed.post",
      "keyword": "test"
    }
  }
}
```

### GET /api/filters
Returns the current filter settings.

**Response:**
```json
{
  "success": true,
  "message": "Current filter settings",
  "data": {
    "repository": "did:plc:abc123",
    "pathPrefix": "app.bsky.feed.post",
    "keyword": "test"
  }
}
```

### POST /api/filters/update
Updates filter settings. Only provided fields will be updated.

**Request Body:**
```json
{
  "repository": "did:plc:newrepo123",     // optional - filter by account DID
  "pathPrefix": "app.bsky.feed.post",     // optional - filter by operation path
  "keyword": "new keyword"                // optional - filter by text content
}
```

**Response:**
```json
{
  "success": true,
  "message": "Filters updated successfully",
  "data": {
    "repository": "did:plc:newrepo123",
    "pathPrefix": "app.bsky.feed.post",
    "keyword": "new keyword"
  }
}
```
