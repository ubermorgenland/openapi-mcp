// Package mcp defines the core types and interfaces for the Model Context Protocol (MCP).
//
// MCP is a protocol for communication between LLM-powered applications and their
// supporting services. This package provides all the essential types, constants,
// and utility functions needed to work with the MCP protocol.
//
// # Core Concepts
//
// The MCP protocol is built around JSON-RPC 2.0 and defines several key concepts:
//   - Resources: Data or content that can be read by LLMs
//   - Prompts: Templates that can be dynamically filled and used with LLMs  
//   - Tools: Functions that can be called by LLMs to perform actions
//
// # Basic Usage
//
//	// Create a tool
//	tool := mcp.NewTool("get_weather",
//		mcp.WithDescription("Get weather for a location"),
//		mcp.WithString("location", mcp.Required(), mcp.Description("City name")),
//	)
//
//	// Create a prompt
//	prompt := mcp.NewPrompt("code_review",
//		mcp.WithPromptDescription("Review code for best practices"),
//		mcp.WithArgument("code", mcp.RequiredArgument(), mcp.ArgumentDescription("Code to review")),
//	)
//
// # Protocol Support
//
// This package supports MCP protocol versions:
//   - 2024-11-05 (stable)
//   - 2025-03-26 (latest)
//
// Use LATEST_PROTOCOL_VERSION constant for the most recent version.
//
// # Message Types
//
// The package defines all standard MCP message types including:
//   - Requests: Initialize, ListTools, ListResources, CallTool, etc.
//   - Responses: Success responses with result data
//   - Notifications: Progress updates and server-initiated messages
//   - Errors: Standardized error responses with codes and details
//
// # Type Safety
//
// All MCP messages are strongly typed with JSON struct tags for proper
// serialization. Use the provided constructors and utility functions
// to ensure protocol compliance.
//
// For higher-level server implementations, see the server package.
// For OpenAPI integration, see the openapi2mcp package.
package mcp