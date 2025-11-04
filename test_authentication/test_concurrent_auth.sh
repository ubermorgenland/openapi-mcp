#!/bin/bash

# Concurrent Authentication Test Suite
# Tests thread safety and concurrent request handling without conflicts
# Verifies that multiple simultaneous requests don't interfere with each other

set -e

# Load test secrets
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/load_test_secrets.sh"

# Configuration
SERVER_URL="http://localhost:8080"
ENDPOINT="/weather"
API_NAME="Concurrent Auth Test"
LOG_FILE="$SCRIPT_DIR/logs/concurrent_auth_test_$(date +%Y%m%d_%H%M%S).log"
NUM_CONCURRENT=10
NUM_ITERATIONS=5

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
CONCURRENT_CONFLICTS=0

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

print_header() {
    echo -e "${BLUE}=================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}=================================${NC}"
}

print_test() {
    echo -e "${YELLOW}TEST: $1${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

print_success() {
    echo -e "${GREEN}✓ PASS: $1${NC}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
}

print_failure() {
    echo -e "${RED}✗ FAIL: $1${NC}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
}

print_concurrent() {
    echo -e "${CYAN}⚡ CONCURRENT: $1${NC}"
}

# Initialize session
initialize_session() {
    local session_response
    session_response=$(curl -s -v -X POST "$SERVER_URL$ENDPOINT" \
        -H "Content-Type: application/json" \
        -d '{
            "jsonrpc": "2.0",
            "method": "initialize",
            "id": 1,
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {"roots": {"listChanged": true}, "sampling": {}},
                "clientInfo": {"name": "Concurrent Auth Test", "version": "1.0.0"}
            }
        }' 2>&1)

    SESSION_ID=$(echo "$session_response" | grep -o "mcp-session-[a-zA-Z0-9-]*" | head -1)
    if [ -z "$SESSION_ID" ]; then
        echo "Failed to get session ID"
        exit 1
    fi
    echo "Session ID: $SESSION_ID"
}

# Single concurrent request test
concurrent_request() {
    local request_id=$1
    local api_key=$2
    local city=$3
    local temp_file="/tmp/concurrent_test_${request_id}.json"
    
    # Make request with unique parameters per thread
    curl -s -X POST "$SERVER_URL$ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Mcp-Session-Id: $SESSION_ID" \
        -d '{
            "jsonrpc": "2.0",
            "method": "tools/call",
            "id": '"$request_id"',
            "params": {
                "name": "realtime-weather",
                "arguments": {
                    "q": "'"$city"'",
                    "key": "'"$api_key"'"
                }
            }
        }' > "$temp_file" 2>&1
    
    # Analyze response
    local success="false"
    local has_conflict="false"
    
    if grep -q '"HTTP GET"' "$temp_file" && grep -q 'Status: 200' "$temp_file"; then
        success="true"
        # Check if this request got data for the wrong city (indicating conflict)
        if ! grep -q "\"name\":\"$city\"" "$temp_file"; then
            has_conflict="true"
            echo "Request $request_id: CONFLICT - got data for wrong city!" >> "$LOG_FILE"
        fi
    elif grep -q '"error"' "$temp_file" && grep -q 'API key is invalid' "$temp_file"; then
        success="true" # Expected for invalid keys
    fi
    
    # Write results
    echo "${request_id}:${api_key}:${city}:${success}:${has_conflict}" >> "/tmp/concurrent_results.tmp"
    rm -f "$temp_file"
}

print_header "$API_NAME Suite"
log "Starting concurrent authentication tests"

# Test 1: Server connectivity
print_test "Server connectivity check"
if curl -s --connect-timeout 5 "$SERVER_URL/health" > /dev/null 2>&1; then
    print_success "Server is accessible"
else
    print_failure "Server is not accessible at $SERVER_URL"
    exit 1
fi

# Initialize session
print_header "Session Initialization"
initialize_session

# Test 2: Sequential baseline
print_header "Sequential Baseline Test"
print_test "Sequential requests (baseline)"

rm -f "/tmp/concurrent_results.tmp"
for i in $(seq 1 5); do
    key_index=$((i % ${#TEST_KEYS[@]}))
    key=${TEST_KEYS[$key_index]}
    city="TestCity$i"
    concurrent_request "$i" "$key" "$city" &
    wait $! # Wait for each to complete (sequential)
done

sequential_results=$(wc -l < "/tmp/concurrent_results.tmp")
if [ "$sequential_results" -eq 5 ]; then
    print_success "Sequential baseline test completed ($sequential_results requests)"
else
    print_failure "Sequential baseline test incomplete ($sequential_results/5 requests)"
fi

# Test 3: Concurrent requests with different keys
print_header "Concurrent Authentication Test"
print_concurrent "Testing $NUM_CONCURRENT concurrent requests per iteration"

rm -f "/tmp/concurrent_results.tmp"

for iteration in $(seq 1 $NUM_ITERATIONS); do
    print_test "Concurrent batch $iteration/$NUM_ITERATIONS"
    
    # Launch multiple concurrent requests
    pids=()
    for i in $(seq 1 $NUM_CONCURRENT); do
        request_id="iter${iteration}_req${i}"
        key_index=$((i % ${#TEST_KEYS[@]}))
        key=${TEST_KEYS[$key_index]}
        city="City${iteration}${i}"
        
        concurrent_request "$request_id" "$key" "$city" &
        pids+=($!)
    done
    
    # Wait for all concurrent requests to complete
    for pid in "${pids[@]}"; do
        wait $pid
    done
    
    print_concurrent "Batch $iteration completed"
done

# Analyze results
print_header "Concurrent Test Analysis"

total_requests=$(wc -l < "/tmp/concurrent_results.tmp")
successful_requests=$(grep ':true:' "/tmp/concurrent_results.tmp" | wc -l)
conflict_requests=$(grep ':true$' "/tmp/concurrent_results.tmp" | wc -l)

print_test "Concurrent request completion"
if [ "$total_requests" -eq $((NUM_CONCURRENT * NUM_ITERATIONS)) ]; then
    print_success "All $total_requests concurrent requests completed"
else
    print_failure "Request completion issue: $total_requests/expected_count completed"
fi

print_test "Authentication isolation (no cross-request conflicts)"
if [ "$conflict_requests" -eq 0 ]; then
    print_success "No authentication conflicts detected across concurrent requests"
else
    print_failure "$conflict_requests authentication conflicts detected"
    CONCURRENT_CONFLICTS=$conflict_requests
fi

print_test "Valid key authentication in concurrent environment" 
valid_key_requests=$(grep "$WEATHER_API_KEY" "/tmp/concurrent_results.tmp" | grep ':true:false$' | wc -l)
if [ "$valid_key_requests" -gt 0 ]; then
    print_success "Valid API key worked correctly in concurrent requests ($valid_key_requests successes)"
else
    print_failure "Valid API key failed in concurrent environment"
fi

print_test "Invalid key rejection in concurrent environment"
invalid_key_requests=$(grep -v "$WEATHER_API_KEY" "/tmp/concurrent_results.tmp" | grep ':true:false$' | wc -l)
expected_invalid=$((total_requests - valid_key_requests))
if [ "$invalid_key_requests" -eq "$expected_invalid" ]; then
    print_success "Invalid API keys correctly rejected in concurrent requests"
else
    print_failure "Invalid key handling inconsistent in concurrent environment"
fi

# Test 4: Mixed authentication methods concurrently
print_header "Mixed Authentication Methods Test"

print_test "Concurrent tool args vs header authentication"
pids=()

# Tool argument authentication
for i in $(seq 1 3); do
    curl -s -X POST "$SERVER_URL$ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Mcp-Session-Id: $SESSION_ID" \
        -d '{
            "jsonrpc": "2.0",
            "method": "tools/call",
            "id": 100,
            "params": {
                "name": "realtime-weather", 
                "arguments": {"q": "ToolArg'$i'", "key": "'$WEATHER_API_KEY'"}
            }
        }' > "/tmp/toolarg_$i.json" &
    pids+=($!)
done

# Header authentication 
for i in $(seq 1 3); do
    curl -s -X POST "$SERVER_URL$ENDPOINT" \
        -H "Content-Type: application/json" \
        -H "Mcp-Session-Id: $SESSION_ID" \
        -H "X-API-Key: $WEATHER_API_KEY" \
        -d '{
            "jsonrpc": "2.0",
            "method": "tools/call",
            "id": 200,
            "params": {
                "name": "realtime-weather",
                "arguments": {"q": "Header'$i'"}
            }
        }' > "/tmp/header_$i.json" &
    pids+=($!)
done

# Wait for all mixed method requests
for pid in "${pids[@]}"; do
    wait $pid
done

# Check mixed method results
mixed_success=0
for i in $(seq 1 3); do
    if grep -q 'Status: 200' "/tmp/toolarg_$i.json" && grep -q "ToolArg$i" "/tmp/toolarg_$i.json"; then
        mixed_success=$((mixed_success + 1))
    fi
    if grep -q 'Status: 200' "/tmp/header_$i.json" && grep -q "Header$i" "/tmp/header_$i.json"; then
        mixed_success=$((mixed_success + 1))
    fi
done

if [ "$mixed_success" -eq 6 ]; then
    print_success "Mixed authentication methods work correctly in concurrent requests"
else
    print_failure "Mixed authentication methods have conflicts ($mixed_success/6 succeeded)"
fi

# Cleanup
rm -f /tmp/concurrent_*.tmp /tmp/toolarg_*.json /tmp/header_*.json /tmp/concurrent_results.tmp

# Final results
print_header "Concurrent Authentication Test Results"

echo -e "${BLUE}Test Environment:${NC}"
echo -e "${BLUE}• Server: $SERVER_URL$ENDPOINT${NC}"
echo -e "${BLUE}• Concurrent Requests: $NUM_CONCURRENT per iteration${NC}" 
echo -e "${BLUE}• Test Iterations: $NUM_ITERATIONS${NC}"
echo -e "${BLUE}• Total Concurrent Requests: $((NUM_CONCURRENT * NUM_ITERATIONS))${NC}"
echo -e "${BLUE}• Authentication Methods Tested: Tool Args, Headers, Mixed${NC}"
echo ""

echo -e "${BLUE}Results Summary:${NC}"
echo -e "${BLUE}Total Tests: $TOTAL_TESTS${NC}"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [ "$CONCURRENT_CONFLICTS" -eq 0 ]; then
    echo -e "${GREEN}🎯 THREAD SAFETY: ✅ VERIFIED${NC}"
    echo -e "${GREEN}✅ No authentication conflicts in concurrent requests${NC}"
    echo -e "${GREEN}✅ Each request maintained isolated authentication context${NC}"
    echo -e "${GREEN}✅ No global state pollution detected${NC}"
else
    echo -e "${RED}🚨 THREAD SAFETY: ❌ CONFLICTS DETECTED${NC}"
    echo -e "${RED}❌ $CONCURRENT_CONFLICTS authentication conflicts found${NC}"
fi

echo ""
if [ $FAILED_TESTS -eq 0 ] && [ $CONCURRENT_CONFLICTS -eq 0 ]; then
    echo -e "${GREEN}🎉 ALL CONCURRENT TESTS PASSED! 🎉${NC}"
    echo -e "${GREEN}System supports concurrent requests without conflicts!${NC}"
    log "All concurrent authentication tests completed successfully"
    exit 0
else
    echo -e "${RED}❌ CONCURRENT TEST FAILURES DETECTED${NC}"
    log "Concurrent authentication tests completed with $FAILED_TESTS failures and $CONCURRENT_CONFLICTS conflicts"
    exit 1
fi

echo ""
echo "Detailed log saved to: $LOG_FILE"