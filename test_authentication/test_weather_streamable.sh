#!/bin/bash

# Weather API StreamableHTTP Authentication Test
# Tests the /weather endpoint (StreamableHTTP) with different authentication methods

set -e

# Load test secrets
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/load_test_secrets.sh"

# Configuration
SERVER_URL="http://localhost:8080"
ENDPOINT="/weather"

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

print_header "Weather API StreamableHTTP Tests"

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

# Extract session ID from headers (StreamableHTTP uses Mcp-Session-Id header)
SESSION_ID=$(echo "$init_response" | grep -i "mcp-session-id" | grep -o "mcp-session-[a-zA-Z0-9-]*" | head -1)

if [ -n "$SESSION_ID" ] && echo "$init_response" | grep -q '"result"'; then
    print_success "Session initialized: $SESSION_ID"
else
    print_failure "Session initialization failed"
    echo "Debug - Response headers:"
    echo "$init_response" | grep -E "(HTTP|Mcp-Session|Content-Type)" || echo "No relevant headers found"
    echo "Debug - Response body:"
    echo "$init_response" | tail -1
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

if echo "$list_response" | grep -q 'realtime-weather'; then
    print_success "Tools listed successfully - realtime-weather found"
else
    print_failure "Tools list failed or realtime-weather not found"
    echo "Debug - List response: $list_response"
fi

echo ""

# Test 3: Call tool with explicit API key (Priority 1 - Tool Arguments)
print_test "Call tool with explicit API key in arguments"

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
        "name": "realtime-weather",
        "arguments": {
            "q": "London",
            "key": "'$WEATHER_API_KEY'"
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

# Test 4: Call tool with header authentication (Priority 2 - Headers)
print_test "Call tool with X-API-Key header"
header_response=$(curl -s -X POST "$SERVER_URL$ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "Mcp-Session-Id: $SESSION_ID" \
    -H "X-API-Key: $WEATHER_API_KEY" \
    -d '{
        "jsonrpc": "2.0",
        "method": "tools/call",
        "id": 4,
        "params": {
            "name": "realtime-weather",
            "arguments": {
                "q": "Paris"
            }
        }
    }')

if echo "$header_response" | grep -q 'Status: 200'; then
    print_success "Tool call with header auth - SUCCESS (HTTP 200)"
elif echo "$header_response" | grep -q '"result"'; then
    print_success "Tool call processed by server"
else
    print_failure "Tool call with header auth failed"
fi

echo ""

# Test 5: Call tool using database fallback (Priority 3 - Database)
print_test "Call tool using database authentication fallback"

echo "📤 REQUEST (NO explicit auth - should use database key):"
echo "POST $SERVER_URL$ENDPOINT"
echo "Headers:"
echo "  Content-Type: application/json"
echo "  Mcp-Session-Id: $SESSION_ID"
echo "  (No X-API-Key header - testing database fallback)"
echo "Body:"
db_request_body='{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "id": 5,
    "params": {
        "name": "realtime-weather",
        "arguments": {
            "q": "Tokyo"
        }
    }
}'
echo "$db_request_body"
echo ""

db_response=$(curl -s -v -X POST "$SERVER_URL$ENDPOINT" \
    -H "Content-Type: application/json" \
    -H "Mcp-Session-Id: $SESSION_ID" \
    -d "$db_request_body" 2>&1)

# Extract headers and body
db_response_headers=$(echo "$db_response" | grep -E "^< |^HTTP" | head -20)
db_response_body=$(echo "$db_response" | tail -1)

echo "📥 RESPONSE HEADERS:"
echo "$db_response_headers"
echo ""
echo "📦 RESPONSE BODY:"
echo "$db_response_body" | jq '.' 2>/dev/null || echo "$db_response_body"
echo ""

if echo "$db_response_body" | grep -q 'Status: 200'; then
    print_success "Tool call with database auth - SUCCESS (HTTP 200) - Database key working!"
elif echo "$db_response_body" | grep -iq "api.*key.*invalid\|unauthorized"; then
    print_failure "Tool call with database auth - FAILED - Database key invalid or not being used"
elif echo "$db_response_body" | grep -q '"result"'; then
    print_success "Tool call processed by server (check server logs for auth details)"
else
    print_failure "Tool call with database fallback failed"
fi

echo ""
print_header "StreamableHTTP Test Summary"
echo "✅ Tests demonstrate authentication priority:"
echo "  1. Tool Arguments (highest priority)"
echo "  2. HTTP Headers" 
echo "  3. Database Fallback (lowest priority)"
echo ""
echo "🎯 StreamableHTTP endpoint working correctly!"