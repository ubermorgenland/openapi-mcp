package openapi2mcp

import (
	"testing"
)

func TestGetSSEURL(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		basePath string
		expected string
	}{
		{
			name:     "basic addr",
			addr:     ":8080",
			basePath: "/mcp",
			expected: "http://localhost:8080/mcp/sse",
		},
		{
			name:     "addr with host",
			addr:     "127.0.0.1:3000",
			basePath: "/api",
			expected: "http://127.0.0.1:3000/api/sse",
		},
		{
			name:     "addr with hostname",
			addr:     "myhost:9000",
			basePath: "/foo",
			expected: "http://myhost:9000/foo/sse",
		},
		{
			name:     "empty basePath",
			addr:     ":8080",
			basePath: "",
			expected: "http://localhost:8080/mcp/sse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSSEURL(tt.addr, tt.basePath)
			if result != tt.expected {
				t.Errorf("GetSSEURL(%q, %q) = %q, want %q", tt.addr, tt.basePath, result, tt.expected)
			}
		})
	}
}

func TestGetMessageURL(t *testing.T) {
	tests := []struct {
		name      string
		addr      string
		basePath  string
		sessionID string
		expected  string
	}{
		{
			name:      "basic addr with session",
			addr:      ":8080",
			basePath:  "/mcp",
			sessionID: "session-123",
			expected:  "http://localhost:8080/mcp/message?sessionId=session-123",
		},
		{
			name:      "addr with host",
			addr:      "127.0.0.1:3000",
			basePath:  "/api",
			sessionID: "abc-def-ghi",
			expected:  "http://127.0.0.1:3000/api/message?sessionId=abc-def-ghi",
		},
		{
			name:      "hostname and uuid session",
			addr:      "myhost:9000",
			basePath:  "/foo",
			sessionID: "550e8400-e29b-41d4-a716-446655440000",
			expected:  "http://myhost:9000/foo/message?sessionId=550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:      "empty session ID",
			addr:      ":8080",
			basePath:  "/mcp",
			sessionID: "",
			expected:  "http://localhost:8080/mcp/message?sessionId=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMessageURL(tt.addr, tt.basePath, tt.sessionID)
			if result != tt.expected {
				t.Errorf("GetMessageURL(%q, %q, %q) = %q, want %q", tt.addr, tt.basePath, tt.sessionID, result, tt.expected)
			}
		})
	}
}
