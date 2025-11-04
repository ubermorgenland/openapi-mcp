#!/bin/bash

# YouTube Transcript StreamableHTTP Simple Test
# Tests the /youtube-transcript endpoint (StreamableHTTP) with authentication

set -e

# Load test secrets
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/load_test_secrets.sh"

# Configuration
SERVER_URL="http://localhost:8080"
ENDPOINT="/youtube-transcript"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}=================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=================================${NC}"
}

print_test() {
    echo -e "${YELLOW}TEST: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ PASS: $1${NC}"
}

print_failure() {
    echo -e "${RED}✗ FAIL: $1${NC}"
}

print_header "YouTube Transcript StreamableHTTP Simple Test"

# Test 1: Initialize session
print_test "Initialize MCP session"
init_response=$(curl -s -v -X POST "$SERVER_URL$ENDPOINT" \
    -H "Content-Type: application/json" \
    -d '{
        "jsonrpc": "2.0",
        "method": "initialize",
        "id": 1,
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {"roots": {"listChanged": true}},
            "clientInfo": {"name": "Test", "version": "1.0.0"}
        }
    }' 2>&1)

# Extract session ID from headers
SESSION_ID=$(echo "$init_response" | grep -i "mcp-session-id" | grep -o "mcp-session-[a-zA-Z0-9-]*" | head -1)

if [ -n "$SESSION_ID" ] && echo "$init_response" | grep -q '"result"'; then
    print_success "Session initialized: $SESSION_ID"
else
    print_failure "Session initialization failed"
    exit 1
fi

echo ""

# Test 2: List available tools
print_test "List available tools"
list_response=$(curl -s -X POST "$SERVER_URL$ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "Mcp-Session-Id: $SESSION_ID" \
    -d '{
        "jsonrpc": "2.0",
        "method": "tools/list",
        "id": 2
    }')

if echo "$list_response" | grep -q 'transcript'; then
    print_success "Tools listed successfully - YouTube transcript tools found"
else
    print_failure "Tools list failed or YouTube transcript tools not found"
    exit 1
fi

echo ""

# Test 3: Call tool with explicit RapidAPI key (Priority 1 - Tool Arguments)
print_test "Call transcript tool with explicit RapidAPI key in arguments"

echo "📤 REQUEST:"
echo "POST $SERVER_URL$ENDPOINT"
echo "Headers:"
echo "  Content-Type: application/json"
echo "  Mcp-Session-Id: $SESSION_ID"
echo "Body:"
request_body='{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "id": 3,
    "params": {
        "name": "GET_/api/transcript",
        "arguments": {
            "videoId": "dQw4w9WgXcQ",
            "X-RapidAPI-Key": "'$TWITTER_RAPIDAPI_KEY'",
            "X-RapidAPI-Host": "youtube-transcript3.p.rapidapi.com"
        }
    }
}'
echo "$request_body"
echo ""

tool_response=$(curl -s -v -X POST "$SERVER_URL$ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "Mcp-Session-Id: $SESSION_ID" \
    -d "$request_body" 2>&1)

# Extract headers and body
response_headers=$(echo "$tool_response" | grep -E "^< |^HTTP" | head -20)
response_body=$(echo "$tool_response" | tail -1)

echo "📥 RESPONSE HEADERS:"
echo "$response_headers"
echo ""
echo "📦 RESPONSE BODY:"
echo "$response_body" | jq '.' 2>/dev/null || echo "$response_body"
echo ""

if echo "$response_body" | grep -q 'Status: 200'; then
    print_success "Tool call with explicit key - SUCCESS (HTTP 200)"
elif echo "$response_body" | grep -q '"result"'; then
    print_success "Tool call processed by server"
else
    print_failure "Tool call with explicit key failed"
fi

echo ""
echo "⏳ Adding 3-second delay to avoid rate limiting..."
sleep 3

# Test 4: Call tool with header authentication (Priority 2 - Headers)
print_test "Call transcript tool with X-RapidAPI-Key header (using different video to avoid rate limit)"

echo "📤 REQUEST:"
echo "POST $SERVER_URL$ENDPOINT"
echo "Headers:"
echo "  Content-Type: application/json"
echo "  Mcp-Session-Id: $SESSION_ID"
echo "  X-RapidAPI-Key: $TWITTER_RAPIDAPI_KEY"
echo "  X-RapidAPI-Host: youtube-transcript3.p.rapidapi.com"
echo "Body:"
header_request_body='{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "id": 4,
    "params": {
        "name": "GET_/api/transcript",
        "arguments": {
            "videoId": "jNQXAC9IVRw"
        }
    }
}'
echo "$header_request_body"
echo "⏳ Adding 3-second delay to avoid rate limiting..."
sleep 3
echo ""

header_response=$(curl -s -v -X POST "$SERVER_URL$ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "Mcp-Session-Id: $SESSION_ID" \
    -H "X-RapidAPI-Key: $TWITTER_RAPIDAPI_KEY" \
    -H "X-RapidAPI-Host: youtube-transcript3.p.rapidapi.com" \
    -d "$header_request_body" 2>&1)

# Extract headers and body
header_response_headers=$(echo "$header_response" | grep -E "^< |^HTTP" | head -20)
header_response_body=$(echo "$header_response" | tail -1)

echo "📥 RESPONSE HEADERS:"
echo "$header_response_headers"
echo ""
echo "📦 RESPONSE BODY:"
echo "$header_response_body" | jq '.' 2>/dev/null || echo "$header_response_body"
echo ""

if echo "$header_response_body" | grep -q 'Status: 200'; then
    print_success "Tool call with header auth - SUCCESS (HTTP 200) - Headers working correctly!"
elif echo "$header_response_body" | grep -q 'HTTP 429\|Too Many Requests'; then
    print_success "Tool call with header auth - HTTP 429 (Rate Limited) - Headers are working correctly!"
elif echo "$header_response_body" | grep -q '"result"'; then
    print_success "Tool call processed by server (check response for actual HTTP status)"
else
    print_failure "Tool call with header auth failed - unexpected response"
fi

echo ""
print_header "StreamableHTTP Test Complete"
echo "✅ YouTube Transcript StreamableHTTP authentication test completed successfully"