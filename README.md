# <img src="https://raw.githubusercontent.com/ubermorgenland/openapi-mcp/main/.github/logo.png.png" alt="openapi-mcp" width="600"/>

# openapi-mcp

> **Expose any OpenAPI 3.x API as a robust, agent-friendly MCP tool server in seconds!**

> **Note**: This project is built upon the excellent work from [jedisct1/openapi-mcp](https://github.com/jedisct1/openapi-mcp) ([documentation](https://jedisct1.github.io/openapi-mcp/)). This version extends the original with additional database-driven features and enhanced authentication handling.

[![Go Version](https://img.shields.io/badge/go-1.21%2B-blue)](https://golang.org/dl/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/jedisct1/openapi-mcp/ci.yml?branch=main)](https://github.com/jedisct1/openapi-mcp/actions)
[![License](https://img.shields.io/github/license/jedisct1/openapi-mcp)](LICENSE)
[![GoDoc](https://pkg.go.dev/badge/github.com/jedisct1/openapi-mcp/pkg/openapi2mcp.svg)](https://pkg.go.dev/github.com/jedisct1/openapi-mcp/pkg/openapi2mcp)

---

**openapi-mcp** transforms any OpenAPI 3.x specification into a powerful, AI-friendly MCP (Model Context Protocol) tool server. In seconds, it validates your OpenAPI spec, generates MCP tools for each operation, and starts serving through stdio or HTTP with structured, machine-readable output.

## 📋 Table of Contents

- [](#)
- [openapi-mcp](#openapi-mcp)
  - [📋 Table of Contents](#-table-of-contents)
  - [✨ Features](#-features)
  - [🤖 AI Agent Integration](#-ai-agent-integration)
  - [🔧 Installation](#-installation)
    - [Prerequisites](#prerequisites)
    - [Build from Source](#build-from-source)
  - [⚡ Quick Start](#-quick-start)
    - [1. Run the MCP Server](#1-run-the-mcp-server)
    - [2. Use the Interactive Client](#2-use-the-interactive-client)
  - [🔒 Authentication](#-authentication)
  - [🛠️ Usage Examples](#️-usage-examples)
    - [Integration with AI Code Editors](#integration-with-ai-code-editors)
    - [OpenAPI Validation and Linting](#openapi-validation-and-linting)
      - [HTTP API for Validation and Linting](#http-api-for-validation-and-linting)
    - [Dry Run (Preview Tools as JSON)](#dry-run-preview-tools-as-json)
    - [Generate Documentation](#generate-documentation)
    - [Filter Operations by Tag, Description, or Function List](#filter-operations-by-tag-description-or-function-list)
    - [Include/Exclude Operations by Description](#includeexclude-operations-by-description)
    - [Print Summary](#print-summary)
    - [Post-Process Schema with External Command](#post-process-schema-with-external-command)
    - [Disable Confirmation for Dangerous Actions](#disable-confirmation-for-dangerous-actions)
  - [🎮 Command-Line Options](#-command-line-options)
    - [Commands](#commands)
    - [Flags](#flags)
  - [📚 Library Usage](#-library-usage)
  - [📊 Output Structure](#-output-structure)
  - [🛡️ Safety Features](#️-safety-features)
  - [📝 Documentation Generation](#-documentation-generation)
  - [🙌 Contributing](#-contributing)
  - [📄 License](#-license)

## ✨ Features

- **Database-Driven Spec Management**: Store and manage OpenAPI specs in PostgreSQL for dynamic loading
  - Import specs from files with `spec-manager import`
  - Activate/deactivate specs without server restarts
  - Combine multiple active specs into a single MCP server
  - Automatic fallback to file-based loading when database unavailable
- **Instant API to MCP Conversion**: Parses any OpenAPI 3.x YAML/JSON spec and generates MCP tools
- **Multiple Transport Options**: Supports stdio (default) and HTTP server modes
- **Complete Parameter Support**: Path, query, header, cookie, and body parameters
- **Authentication**: API key, Bearer token, Basic auth, and OAuth2 support
- **Structured Output**: All responses have consistent, well-structured formats with type information
- **Validation & Linting**: Comprehensive OpenAPI validation and linting with actionable suggestions
  - `validate` command for critical issues (missing operationIds, schema errors)
  - `lint` command for best practices (summaries, descriptions, tags, parameter recommendations)
- **Safety Features**: Confirmation required for dangerous operations (PUT/POST/DELETE)
- **Documentation**: Built-in documentation generation in Markdown or HTML
- **AI-Optimized**: Unique features specifically designed to enhance AI agent interactions:
  - Consistent output structures with OutputFormat and OutputType for reliable parsing
  - Rich machine-readable schema information with constraints and examples
  - Streamlined, agent-friendly response format with minimal verbosity
  - Intelligent error messages with suggestions for correction
  - Automatic handling of authentication, pagination, and complex data structures
- **Interactive Client**: Includes an MCP client with readline support and command history
- **Flexible Configuration**: Environment variables or command-line flags
- **CI/Testing Support**: Summary options, exit codes, and dry-run mode

## 🤖 AI Agent Integration

openapi-mcp is designed for seamless integration with AI coding agents, LLMs, and automation tools with unique features that set it apart from other API-to-tool converters:

- **Structured JSON Responses**: Every response includes `OutputFormat` and `OutputType` fields for consistent parsing
- **Rich Schema Information**: All tools provide detailed parameter constraints and examples that help AI agents understand API requirements
- **Actionable Error Messages**: Validation errors include detailed information and suggestions that guide agents toward correct usage
- **Safety Confirmations**: Standardized confirmation workflow for dangerous operations prevents unintended consequences
- **Self-Describing API**: The `describe` tool provides complete, machine-readable documentation for all operations
- **Minimal Verbosity**: No redundant warnings or messages to confuse agents—outputs are optimized for machine consumption
- **Smart Parameter Handling**: Automatic conversion between OpenAPI parameter types and MCP tool parameters
- **Contextual Examples**: Every tool includes context-aware examples based on the OpenAPI specification
- **Intelligent Default Values**: Sensible defaults are provided whenever possible to simplify API usage

## 🔧 Installation

### Prerequisites

- Go 1.21+
- An OpenAPI 3.x YAML or JSON specification file

### Build from Source

```sh
# Clone the repository
git clone <repo-url>
cd openapi-mcp

# Build the binaries
make

# This will create:
# - bin/openapi-mcp (main tool)  
# - bin/mcp-client (interactive client)
# - bin/spec-manager (database spec management)
# - bin/seed-database (database seeding utility)
```

## 🗃️ Database Setup (Optional but Recommended)

openapi-mcp supports storing and managing OpenAPI specs in PostgreSQL for dynamic loading:

### Prerequisites
- PostgreSQL 9.5+ with a database created
- Connection string in `DATABASE_URL` environment variable

### Database Seeding Options

**Option 1: Quick Auto-Seed (Recommended)**
```sh
export DATABASE_URL="postgresql://username:password@host:port/database"
make all
make seed-database  # Imports specs with smart defaults
```

**Option 2: Custom Configuration**
```sh
# Edit seed_config.yaml to customize which specs are active
make seed-from-config
```

**Option 3: Manual Import**
```sh
bin/spec-manager import specs/weather.json weather /weather
bin/spec-manager import specs/twitter.yml twitter /twitter
```

**Option 4: Bulk Import**
```sh
make import-specs-from-files  # Imports all specs from specs/ directory
```

### Benefits
- **Dynamic Management**: Add/remove specs without server restarts
- **Centralized Storage**: All specs in database, not scattered files  
- **Version Control**: Track spec changes and metadata
- **Selective Loading**: Enable/disable specific APIs as needed
- **Easy Deployment**: No file management in containers

See [DATABASE_SETUP.md](DATABASE_SETUP.md) for comprehensive documentation.

## ⚡ Quick Start

### 1. Database-Driven Spec Loading (Recommended)

openapi-mcp now supports loading OpenAPI specs from a PostgreSQL database for dynamic management:

```sh
# Set your database connection
export DATABASE_URL="postgresql://username:password@host:port/database"

# Build all tools including spec management utilities
make all

# Seed database with existing specs
make seed-database

# Start server - automatically loads active specs from database
bin/openapi-mcp

# Or as HTTP server
bin/openapi-mcp --http=:8080
```

### 2. Traditional File-Based Loading

```sh
# Basic usage (stdio mode)
bin/openapi-mcp examples/fastly-openapi-mcp.yaml

# With API key
API_KEY=your_api_key bin/openapi-mcp examples/fastly-openapi-mcp.yaml

# As HTTP server
bin/openapi-mcp --http=:8080 examples/fastly-openapi-mcp.yaml

# Override base URL
bin/openapi-mcp --base-url=https://api.example.com examples/fastly-openapi-mcp.yaml
```

### 3. Managing Database Specs

**CLI Management:**
```sh
# List all specs in database
bin/spec-manager list

# Import a spec from file
bin/spec-manager import specs/weather.json weather /weather

# Activate/deactivate specs
bin/spec-manager activate 1
bin/spec-manager deactivate 2

# Set or update API key tokens
bin/spec-manager set-token 1 "sk-1234567890abcdef"
bin/spec-manager set-token 2 ""  # Clear token

# View only active specs
bin/spec-manager active
```

**HTTP API Management:**
```sh
# Start the management API server
bin/spec-api-server :8090

# Import spec via HTTP POST
curl -X POST http://localhost:8090/specs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-api",
    "endpoint_path": "/my-api",
    "spec_content": "{\"openapi\":\"3.0.0\"...}",
    "api_key_token": "sk-mytoken-123456789",
    "active": true
  }'

# List all specs
curl http://localhost:8090/specs | jq '.'

# Activate/deactivate specs
curl -X POST http://localhost:8090/specs/1/activate
curl -X POST http://localhost:8090/specs/1/deactivate

# Update API key token
curl -X PUT http://localhost:8090/specs/1/token \
  -H "Content-Type: application/json" \
  -d '{"api_key_token": "sk-new-token-123456789"}'
```

See [SPEC_API_EXAMPLES.md](SPEC_API_EXAMPLES.md) for comprehensive API examples.

### 2. Use the Interactive Client

```sh
# Start the client (connects to openapi-mcp via stdio)
bin/mcp-client bin/openapi-mcp examples/fastly-openapi-mcp.yaml

# Client commands
mcp> list                              # List available tools
mcp> schema <tool-name>                # Show tool schema
mcp> call <tool-name> {arg1: value1}   # Call a tool with arguments
mcp> describe                          # Get full API documentation
```

## 🔒 Authentication

openapi-mcp supports all standard OpenAPI authentication methods with intelligent priority handling for API keys:

### API Key Token Priority (Database Mode)

When using database-driven spec loading, authentication tokens are intelligently resolved:

#### 🧠 Smart Authentication Detection
The system automatically detects authentication type from the OpenAPI spec and uses the `api_key_token` appropriately:
- **Bearer Token APIs** (like Perplexity) → Uses `api_key_token` as `BEARER_TOKEN`
- **API Key APIs** (like Weather, Twitter) → Uses `api_key_token` as `API_KEY`  
- **Basic Auth APIs** → Uses `api_key_token` as `BASIC_AUTH`

#### 🏆 Priority Order
1. **🥇 Database Spec Token** - `api_key_token` field (applied based on detected auth type)
2. **🥈 Endpoint-Specific Environment Variable** - `{ENDPOINT}_API_KEY`, `{ENDPOINT}_BEARER_TOKEN`, etc.
3. **🥉 General Environment Variable** - `GENERAL_API_KEY`, `GENERAL_BEARER_TOKEN`, etc.

This allows you to:
- Store any authentication token type in the single `api_key_token` field
- Automatic application based on OpenAPI security scheme detection
- Override with environment variables when needed  
- Have global fallback authentication for any auth type

**Example:**
```sh
# Set tokens for different authentication types
bin/spec-manager set-token 1 "sk-weather-api-12345"           # API key for Weather
bin/spec-manager set-token 5 "pplx-bearer-token-67890"       # Bearer token for Perplexity

# Start server - automatically detects and applies correct auth type
export DATABASE_URL="postgresql://user:pass@localhost/db" 
bin/openapi-mcp

# Weather API uses "sk-weather-api-12345" as API_KEY (detected from spec)
# Perplexity API uses "pplx-bearer-token-67890" as BEARER_TOKEN (detected from spec)
# No need to specify authentication type - it's automatic!
```

### Command-Line Flags & Environment Variables

```sh
# API Key authentication
bin/openapi-mcp --api-key=your_api_key examples/fastly-openapi-mcp.yaml
# or use environment variable
API_KEY=your_api_key bin/openapi-mcp examples/fastly-openapi-mcp.yaml

# Bearer token / OAuth2
bin/openapi-mcp --bearer-token=your_token examples/fastly-openapi-mcp.yaml
# or use environment variable
BEARER_TOKEN=your_token bin/openapi-mcp examples/fastly-openapi-mcp.yaml

# Basic authentication
bin/openapi-mcp --basic-auth=username:password examples/fastly-openapi-mcp.yaml
# or use environment variable
BASIC_AUTH=username:password bin/openapi-mcp examples/fastly-openapi-mcp.yaml
```

### HTTP Header Authentication (HTTP Mode Only)

When using HTTP mode (`--http=:8080`), you can provide authentication via HTTP headers in your requests:

```sh
# API Key via headers
curl -H "X-API-Key: your_api_key" http://localhost:8080/mcp -d '...'
curl -H "Api-Key: your_api_key" http://localhost:8080/mcp -d '...'

# Bearer token
curl -H "Authorization: Bearer your_token" http://localhost:8080/mcp -d '...'

# Basic authentication
curl -H "Authorization: Basic base64_credentials" http://localhost:8080/mcp -d '...'
```

**Supported Authentication Headers:**
- `X-API-Key` or `Api-Key` - for API key authentication
- `Authorization: Bearer <token>` - for OAuth2/Bearer token authentication
- `Authorization: Basic <credentials>` - for Basic authentication

Authentication is automatically applied to the appropriate endpoints as defined in your OpenAPI spec. HTTP header authentication takes precedence over environment variables for the duration of each request.

When using HTTP mode, openapi-mcp serves a StreamableHTTP-based MCP server by default. For developers building HTTP clients, the package provides convenient URL helper functions:

```go
import "github.com/jedisct1/openapi-mcp/pkg/openapi2mcp"

// Get the Streamable HTTP endpoint URL
streamableURL := openapi2mcp.GetStreamableHTTPURL(":8080", "/mcp")
// Returns: "http://localhost:8080/mcp"

// For SSE mode (when using --http-transport=sse), you can use:
sseURL := openapi2mcp.GetSSEURL(":8080", "/mcp")
// Returns: "http://localhost:8080/mcp/sse"

messageURL := openapi2mcp.GetMessageURL(":8080", "/mcp", sessionID)
// Returns: "http://localhost:8080/mcp/message?sessionId=<sessionID>"
```

**StreamableHTTP Client Connection Flow:**
1. Send POST requests to the Streamable HTTP endpoint for requests/notifications
2. Send GET requests to the same endpoint to listen for notifications
3. Send DELETE requests to terminate the session

**Example with curl:**
```sh
# Step 1: Initialize the session
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}'

# The response will include a Mcp-Session-Id header

# Step 2: Send JSON-RPC requests
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: <session-id>" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'

# Step 3: Listen for notifications
curl -N http://localhost:8080/mcp \
  -H "Mcp-Session-Id: <session-id>"
```

**SSE Client Connection Flow (when using --http-transport=sse):**
1. Connect to the SSE endpoint to establish a persistent connection
2. Receive an `endpoint` event containing the session ID
3. Send JSON-RPC requests to the message endpoint using the session ID
4. Receive responses and notifications via the SSE stream

**Example with curl (SSE mode):**
```sh
# Step 1: Connect to SSE endpoint (keep connection open)
curl -N http://localhost:8080/mcp/sse

# Output: event: endpoint
#         data: /mcp/message?sessionId=<session-id>

# Step 2: Send JSON-RPC requests (in another terminal)
curl -X POST http://localhost:8080/mcp/message?sessionId=<session-id> \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## 🛠️ Usage Examples

### Integration with AI Code Editors

You can easily integrate openapi-mcp with AI code editors that support MCP tools, such as Roo Code:

```json
{
    "fastly": {
        "command": "/opt/bin/openapi-mcp",
        "args": [
            "-api-key",
            "YOUR_API_KEY",
            "/opt/etc/openapi/fastly-openapi-mcp.yaml"
        ]
    }
}
```

Add this configuration to your editor's MCP tools configuration to provide AI assistants with direct access to the API. The assistant can then discover and use the API operations without additional setup.

### OpenAPI Validation and Linting

openapi-mcp includes powerful OpenAPI validation and linting capabilities to help you improve your API specifications:

```sh
# Validate OpenAPI spec and check for critical issues
bin/openapi-mcp validate examples/fastly-openapi-mcp.yaml

# Comprehensive linting with detailed suggestions
bin/openapi-mcp lint examples/fastly-openapi-mcp.yaml

# Start HTTP validation service
bin/openapi-mcp --http=:8080 validate

# Start HTTP linting service
bin/openapi-mcp --http=:8080 lint
```

The **validate** command performs essential checks:
- Missing `operationId` fields (required for MCP tool generation)
- Schema validation errors
- Basic structural issues

The **lint** command provides comprehensive analysis with suggestions for:
- Missing summaries and descriptions
- Untagged operations
- Parameter naming and type recommendations
- Security scheme validation
- Best practices for API design

Both commands exit with non-zero status codes when issues are found, making them perfect for CI/CD pipelines.

#### HTTP API for Validation and Linting

Both validate and lint commands can be run as HTTP services using the `--http` flag, allowing you to validate OpenAPI specs via REST API. Note that these endpoints are only available when using the `validate` or `lint` commands, not during normal MCP server operation:

```sh
# Start validation HTTP service
bin/openapi-mcp --http=:8080 validate

# Start linting HTTP service
bin/openapi-mcp --http=:8080 lint
```

**API Endpoints:**

- `POST /validate` - Validate OpenAPI specs for critical issues
- `POST /lint` - Comprehensive linting with detailed suggestions
- `GET /health` - Health check endpoint

**Request Format:**
```json
{
  "openapi_spec": "openapi: 3.0.0\ninfo:\n  title: My API\n  version: 1.0.0\npaths: {}"
}
```

**Response Format:**
```json
{
  "success": false,
  "error_count": 1,
  "warning_count": 2,
  "issues": [
    {
      "type": "error",
      "message": "Operation missing operationId",
      "suggestion": "Add an operationId field",
      "operation": "GET_/users",
      "path": "/users",
      "method": "GET"
    }
  ],
  "summary": "OpenAPI linting completed with issues: 1 errors, 2 warnings."
}
```

**Example Usage:**
```sh
curl -X POST http://localhost:8080/lint \
  -H "Content-Type: application/json" \
  -d '{"openapi_spec": "..."}'
```

### Dry Run (Preview Tools as JSON)

```sh
bin/openapi-mcp --dry-run examples/fastly-openapi-mcp.yaml
```

### Generate Documentation

```sh
bin/openapi-mcp --doc=tools.md examples/fastly-openapi-mcp.yaml
```

### Filter Operations by Tag, Description, or Function List

```sh
bin/openapi-mcp filter --tag=admin examples/fastly-openapi-mcp.yaml
bin/openapi-mcp filter --include-desc-regex="user|account" examples/fastly-openapi-mcp.yaml
bin/openapi-mcp filter --exclude-desc-regex="deprecated" examples/fastly-openapi-mcp.yaml
bin/openapi-mcp filter --function-list-file=funcs.txt examples/fastly-openapi-mcp.yaml
```

You can use `--function-list-file=funcs.txt` to restrict the output to only the operations whose `operationId` is listed (one per line) in the given file. This filter is applied after tag and description filters.

### Print Summary

```sh
bin/openapi-mcp --summary --dry-run examples/fastly-openapi-mcp.yaml
```

### Post-Process Schema with External Command

```sh
bin/openapi-mcp --doc=tools.md --post-hook-cmd='jq . | tee /tmp/filtered.json' examples/fastly-openapi-mcp.yaml
```

### Disable Confirmation for Dangerous Actions

```sh
bin/openapi-mcp --no-confirm-dangerous examples/fastly-openapi-mcp.yaml
```

## 🎮 Command-Line Options

### Commands

| Command           | Description                                                                                                                    |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `validate <spec>` | Validate OpenAPI spec and report critical issues (missing operationIds, schema errors)                                         |
| `lint <spec>`     | Comprehensive linting with detailed suggestions for best practices                                                             |
| `filter <spec>`   | Output a filtered list of operations as JSON, applying `--tag`, `--include-desc-regex`, `--exclude-desc-regex`, and `--function-list-file` (no server) |

### Flags

| Flag                     | Environment Variable | Description                                              |
| ------------------------ | -------------------- | -------------------------------------------------------- |
| `--api-key`              | `API_KEY`            | API key for authentication                               |
| `--bearer-token`         | `BEARER_TOKEN`       | Bearer token for Authorization header                    |
| `--basic-auth`           | `BASIC_AUTH`         | Basic auth credentials (user:pass)                       |
| `--base-url`             | `OPENAPI_BASE_URL`   | Override base URL for HTTP calls                         |
| `--http`                 | -                    | Serve MCP over HTTP instead of stdio                     |
| `--tag`                  | `OPENAPI_TAG`        | Only include operations with this tag                    |
| `--include-desc-regex`   | `INCLUDE_DESC_REGEX` | Only include APIs matching regex                         |
| `--exclude-desc-regex`   | `EXCLUDE_DESC_REGEX` | Exclude APIs matching regex                              |
| `--dry-run`              | -                    | Print tool schemas as JSON and exit                      |
| `--summary`              | -                    | Print operation count summary                            |
| `--doc`                  | -                    | Generate documentation file                              |
| `--doc-format`           | -                    | Documentation format (markdown or html)                  |
| `--post-hook-cmd`        | -                    | Command to post-process schema JSON                      |
| `--no-confirm-dangerous` | -                    | Disable confirmation for dangerous actions               |
| `--extended`             | -                    | Enable human-friendly output (default is agent-friendly) |
| `--function-list-file`   | -                    | Only include operations whose operationId is listed (one per line) in the given file (for filter command) |

### Database Management Commands

#### CLI Commands
| Command                           | Description                                                    |
| --------------------------------- | -------------------------------------------------------------- |
| `spec-manager list`               | List all specs in database with status and metadata           |
| `spec-manager active`             | List only active specs that will be loaded                    |
| `spec-manager import <file> <name> <endpoint>` | Import OpenAPI spec from file to database  |
| `spec-manager activate <id>`      | Activate a spec by ID                                          |
| `spec-manager deactivate <id>`    | Deactivate a spec by ID                                        |
| `spec-manager set-token <id> <token>` | Set or clear API key token for a spec                    |
| `spec-manager delete <id>`        | Delete a spec from database                                    |
| `make seed-database`              | Auto-seed database with predefined spec configuration         |
| `make seed-from-config`           | Seed database using custom seed_config.yaml                   |
| `make import-specs-from-files`    | Bulk import all specs from specs/ directory                   |

#### HTTP API Server
| Command                           | Description                                                    |
| --------------------------------- | -------------------------------------------------------------- |
| `spec-api-server :8090`           | Start HTTP API server for remote spec management              |

#### HTTP API Endpoints
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/specs` | List all specs with full metadata |
| `POST` | `/specs` | Import new spec from JSON payload |
| `GET` | `/specs/active` | List only active specs |
| `DELETE` | `/specs/{id}` | Delete spec by ID |
| `POST` | `/specs/{id}/activate` | Activate spec by ID |
| `POST` | `/specs/{id}/deactivate` | Deactivate spec by ID |
| `PUT` | `/specs/{id}/token` | Update API key token for spec |
| `GET` | `/health` | Health check endpoint |
| `GET` | `/swagger` | OpenAPI specification for this API |

### Environment Variables

| Variable        | Description                                                          |
| --------------- | -------------------------------------------------------------------- |
| `DATABASE_URL`  | PostgreSQL connection string for database-driven spec loading       |

## 🔗 Available Endpoints

When running openapi-mcp as an HTTP server, various endpoints become available depending on the mode:

### MCP Server Endpoints

```bash
# Start MCP server with database specs
export DATABASE_URL="postgresql://..."
bin/openapi-mcp --http :8080
```

**Core MCP Protocol:**
- `GET/POST /mcp` - Main MCP server endpoint (StreamableHTTP transport)
- `GET /mcp/sse` - Server-Sent Events endpoint (with `--http-transport=sse`)
- `POST /mcp/message` - Message endpoint for SSE mode
- `GET /health` - Health check endpoint

### API-Specific Endpoints (Database-Driven)

Each active spec in the database creates its own endpoint based on the `endpoint_path` field:

**Default Active Endpoints** (after `make seed-database`):
- `/weather` - Weather API operations
- `/google-keywords` - Google Keywords API operations  
- `/youtube-transcript` - YouTube Transcript API operations
- `/perplexity` - Perplexity AI API operations

**Optional Endpoints** (inactive by default, activate with `spec-manager activate <id>`):
- `/twitter` - Twitter API operations
- `/alpha-vantage` - Alpha Vantage financial API operations
- `/google-finance` - Google Finance API operations

### Validation/Linting API Endpoints

```bash
# Start validation server
bin/openapi-mcp --http=:8080 validate

# Start linting server  
bin/openapi-mcp --http=:8080 lint
```

**Available Endpoints:**
- `POST /validate` - Validate OpenAPI specifications
- `POST /lint` - Comprehensive OpenAPI linting with suggestions
- `GET /health` - Health check endpoint
- `OPTIONS /validate` - CORS preflight for validate
- `OPTIONS /lint` - CORS preflight for lint

**Example Usage:**
```bash
# Validate a spec
curl -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d '{"openapi_spec": "openapi: 3.0.0\n..."}'

# Lint a spec  
curl -X POST http://localhost:8080/lint \
  -H "Content-Type: application/json" \
  -d '{"openapi_spec": "openapi: 3.0.0\n..."}'
```

### Multi-Mount Endpoints (File-Based)

```bash
# Mount multiple specs at custom paths
bin/openapi-mcp --http=:8080 \
  --mount /api1:specs/weather.json \
  --mount /api2:specs/twitter.yml
```

Creates endpoints at `/api1` and `/api2` respectively.

### Check Active Endpoints

```bash
# See what endpoints are currently active
bin/spec-manager active

# Example output:
# ID   Name                 Endpoint
# 1    weather             /weather  
# 2    google-keywords     /google-keywords
# 3    youtube-transcript  /youtube-transcript
```

**Dynamic Endpoint Management:**
```bash
# Activate an endpoint
bin/spec-manager activate 5  # Activates /twitter endpoint

# Deactivate an endpoint  
bin/spec-manager deactivate 5  # Deactivates /twitter endpoint

# Changes take effect on next server restart
```

## 🔧 Managing OpenAPI Specs with curl

While the `spec-manager` CLI tool is the primary way to manage specs, you can also create a simple HTTP API wrapper for remote management. Here are examples of how you might interact with the database directly or through a custom API:

### Database Management Examples

**Set up database connection:**
```bash
export DATABASE_URL="postgresql://username:password@host:port/database"
```

### CLI Management with curl-like Interface

**1. List All Specs:**
```bash
# Using spec-manager (recommended)
bin/spec-manager list

# Example output:
# ID   Name                 Title                          Version    Active   Format     Endpoint
# 1    weather             OpenWeatherMap API             2.5        true     json       /weather
# 2    twitter             Twitter API                    2.0        false    yaml       /twitter
# 3    google-keywords     Google Keywords API            1.0        true     yml        /google-keywords
```

**2. List Only Active Specs:**
```bash
bin/spec-manager active

# Example output:
# ID   Name                 Title                          Version    Format     Endpoint
# 1    weather             OpenWeatherMap API             2.5        json       /weather
# 3    google-keywords     Google Keywords API            1.0        yml        /google-keywords
```

**3. Import New Spec:**
```bash
# Import from local file
bin/spec-manager import specs/new-api.yaml "new-api" "/new-api"

# Import from URL (download first)
curl -o temp-spec.yaml https://api.example.com/openapi.yaml
bin/spec-manager import temp-spec.yaml "example-api" "/example"
rm temp-spec.yaml
```

**4. Activate/Deactivate Specs:**
```bash
# Activate a spec
bin/spec-manager activate 2
# Output: Successfully activated spec with ID 2

# Deactivate a spec
bin/spec-manager deactivate 2
# Output: Successfully deactivated spec with ID 2
```

**5. Delete Specs:**
```bash
# Delete a spec permanently
bin/spec-manager delete 2
# Output: Successfully deleted spec with ID 2
```

### Remote Management API Examples

For remote management, you could create a simple HTTP wrapper around the spec-manager functionality:

**Create a simple management API server (`management-server.go`):**
```go
package main

import (
    "encoding/json"
    "net/http"
    "strconv"
    
    "github.com/jedisct1/openapi-mcp/pkg/database"
    "github.com/jedisct1/openapi-mcp/pkg/services"
)

func main() {
    database.InitializeDatabase()
    specLoader := services.NewSpecLoaderService(database.DB)
    
    http.HandleFunc("/specs", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
            specs, _ := specLoader.GetAllSpecs()
            json.NewEncoder(w).Encode(specs)
        }
    })
    
    http.HandleFunc("/specs/active", func(w http.ResponseWriter, r *http.Request) {
        specs, _ := specLoader.GetActiveSpecs()
        json.NewEncoder(w).Encode(specs)
    })
    
    http.ListenAndServe(":9090", nil)
}
```

**Then use curl with the management API:**

**List all specs:**
```bash
curl -X GET http://localhost:9090/specs | jq '.'

# Example response:
# [
#   {
#     "id": 1,
#     "name": "weather",
#     "title": "OpenWeatherMap API",
#     "version": "2.5",
#     "endpoint_path": "/weather",
#     "file_format": "json",
#     "is_active": true,
#     "created_at": "2024-08-27T10:30:00Z",
#     "updated_at": "2024-08-27T10:30:00Z"
#   },
#   {
#     "id": 2,
#     "name": "twitter",
#     "title": "Twitter API", 
#     "version": "2.0",
#     "endpoint_path": "/twitter",
#     "file_format": "yaml",
#     "is_active": false,
#     "created_at": "2024-08-27T10:31:00Z",
#     "updated_at": "2024-08-27T10:31:00Z"
#   }
# ]
```

**List active specs only:**
```bash
curl -X GET http://localhost:9090/specs/active | jq '.[].name'

# Example response:
# "weather"
# "google-keywords" 
# "youtube-transcript"
# "perplexity"
```

### Bulk Operations with curl and jq

**Activate multiple specs by name:**
```bash
# Get IDs of specs to activate
SPEC_IDS=$(bin/spec-manager list | grep -E "(twitter|alpha-vantage)" | awk '{print $1}')

# Activate each one
for id in $SPEC_IDS; do
    bin/spec-manager activate $id
    echo "Activated spec ID: $id"
done
```

**Deactivate all specs except core ones:**
```bash
# Get all active spec IDs except weather and google-keywords
DEACTIVATE_IDS=$(bin/spec-manager active | grep -v -E "(weather|google-keywords)" | tail -n +3 | awk '{print $1}')

# Deactivate them
for id in $DEACTIVATE_IDS; do
    bin/spec-manager deactivate $id
    echo "Deactivated spec ID: $id"
done
```

### Automated Spec Management

**Check and reload server when specs change:**
```bash
#!/bin/bash
# check-and-reload.sh

# Get current active specs
CURRENT_SPECS=$(bin/spec-manager active | wc -l)

# Store in temp file for comparison
echo $CURRENT_SPECS > /tmp/current_spec_count

# Function to restart server
restart_server() {
    echo "Spec changes detected, restarting server..."
    pkill -f "openapi-mcp --http"
    sleep 2
    nohup bin/openapi-mcp --http :8080 > /tmp/server.log 2>&1 &
    echo "Server restarted"
}

# Compare with previous count
if [[ -f /tmp/previous_spec_count ]]; then
    PREVIOUS_SPECS=$(cat /tmp/previous_spec_count)
    if [[ $CURRENT_SPECS -ne $PREVIOUS_SPECS ]]; then
        restart_server
    fi
fi

# Update previous count
cp /tmp/current_spec_count /tmp/previous_spec_count
```

**API Health Check with Specific Endpoints:**
```bash
# Check if specific API endpoints are responding
check_endpoint() {
    local endpoint=$1
    local response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080${endpoint})
    
    if [[ $response == "200" ]]; then
        echo "✅ $endpoint is healthy"
    else
        echo "❌ $endpoint returned $response"
    fi
}

# Check all active endpoints
echo "Checking API endpoint health..."
check_endpoint "/weather"
check_endpoint "/google-keywords" 
check_endpoint "/youtube-transcript"
check_endpoint "/perplexity"
```

**Import specs from a directory with curl-style batch processing:**
```bash
#!/bin/bash
# batch-import.sh

SPECS_DIR="./specs"
SUCCESS_COUNT=0
FAIL_COUNT=0

echo "Batch importing specs from $SPECS_DIR..."

for spec_file in "$SPECS_DIR"/*.{json,yaml,yml}; do
    if [[ -f "$spec_file" ]]; then
        filename=$(basename "$spec_file")
        name="${filename%.*}"
        endpoint="/${name//_/-}"
        
        echo "Importing $filename as '$name' with endpoint '$endpoint'..."
        
        if bin/spec-manager import "$spec_file" "$name" "$endpoint"; then
            echo "✅ Successfully imported $filename"
            ((SUCCESS_COUNT++))
        else
            echo "❌ Failed to import $filename"
            ((FAIL_COUNT++))
        fi
    fi
done

echo ""
echo "Import completed: $SUCCESS_COUNT successful, $FAIL_COUNT failed"
```

These examples show comprehensive spec management workflows that can be automated or used interactively for both local and remote OpenAPI spec management.

## 📚 Library Usage

openapi-mcp can be imported as a Go module in your projects:

```go
package main

import (
        "github.com/jedisct1/openapi-mcp/pkg/openapi2mcp"
)

func main() {
        // Load OpenAPI spec
        doc, err := openapi2mcp.LoadOpenAPISpec("openapi.yaml")
        if err != nil {
                panic(err)
        }

        // Create MCP server
        srv := openapi2mcp.NewServer("myapi", doc.Info.Version, doc)

        // Serve over HTTP
        if err := openapi2mcp.ServeHTTP(srv, ":8080"); err != nil {
                panic(err)
        }

        // Or serve over stdio
        // if err := openapi2mcp.ServeStdio(srv); err != nil {
        //    panic(err)
        // }
}
```

See [GoDoc](https://pkg.go.dev/github.com/jedisct1/openapi-mcp/pkg/openapi2mcp) for complete API documentation.

## 📊 Output Structure

All tool results include consistent structure for machine readability:

```json
{
  "OutputFormat": "structured",
  "OutputType": "json",
  "type": "api_response",
  "data": {
    // API-specific response data
  },
  "metadata": {
    "status_code": 200,
    "headers": {
      // Response headers
    }
  }
}
```

For errors, you'll receive:

```json
{
  "OutputFormat": "structured",
  "OutputType": "json",
  "type": "error",
  "error": {
    "code": "validation_error",
    "message": "Invalid parameter",
    "details": {
      "field": "username",
      "reason": "required field missing"
    },
    "suggestions": [
      "Provide a username parameter"
    ]
  }
}
```

## 🛡️ Safety Features

For any operation that performs a PUT, POST, or DELETE, openapi-mcp requires confirmation:

```json
{
  "type": "confirmation_request",
  "confirmation_required": true,
  "message": "This action is irreversible. Proceed?",
  "action": "delete_resource"
}
```

To proceed, retry the call with:

```json
{
  "original_parameters": {},
  "__confirmed": true
}
```

This confirmation workflow can be disabled with `--no-confirm-dangerous`.

## 📝 Documentation Generation

Generate comprehensive documentation for all tools:

```sh
# Markdown documentation
bin/openapi-mcp --doc=tools.md examples/fastly-openapi-mcp.yaml

# HTML documentation
bin/openapi-mcp --doc=tools.html --doc-format=html examples/fastly-openapi-mcp.yaml
```

The documentation includes:
- Complete tool schemas with parameter types, constraints, and descriptions
- Example calls for each tool
- Response formats and examples
- Authentication requirements

## 🙌 Contributing

We welcome contributions from the community! Whether you're fixing bugs, adding features, improving documentation, or sharing feedback, your help makes this project better.

### 🚀 Quick Start for Contributors

1. **Fork and Clone**
   ```sh
   git clone https://github.com/yourusername/openapi-mcp.git
   cd openapi-mcp
   ```

2. **Set Up Development Environment**
   ```sh
   go mod download
   make all  # Build all binaries
   go test ./...  # Verify everything works
   ```

3. **Make Your Changes**
   - Create feature branch: `git checkout -b feature/your-feature`
   - Follow existing code patterns and add tests
   - Update documentation for new features

4. **Submit Your Contribution**
   - Run tests and linting: `go test ./... && go fmt ./...`
   - Commit with clear message: `git commit -m "feat: describe your change"`
   - Open a Pull Request with detailed description

### 🎯 Ways to Contribute

- **🐛 Bug Reports**: Found an issue? Help us fix it!
- **✨ Feature Requests**: Have ideas? We'd love to hear them!
- **📝 Documentation**: Improve guides, examples, or API docs
- **🧪 Testing**: Add test cases or improve existing ones
- **🔧 Code**: Fix bugs, add features, or optimize performance
- **💡 Ideas**: Share thoughts in GitHub Discussions

### 📋 Contribution Guidelines

**Before Contributing:**
- Read our [Contributing Guide](CONTRIBUTING.md) for detailed instructions
- Check existing issues to avoid duplicates
- Review our [Code of Conduct](CODE_OF_CONDUCT.md)

**For New Contributors:**
- Look for issues labeled `good-first-issue`
- Start with documentation or test improvements
- Ask questions in GitHub Discussions - we're here to help!

**For Feature Development:**
- Open an issue to discuss major changes first
- Follow Go best practices and existing code patterns  
- Include comprehensive tests and documentation
- Consider backwards compatibility and API stability

### 🤝 Community

- **GitHub Discussions**: Ask questions, share ideas, get help
- **Issues**: Report bugs and request features  
- **Pull Requests**: Submit code contributions
- **Documentation**: Help others learn and use the project

We believe in fostering an inclusive, welcoming community where everyone can contribute meaningfully. New contributors are especially welcome - don't hesitate to ask questions or start with small improvements!

## 📚 Documentation & Learning

### 📖 Comprehensive Guides
- **[Getting Started Guide](GETTING_STARTED.md)** - Step-by-step tutorial for new users
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute code, docs, and ideas
- **[Database Setup](DATABASE_SETUP.md)** - Complete database configuration guide
- **[API Examples](SPEC_API_EXAMPLES.md)** - HTTP API usage examples and recipes

### 🎓 Learning Resources
- **Package Documentation**: Full API docs available on [pkg.go.dev](https://pkg.go.dev/github.com/ubermorgenland/openapi-mcp)
- **Examples Directory**: Real-world OpenAPI specifications and usage patterns
- **Command Help**: Run `bin/openapi-mcp --help` for all available options
- **Interactive Testing**: Use `bin/mcp-client` to explore APIs interactively

## 🤝 Community & Support

### 💬 Getting Help
- **🐛 Bug Reports**: [Create an issue](../../issues/new?template=bug_report.yml) with our bug template
- **✨ Feature Requests**: [Suggest features](../../issues/new?template=feature_request.yml) you'd like to see
- **❓ Questions**: [Ask questions](../../issues/new?template=question.yml) or use [GitHub Discussions](../../discussions)
- **📚 Documentation**: [Improve docs](../../issues/new?template=documentation.yml) for everyone's benefit

### 🌟 Community Resources
- **[Code of Conduct](CODE_OF_CONDUCT.md)** - Our community guidelines and values
- **[GitHub Discussions](../../discussions)** - General questions, ideas, and community chat
- **[Issue Templates](../../issues/new/choose)** - Structured ways to report bugs and request features
- **[Pull Request Guidelines](CONTRIBUTING.md#pull-request-guidelines)** - How to contribute code changes

### 🏆 Recognition
We value all contributions and recognize them through:
- Contributor acknowledgments in releases
- Community highlights for significant contributions
- Mentorship opportunities for new contributors
- Feature showcases for innovative use cases

### 💡 Ways to Get Involved
- **First-time Contributors**: Look for [`good-first-issue`](../../issues?q=label%3A%22good-first-issue%22) labels
- **Documentation Lovers**: Help improve guides, examples, and API docs
- **Testing Enthusiasts**: Add test cases and help verify functionality
- **Integration Experts**: Share examples and integration patterns
- **Community Builders**: Help answer questions and welcome newcomers

## 📄 License

This project is licensed under the [MIT License](LICENSE).