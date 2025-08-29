package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ubermorgenland/openapi-mcp/pkg/mcp/mcp"
)

func TestStreamableHTTPServer_GET(t *testing.T) {
	// Create a basic MCP server
	mcpServer := NewMCPServer("test-server", "1.0.0")

	// Create the streamable HTTP server
	httpServer := NewStreamableHTTPServer(mcpServer)

	// Create a test server
	testServer := httptest.NewServer(httpServer)
	defer testServer.Close()

	t.Run("GET request establishes SSE connection", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", testServer.URL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send GET request: %v", err)
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status %d, got %d", http.StatusAccepted, resp.StatusCode)
		}

		// Check content type
		if resp.Header.Get("Content-Type") != "text/event-stream" {
			t.Errorf("Expected Content-Type text/event-stream, got %s", resp.Header.Get("Content-Type"))
		}

		// Check cache control
		if resp.Header.Get("Cache-Control") != "no-cache" {
			t.Errorf("Expected Cache-Control no-cache, got %s", resp.Header.Get("Cache-Control"))
		}

		// Check connection header
		if resp.Header.Get("Connection") != "keep-alive" {
			t.Errorf("Expected Connection keep-alive, got %s", resp.Header.Get("Connection"))
		}

		// Read the initial endpoint event
		reader := bufio.NewReader(resp.Body)

		// Read event line
		eventLine, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read event line: %v", err)
		}

		if !strings.HasPrefix(eventLine, "event: endpoint") {
			t.Errorf("Expected initial event to be 'endpoint', got %s", strings.TrimSpace(eventLine))
		}

		// Read data line
		dataLine, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read data line: %v", err)
		}

		if !strings.HasPrefix(dataLine, "data: ?sessionId=") {
			t.Errorf("Expected data line to contain sessionId, got %s", strings.TrimSpace(dataLine))
		}

		// Read empty line
		emptyLine, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read empty line: %v", err)
		}

		if strings.TrimSpace(emptyLine) != "" {
			t.Errorf("Expected empty line after SSE event, got %s", strings.TrimSpace(emptyLine))
		}
	})

	t.Run("POST request with notifications upgrades to SSE", func(t *testing.T) {
		// Add a tool that sends notifications
		mcpServer.AddTool(mcp.Tool{
			Name: "test-tool",
		}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Send a notification to the client
			server := ServerFromContext(ctx)
			if server != nil {
				// Send notification immediately - the upgrade should happen
				err := server.SendNotificationToClient(ctx, "test/notification", map[string]any{
					"message": "test notification",
				})
				if err != nil {
					t.Logf("Failed to send notification: %v", err)
				} else {
					t.Logf("Notification sent successfully")
				}
				// Add a small delay to ensure the notification is processed before response
				time.Sleep(20 * time.Millisecond)
			} else {
				t.Logf("Server not found in context")
			}
			return mcp.NewToolResultText("done", nil, nil, nil, "", nil), nil
		})

		// First, initialize the session
		initRequest := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]any{
				"protocolVersion": "2025-03-26",
				"clientInfo": map[string]any{
					"name":    "test-client",
					"version": "1.0.0",
				},
			},
		}

		initBody, _ := json.Marshal(initRequest)
		resp, err := http.Post(testServer.URL, "application/json", strings.NewReader(string(initBody)))
		if err != nil {
			t.Fatalf("Failed to send initialize request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 for initialize, got %d", resp.StatusCode)
		}

		sessionID := resp.Header.Get("Mcp-Session-Id")
		if sessionID == "" {
			t.Fatal("Expected session ID in response header")
		}

		// Now call the tool that sends notifications
		toolRequest := map[string]any{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "tools/call",
			"params": map[string]any{
				"name": "test-tool",
			},
		}

		toolBody, _ := json.Marshal(toolRequest)
		req, err := http.NewRequest("POST", testServer.URL, strings.NewReader(string(toolBody)))
		if err != nil {
			t.Fatalf("Failed to create tool request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Mcp-Session-Id", sessionID)

		resp2, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send tool request: %v", err)
		}
		defer resp2.Body.Close()

		// Should upgrade to SSE when notifications are sent
		if resp2.StatusCode != http.StatusAccepted {
			t.Errorf("Expected status 202 for SSE upgrade, got %d", resp2.StatusCode)
		}

		if resp2.Header.Get("Content-Type") != "text/event-stream" {
			t.Errorf("Expected Content-Type text/event-stream for SSE upgrade, got %s", resp2.Header.Get("Content-Type"))
		}
	})
}
