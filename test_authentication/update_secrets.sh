#!/bin/bash

# Script to update all test files to use centralized secrets loading

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# List of files to update
TEST_FILES=(
    "test_weather_sse.sh"
    "test_twitter_streamable.sh"
    "test_twitter_sse.sh"
    "test_youtube_streamable.sh"
    "test_youtube_sse.sh"
    "test_perplexity_sse.sh"
    "test_perplexity_streamable.sh"
    "test_google_keywords_sse.sh"
    "test_google_keywords_streamable.sh"
    "test_google_finance_streamable.sh"
    "test_alpha_vantage_streamable.sh"
    "test_alpha_vantage_sse.sh"
    "test_google_finance_sse.sh"
)

# Function to get new variable name for old variable
get_new_var() {
    case "$1" in
        "VALID_API_KEY") echo "WEATHER_API_KEY" ;;
        "VALID_RAPIDAPI_KEY") echo "TWITTER_RAPIDAPI_KEY" ;;
        "VALID_BEARER_TOKEN") echo "PERPLEXITY_BEARER_TOKEN" ;;
        "GOOGLE_FINANCE_RAPIDAPI_KEY") echo "GOOGLE_FINANCE_RAPIDAPI_KEY" ;;
        "GOOGLE_KEYWORDS_RAPIDAPI_KEY") echo "GOOGLE_KEYWORDS_RAPIDAPI_KEY" ;;
        "ALPHA_VANTAGE_API_KEY") echo "ALPHA_VANTAGE_API_KEY" ;;
        "YOUTUBE_RAPIDAPI_KEY") echo "YOUTUBE_RAPIDAPI_KEY" ;;
        *) echo "$1" ;;
    esac
}

echo "🔄 Updating test files to use centralized secrets..."

for file in "${TEST_FILES[@]}"; do
    if [[ -f "$SCRIPT_DIR/$file" ]]; then
        echo "📝 Processing $file..."
        
        # Add secrets loading at the top (after set -e)
        if ! grep -q "load_test_secrets.sh" "$SCRIPT_DIR/$file"; then
            sed -i.bak '/^set -e$/a\
\
# Load test secrets\
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"\
source "$SCRIPT_DIR/load_test_secrets.sh"\
' "$SCRIPT_DIR/$file"
        fi
        
        # Remove hardcoded API key declarations and replace usage
        old_vars=("VALID_API_KEY" "VALID_RAPIDAPI_KEY" "VALID_BEARER_TOKEN" "GOOGLE_FINANCE_RAPIDAPI_KEY" "GOOGLE_KEYWORDS_RAPIDAPI_KEY" "ALPHA_VANTAGE_API_KEY" "YOUTUBE_RAPIDAPI_KEY")
        
        for old_var in "${old_vars[@]}"; do
            new_var=$(get_new_var "$old_var")
            
            # Remove variable declarations like: VALID_API_KEY="..."
            sed -i.bak "/${old_var}=\".*\"/d" "$SCRIPT_DIR/$file"
            
            # Replace variable usage
            sed -i.bak "s/\$${old_var}/\$${new_var}/g" "$SCRIPT_DIR/$file"
        done
        
        # Clean up backup files
        rm -f "$SCRIPT_DIR/${file}.bak"
        
        echo "✅ Updated $file"
    else
        echo "⚠️  File not found: $file"
    fi
done

echo "🎉 All test files updated successfully!"
echo "📋 Updated files now use centralized secrets from .env.test via load_test_secrets.sh"