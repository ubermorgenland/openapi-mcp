// server.go
package openapi2mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	mcpserver "github.com/ubermorgenland/openapi-mcp/pkg/mcp/server"
)

// authContextFunc extracts authentication from environment variables first, then falls back to HTTP headers.
// This ensures environment variables take priority over per-request authentication.
// Priority order: Environment Variables > HTTP Headers
func authContextFunc(ctx context.Context, r *http.Request) context.Context {
	// Save original environment values to restore them later
	origAPIKey := os.Getenv("API_KEY")
	origBearerToken := os.Getenv("BEARER_TOKEN")
	origBasicAuth := os.Getenv("BASIC_AUTH")

	// Phase 1: Environment variables are already set and take priority
	// No need to override if they exist - they're the primary source

	// Phase 2: Extract authentication from HTTP headers ONLY as fallback
	// Only set from headers if environment variables are not already set

	// API Key fallback: Try common API key header patterns
	if os.Getenv("API_KEY") == "" {
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			os.Setenv("API_KEY", apiKey)
		} else if apiKey := r.Header.Get("Api-Key"); apiKey != "" {
			os.Setenv("API_KEY", apiKey)
		} else if apiKey := r.Header.Get("x-rapidapi-key"); apiKey != "" {
			// Handle RapidAPI specific header
			os.Setenv("API_KEY", apiKey)
		}
	}

	// Bearer Token fallback: Extract from Authorization header
	if os.Getenv("BEARER_TOKEN") == "" {
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
				os.Setenv("BEARER_TOKEN", authHeader[7:])
			}
		}
	}

	// Basic Auth fallback: Extract from Authorization header
	if os.Getenv("BASIC_AUTH") == "" {
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			if len(authHeader) > 6 && strings.HasPrefix(authHeader, "Basic ") {
				os.Setenv("BASIC_AUTH", authHeader[6:])
			}
		}
	}

	// Create a context that restores the original environment when done
	return &authContext{
		Context:         ctx,
		origAPIKey:      origAPIKey,
		origBearerToken: origBearerToken,
		origBasicAuth:   origBasicAuth,
	}
}

// authContext wraps a context and restores original environment variables when done
type authContext struct {
	context.Context
	origAPIKey      string
	origBearerToken string
	origBasicAuth   string
}

// Done restores the original environment variables when the context is done
func (c *authContext) Done() <-chan struct{} {
	done := c.Context.Done()
	if done != nil {
		go func() {
			<-done
			c.restoreEnv()
		}()
	}
	return done
}

func (c *authContext) restoreEnv() {
	// Restore original environment variables
	if c.origAPIKey != "" {
		os.Setenv("API_KEY", c.origAPIKey)
	} else {
		os.Unsetenv("API_KEY")
	}
	if c.origBearerToken != "" {
		os.Setenv("BEARER_TOKEN", c.origBearerToken)
	} else {
		os.Unsetenv("BEARER_TOKEN")
	}
	if c.origBasicAuth != "" {
		os.Setenv("BASIC_AUTH", c.origBasicAuth)
	} else {
		os.Unsetenv("BASIC_AUTH")
	}
}

// NewServer creates a new MCP server, registers all OpenAPI tools, and returns the server.
// Equivalent to calling RegisterOpenAPITools with all operations from the spec.
// Example usage for NewServer:
//
//	doc, _ := openapi2mcp.LoadOpenAPISpec("petstore.yaml")
//	srv := openapi2mcp.NewServer("petstore", doc.Info.Version, doc)
//	openapi2mcp.ServeHTTP(srv, ":8080")
func NewServer(name, version string, doc *openapi3.T) *mcpserver.MCPServer {
	ops := ExtractOpenAPIOperations(doc)
	srv := mcpserver.NewMCPServer(name, version)
	RegisterOpenAPITools(srv, ops, doc, nil)
	return srv
}

// NewServerWithOps creates a new MCP server, registers the provided OpenAPI operations, and returns the server.
// Example usage for NewServerWithOps:
//
//	doc, _ := openapi2mcp.LoadOpenAPISpec("petstore.yaml")
//	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
//	srv := openapi2mcp.NewServerWithOps("petstore", doc.Info.Version, doc, ops)
//	openapi2mcp.ServeHTTP(srv, ":8080")
func NewServerWithOps(name, version string, doc *openapi3.T, ops []OpenAPIOperation) *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer(name, version)
	RegisterOpenAPITools(srv, ops, doc, nil)
	return srv
}

// ServeStdio starts the MCP server using stdio (wraps mcpserver.ServeStdio).
// Returns an error if the server fails to start.
// Example usage for ServeStdio:
//
//	openapi2mcp.ServeStdio(srv)
func ServeStdio(server *mcpserver.MCPServer) error {
	return mcpserver.ServeStdio(server)
}

// ServeHTTP starts the MCP server using HTTP SSE (wraps mcpserver.NewSSEServer and Start).
// addr is the address to listen on, e.g. ":8080".
// basePath is the base HTTP path to mount the MCP server (e.g. "/mcp").
// Returns an error if the server fails to start.
// Example usage for ServeHTTP:
//
//	srv, _ := openapi2mcp.NewServer("petstore", "1.0.0", doc)
//	openapi2mcp.ServeHTTP(srv, ":8080", "/custom-base")
func ServeHTTP(server *mcpserver.MCPServer, addr string, basePath string) error {
	// Convert the authContextFunc to SSEContextFunc signature
	sseAuthContextFunc := func(ctx context.Context, r *http.Request) context.Context {
		return authContextFunc(ctx, r)
	}

	if basePath == "" {
		basePath = "/mcp"
	}

	sseServer := mcpserver.NewSSEServer(server,
		mcpserver.WithSSEContextFunc(sseAuthContextFunc),
		mcpserver.WithStaticBasePath(basePath),
		mcpserver.WithSSEEndpoint("/sse"),
		mcpserver.WithMessageEndpoint("/message"))
	return sseServer.Start(addr)
}

// GetSSEURL returns the URL for establishing an SSE connection to the MCP server.
// addr is the address the server is listening on (e.g., ":8080", "0.0.0.0:8080", "localhost:8080").
// basePath is the base HTTP path (e.g., "/mcp").
// Example usage:
//
//	url := openapi2mcp.GetSSEURL(":8080", "/custom-base")
//	// Returns: "http://localhost:8080/custom-base/sse"
func GetSSEURL(addr, basePath string) string {
	if basePath == "" {
		basePath = "/mcp"
	}
	host := normalizeAddrToHost(addr)
	return "http://" + host + basePath + "/sse"
}

// GetMessageURL returns the URL for sending JSON-RPC requests to the MCP server.
// addr is the address the server is listening on (e.g., ":8080", "0.0.0.0:8080", "localhost:8080").
// basePath is the base HTTP path (e.g., "/mcp").
// sessionID should be the session ID received from the SSE endpoint event.
// Example usage:
//
//	url := openapi2mcp.GetMessageURL(":8080", "/custom-base", "session-id-123")
//	// Returns: "http://localhost:8080/custom-base/message?sessionId=session-id-123"
func GetMessageURL(addr, basePath, sessionID string) string {
	if basePath == "" {
		basePath = "/mcp"
	}
	host := normalizeAddrToHost(addr)
	return fmt.Sprintf("http://%s%s/message?sessionId=%s", host, basePath, sessionID)
}

// GetStreamableHTTPURL returns the URL for the Streamable HTTP endpoint of the MCP server.
// addr is the address the server is listening on (e.g., ":8080", "0.0.0.0:8080", "localhost:8080").
// basePath is the base HTTP path (e.g., "/mcp").
// Example usage:
//
//	url := openapi2mcp.GetStreamableHTTPURL(":8080", "/custom-base")
//	// Returns: "http://localhost:8080/custom-base"
func GetStreamableHTTPURL(addr, basePath string) string {
	if basePath == "" {
		basePath = "/mcp"
	}
	host := normalizeAddrToHost(addr)
	return "http://" + host + basePath
}

// normalizeAddrToHost converts an addr (as used by net/http) to a host:port string suitable for URLs.
// If addr is just ":8080", returns "localhost:8080". If it already includes a host, returns as is.
func normalizeAddrToHost(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "localhost"
	}
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

// HandlerForBasePath returns an http.Handler that serves the given MCP server at the specified basePath.
// This is useful for multi-mount HTTP servers, where you want to serve multiple OpenAPI schemas at different URL paths.
// Example usage:
//
//	handler := openapi2mcp.HandlerForBasePath(srv, "/petstore")
//	mux.Handle("/petstore/", handler)
func HandlerForBasePath(server *mcpserver.MCPServer, basePath string) http.Handler {
	sseAuthContextFunc := func(ctx context.Context, r *http.Request) context.Context {
		return authContextFunc(ctx, r)
	}
	if basePath == "" {
		basePath = "/mcp"
	}
	sseServer := mcpserver.NewSSEServer(server,
		mcpserver.WithSSEContextFunc(sseAuthContextFunc),
		mcpserver.WithStaticBasePath(basePath),
		mcpserver.WithSSEEndpoint("/sse"),
		mcpserver.WithMessageEndpoint("/message"),
	)
	return sseServer
}

// ServeStreamableHTTP starts the MCP server using HTTP StreamableHTTP (wraps mcpserver.NewStreamableHTTPServer and Start).
// addr is the address to listen on, e.g. ":8080".
// basePath is the base HTTP path to mount the MCP server (e.g. "/mcp").
// Returns an error if the server fails to start.
// Example usage for ServeStreamableHTTP:
//
//	srv, _ := openapi2mcp.NewServer("petstore", "1.0.0", doc)
//	openapi2mcp.ServeStreamableHTTP(srv, ":8080", "/custom-base")
func ServeStreamableHTTP(server *mcpserver.MCPServer, addr string, basePath string) error {
	streamableAuthContextFunc := func(ctx context.Context, r *http.Request) context.Context {
		return authContextFunc(ctx, r)
	}

	if basePath == "" {
		basePath = "/mcp"
	}

	streamableServer := mcpserver.NewStreamableHTTPServer(server,
		mcpserver.WithHTTPContextFunc(streamableAuthContextFunc),
		mcpserver.WithEndpointPath(basePath),
	)
	return streamableServer.Start(addr)
}

// HandlerForStreamableHTTP returns an http.Handler that serves the given MCP server at the specified basePath using StreamableHTTP.
// This is useful for multi-mount HTTP servers, where you want to serve multiple OpenAPI schemas at different URL paths.
// Example usage:
//
//	handler := openapi2mcp.HandlerForStreamableHTTP(srv, "/petstore")
//	mux.Handle("/petstore", handler)
func HandlerForStreamableHTTP(server *mcpserver.MCPServer, basePath string) http.Handler {
	streamableAuthContextFunc := func(ctx context.Context, r *http.Request) context.Context {
		return authContextFunc(ctx, r)
	}
	if basePath == "" {
		basePath = "/mcp"
	}
	streamableServer := mcpserver.NewStreamableHTTPServer(server,
		mcpserver.WithHTTPContextFunc(streamableAuthContextFunc),
		mcpserver.WithEndpointPath(basePath),
	)
	return streamableServer
}
