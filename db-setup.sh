#!/bin/bash

# Database connection setup script
# Use this to set the correct DATABASE_URL for your environment

# Set your database URL - replace with your actual credentials:
# export DATABASE_URL="postgresql://username:password@localhost:5432/database_name?sslmode=disable"

echo "⚠️  Please set your DATABASE_URL environment variable before running this script"
echo "📝 Example: export DATABASE_URL=\"postgresql://username:password@localhost:5432/database_name?sslmode=disable\""
echo "🔗 Connection will be established once you set the DATABASE_URL"
echo ""
echo "📊 Current database specs:"
./bin/spec-manager active
echo ""
echo "🚀 To start the server with database specs:"
echo "   source db-setup.sh"
echo "   ./bin/openapi-mcp-main"
echo ""
echo "🔧 To manage specs:"
echo "   ./bin/spec-manager list"
echo "   ./bin/spec-manager set-token <id> '<token>'"
echo "   ./bin/spec-manager activate <id>"
echo "   ./bin/spec-manager deactivate <id>"