// server.go
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/database"
	"github.com/ubermorgenland/openapi-mcp/pkg/mcp/mcp"
	mcpserver "github.com/ubermorgenland/openapi-mcp/pkg/mcp/server"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
	"github.com/ubermorgenland/openapi-mcp/pkg/services"
)

// tryLoadFromDatabase attempts to load specs from database if DATABASE_URL is set
func tryLoadFromDatabase() ([]openapi2mcp.OpenAPIOperation, []*openapi3.T, bool) {
	if os.Getenv("DATABASE_URL") == "" {
		return nil, nil, false
	}

	// Try to initialize database connection
	if err := database.InitializeDatabase(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize database: %v\nFalling back to file loading\n", err)
		return nil, nil, false
	}

	// Create spec loader service
	specLoader := services.NewSpecLoaderService(database.DB)

	// Try to load specs from database
	ops, docs, err := specLoader.LoadFromDatabase()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load specs from database: %v\nFalling back to file loading\n", err)
		return nil, nil, false
	}

	fmt.Fprintf(os.Stderr, "Successfully loaded %d operations from %d database specs\n", len(ops), len(docs))
	return ops, docs, true
}

// startServer starts the MCP server in stdio or HTTP mode, based on CLI flags.
// It registers all OpenAPI operations as MCP tools and starts the server.
func startServer(flags *cliFlags, ops []openapi2mcp.OpenAPIOperation, doc *openapi3.T) {
	if flags.httpAddr != "" && len(flags.mounts) > 0 {
		// Check for duplicate base paths
		basePathCount := make(map[string]int)
		for _, m := range flags.mounts {
			basePathCount[m.BasePath]++
		}
		var dups []string
		for base, count := range basePathCount {
			if count > 1 {
				dups = append(dups, base)
			}
		}
		if len(dups) > 0 {
			fmt.Fprintf(os.Stderr, "Error: duplicate --mount base path(s): %v\nEach base path may only be used once.\n", dups)
			os.Exit(2)
		}
		if len(flags.args) > 0 {
			fmt.Fprintln(os.Stderr, "[WARN] Positional OpenAPI spec arguments are ignored when using --mount. Only --mount will be used.")
		}
		mux := http.NewServeMux()
		for _, m := range flags.mounts {
			fmt.Fprintf(os.Stderr, "Loading OpenAPI spec for mount %s: %s...\n", m.BasePath, m.SpecPath)
			d, err := openapi3.NewLoader().LoadFromFile(m.SpecPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load OpenAPI spec for %s: %v\n", m.BasePath, err)
				os.Exit(1)
			}
			ops = openapi2mcp.ExtractOpenAPIOperations(d)
			srv, logFileHandle := createServerWithOptions("openapi-mcp", d.Info.Version, d, ops, flags.logFile, flags.noLogTruncation)
			if logFileHandle != nil {
				defer logFileHandle.Close()
			}
			var handler http.Handler
			if flags.httpTransport == "streamable" {
				handler = openapi2mcp.HandlerForStreamableHTTP(srv, m.BasePath)
			} else {
				handler = openapi2mcp.HandlerForBasePath(srv, m.BasePath)
			}
			mux.Handle(m.BasePath+"/", handler)
			mux.Handle(m.BasePath, handler) // allow both /base and /base/
			fmt.Fprintf(os.Stderr, "Mounted %s at %s\n", m.SpecPath, m.BasePath)
		}
		fmt.Fprintf(os.Stderr, "Starting multi-mount MCP HTTP server on %s...\n", flags.httpAddr)
		if err := http.ListenAndServe(flags.httpAddr, mux); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start MCP HTTP server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if flags.httpAddr != "" {
		// Try database first for HTTP mode
		dbOps, dbDocs, fromDB := tryLoadFromDatabase()
		if fromDB && len(dbDocs) > 0 {
			// Use database specs
			d := dbDocs[0]
			ops := dbOps
			srv, logFileHandle := createServerWithOptions("openapi-mcp", d.Info.Version, d, ops, flags.logFile, flags.noLogTruncation)
			if logFileHandle != nil {
				defer logFileHandle.Close()
			}
			fmt.Fprintf(os.Stderr, "Starting MCP server (HTTP, %s transport) with database specs on %s...\n", flags.httpTransport, flags.httpAddr)
			if flags.httpTransport == "streamable" {
				if err := openapi2mcp.ServeStreamableHTTP(srv, flags.httpAddr, "/mcp"); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to start MCP HTTP server: %v\n", err)
					os.Exit(1)
				}
			} else {
				if err := openapi2mcp.ServeHTTP(srv, flags.httpAddr, "/mcp"); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to start MCP HTTP server: %v\n", err)
					os.Exit(1)
				}
			}
		} else {
			// Fall back to file-based loading
			if len(flags.args) != 1 {
				fmt.Fprintln(os.Stderr, "Usage: openapi-mcp --http=:8080 <openapi-spec-path>")
				os.Exit(2)
			}
			specPath := flags.args[0]
			d, err := openapi3.NewLoader().LoadFromFile(specPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load OpenAPI spec: %v\n", err)
				os.Exit(1)
			}
			ops := openapi2mcp.ExtractOpenAPIOperations(d)
			srv, logFileHandle := createServerWithOptions("openapi-mcp", d.Info.Version, d, ops, flags.logFile, flags.noLogTruncation)
			if logFileHandle != nil {
				defer logFileHandle.Close()
			}
			fmt.Fprintf(os.Stderr, "Starting MCP server (HTTP, %s transport) on %s...\n", flags.httpTransport, flags.httpAddr)
			if flags.httpTransport == "streamable" {
				if err := openapi2mcp.ServeStreamableHTTP(srv, flags.httpAddr, "/mcp"); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to start MCP HTTP server: %v\n", err)
					os.Exit(1)
				}
			} else {
				if err := openapi2mcp.ServeHTTP(srv, flags.httpAddr, "/mcp"); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to start MCP HTTP server: %v\n", err)
					os.Exit(1)
				}
			}
		}
		return
	}

	// stdio mode: try database first, then fall back to file loading
	dbOps, dbDocs, fromDB := tryLoadFromDatabase()
	if fromDB && len(dbDocs) > 0 {
		// Use first doc for server info, combine all operations
		d := dbDocs[0]
		ops = dbOps
		srv, logFileHandle := createServerWithOptions("openapi-mcp", d.Info.Version, d, ops, flags.logFile, flags.noLogTruncation)
		if logFileHandle != nil {
			defer logFileHandle.Close()
		}
		fmt.Fprintln(os.Stderr, "Registered all OpenAPI operations from database as MCP tools.")
		fmt.Fprintln(os.Stderr, "Starting MCP server (stdio) with database specs...")
		if err := openapi2mcp.ServeStdio(srv); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start MCP server: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Fall back to original file-based loading
		if len(flags.args) != 1 {
			fmt.Fprintln(os.Stderr, "Usage: openapi-mcp <openapi-spec-path>")
			os.Exit(2)
		}
		specPath := flags.args[0]
		d, err := openapi3.NewLoader().LoadFromFile(specPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load OpenAPI spec: %v\n", err)
			os.Exit(1)
		}
		ops = openapi2mcp.ExtractOpenAPIOperations(d)
		srv, logFileHandle := createServerWithOptions("openapi-mcp", d.Info.Version, d, ops, flags.logFile, flags.noLogTruncation)
		if logFileHandle != nil {
			defer logFileHandle.Close()
		}
		fmt.Fprintln(os.Stderr, "Registered all OpenAPI operations as MCP tools.")
		fmt.Fprintln(os.Stderr, "Starting MCP server (stdio)...")
		if err := openapi2mcp.ServeStdio(srv); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start MCP server: %v\n", err)
			os.Exit(1)
		}
	}
}

// makeMCPHandler returns an http.Handler that serves the MCP server at the given basePath.
func makeMCPHandler(srv *mcpserver.MCPServer, basePath string) http.Handler {
	return openapi2mcp.HandlerForBasePath(srv, basePath)
}

// formatHumanReadableLog creates a human-readable log entry for MCP transactions
func formatHumanReadableLog(timestamp, logType, method string, id any, data interface{}, err error, noTruncation bool) string {
	var log strings.Builder

	// Header with timestamp and type
	log.WriteString("═══════════════════════════════════════════════════════════════════════════════\n")
	log.WriteString(fmt.Sprintf("🕐 %s | %s | Method: %s",
		timestamp, strings.ToUpper(logType), method))

	if id != nil {
		log.WriteString(fmt.Sprintf(" | ID: %v", id))
	}
	log.WriteString("\n")

	// Content based on type
	switch logType {
	case "request":
		log.WriteString("📤 INCOMING REQUEST\n")

		// Handle typed MCP request objects
		switch req := data.(type) {
		case *mcp.CallToolRequest:
			// Handle CallToolRequest directly
			log.WriteString(fmt.Sprintf("🔧 Tool: %s\n", req.Params.Name))
			args := req.GetArguments()
			if len(args) > 0 {
				log.WriteString("📝 Arguments:\n")
				for key, value := range args {
					valueStr := formatValue(value, noTruncation)
					log.WriteString(fmt.Sprintf("   %s: %s\n", key, valueStr))
				}
			} else {
				log.WriteString("📝 Arguments: (none)\n")
			}

		case *mcp.ListToolsRequest:
			// ListToolsRequest typically has pagination params
			log.WriteString("📝 Method: tools/list\n")
			if req.Params.Cursor != "" {
				log.WriteString(fmt.Sprintf("   Cursor: %s\n", req.Params.Cursor))
			}

		case *mcp.InitializeRequest:
			log.WriteString("📝 Method: initialize\n")
			log.WriteString(fmt.Sprintf("   Protocol Version: %s\n", req.Params.ProtocolVersion))
			if req.Params.ClientInfo.Name != "" {
				log.WriteString(fmt.Sprintf("   Client: %s/%s\n", req.Params.ClientInfo.Name, req.Params.ClientInfo.Version))
			}

		case *mcp.PingRequest:
			log.WriteString("📝 Method: ping\n")

		default:
			// For other request types or if we can't determine the type,
			// try to marshal to JSON and display
			if jsonData, err := json.MarshalIndent(data, "   ", "  "); err == nil {
				log.WriteString(fmt.Sprintf("📝 Request:\n   %s\n", string(jsonData)))
			} else {
				log.WriteString(fmt.Sprintf("📝 Request type: %T\n", data))
			}
		}

	case "response":
		log.WriteString("📥 OUTGOING RESPONSE\n")
		if os.Getenv("DEBUG_RESPONSE") != "" {
			log.WriteString(fmt.Sprintf("🐛 Data type: %T\n", data))
			if data != nil {
				dataJSON, _ := json.MarshalIndent(data, "   ", "  ")
				log.WriteString(fmt.Sprintf("🐛 Data content: %s\n", string(dataJSON)))
			}
		}
		// Handle specific MCP result types
		switch result := data.(type) {
		case *mcp.ListToolsResult:
			tools := result.Tools
			log.WriteString(fmt.Sprintf("🔧 Tools Listed: %d tools\n", len(tools)))
			if noTruncation || len(tools) <= 10 {
				// Show all tools if no truncation or 10 or fewer
				for i, tool := range tools {
					desc := ""
					if len(tool.Description) > 0 {
						// Extract first line of description for brevity
						lines := strings.Split(tool.Description, "\n")
						if len(lines) > 0 {
							desc = lines[0]
							if !noTruncation && len(desc) > 80 {
								desc = desc[:80] + "..."
							}
						}
					}
					log.WriteString(fmt.Sprintf("   [%d] %s: %s\n", i+1, tool.Name, desc))
				}
			} else {
				// Show first 5 tools and mention there are more
				for i := 0; i < 5; i++ {
					desc := ""
					if len(tools[i].Description) > 0 {
						lines := strings.Split(tools[i].Description, "\n")
						if len(lines) > 0 {
							desc = lines[0]
							if !noTruncation && len(desc) > 80 {
								desc = desc[:80] + "..."
							}
						}
					}
					log.WriteString(fmt.Sprintf("   [%d] %s: %s\n", i+1, tools[i].Name, desc))
				}
				log.WriteString(fmt.Sprintf("   ... and %d more tools\n", len(tools)-5))
			}
		case *mcp.CallToolResult:
			if len(result.Content) > 0 {
				log.WriteString("📋 Response Content:\n")
				for i, item := range result.Content {
					if textContent, ok := item.(mcp.TextContent); ok {
						log.WriteString(fmt.Sprintf("   [%d] Type: %s\n", i+1, textContent.Type))
						// Truncate very long responses
						if !noTruncation && len(textContent.Text) > 500 {
							log.WriteString(fmt.Sprintf("   [%d] Text: %s... (%d chars total)\n",
								i+1, textContent.Text[:500], len(textContent.Text)))
						} else {
							log.WriteString(fmt.Sprintf("   [%d] Text: %s\n", i+1, textContent.Text))
						}
					}
				}
			}
		default:
			// Handle generic map[string]interface{} responses
			if result, ok := data.(map[string]interface{}); ok {
				if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
					log.WriteString("📋 Response Content:\n")
					for i, item := range content {
						if contentItem, ok := item.(map[string]interface{}); ok {
							if contentType, ok := contentItem["type"].(string); ok {
								log.WriteString(fmt.Sprintf("   [%d] Type: %s\n", i+1, contentType))
							}
							if text, ok := contentItem["text"].(string); ok {
								// Truncate very long responses
								if !noTruncation && len(text) > 500 {
									log.WriteString(fmt.Sprintf("   [%d] Text: %s... (%d chars total)\n",
										i+1, text[:500], len(text)))
								} else {
									log.WriteString(fmt.Sprintf("   [%d] Text: %s\n", i+1, text))
								}
							}
						}
					}
				}
			} else if tools, ok := result["tools"].([]interface{}); ok {
				log.WriteString(fmt.Sprintf("🔧 Tools Listed: %d tools\n", len(tools)))
				if noTruncation || len(tools) <= 10 {
					// Show all tools if no truncation or 10 or fewer
					for i, tool := range tools {
						if toolItem, ok := tool.(map[string]interface{}); ok {
							if name, ok := toolItem["name"].(string); ok {
								desc := ""
								if description, ok := toolItem["description"].(string); ok && len(description) > 0 {
									// Extract first line of description for brevity
									lines := strings.Split(description, "\\n")
									if len(lines) > 0 {
										desc = lines[0]
										if len(desc) > 80 {
											desc = desc[:80] + "..."
										}
									}
								}
								log.WriteString(fmt.Sprintf("   [%d] %s: %s\n", i+1, name, desc))
							}
						}
					}
				} else {
					// Show first 5 tools and mention there are more
					for i := 0; i < 5; i++ {
						if toolItem, ok := tools[i].(map[string]interface{}); ok {
							if name, ok := toolItem["name"].(string); ok {
								desc := ""
								if description, ok := toolItem["description"].(string); ok && len(description) > 0 {
									lines := strings.Split(description, "\\n")
									if len(lines) > 0 {
										desc = lines[0]
										if len(desc) > 80 {
											desc = desc[:80] + "..."
										}
									}
								}
								log.WriteString(fmt.Sprintf("   [%d] %s: %s\n", i+1, name, desc))
							}
						}
					}
					log.WriteString(fmt.Sprintf("   ... and %d more tools\n", len(tools)-5))
				}
			} else {
				// Generic response formatting - show structure for debugging
				prettyJSON, _ := json.MarshalIndent(result, "   ", "  ")
				if len(string(prettyJSON)) > 2000 {
					log.WriteString(fmt.Sprintf("📋 Result: %s... (%d chars total)\n", string(prettyJSON)[:2000], len(string(prettyJSON))))
				} else {
					log.WriteString(fmt.Sprintf("📋 Result:\n   %s\n", string(prettyJSON)))
				}
			}
		}

	case "error":
		log.WriteString("❌ ERROR OCCURRED\n")
		log.WriteString(fmt.Sprintf("💥 Error: %s\n", err.Error()))
		if data != nil {
			prettyJSON, _ := json.MarshalIndent(data, "   ", "  ")
			log.WriteString(fmt.Sprintf("📝 Request Data:\n   %s\n", string(prettyJSON)))
		}
	}

	log.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")
	return log.String()
}

// formatValue formats a value for human-readable display
func formatValue(value interface{}, noTruncation bool) string {
	switch v := value.(type) {
	case string:
		if !noTruncation && len(v) > 100 {
			return fmt.Sprintf("\"%s...\" (%d chars)", v[:100], len(v))
		}
		return fmt.Sprintf("\"%s\"", v)
	case map[string]interface{}:
		if len(v) == 0 {
			return "{}"
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		if !noTruncation && len(keys) > 3 {
			return fmt.Sprintf("{%s, ...} (%d keys)", strings.Join(keys[:3], ", "), len(keys))
		}
		return fmt.Sprintf("{%s}", strings.Join(keys, ", "))
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// createLoggingHooks creates MCP hooks for logging requests and responses to a file
func createLoggingHooks(logFilePath string, noLogTruncation bool) (*mcpserver.Hooks, *os.File, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(logFile, "", 0) // No prefix, we'll format our own output

	hooks := &mcpserver.Hooks{}

	// Log requests with human-readable format
	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		timestamp := time.Now().Format("2006-01-02 15:04:05 MST")
		humanLog := formatHumanReadableLog(timestamp, "request", string(method), id, message, nil, noLogTruncation)
		logger.Print(humanLog)
	})

	// Log successful responses with human-readable format
	hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
		timestamp := time.Now().Format("2006-01-02 15:04:05 MST")
		humanLog := formatHumanReadableLog(timestamp, "response", string(method), id, result, nil, noLogTruncation)
		logger.Print(humanLog)
	})

	// Log errors with human-readable format
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		timestamp := time.Now().Format("2006-01-02 15:04:05 MST")
		humanLog := formatHumanReadableLog(timestamp, "error", string(method), id, message, err, noLogTruncation)
		logger.Print(humanLog)
	})

	return hooks, logFile, nil
}

// createServerWithOptions creates a new MCP server with the given operations and optional logging
func createServerWithOptions(name, version string, doc *openapi3.T, ops []openapi2mcp.OpenAPIOperation, logFile string, noLogTruncation bool) (*mcpserver.MCPServer, *os.File) {
	var opts []mcpserver.ServerOption
	var logFileHandle *os.File

	if logFile != "" {
		hooks, fileHandle, err := createLoggingHooks(logFile, noLogTruncation)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create logging hooks: %v\n", err)
			os.Exit(1)
		}
		logFileHandle = fileHandle
		opts = append(opts, mcpserver.WithHooks(hooks))
		fmt.Fprintf(os.Stderr, "Logging MCP requests and responses to: %s\n", logFile)
	}

	srv := mcpserver.NewMCPServer(name, version, opts...)
	openapi2mcp.RegisterOpenAPITools(srv, ops, doc, nil)
	return srv, logFileHandle
}
