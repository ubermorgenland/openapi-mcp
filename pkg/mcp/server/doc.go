// Package server provides MCP (Model Context Protocol) server implementations.
//
// The server package enables you to create MCP servers that can handle resources,
// prompts, and tools. It supports multiple transport methods including stdio,
// HTTP Server-Sent Events (SSE), and streamable HTTP.
//
// # Quick Start
//
// Creating a basic MCP server:
//
//	server := server.NewMCPServer("my-server", "1.0.0")
//	server.AddTool(tool, handler)
//	server.ServeStdio()
//
// # HTTP Server Example
//
//	server := server.NewMCPServer("my-server", "1.0.0")
//	sseServer := server.NewSSEServer(server)
//	sseServer.Start(":8080")
//
// # Streamable HTTP Example
//
//	server := server.NewMCPServer("my-server", "1.0.0")
//	streamableServer := server.NewStreamableHTTPServer(server)
//	streamableServer.Start(":8080")
//
// For more examples and higher-level abstractions for OpenAPI-based MCP servers,
// see the openapi2mcp package.
//
// # Transport Types
//
// The server package supports three transport methods:
//   - Stdio: Direct process communication via stdin/stdout
//   - SSE: Server-Sent Events over HTTP for web applications
//   - StreamableHTTP: HTTP-based transport with JSON-RPC over HTTP POST/GET
//
// # Server Configuration
//
// Servers can be configured with various options:
//   - Custom capability settings (tools, resources, prompts)
//   - Request hooks for authentication and logging
//   - Custom error handling and response formatting
//
// See ServerOption functions for configuration details.
package server