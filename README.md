# atp-test
Testing firehose read and filter of events

A server that reads the AT Protocol firehose and filters events based on repository and keyword criteria.

## Installation

```bash
npm install
```

## Building

```bash
npm run build
```

## Usage

**Note:** This server requires internet connectivity to connect to the AT Protocol firehose at `wss://bsky.network`.

### Basic usage (show all events with text):
```bash
npm start
```

### Filter by keyword:
```bash
npm start -- --keyword "hello"
```

### Filter by repository DID:
```bash
npm start -- --repository did:plc:abc123xyz
```

### Filter by both repository and keyword:
```bash
npm start -- --repository did:plc:abc123xyz --keyword "test"
```

### Show help:
```bash
npm start -- --help
```

## Command Line Options

- `-r, --repository <repo>` - Filter by repository DID
- `-k, --keyword <keyword>` - Filter by keyword in text (case-insensitive)
- `-h, --help` - Show help message

## How it works

The server connects to the AT Protocol firehose at `wss://bsky.network` and:
1. Receives all events from the firehose
2. Filters events based on the provided repository DID (if specified)
3. Further filters events that contain the specified keyword in their text content (if specified)
4. Logs matching events to the console with detailed information

### Example Filter Logic

You can test the filtering logic locally without connecting to the firehose:

```bash
node example-filter.js
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
Filters:
  Repository: ALL
  Keyword: test
Connecting to firehose...

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

================================================================================
[2024-10-04T21:15:35.456Z] UPDATE event
--------------------------------------------------------------------------------
Repository: did:plc:def789ghi012
Collection: app.bsky.feed.post
Record Key: 9m8n7b6v5c4x
URI: at://did:plc:def789ghi012/app.bsky.feed.post/9m8n7b6v5c4x
Text: Testing the new firehose filter functionality
Reply to: {"parent":{"uri":"at://did:plc:other123/app.bsky.feed.post/abc"}}
================================================================================
```

Press `Ctrl+C` to stop the server.
