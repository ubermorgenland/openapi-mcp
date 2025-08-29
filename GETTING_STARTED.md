# Getting Started with openapi-mcp

Welcome to openapi-mcp! This guide will help you get up and running quickly with transforming your OpenAPI specifications into powerful MCP (Model Context Protocol) servers.

## 🎯 What You'll Learn

By the end of this guide, you'll know how to:
- Set up openapi-mcp for your project
- Convert OpenAPI specs to MCP tools
- Configure authentication for different APIs
- Use database-driven spec management
- Integrate with AI agents and LLMs

## 📋 Prerequisites

Before starting, ensure you have:

- **Go 1.22+**: [Install Go](https://golang.org/dl/) if you haven't already
- **Basic OpenAPI Knowledge**: Familiarity with OpenAPI 3.x specifications
- **Terminal/Command Line**: Comfort with basic command-line operations
- **PostgreSQL** (optional): For advanced database features

## 🚀 Quick Start (5 Minutes)

### Step 1: Get openapi-mcp

**Option A: Download Binary** (Coming Soon)
```sh
# Download from GitHub releases (when available)
curl -L https://github.com/ubermorgenland/openapi-mcp/releases/latest/download/openapi-mcp-linux-amd64.tar.gz | tar xz
```

**Option B: Build from Source**
```sh
# Clone the repository
git clone https://github.com/ubermorgenland/openapi-mcp.git
cd openapi-mcp

# Build all tools
make all
# This creates: bin/openapi-mcp, bin/mcp-client, bin/spec-manager
```

### Step 2: Test with Example API

```sh
# Run with the included weather API example
bin/openapi-mcp examples/weather.yaml

# The server is now running in stdio mode!
# You should see: Server initialized and ready for requests
```

### Step 3: Try the Interactive Client

```sh
# In another terminal, start the interactive client
bin/mcp-client bin/openapi-mcp examples/weather.yaml

# Try these commands in the client:
mcp> list                    # See available tools
mcp> schema getCurrentWeather # View tool parameters
mcp> describe               # Get full API documentation
```

**🎉 Congratulations!** You've successfully converted an OpenAPI specification into an MCP server.

## 🔧 Basic Configuration

### File-Based Mode (Simple)

```sh
# Basic usage with API key
API_KEY=your_api_key bin/openapi-mcp your-api-spec.yaml

# HTTP server mode
bin/openapi-mcp --http :8080 your-api-spec.yaml

# With custom base URL
bin/openapi-mcp --base-url=https://api.yourcompany.com your-api-spec.yaml
```

### Database-Driven Mode (Recommended for Production)

```sh
# 1. Set up PostgreSQL database
export DATABASE_URL="postgresql://username:password@localhost:5432/your_database"

# 2. Build tools and seed database
make all
make seed-database  # Imports example specs

# 3. Start server (loads all active specs from database)
bin/openapi-mcp --http :8080

# 4. Manage specs dynamically
bin/spec-manager list                           # View all specs
bin/spec-manager import myapi.yaml myapi /myapi # Add new API
bin/spec-manager activate 1                     # Enable an API
bin/spec-manager set-token 1 "api-key-123"     # Set authentication
```

## 🔐 Authentication Setup

### Single API Authentication

```sh
# API Key
API_KEY=your_key bin/openapi-mcp api-spec.yaml

# Bearer Token  
BEARER_TOKEN=your_token bin/openapi-mcp api-spec.yaml

# Basic Auth
BASIC_AUTH=user:pass bin/openapi-mcp api-spec.yaml
```

### Multi-API Authentication (Database Mode)

```sh
# Import APIs with different auth types
bin/spec-manager import weather.yaml weather /weather
bin/spec-manager import twitter.yaml twitter /twitter

# Set authentication tokens (auto-detected by API type)
bin/spec-manager set-token 1 "weather-api-key-123"      # API key
bin/spec-manager set-token 2 "twitter-bearer-token-456"  # Bearer token

# Start server - authentication is automatic!
bin/openapi-mcp --http :8080
```

### HTTP Header Authentication

```sh
# Start HTTP server
bin/openapi-mcp --http :8080 api-spec.yaml

# Use authentication headers in requests
curl -H "X-API-Key: your_key" http://localhost:8080/mcp -d '{...}'
curl -H "Authorization: Bearer your_token" http://localhost:8080/mcp -d '{...}'
```

## 🎮 Usage Examples

### Example 1: Weather API Integration

```sh
# 1. Get a weather API key from OpenWeatherMap
export WEATHER_API_KEY="your_weather_api_key"

# 2. Run the weather API example
bin/openapi-mcp examples/weather.yaml

# 3. In another terminal, test it
bin/mcp-client bin/openapi-mcp examples/weather.yaml

# 4. In the client, get current weather
mcp> call getCurrentWeather {"q": "London", "appid": "auto"}
```

### Example 2: Multiple APIs with Database

```sh
# 1. Set up database
export DATABASE_URL="postgresql://user:pass@localhost:5432/mydb"

# 2. Import your APIs
bin/spec-manager import examples/weather.yaml weather /weather
bin/spec-manager import examples/twitter.yml twitter /twitter

# 3. Configure authentication
bin/spec-manager set-token 1 "$WEATHER_API_KEY"
bin/spec-manager set-token 2 "$TWITTER_BEARER_TOKEN"

# 4. Activate the APIs you want
bin/spec-manager activate 1  # Enable weather
bin/spec-manager activate 2  # Enable twitter

# 5. Start the combined server
bin/openapi-mcp --http :8080

# 6. Both APIs are now available at their endpoints:
# http://localhost:8080/weather (weather operations)
# http://localhost:8080/twitter (twitter operations)
```

### Example 3: AI Agent Integration

```sh
# 1. Start MCP server
bin/openapi-mcp --http :8080 examples/weather.yaml

# 2. Configure your AI agent to connect to:
# HTTP Endpoint: http://localhost:8080/mcp
# Authentication: X-API-Key header with your API key

# 3. The AI can now use tools like:
# - getCurrentWeather(q: "Paris")
# - getForecast(q: "Tokyo", cnt: 5)
# - And more from your OpenAPI spec!
```

## 🛠️ Common Workflows

### Adding a New API

```sh
# 1. Create or obtain OpenAPI 3.x specification
# 2. Validate the spec
bin/openapi-mcp validate myapi.yaml

# 3. Import to database
bin/spec-manager import myapi.yaml myapi /myapi

# 4. Set authentication if needed
bin/spec-manager set-token <id> "your-api-token"

# 5. Activate the API
bin/spec-manager activate <id>

# 6. Restart server to load changes
# Server automatically loads all active specs
```

### Debugging API Integration

```sh
# 1. Validate your OpenAPI spec
bin/openapi-mcp validate your-spec.yaml

# 2. Lint for best practices
bin/openapi-mcp lint your-spec.yaml

# 3. Preview generated tools
bin/openapi-mcp --dry-run your-spec.yaml

# 4. Test with verbose output
bin/openapi-mcp --extended your-spec.yaml

# 5. Check tool summary
bin/openapi-mcp --summary --dry-run your-spec.yaml
```

### Managing Database Specs

```sh
# View all specs
bin/spec-manager list

# View only active specs
bin/spec-manager active

# Activate/deactivate APIs without server restart
bin/spec-manager activate 3
bin/spec-manager deactivate 5

# Update authentication tokens
bin/spec-manager set-token 1 "new-api-key-123"

# Delete specs you no longer need
bin/spec-manager delete 7
```

## 🔍 Troubleshooting

### Common Issues

**"No such file or directory"**
- Check file paths are correct
- Ensure OpenAPI files are valid YAML/JSON
- Verify you're in the right directory

**"Failed to initialize database"**
- Check PostgreSQL is running
- Verify DATABASE_URL format: `postgresql://user:pass@host:port/db`
- Ensure database exists and is accessible

**"Authentication failed"**
- Verify API keys are correct and not expired
- Check authentication type matches API requirements
- Ensure environment variables are set correctly

**"Tool not found"**
- Check OpenAPI spec has `operationId` for each operation
- Verify the API is active: `bin/spec-manager active`
- Run validation: `bin/openapi-mcp validate your-spec.yaml`

### Getting Help

1. **Check Documentation**: README, examples, and issue templates
2. **Search Issues**: Look for similar problems and solutions
3. **Ask Questions**: Use GitHub Issues with the question template
4. **Community Discussion**: Use GitHub Discussions for broader topics

## 🎓 Advanced Topics

Once you're comfortable with the basics, explore these advanced features:

### Custom Tool Filtering
```sh
# Only expose admin operations
bin/openapi-mcp filter --tag=admin your-spec.yaml

# Exclude deprecated operations
bin/openapi-mcp filter --exclude-desc-regex="deprecated" your-spec.yaml
```

### HTTP API Management
```sh
# Start the management API server
bin/spec-api-server :9090

# Manage specs via HTTP API
curl -X GET http://localhost:9090/specs
curl -X POST http://localhost:9090/specs -d '{"name":"myapi",...}'
```

### Library Integration
```go
package main

import (
    "github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
)

func main() {
    // Load and serve OpenAPI spec programmatically
    doc, _ := openapi2mcp.LoadOpenAPISpec("myapi.yaml")
    srv := openapi2mcp.NewServer("myapi", doc.Info.Version, doc)
    openapi2mcp.ServeHTTP(srv, ":8080")
}
```

## 🎉 What's Next?

Now that you have openapi-mcp running, consider:

1. **Integrate with AI Tools**: Connect Claude, ChatGPT, or other LLMs
2. **Add Your APIs**: Import your organization's OpenAPI specifications
3. **Automate Deployment**: Set up CI/CD for automatic spec updates  
4. **Contribute Back**: Share improvements, report bugs, or help with documentation

## 📚 Additional Resources

- **[Full Documentation](README.md)**: Comprehensive feature reference
- **[Contributing Guide](CONTRIBUTING.md)**: How to contribute to the project
- **[Database Setup](DATABASE_SETUP.md)**: Detailed database configuration
- **[API Examples](SPEC_API_EXAMPLES.md)**: HTTP API usage examples
- **[Original Project](https://github.com/jedisct1/openapi-mcp)**: The foundation this builds upon

---

**Questions?** Open an issue or start a discussion - we're here to help! 🤗