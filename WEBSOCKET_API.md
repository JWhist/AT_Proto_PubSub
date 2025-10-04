# WebSocket API Documentation - System Operational ✅

## Overview

The ATP Test WebSocket system is **now fully operational** and provides real-time filtered event streaming from the AT Protocol firehose. Users can create filter subscriptions and connect via WebSocket to receive only the events that match their criteria.

## System Status ✅
- **Server**: Running and processing live AT Protocol events
- **WebSocket**: Fully functional with real-time event streaming  
- **Filters**: Working with keyword and path prefix filtering
- **API**: All endpoints operational
- **Testing**: Successfully validated with multiple filter types

The system allows users to:
1. Create filter subscriptions that generate unique filter keys
2. Use those keys to establish WebSocket connections for real-time event streaming
3. Receive filtered AT Protocol events in real-time based on their subscription criteria

## API Endpoints

### Create Filter Subscription
```
POST /api/filters/create
Content-Type: application/json

{
  "options": {
    "repository": "did:plc:user123",    // Optional: filter by repository DID
    "pathPrefix": "app.bsky.feed.post", // Optional: filter by path prefix  
    "keyword": "test"                   // Optional: filter by keyword in text
  }
}
```

Response:
```json
{
  "success": true,
  "message": "Filter created successfully",
  "data": {
    "filterKey": "abc123def456...",
    "options": {
      "repository": "did:plc:user123",
      "pathPrefix": "app.bsky.feed.post",
      "keyword": "test"
    },
    "createdAt": "2024-01-01T12:00:00Z"
  }
}
```

### List Subscriptions
```
GET /api/subscriptions
```

Response:
```json
{
  "success": true,
  "message": "Filter subscriptions retrieved successfully",
  "data": [
    {
      "filterKey": "abc123def456...",
      "options": {
        "repository": "did:plc:user123",
        "pathPrefix": "app.bsky.feed.post",
        "keyword": "test"
      },
      "createdAt": "2024-01-01T12:00:00Z",
      "connections": 2
    }
  ]
}
```

### Delete Filter Subscription
```
DELETE /api/filters/delete/{filterKey}
```

Response:
```json
{
  "success": true,
  "message": "Filter deleted successfully",
  "data": {
    "filterKey": "abc123def456..."
  }
}
```

### WebSocket Connection
```
GET /ws/{filterKey}
Upgrade: websocket
Connection: Upgrade
```

The WebSocket connection will receive real-time events matching the filter criteria.

## WebSocket Message Format

Events are sent as JSON messages with this structure:

```json
{
  "type": "event",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": {
    "event": "commit",
    "did": "did:plc:user123",
    "time": "2024-01-01T12:00:00Z",
    "kind": "commit",
    "ops": [
      {
        "action": "create",
        "path": "app.bsky.feed.post/abc123",
        "collection": "app.bsky.feed.post",
        "rkey": "abc123",
        "record": {
          "text": "This is a test post",
          "createdAt": "2024-01-01T12:00:00Z"
        },
        "cid": "bafyreiabc123..."
      }
    ]
  }
}
```

## Filter Options

All filter options are optional. If no filters are specified, all events will be matched.

- **repository**: Filter events by the repository DID (exact match)
- **pathPrefix**: Filter events by operation path prefix (e.g., "app.bsky.feed.post" for posts)
- **keyword**: Filter events by keyword in text content (case-insensitive, searches "text", "message", and "content" fields)

## Architecture

The system consists of several components:

1. **Subscription Manager** (`internal/subscription/manager.go`): Core component that manages filter subscriptions, WebSocket connections, and event broadcasting
2. **API Handlers** (`internal/api/handlers.go`): HTTP endpoints for creating, listing, and deleting filter subscriptions
3. **WebSocket Handler** (`internal/api/handlers.go`): WebSocket upgrade and connection management
4. **Firehose Integration** (`internal/firehose/client.go`): Event callback mechanism to broadcast AT Protocol events to subscribers

## Usage Example

### 1. Create a Filter Subscription
```bash
curl -X POST http://localhost:8080/api/filters/create \
  -H "Content-Type: application/json" \
  -d '{
    "options": {
      "repository": "did:plc:user123",
      "pathPrefix": "app.bsky.feed.post",
      "keyword": "test"
    }
  }'
```

### 2. Connect via WebSocket
```javascript
const filterKey = "abc123def456..."; // From step 1 response
const ws = new WebSocket(`ws://localhost:8080/ws/${filterKey}`);

ws.onmessage = function(event) {
  const message = JSON.parse(event.data);
  console.log('Received event:', message.data);
};

ws.onopen = function() {
  console.log('WebSocket connected');
};

ws.onclose = function() {
  console.log('WebSocket disconnected');
};
```

### 3. Clean Up
```bash
curl -X DELETE http://localhost:8080/api/filters/delete/abc123def456...
```

## Testing

The system includes comprehensive tests:

- **Subscription Manager Tests**: Unit tests for filter creation, deletion, event matching, and connection management
- **API Handler Tests**: HTTP endpoint testing including WebSocket upgrade scenarios
- **Firehose Client Tests**: Event callback functionality and concurrency testing

Run tests with:
```bash
go test ./...
```

Get test coverage with:
```bash
go test -cover ./...
```

Current test coverage:
- API: 54.3%
- Subscription Manager: 60.3%
- Firehose Client: 52.3%
- Car Parser: 47.5%

## Configuration

The WebSocket upgrader is configured to:
- Allow all origins (development mode)
- Use default buffer sizes
- Handle connection lifecycle automatically

## Concurrency

The system is designed to handle concurrent access:
- All subscription operations are thread-safe using sync.RWMutex
- Event broadcasting supports multiple simultaneous WebSocket connections
- Filter key generation uses cryptographically secure random numbers

## Error Handling

Common error scenarios:
- Invalid filter key for WebSocket connection: Returns 400 Bad Request
- Filter not found for deletion: Returns 404 Not Found
- WebSocket upgrade failure: Logged and connection closed
- Invalid JSON in requests: Returns 400 Bad Request

## Performance Considerations

- Filter keys are 32-character hex strings (16 random bytes)
- Event filtering is performed in-memory for real-time performance
- WebSocket connections are managed efficiently with proper cleanup
- Subscription data is stored in memory (consider persistence for production)