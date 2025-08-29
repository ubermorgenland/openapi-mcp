#!/bin/bash

# Demo script showing smart authentication type detection
# Run this to see how the system automatically detects Bearer vs API key authentication

echo "🚀 OpenAPI MCP Smart Authentication Demo"
echo "========================================"

# Set the database URL
# Set your database URL - replace with your actual credentials:
# export DATABASE_URL="postgresql://username:password@localhost:5432/database_name?sslmode=disable"

echo "📊 Current specs in database:"
./bin/spec-manager active
echo ""

echo "🧠 Starting server with smart authentication detection..."
echo "   Look for these log lines:"
echo "   - 'Found security scheme' - shows detected authentication type"
echo "   - 'Will use database token as BEARER TOKEN' - for Bearer auth APIs"  
echo "   - 'Will use database token as API KEY' - for API key APIs"
echo ""

# Start server for a few seconds to show the authentication detection
timeout 10s ./bin/openapi-mcp-main 2>&1 | grep -E "(Found security scheme|Will use database token|Using database token as|Mounted)" | head -15

echo ""
echo "✨ Key Features Demonstrated:"
echo "🔍 Authentication Type Detection:"
echo "   • Perplexity API → Bearer Token (automatically detected)"
echo "   • Weather API → API Key in query parameter (automatically detected)"
echo "   • Twitter/YouTube/Google APIs → API Key in header (automatically detected)"
echo ""
echo "🎯 Single Token Field:"
echo "   • All tokens stored in 'api_key_token' database field"
echo "   • System automatically applies as Bearer or API key based on OpenAPI spec"
echo "   • No need for separate fields or manual configuration"
echo ""
echo "🏆 Priority System:"
echo "   1. Database token (applied as correct auth type)"
echo "   2. Environment variables ({ENDPOINT}_API_KEY, {ENDPOINT}_BEARER_TOKEN)"
echo "   3. General environment variables (GENERAL_API_KEY, GENERAL_BEARER_TOKEN)"