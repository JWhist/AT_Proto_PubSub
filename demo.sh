#!/bin/bash

echo "🚀 ATP Test WebSocket Subscription System Demo"
echo "=============================================="
echo

echo "📊 Current server stats:"
curl -s http://localhost:8080/api/stats | jq '.'
echo

echo "📋 Active subscriptions:"
curl -s http://localhost:8080/api/subscriptions | jq '.'
echo

echo "🔗 Testing WebSocket connections..."
echo

# Function to start a WebSocket client in background
start_ws_client() {
    local filter_key=$1
    local name=$2
    echo "Starting WebSocket client for $name (Filter: $filter_key)"
    
    # Use websocat if available, otherwise use our Go client
    if command -v websocat >/dev/null 2>&1; then
        websocat "ws://localhost:8080/ws/$filter_key" &
    else
        go run test_websocket_client.go "$filter_key" &
    fi
    
    local pid=$!
    echo "WebSocket client PID: $pid"
    return $pid
}

echo "💡 You can connect to these WebSocket endpoints:"
echo "   Posts with 'test': ws://localhost:8080/ws/8a3ce5f31b47d4788df91aeb38a565fe"
echo "   Follow events: ws://localhost:8080/ws/cd79243732b5a0fce01cfb1a051eb7ab"
echo
echo "💭 To test manually, run:"
echo "   go run test_websocket_client.go 8a3ce5f31b47d4788df91aeb38a565fe"
echo "   go run test_websocket_client.go cd79243732b5a0fce01cfb1a051eb7ab"
echo
echo "🎯 The system is now filtering live AT Protocol events!"