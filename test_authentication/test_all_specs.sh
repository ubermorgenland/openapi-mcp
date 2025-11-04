#!/bin/bash

# =============================================================================
# Master Authentication Test Suite - All API Specifications
# =============================================================================
# This script runs all individual API authentication tests and provides a
# comprehensive report of the OpenAPI MCP server authentication functionality
# across all supported API specifications.
# =============================================================================

set -e

# Load test secrets
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/load_test_secrets.sh"

# Configuration
SERVER_PORT=${SERVER_PORT:-8080}
BASE_URL="http://localhost:${SERVER_PORT}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_FILE="$SCRIPT_DIR/logs/master_auth_test_$(date +%s).log"
REPORT_FILE="$SCRIPT_DIR/logs/master_auth_report_$(date +%s).txt"

# Test tracking
TOTAL_API_TESTS=0
PASSED_API_TESTS=0
FAILED_API_TESTS=0
SKIPPED_API_TESTS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Test definitions
TEST_FILES=("google_keywords" "perplexity" "weather" "twitter" "google_finance" "alpha_vantage")
TEST_NAMES=("Google Keywords API" "Perplexity AI API" "WeatherAPI.com" "Twitter Social API" "Google Finance API" "Alpha Vantage Financial API")

# Results arrays
RESULTS=()
DURATIONS=()
DETAILS=()

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Enhanced logging for reports
report_log() {
    echo "$1" | tee -a "$REPORT_FILE"
}

# Print header
print_header() {
    echo -e "${PURPLE}${BOLD}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}${BOLD}║                    OpenAPI MCP Authentication Test Suite                    ║${NC}"
    echo -e "${PURPLE}${BOLD}║                           Master Test Runner                               ║${NC}"
    echo -e "${PURPLE}${BOLD}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo
    echo -e "${BLUE}Testing all API specifications for authentication functionality${NC}"
    echo -e "${BLUE}Server: ${BASE_URL}${NC}"
    echo -e "${BLUE}Test Directory: ${SCRIPT_DIR}${NC}"
    echo -e "${BLUE}Master Log: ${LOG_FILE}${NC}"
    echo -e "${BLUE}Final Report: ${REPORT_FILE}${NC}"
    echo
}

# Check server connectivity
check_server() {
    echo -e "${CYAN}=== Pre-flight Server Check ===${NC}"
    
    if curl -s --max-time 10 "$BASE_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Server is responsive at $BASE_URL${NC}"
        log "Server connectivity check: PASSED"
        return 0
    else
        echo -e "${RED}✗ Server not accessible at $BASE_URL${NC}"
        echo -e "${YELLOW}Please ensure the OpenAPI MCP server is running on port $SERVER_PORT${NC}"
        log "Server connectivity check: FAILED"
        return 1
    fi
}

# Get test name by index
get_test_name() {
    local index=$1
    echo "${TEST_NAMES[$index]}"
}

# Run individual test
run_individual_test() {
    local index=$1
    local test_key="${TEST_FILES[$index]}"
    local test_name="$(get_test_name $index)"
    local test_file="${SCRIPT_DIR}/test_${test_key}.sh"
    
    echo -e "${BLUE}┌─────────────────────────────────────────────────────────────────────────────┐${NC}"
    echo -e "${BLUE}│ Testing: ${test_name}${NC}"
    echo -e "${BLUE}└─────────────────────────────────────────────────────────────────────────────┘${NC}"
    
    TOTAL_API_TESTS=$((TOTAL_API_TESTS + 1))
    
    # Check if test file exists
    if [ ! -f "$test_file" ]; then
        echo -e "${RED}✗ Test file not found: $test_file${NC}"
        RESULTS[$index]="MISSING"
        DETAILS[$index]="Test file not found"
        SKIPPED_API_TESTS=$((SKIPPED_API_TESTS + 1))
        log "Test $test_key: SKIPPED - file not found"
        return
    fi
    
    # Check if test file is executable
    if [ ! -x "$test_file" ]; then
        echo -e "${YELLOW}⚠ Making test file executable...${NC}"
        chmod +x "$test_file"
    fi
    
    # Run the test and capture results
    local start_time=$(date +%s)
    local test_output
    local test_exit_code
    
    echo -e "${CYAN}Running $test_file...${NC}"
    
    # Capture both stdout and exit code
    if test_output=$(timeout 300 "$test_file" 2>&1); then
        test_exit_code=0
    else
        test_exit_code=$?
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    DURATIONS[$index]=$duration
    
    # Parse test results from output
    local total_tests=$(echo "$test_output" | grep -o "Total Tests: [0-9]*" | grep -o "[0-9]*" | head -1)
    local passed_tests=$(echo "$test_output" | grep -o "Passed: [0-9]*" | grep -o "[0-9]*" | head -1)
    local failed_tests=$(echo "$test_output" | grep -o "Failed: [0-9]*" | grep -o "[0-9]*" | head -1)
    
    # Set defaults if parsing failed
    total_tests=${total_tests:-0}
    passed_tests=${passed_tests:-0}
    failed_tests=${failed_tests:-0}
    
    # Determine overall test result
    if [ $test_exit_code -eq 0 ] && [ "$failed_tests" -eq 0 ] && [ "$passed_tests" -gt 0 ]; then
        RESULTS[$index]="PASS"
        DETAILS[$index]="$passed_tests/$total_tests tests passed in ${duration}s"
        PASSED_API_TESTS=$((PASSED_API_TESTS + 1))
        echo -e "${GREEN}✅ $test_name: ALL TESTS PASSED ($passed_tests/$total_tests)${NC}"
    elif [ $test_exit_code -eq 124 ]; then
        RESULTS[$index]="TIMEOUT"
        DETAILS[$index]="Test timed out after 300 seconds"
        FAILED_API_TESTS=$((FAILED_API_TESTS + 1))
        echo -e "${RED}⏱️ $test_name: TIMEOUT${NC}"
    elif [ "$total_tests" -gt 0 ] && [ "$passed_tests" -gt 0 ]; then
        RESULTS[$index]="PARTIAL"
        DETAILS[$index]="$passed_tests/$total_tests tests passed, $failed_tests failed in ${duration}s"
        FAILED_API_TESTS=$((FAILED_API_TESTS + 1))
        echo -e "${YELLOW}⚠️ $test_name: PARTIAL SUCCESS ($passed_tests/$total_tests passed)${NC}"
    else
        RESULTS[$index]="FAIL"
        DETAILS[$index]="All tests failed or no tests found in ${duration}s"
        FAILED_API_TESTS=$((FAILED_API_TESTS + 1))
        echo -e "${RED}❌ $test_name: FAILED${NC}"
    fi
    
    # Log detailed results
    log "Test $test_key ($test_name): ${RESULTS[$index]} - ${DETAILS[$index]}"
    
    echo
}

# Generate comprehensive report
generate_report() {
    echo -e "${PURPLE}${BOLD}=== GENERATING COMPREHENSIVE REPORT ===${NC}"
    
    # Report header
    report_log "╔══════════════════════════════════════════════════════════════════════════════╗"
    report_log "║                OpenAPI MCP Authentication Test Report                       ║"
    report_log "║                          $(date '+%Y-%m-%d %H:%M:%S')                          ║"
    report_log "╚══════════════════════════════════════════════════════════════════════════════╝"
    report_log ""
    report_log "Server: $BASE_URL"
    report_log "Test Directory: $SCRIPT_DIR"
    report_log "Report Generated: $(date)"
    report_log ""
    
    # Executive Summary
    report_log "EXECUTIVE SUMMARY"
    report_log "=================="
    report_log "Total API Tests:    $TOTAL_API_TESTS"
    report_log "Passed:             $PASSED_API_TESTS"
    report_log "Failed/Partial:     $FAILED_API_TESTS"
    report_log "Skipped:            $SKIPPED_API_TESTS"
    
    local success_rate=0
    if [ $TOTAL_API_TESTS -gt 0 ]; then
        success_rate=$(( (PASSED_API_TESTS * 100) / TOTAL_API_TESTS ))
    fi
    report_log "Success Rate:       ${success_rate}%"
    report_log ""
    
    # Detailed Results
    report_log "DETAILED RESULTS BY API"
    report_log "======================="
    
    for i in $(seq 0 $((${#TEST_FILES[@]} - 1))); do
        local test_name="$(get_test_name $i)"
        local result="${RESULTS[$i]:-NOT_RUN}"
        local details="${DETAILS[$i]:-No details available}"
        local duration="${DURATIONS[$i]:-0}"
        
        report_log ""
        report_log "API: $test_name"
        report_log "  Status:    $result"
        report_log "  Duration:  ${duration}s"
        report_log "  Details:   $details"
        
        # Add status symbol for quick visual scanning
        case "$result" in
            "PASS")    report_log "  Result:    ✅ FULLY OPERATIONAL" ;;
            "PARTIAL") report_log "  Result:    ⚠️  PARTIALLY FUNCTIONAL" ;;
            "FAIL")    report_log "  Result:    ❌ AUTHENTICATION ISSUES" ;;
            "TIMEOUT") report_log "  Result:    ⏱️  TIMED OUT" ;;
            "MISSING") report_log "  Result:    🔍 TEST FILE MISSING" ;;
            *)         report_log "  Result:    ❓ UNKNOWN STATUS" ;;
        esac
    done
    
    # Authentication Analysis
    report_log ""
    report_log "AUTHENTICATION SYSTEM ANALYSIS"
    report_log "==============================="
    
    if [ $PASSED_API_TESTS -eq $TOTAL_API_TESTS ] && [ $TOTAL_API_TESTS -gt 0 ]; then
        report_log "🎉 EXCELLENT: All API authentications working perfectly!"
        report_log "   ✓ Database integration successful across all APIs"
        report_log "   ✓ Authentication priority hierarchy functioning"
        report_log "   ✓ Secure context-based authentication operational"
    elif [ $PASSED_API_TESTS -gt 0 ]; then
        report_log "⚠️  MIXED RESULTS: Some APIs working, others need attention"
        report_log "   • Working APIs indicate core authentication system is functional"
        report_log "   • Failed APIs may have invalid keys or connectivity issues"
        report_log "   • Recommend checking database API key values for failed services"
    else
        report_log "🚨 CRITICAL: No APIs are fully functional"
        report_log "   • Core authentication system may have fundamental issues"
        report_log "   • Check server configuration and database connectivity"
        report_log "   • Verify authentication context implementation"
    fi
}

# Display final summary
display_summary() {
    echo
    echo -e "${PURPLE}${BOLD}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${PURPLE}${BOLD}║                              FINAL SUMMARY                                  ║${NC}"
    echo -e "${PURPLE}${BOLD}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo
    
    # Overall statistics
    echo -e "${BOLD}Authentication Test Results:${NC}"
    echo -e "├─ Total API Tests: ${TOTAL_API_TESTS}"
    echo -e "├─ ${GREEN}Passed: ${PASSED_API_TESTS}${NC}"
    echo -e "├─ ${RED}Failed/Partial: ${FAILED_API_TESTS}${NC}"
    echo -e "└─ ${YELLOW}Skipped: ${SKIPPED_API_TESTS}${NC}"
    echo
    
    # Success rate
    local success_rate=0
    if [ $TOTAL_API_TESTS -gt 0 ]; then
        success_rate=$(( (PASSED_API_TESTS * 100) / TOTAL_API_TESTS ))
    fi
    
    echo -e "${BOLD}Overall Success Rate: ${NC}"
    if [ $success_rate -ge 90 ]; then
        echo -e "${GREEN}${BOLD}${success_rate}% - EXCELLENT${NC} 🎉"
    elif [ $success_rate -ge 70 ]; then
        echo -e "${YELLOW}${BOLD}${success_rate}% - GOOD${NC} 👍"
    elif [ $success_rate -ge 50 ]; then
        echo -e "${YELLOW}${BOLD}${success_rate}% - FAIR${NC} ⚠️"
    else
        echo -e "${RED}${BOLD}${success_rate}% - NEEDS ATTENTION${NC} 🚨"
    fi
    echo
    
    # Individual results summary
    echo -e "${BOLD}API Status Overview:${NC}"
    for i in $(seq 0 $((${#TEST_FILES[@]} - 1))); do
        local test_name="$(get_test_name $i)"
        local result="${RESULTS[$i]:-NOT_RUN}"
        
        case "$result" in
            "PASS")    echo -e "├─ ${GREEN}✅ $test_name${NC}" ;;
            "PARTIAL") echo -e "├─ ${YELLOW}⚠️  $test_name${NC}" ;;
            "FAIL")    echo -e "├─ ${RED}❌ $test_name${NC}" ;;
            "TIMEOUT") echo -e "├─ ${RED}⏱️  $test_name${NC}" ;;
            "MISSING") echo -e "├─ ${YELLOW}🔍 $test_name${NC}" ;;
            *)         echo -e "├─ ${YELLOW}❓ $test_name${NC}" ;;
        esac
    done
    echo
    
    # File references
    echo -e "${BOLD}Generated Files:${NC}"
    echo -e "├─ Detailed Report: ${CYAN}$REPORT_FILE${NC}"
    echo -e "└─ Master Log:      ${CYAN}$LOG_FILE${NC}"
    echo
}

# Main execution
main() {
    print_header
    
    # Pre-flight checks
    if ! check_server; then
        echo -e "${RED}Cannot proceed without server connectivity${NC}"
        exit 1
    fi
    
    echo
    echo -e "${CYAN}Starting individual API authentication tests...${NC}"
    echo
    
    # Initialize results arrays
    for i in $(seq 0 $((${#TEST_FILES[@]} - 1))); do
        RESULTS[$i]="NOT_RUN"
        DURATIONS[$i]=0
        DETAILS[$i]="Not executed"
    done
    
    # Run all individual tests
    for i in $(seq 0 $((${#TEST_FILES[@]} - 1))); do
        run_individual_test $i
    done
    
    # Generate comprehensive report
    generate_report
    
    # Display summary
    display_summary
    
    # Set exit code based on overall results
    if [ $FAILED_API_TESTS -eq 0 ] && [ $PASSED_API_TESTS -gt 0 ]; then
        echo -e "${GREEN}${BOLD}🎉 All authentication tests completed successfully!${NC}"
        exit 0
    elif [ $PASSED_API_TESTS -gt 0 ]; then
        echo -e "${YELLOW}${BOLD}⚠️  Some tests failed - see report for details${NC}"
        exit 1
    else
        echo -e "${RED}${BOLD}🚨 Critical authentication issues detected${NC}"
        exit 2
    fi
}

# Handle script interruption
trap 'echo -e "\n${YELLOW}Master test suite interrupted by user${NC}"; exit 130' INT

# Execute main function
main "$@"