# AT Proto Pub Sub
Real-time AT Protocol WebSocket Subscription System

A WebSocket-based subscription system that connects to the AT Protocol firehose and provides real-time filtered event streaming to multiple clients. Users can create custom filters and subscribe to specific types of AT Protocol events via WebSocket connections.

## Features

- **Real-time WebSocket Subscriptions**: Multiple clients can subscribe to filtered AT Protocol events
- **Custom Filter Creation**: Create filters based on repository, path prefix, and keyword criteria
- **Live Event Broadcasting**: Events are broadcast to all matching subscriptions in real-time
- **RESTful API**: HTTP endpoints for filter management and subscription stats
- **Concurrent Client Support**: Multiple WebSocket clients can connect simultaneously
- **Automatic Resource Management**: Filters remain available indefinitely with no manual cleanup needed
- **Enhanced Logging**: Detailed logging for debugging and monitoring

## Prerequisites

- Go 1.24 or later
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
go build -o atp-test

# Or run directly
go run .
```

## Usage

### Starting the Server

```bash
# Run from source
go run .

# Or run the compiled binary
./atp-test
```

The server will start:
- **Firehose Connection**: Connects to AT Protocol firehose at `wss://bsky.network`
- **HTTP API Server**: Listens on `http://localhost:8080` for filter management
- **WebSocket Server**: Accepts WebSocket connections for real-time event streaming

### API Endpoints

#### Get Server Status
```bash
curl http://localhost:8080/api/status
```

#### Create a New Filter
```bash
# Create a filter for posts containing "test"
curl -X POST http://localhost:8080/api/filters/create \
  -H "Content-Type: application/json" \
  -d '{
    "options": {
      "pathPrefix": "app.bsky.feed.post",
      "keyword": "test"
    }
  }'

# Returns a filter key like: {"filterKey": "8a3ce5f31b47d4788df91aeb38a565fe"}
```

#### Get Filter Details
```bash
curl http://localhost:8080/api/filters/{filterKey}
```

#### List All Filters
```bash
curl http://localhost:8080/api/filters
```

#### Get Subscription Statistics
```bash
curl http://localhost:8080/api/stats
```

### WebSocket Connection

Once you have a filter key, you can connect via WebSocket to receive real-time events:

#### WebSocket URL Format
```
ws://localhost:8080/ws/{filterKey}
```

#### Example WebSocket Client
```bash
# Use the provided test client
go run test_websocket_client.go {filterKey}
```

Or connect using any WebSocket client to `ws://localhost:8080/ws/8a3ce5f31b47d4788df91aeb38a565fe`

### Filter Types

#### Repository Filter
Filters events by repository DID (the source account):
```json
{
  "options": {
    "repository": "did:plc:abc123xyz"
  }
}
```

#### Path Prefix Filter  
Filters events by operation path/collection prefix:
```json
{
  "options": {
    "pathPrefix": "app.bsky.feed.post"
  }
}
```

#### Keyword Filter
Filters events by text content within the record:
```json
{
  "options": {
    "keyword": "hello world"
  }
}
```

#### Combined Filters
All filter options can be combined:
```json
{
  "options": {
    "repository": "did:plc:abc123xyz",
    "pathPrefix": "app.bsky.feed.post",
    "keyword": "test"
  }
}
```

## How it Works

The system uses a **publish-subscribe architecture** with the following components:

### 1. Firehose Connection
- Connects to `wss://bsky.network` AT Protocol firehose
- Receives real-time events from the entire AT Protocol network
- Processes and broadcasts events to the subscription manager

### 2. Subscription Manager
- Manages multiple filter subscriptions with unique keys
- Maintains WebSocket connections for each active subscription
- Filters incoming events against all active subscriptions
- Broadcasts matching events to connected WebSocket clients

### 3. HTTP API Server
- Provides REST endpoints for filter management
- Handles filter creation, retrieval, and deletion
- Serves subscription statistics and server status

### 4. WebSocket Server
- Accepts WebSocket connections using filter keys
- Streams real-time filtered events to connected clients
- Handles connection lifecycle and cleanup

### Event Processing Flow

1. **Filter Creation**: Client creates a filter via POST `/api/filters/create`
2. **WebSocket Connection**: Client connects to `ws://localhost:8080/ws/{filterKey}`
3. **Event Filtering**: Incoming firehose events are filtered by subscription manager
4. **Real-time Broadcasting**: Matching events are sent to all relevant WebSocket connections

### Filter Lifecycle

Filters are **persistent and lightweight** - once created, they remain available indefinitely. There's no need for manual cleanup since:
- Unused filters consume minimal resources (just stored filter criteria)
- Filters only broadcast events when clients are actively connected
- Connection cleanup happens automatically when WebSocket clients disconnect
- The same filter can be reused by multiple clients over time

### WebSocket Message Format

When connected, you'll receive JSON messages in this format:

```json
{
  "type": "event",
  "data": {
    "event": {
      "commit": {
        "rev": "3l4k5j6h7g8f9h0i1j2k3l4m5n6o7p8q",
        "time": "2024-10-04T21:15:32.123Z"
      },
      "did": "did:plc:abc123xyz456",
      "ops": [
        {
          "action": "create",
          "path": "app.bsky.feed.post/3l4k5j6h7g8f",
          "record": {
            "text": "This is a test post!",
            "langs": ["en"],
            "createdAt": "2024-10-04T21:15:32.123Z"
          }
        }
      ]
    },
    "filterKey": "8a3ce5f31b47d4788df91aeb38a565fe"
  }
}
```

### Connection Messages
You'll also receive connection status messages:
```json
{
  "type": "connection",
  "data": {
    "status": "connected",
    "filterKey": "8a3ce5f31b47d4788df91aeb38a565fe",
    "message": "Connected to filter"
  }
}
```

## Quick Start Example

### 1. Start the Server
```bash
go run .
```

### 2. Create a Filter
```bash
# Create a filter for Bluesky posts containing "hello"
curl -X POST http://localhost:8080/api/filters/create \
  -H "Content-Type: application/json" \
  -d '{
    "options": {
      "pathPrefix": "app.bsky.feed.post",
      "keyword": "hello"
    }
  }'

# Response: {"filterKey": "abc123def456..."}
```

### 3. Connect via WebSocket
```bash
# Use the test client with your filter key
go run test_websocket_client.go abc123def456...
```

### 4. Watch Real-time Events
You'll now see matching AT Protocol events in real-time as they happen!

## Testing

### Running Tests
```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/subscription/
```

### Manual Testing

#### Test Client
Use the included WebSocket test client:
```bash
go run test_websocket_client.go {filterKey}
```

#### Filter Testing
```bash
# Create different types of filters
curl -X POST http://localhost:8080/api/filters/create \
  -H "Content-Type: application/json" \
  -d '{"options": {"repository": "did:plc:specific-user"}}'

curl -X POST http://localhost:8080/api/filters/create \
  -H "Content-Type: application/json" \
  -d '{"options": {"pathPrefix": "app.bsky.graph.follow"}}'

curl -X POST http://localhost:8080/api/filters/create \
  -H "Content-Type: application/json" \
  -d '{"options": {"keyword": "bluesky"}}'
```

## API Reference

### GET /api/status
Returns server status and basic statistics.

**Response:**
```json
{
  "status": "active",
  "uptime": "2h34m12s",
  "totalEvents": 15234,
  "activeFilters": 3,
  "activeConnections": 2
}
```

### POST /api/filters/create
Creates a new filter and returns a unique filter key.

**Request:**
```json
{
  "options": {
    "repository": "did:plc:abc123",      // optional
    "pathPrefix": "app.bsky.feed.post",  // optional  
    "keyword": "hello world"             // optional
  }
}
```

**Response:**
```json
{
  "filterKey": "8a3ce5f31b47d4788df91aeb38a565fe"
}
```

### GET /api/filters/{filterKey}
Retrieves details for a specific filter.

**Response:**
```json
{
  "filterKey": "8a3ce5f31b47d4788df91aeb38a565fe",
  "options": {
    "repository": "",
    "pathPrefix": "app.bsky.feed.post",
    "keyword": "hello"
  },
  "connections": 1,
  "created": "2024-10-04T21:15:32.123Z"
}
```

### GET /api/filters
Lists all active filters.

**Response:**
```json
{
  "filters": [
    {
      "filterKey": "8a3ce5f31b47d4788df91aeb38a565fe",
      "options": {
        "pathPrefix": "app.bsky.feed.post",
        "keyword": "hello"
      },
      "connections": 1
    }
  ]
}
```

### GET /api/stats
Returns detailed subscription statistics.

**Response:**
```json
{
  "active_filters": 3,
  "total_connections": 2,
  "events_processed": 15234,
  "events_broadcasted": 89
}
```

### WebSocket Endpoint
**URL:** `ws://localhost:8080/ws/{filterKey}`

Connects to receive real-time filtered events. The connection will receive:
- Event messages when matching AT Protocol events occur
- Connection status messages
- Error messages if issues occur

## Development

### Code Structure
```
├── main.go                           # Application entry point
├── internal/
│   ├── api/
│   │   └── handlers.go              # HTTP and WebSocket handlers
│   ├── firehose/
│   │   └── client.go                # AT Protocol firehose client  
│   ├── models/
│   │   └── types.go                 # Data structures
│   └── subscription/
│       ├── manager.go               # Subscription management
│       └── manager_test.go          # Tests
├── test_websocket_client.go         # WebSocket test client
└── README.md
```

### Adding New Features

1. **New Filter Types**: Add to `FilterOptions` in `models/types.go`
2. **Custom Event Processing**: Modify `BroadcastEvent` in `subscription/manager.go`
3. **Additional API Endpoints**: Add to `api/handlers.go`

### Debugging

The application includes comprehensive logging:
- **Server logs**: Connection status, filter operations, errors
- **Client logs**: Event reception, connection status, message parsing

Set debug level logging by modifying the log configuration in `main.go`.

## Deployment

### Docker (Optional)
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o atp-test

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/atp-test .
EXPOSE 8080
CMD ["./atp-test"]
```

### Production Considerations
- Use a reverse proxy (nginx) for production deployment
- Implement rate limiting for API endpoints
- Add authentication for filter management
- Monitor WebSocket connection limits
- Consider horizontal scaling with Redis for shared state
