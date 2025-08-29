# openapi2mcp Go Library

This package provides a Go library for converting OpenAPI 3.x specifications into MCP (Model Context Protocol) tool servers.

## Installation

```bash
go get github.com/jedisct1/openapi-mcp/pkg/openapi2mcp
```

For direct access to MCP types and tools:
```bash
go get github.com/jedisct1/openapi-mcp/pkg/mcp
```

## Usage

```go
package main

import (
        "log"
        "github.com/jedisct1/openapi-mcp/pkg/openapi2mcp"
)

func main() {
        // Load OpenAPI spec
        doc, err := openapi2mcp.LoadOpenAPISpec("openapi.yaml")
        if err != nil {
                log.Fatal(err)
        }

        // Create MCP server
        srv := openapi2mcp.NewServer("myapi", doc.Info.Version, doc)

        // Serve over HTTP (StreamableHTTP is now the default)
        if err := openapi2mcp.ServeStreamableHTTP(srv, ":8080", "/mcp"); err != nil {
                log.Fatal(err)
        }

        // Or serve over stdio
        // if err := openapi2mcp.ServeStdio(srv); err != nil {
        //     log.Fatal(err)
        // }
}
```

### Using MCP Package Directly

For more advanced usage, you can work with MCP types and tools directly:

```go
package main

import (
        "context"
        "log"

        "github.com/jedisct1/openapi-mcp/pkg/mcp/mcp"
        "github.com/jedisct1/openapi-mcp/pkg/mcp/server"
        "github.com/jedisct1/openapi-mcp/pkg/openapi2mcp"
)

func main() {
        // Load OpenAPI spec
        doc, err := openapi2mcp.LoadOpenAPISpec("openapi.yaml")
        if err != nil {
                log.Fatal(err)
        }

        // Create MCP server manually
        srv := server.NewMCPServer("myapi", doc.Info.Version)

        // Register OpenAPI tools
        ops := openapi2mcp.ExtractOpenAPIOperations(doc)
        openapi2mcp.RegisterOpenAPITools(srv, ops, doc, nil)

        // Add custom tools using the MCP package directly
        customTool := mcp.NewTool("custom",
                mcp.WithDescription("A custom tool"),
                mcp.WithString("message", mcp.Description("Message to process"), mcp.Required()),
        )

        srv.AddTool(customTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
                args := req.GetArguments()
                message := args["message"].(string)

                return &mcp.CallToolResult{
                        Content: []mcp.Content{
                                mcp.TextContent{
                                        Type: "text",
                                        Text: "Processed: " + message,
                                },
                        },
                }, nil
        })

        // Serve
        if err := server.ServeStdio(srv); err != nil {
                log.Fatal(err)
        }
}
```

## Features

- Convert OpenAPI 3.x specifications to MCP tool servers
- Support for HTTP (StreamableHTTP is default, SSE also available) and stdio transport
- Automatic tool generation from OpenAPI operations
- Built-in validation and error handling
- AI-optimized responses with structured output

## API Documentation

See [GoDoc](https://pkg.go.dev/github.com/jedisct1/openapi-mcp/pkg/openapi2mcp) for complete API documentation.

### HTTP Client Development

When using HTTP mode, openapi-mcp now serves a StreamableHTTP-based MCP server by default. For developers building HTTP clients, you can interact with the `/mcp` endpoint using POST/GET/DELETE as per the StreamableHTTP protocol. SSE is still available by running with the `--http-transport=sse` flag or using `ServeHTTP` in Go.

See the [StreamableHTTP specification](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http) for protocol details.

If you need SSE, you can still use:

```go
// Serve over HTTP using SSE
if err := openapi2mcp.ServeHTTP(srv, ":8080", "/mcp"); err != nil {
    log.Fatal(err)
}
```
