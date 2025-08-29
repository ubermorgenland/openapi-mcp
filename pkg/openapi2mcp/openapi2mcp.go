// Package openapi2mcp transforms OpenAPI 3.x specifications into MCP (Model Context Protocol) tool servers.
//
// This package provides the core functionality to automatically convert any OpenAPI 3.x
// specification into a fully functional MCP server that can be used by AI agents and LLMs.
// It handles the complete conversion pipeline: parsing OpenAPI specs, generating MCP tools
// for each operation, and serving them via stdio or HTTP transports.
//
// # Quick Start
//
//	// Load an OpenAPI specification
//	doc, err := openapi2mcp.LoadOpenAPISpec("petstore.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Create and start MCP server
//	srv := openapi2mcp.NewServer("petstore", doc.Info.Version, doc)
//	openapi2mcp.ServeStdio(srv) // or ServeHTTP(srv, ":8080")
//
// # Authentication Support
//
// The package automatically handles OpenAPI authentication schemes:
//   - API Keys (header, query, cookie)
//   - Bearer tokens (OAuth2, JWT)
//   - Basic authentication
//   - Custom security schemes
//
// Authentication can be provided via environment variables, command-line flags,
// or HTTP headers (in HTTP mode).
//
// # Advanced Usage
//
//	// Extract operations for custom processing
//	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
//
//	// Create server with custom options
//	srv := mcpserver.NewMCPServer("myapi", "1.0.0")
//	opts := &ToolGenOptions{
//		TagFilter: []string{"admin"},
//		ConfirmDangerousActions: true,
//	}
//	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, opts)
//
// # Validation and Linting
//
// The package includes comprehensive OpenAPI validation:
//
//	// Validate specification
//	issues := openapi2mcp.ValidateOpenAPISpec(doc)
//
//	// Lint for best practices
//	lintResults := openapi2mcp.LintOpenAPISpec(doc)
//
// # Output Formats
//
// All tool responses use structured formats optimized for AI agent consumption,
// with consistent OutputFormat and OutputType fields for reliable parsing.
//
// For server implementations, see the server package.
// For core MCP protocol types, see the mcp package.
package openapi2mcp

import (
	"github.com/getkin/kin-openapi/openapi3"
)

// OpenAPIOperation describes a single OpenAPI operation to be mapped to an MCP tool.
// It includes the operation's ID, summary, description, HTTP path/method, parameters, request body, and tags.
type OpenAPIOperation struct {
	OperationID string
	Summary     string
	Description string
	Path        string
	Method      string
	Parameters  openapi3.Parameters
	RequestBody *openapi3.RequestBodyRef
	Tags        []string
	Security    openapi3.SecurityRequirements
}

// ToolGenOptions controls tool generation and output for OpenAPI-MCP conversion.
//
// NameFormat: function to format tool names (e.g., strings.ToLower)
// TagFilter: only include operations with at least one of these tags (if non-empty)
// DryRun: if true, only print the generated tool schemas, don't register
// PrettyPrint: if true, pretty-print the output
// Version: version string to embed in tool annotations
// PostProcessSchema: optional hook to modify each tool's input schema before registration/output
// ConfirmDangerousActions: if true (default), require confirmation for PUT/POST/DELETE tools
//
//	func(toolName string, schema map[string]any) map[string]any
type ToolGenOptions struct {
	NameFormat              func(string) string
	TagFilter               []string
	DryRun                  bool
	PrettyPrint             bool
	Version                 string
	PostProcessSchema       func(toolName string, schema map[string]any) map[string]any
	ConfirmDangerousActions bool // if true, add confirmation prompt for dangerous actions
}
