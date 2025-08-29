# OpenAPI Spec Management API Examples

This document provides comprehensive curl examples for managing OpenAPI specifications via the HTTP API.

## 🚀 Quick Start

```bash
# Set up database connection
export DATABASE_URL="postgresql://username:password@host:port/database"

# Build and start the API server
make all
bin/spec-api-server :8090
```

The API server will be available at `http://localhost:8090`

## 📚 Available Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/specs` | List all specs |
| `POST` | `/specs` | Create/import new spec |
| `GET` | `/specs/active` | List only active specs |
| `GET` | `/specs/{id}` | Get specific spec by ID |
| `PUT` | `/specs/{id}` | Update spec |
| `DELETE` | `/specs/{id}` | Delete spec |
| `POST` | `/specs/{id}/activate` | Activate spec |
| `POST` | `/specs/{id}/deactivate` | Deactivate spec |
| `PUT` | `/specs/{id}/token` | Update API key token |
| `GET` | `/health` | Health check |
| `GET` | `/swagger` | OpenAPI specification |

## 🔧 curl Examples

### 1. Health Check

```bash
curl -X GET http://localhost:8090/health
```

**Response:**
```json
{
  "status": "healthy"
}
```

### 2. List All Specs

```bash
curl -X GET http://localhost:8090/specs | jq '.'
```

**Response:**
```json
{
  "success": true,
  "message": "Specs retrieved successfully",
  "data": [
    {
      "id": 1,
      "name": "weather",
      "title": "OpenWeatherMap API",
      "version": "2.5",
      "endpoint_path": "/weather",
      "file_format": "json",
      "file_size": 65561,
      "api_key_token": "sk-abc123...",
      "is_active": true,
      "created_at": "2024-08-27T10:30:00Z",
      "updated_at": "2024-08-27T10:30:00Z"
    },
    {
      "id": 2,
      "name": "twitter",
      "title": "Twitter API v2",
      "version": "2.0",
      "endpoint_path": "/twitter",
      "file_format": "yaml",
      "file_size": 672605,
      "is_active": false,
      "created_at": "2024-08-27T10:31:00Z",
      "updated_at": "2024-08-27T10:31:00Z"
    }
  ]
}
```

### 3. List Active Specs Only

```bash
curl -X GET http://localhost:8090/specs/active | jq '.data[].name'
```

**Response:**
```json
{
  "success": true,
  "message": "Active specs retrieved successfully", 
  "data": [
    {
      "id": 1,
      "name": "weather",
      "title": "OpenWeatherMap API",
      "endpoint_path": "/weather",
      "is_active": true
    }
  ]
}
```

### 4. Import/Create New Spec

#### From JSON Content

```bash
curl -X POST http://localhost:8090/specs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "petstore",
    "endpoint_path": "/petstore",
    "file_format": "json",
    "active": true,
    "spec_content": "{\"openapi\":\"3.0.0\",\"info\":{\"title\":\"Pet Store API\",\"version\":\"1.0.0\"},\"paths\":{\"/pets\":{\"get\":{\"operationId\":\"listPets\",\"responses\":{\"200\":{\"description\":\"List of pets\"}}}}}}"
  }' | jq '.'
```

#### From JSON Content with API Key Token

```bash
curl -X POST http://localhost:8090/specs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "petstore-secured",
    "endpoint_path": "/petstore-secured",
    "file_format": "json",
    "api_key_token": "pk-petstore-abc123xyz789",
    "active": true,
    "spec_content": "{\"openapi\":\"3.0.0\",\"info\":{\"title\":\"Secured Pet Store API\",\"version\":\"1.0.0\"},\"paths\":{\"/pets\":{\"get\":{\"operationId\":\"listPets\",\"responses\":{\"200\":{\"description\":\"List of pets\"}}}}}}"
  }' | jq '.'
```

#### From YAML Content

```bash
curl -X POST http://localhost:8090/specs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "bookstore",
    "endpoint_path": "/bookstore", 
    "file_format": "yaml",
    "active": true,
    "spec_content": "openapi: 3.0.0\ninfo:\n  title: Book Store API\n  version: 1.0.0\npaths:\n  /books:\n    get:\n      operationId: listBooks\n      responses:\n        200:\n          description: List of books"
  }' | jq '.'
```

#### Import from URL

```bash
# Download spec and import
SPEC_CONTENT=$(curl -s https://petstore3.swagger.io/api/v3/openapi.json)

curl -X POST http://localhost:8090/specs \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"petstore-v3\",
    \"endpoint_path\": \"/petstore-v3\",
    \"file_format\": \"json\",
    \"active\": true,
    \"spec_content\": $(echo "$SPEC_CONTENT" | jq -R -s .)
  }" | jq '.'
```

#### Import from Local File

```bash
# Read local file and import
SPEC_CONTENT=$(cat specs/weather.json | jq -R -s .)

curl -X POST http://localhost:8090/specs \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"weather-api\",
    \"endpoint_path\": \"/weather\",
    \"file_format\": \"json\", 
    \"active\": true,
    \"spec_content\": $SPEC_CONTENT
  }" | jq '.'
```

**Success Response:**
```json
{
  "success": true,
  "message": "Spec imported successfully",
  "data": {
    "name": "weather-api",
    "endpoint_path": "/weather",
    "active": true,
    "has_api_token": false
  }
}
```

### 5. Activate Spec

```bash
curl -X POST http://localhost:8090/specs/2/activate | jq '.'
```

**Response:**
```json
{
  "success": true,
  "message": "Spec activated successfully",
  "data": {
    "id": 2
  }
}
```

### 6. Deactivate Spec

```bash
curl -X POST http://localhost:8090/specs/2/deactivate | jq '.'
```

**Response:**
```json
{
  "success": true,
  "message": "Spec deactivated successfully", 
  "data": {
    "id": 2
  }
}
```

### 7. Delete Spec

```bash
curl -X DELETE http://localhost:8090/specs/2 | jq '.'
```

**Response:**
```json
{
  "success": true,
  "message": "Spec deleted successfully",
  "data": {
    "id": 2
  }
}
```

### 8. Update API Key Token

```bash
# Set API key token for a spec
curl -X PUT http://localhost:8090/specs/1/token \
  -H "Content-Type: application/json" \
  -d '{
    "api_key_token": "sk-new-api-key-123456789"
  }' | jq '.'
```

**Response:**
```json
{
  "success": true,
  "message": "API key token updated successfully",
  "data": {
    "id": 1,
    "api_key_token_updated": true
  }
}
```

#### Clear API Key Token

```bash
# Clear/remove API key token from a spec
curl -X PUT http://localhost:8090/specs/1/token \
  -H "Content-Type: application/json" \
  -d '{
    "api_key_token": null
  }' | jq '.'
```

**Response:**
```json
{
  "success": true,
  "message": "API key token updated successfully",
  "data": {
    "id": 1,
    "api_key_token_updated": true
  }
}
```

### 9. Get OpenAPI Specification

```bash
# Get the OpenAPI specification for the spec management API
curl -X GET http://localhost:8090/swagger | jq '.'
```

**Response:**
```json
{
  "openapi": "3.0.0",
  "info": {
    "title": "OpenAPI Spec Management API",
    "description": "REST API for managing OpenAPI specifications in the database",
    "version": "1.0.0",
    "contact": {
      "name": "OpenAPI MCP",
      "url": "https://github.com/jedisct1/openapi-mcp"
    }
  },
  "servers": [
    {
      "url": "http://localhost:8090",
      "description": "Local development server"
    }
  ],
  "paths": {
    "/specs": {
      "get": {
        "summary": "List all OpenAPI specs",
        "operationId": "listSpecs",
        "responses": {
          "200": {
            "description": "Successfully retrieved specs"
          }
        }
      }
    }
  }
}
```

**Use with Swagger UI or other OpenAPI tools:**
```bash
# Use with swagger-ui-serve
npx swagger-ui-serve http://localhost:8090/swagger

# Use with redoc-cli  
npx redoc-cli serve http://localhost:8090/swagger

# Use with openapi-generator
openapi-generator generate -i http://localhost:8090/swagger -g typescript-fetch -o ./client
```

## 🔄 Batch Operations

### Import Multiple Specs from Directory

```bash
#!/bin/bash
# batch-import-api.sh

API_BASE="http://localhost:8090"
SPECS_DIR="./specs"

for spec_file in "$SPECS_DIR"/*.{json,yaml,yml}; do
    if [[ -f "$spec_file" ]]; then
        filename=$(basename "$spec_file")
        name="${filename%.*}"
        endpoint="/${name//_/-}"
        
        # Determine format
        if [[ "$filename" == *.json ]]; then
            format="json"
            content=$(jq -R -s . < "$spec_file")
        else
            format="yaml" 
            content=$(cat "$spec_file" | jq -R -s .)
        fi
        
        echo "Importing $filename..."
        
        # Optional: Add API key token from environment variable
        # Example: export API_TOKEN_PREFIX="sk-mytoken-"
        api_token=""
        if [[ -n "$API_TOKEN_PREFIX" ]]; then
            api_token="\"api_key_token\": \"${API_TOKEN_PREFIX}${name}\","
        fi
        
        response=$(curl -s -X POST "$API_BASE/specs" \
            -H "Content-Type: application/json" \
            -d "{
                \"name\": \"$name\",
                \"endpoint_path\": \"$endpoint\",
                \"file_format\": \"$format\",
                ${api_token}
                \"active\": true,
                \"spec_content\": $content
            }")
        
        if echo "$response" | jq -e '.success' > /dev/null; then
            echo "✅ Successfully imported $filename"
        else
            echo "❌ Failed to import $filename"
            echo "$response" | jq '.message'
        fi
    fi
done
```

### Bulk Activate/Deactivate

```bash
# Activate specific specs by name
API_BASE="http://localhost:8090"

# Get all specs and filter by names
SPEC_IDS=$(curl -s "$API_BASE/specs" | jq -r '.data[] | select(.name | test("twitter|alpha-vantage")) | .id')

for id in $SPEC_IDS; do
    curl -s -X POST "$API_BASE/specs/$id/activate" | jq '.message'
done
```

### Check Status of All Specs

```bash
#!/bin/bash
# check-spec-status.sh

API_BASE="http://localhost:8090"

echo "📊 Spec Status Report"
echo "===================="

# Get all specs
response=$(curl -s "$API_BASE/specs")

if ! echo "$response" | jq -e '.success' > /dev/null; then
    echo "❌ Failed to get specs"
    exit 1
fi

# Parse and display
echo "$response" | jq -r '.data[] | 
    "\(.id)\t\(.name)\t" + 
    (if .is_active then "✅ Active" else "❌ Inactive" end) + 
    "\t\(.endpoint_path)\t\(.file_format)"' | \
column -t -s $'\t' -N "ID,Name,Status,Endpoint,Format"

# Summary
total=$(echo "$response" | jq '.data | length')
active=$(echo "$response" | jq '.data | map(select(.is_active)) | length')

echo ""
echo "Summary: $active/$total specs active"
```

## 🔍 Advanced Examples

### Import with Validation

```bash
#!/bin/bash
# import-with-validation.sh

validate_spec() {
    local spec_content="$1"
    
    # Use the validation API
    response=$(curl -s -X POST http://localhost:8080/validate \
        -H "Content-Type: application/json" \
        -d "{\"openapi_spec\": $spec_content}")
    
    if echo "$response" | jq -e '.success' > /dev/null; then
        echo "✅ Validation passed"
        return 0
    else
        echo "❌ Validation failed:"
        echo "$response" | jq '.summary'
        return 1
    fi
}

import_spec() {
    local name="$1"
    local endpoint="$2" 
    local spec_file="$3"
    
    echo "Validating $spec_file..."
    
    spec_content=$(cat "$spec_file" | jq -R -s .)
    
    if validate_spec "$spec_content"; then
        echo "Importing $name..."
        
        curl -X POST http://localhost:8090/specs \
            -H "Content-Type: application/json" \
            -d "{
                \"name\": \"$name\",
                \"endpoint_path\": \"$endpoint\",
                \"active\": true,
                \"spec_content\": $spec_content
            }" | jq '.message'
    else
        echo "Skipping $name due to validation errors"
    fi
}

# Usage
import_spec "my-api" "/my-api" "specs/my-api.json"
```

### Monitor and Auto-restart Server

```bash
#!/bin/bash
# monitor-and-restart.sh

API_BASE="http://localhost:8090"
MCP_SERVER_PID=""

start_mcp_server() {
    echo "Starting MCP server..."
    export DATABASE_URL="postgresql://..."
    nohup bin/openapi-mcp --http :8080 > /tmp/mcp-server.log 2>&1 &
    MCP_SERVER_PID=$!
    echo "MCP server started with PID: $MCP_SERVER_PID"
}

check_specs_changed() {
    local current_hash=$(curl -s "$API_BASE/specs/active" | jq -S . | sha256sum)
    local previous_hash_file="/tmp/active_specs_hash"
    
    if [[ -f "$previous_hash_file" ]]; then
        local previous_hash=$(cat "$previous_hash_file")
        if [[ "$current_hash" != "$previous_hash" ]]; then
            echo "Active specs changed, restarting MCP server..."
            
            # Kill old server
            if [[ -n "$MCP_SERVER_PID" ]]; then
                kill $MCP_SERVER_PID 2>/dev/null
                sleep 2
            fi
            
            # Start new server
            start_mcp_server
        fi
    fi
    
    echo "$current_hash" > "$previous_hash_file"
}

# Initial start
start_mcp_server

# Monitor loop
while true; do
    sleep 10
    check_specs_changed
done
```

## 🛡️ Error Handling

### Common Error Responses

**400 Bad Request:**
```json
{
  "error": "Bad Request",
  "message": "Name is required",
  "code": 400
}
```

**409 Conflict (Duplicate name/endpoint):**
```json
{
  "error": "Conflict", 
  "message": "Failed to import spec: name already exists",
  "code": 409
}
```

**404 Not Found:**
```json
{
  "error": "Not Found",
  "message": "Failed to activate spec: openapi spec with id 999 not found", 
  "code": 404
}
```

### Error Handling in Scripts

```bash
handle_api_error() {
    local response="$1"
    
    if ! echo "$response" | jq -e '.success' > /dev/null; then
        echo "❌ API Error:" 
        echo "$response" | jq -r '.message // .error'
        return 1
    fi
    
    return 0
}

# Usage
response=$(curl -s -X POST http://localhost:8090/specs \
    -H "Content-Type: application/json" \
    -d '{"name": "test", "endpoint_path": "/test", "spec_content": "invalid"}')

if handle_api_error "$response"; then
    echo "✅ Success: $(echo "$response" | jq -r '.message')"
else
    exit 1
fi
```

## 🔗 Integration with Main Server

After managing specs via the API, restart the main MCP server to load changes:

```bash
# Check active specs
curl -s http://localhost:8090/specs/active | jq -r '.data[].endpoint_path'

# Restart MCP server with database loading
export DATABASE_URL="postgresql://..."
bin/openapi-mcp --http :8080
```

The MCP server will automatically load all active specs from the database and create endpoints for each one.