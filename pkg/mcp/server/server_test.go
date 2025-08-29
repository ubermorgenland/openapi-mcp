package server

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ubermorgenland/openapi-mcp/pkg/mcp/mcp"
)

func TestMCPServer_ProtocolNegotiation(t *testing.T) {
	tests := []struct {
		name            string
		clientVersion   string
		expectedVersion string
	}{
		{
			name:            "Server supports client version - should respond with same version",
			clientVersion:   "2024-11-05",
			expectedVersion: "2024-11-05",
		},
		{
			name:            "Client requests current latest - should respond with same version",
			clientVersion:   mcp.LATEST_PROTOCOL_VERSION,
			expectedVersion: mcp.LATEST_PROTOCOL_VERSION,
		},
		{
			name:            "Client requests unsupported future version - should respond with server's latest",
			clientVersion:   "2026-01-01",
			expectedVersion: mcp.LATEST_PROTOCOL_VERSION,
		},
		{
			name:            "Client requests unsupported old version - should respond with server's latest",
			clientVersion:   "2023-01-01",
			expectedVersion: mcp.LATEST_PROTOCOL_VERSION,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewMCPServer("test-server", "1.0.0")

			params := struct {
				ProtocolVersion string                 `json:"protocolVersion"`
				ClientInfo      mcp.Implementation     `json:"clientInfo"`
				Capabilities    mcp.ClientCapabilities `json:"capabilities"`
			}{
				ProtocolVersion: tt.clientVersion,
				ClientInfo: mcp.Implementation{
					Name:    "test-client",
					Version: "1.0.0",
				},
			}

			// Create initialize request with specific protocol version
			initRequest := mcp.JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      mcp.NewRequestId(int64(1)),
				Request: mcp.Request{
					Method: "initialize",
				},
				Params: params,
			}

			messageBytes, err := json.Marshal(initRequest)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			response := server.HandleMessage(context.Background(), messageBytes)
			if response == nil {
				t.Fatalf("No response from server")
			}

			resp, ok := response.(mcp.JSONRPCResponse)
			if !ok {
				t.Fatalf("Response is not JSONRPCResponse: %T", response)
			}

			initResult, ok := resp.Result.(mcp.InitializeResult)
			if !ok {
				t.Fatalf("Result is not InitializeResult: %T", resp.Result)
			}

			if initResult.ProtocolVersion != tt.expectedVersion {
				t.Errorf("ProtocolVersion = %q, want %q", initResult.ProtocolVersion, tt.expectedVersion)
			}
		})
	}
}
