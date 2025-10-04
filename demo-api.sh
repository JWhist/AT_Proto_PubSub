#!/bin/bash

# Demo script for AT Protocol Firehose Filter API

echo "=== AT Protocol Firehose Filter Server API Demo ==="
echo

# Check if server is running
if ! curl -s http://localhost:8080/api/status >/dev/null 2>&1; then
    echo "❌ Server is not running on localhost:8080"
    echo "   Please start it with: go run main.go"
    exit 1
fi

echo "✅ Server is running!"
echo

echo "📊 Current server status:"
curl -s http://localhost:8080/api/status | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(f\"   Status: {data['data']['status']}\")
    print(f\"   Repository filter: {data['data']['filters']['repository'] or 'ALL'}\")
    print(f\"   Keyword filter: {data['data']['filters']['keyword'] or 'ALL'}\")
except:
    print('   Error parsing response')
"
echo

echo "🔧 Setting keyword filter to 'test'..."
curl -s -X POST -H "Content-Type: application/json" \
    -d '{"keyword":"test"}' \
    http://localhost:8080/api/filters/update | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if data['success']:
        print(f\"   ✅ {data['message']}\")
    else:
        print(f\"   ❌ {data['message']}\")
except:
    print('   ❌ Error parsing response')
"
echo

echo "🔧 Setting repository filter to 'did:plc:example123'..."
curl -s -X POST -H "Content-Type: application/json" \
    -d '{"repository":"did:plc:example123"}' \
    http://localhost:8080/api/filters/update | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if data['success']:
        print(f\"   ✅ {data['message']}\")
    else:
        print(f\"   ❌ {data['message']}\")
except:
    print('   ❌ Error parsing response')
"
echo

echo "📋 Current filter settings:"
curl -s http://localhost:8080/api/filters | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    filters = data['data']
    print(f\"   Repository: {filters['repository'] or 'ALL'}\")
    print(f\"   Keyword: {filters['keyword'] or 'ALL'}\")
except:
    print('   Error parsing response')
"
echo

echo "🧹 Clearing all filters..."
curl -s -X POST -H "Content-Type: application/json" \
    -d '{"repository":"","keyword":""}' \
    http://localhost:8080/api/filters/update | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if data['success']:
        print(f\"   ✅ {data['message']}\")
    else:
        print(f\"   ❌ {data['message']}\")
except:
    print('   ❌ Error parsing response')
"
echo

echo "📋 Final filter settings:"
curl -s http://localhost:8080/api/filters | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    filters = data['data']
    print(f\"   Repository: {filters['repository'] or 'ALL'}\")
    print(f\"   Keyword: {filters['keyword'] or 'ALL'}\")
except:
    print('   Error parsing response')
"
echo

echo "🎉 Demo complete!"
echo "   The server is now ready to filter firehose events based on your API settings."
echo "   Use the API endpoints to control filtering in real-time."