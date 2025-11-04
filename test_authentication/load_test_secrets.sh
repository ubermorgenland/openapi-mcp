#!/bin/bash

# Test Secrets Loader
# This script loads test secrets from .env.test file
# Source this file in your test scripts: source ./load_test_secrets.sh

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env.test"

# Check if .env.test file exists
if [[ ! -f "$ENV_FILE" ]]; then
    echo "ERROR: Test secrets file not found: $ENV_FILE"
    echo "Please create .env.test file with your test credentials."
    echo "See .env.test.example for the required format."
    exit 1
fi

# Load environment variables from .env.test
set -a  # automatically export all variables
source "$ENV_FILE"
set +a  # stop automatically exporting

# Validate that required secrets are set (not placeholder values)
validate_secret() {
    local var_name="$1"
    local var_value="${!var_name}"
    
    if [[ -z "$var_value" ]] || [[ "$var_value" == "REPLACE_WITH_YOUR_"* ]]; then
        echo "WARNING: $var_name is not set or still contains placeholder value"
        echo "Please update .env.test with your actual test credentials"
        return 1
    fi
    return 0
}

# Function to check if secrets are properly configured
check_test_secrets() {
    local missing_secrets=0
    
    echo "Checking test secrets configuration..."
    
    # Check each required secret
    validate_secret "TWITTER_RAPIDAPI_KEY" || ((missing_secrets++))
    validate_secret "GOOGLE_ANALYTICS_BEARER_TOKEN" || ((missing_secrets++))
    validate_secret "GOOGLE_FINANCE_RAPIDAPI_KEY" || ((missing_secrets++))
    validate_secret "GOOGLE_KEYWORDS_RAPIDAPI_KEY" || ((missing_secrets++))
    validate_secret "ALPHA_VANTAGE_API_KEY" || ((missing_secrets++))
    validate_secret "WEATHER_API_KEY" || ((missing_secrets++))
    validate_secret "PERPLEXITY_BEARER_TOKEN" || ((missing_secrets++))
    validate_secret "YOUTUBE_RAPIDAPI_KEY" || ((missing_secrets++))
    
    if [[ $missing_secrets -gt 0 ]]; then
        echo "ERROR: $missing_secrets test secrets need to be configured"
        echo "Please update $ENV_FILE with your actual test credentials"
        return 1
    fi
    
    echo "✓ All test secrets are configured"
    return 0
}

# Set legacy variable names for backward compatibility
export VALID_RAPIDAPI_KEY="$TWITTER_RAPIDAPI_KEY"
export VALID_BEARER_TOKEN="$GOOGLE_ANALYTICS_BEARER_TOKEN"
export VALID_API_KEY="$WEATHER_API_KEY"
export TEST_PROPERTY_ID="$GOOGLE_ANALYTICS_PROPERTY_ID"

# Update concurrent test keys array
if [[ -n "$WEATHER_API_KEY" ]] && [[ "$WEATHER_API_KEY" != "REPLACE_WITH_YOUR_"* ]]; then
    export VALID_KEY="$WEATHER_API_KEY"
    export TEST_KEYS=("key1_invalid" "key2_invalid" "key3_invalid" "$VALID_KEY" "key5_invalid")
fi

echo "Test secrets loaded from $ENV_FILE"