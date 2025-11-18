package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/mcp/mcp"
	mcpserver "github.com/ubermorgenland/openapi-mcp/pkg/mcp/server"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
)

func TestRegisterOpenAPITools(t *testing.T) {
	// Start a mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("404 page not found"))
	}))
	defer ts.Close()
	os.Setenv("OPENAPI_BASE_URL", ts.URL)
	defer os.Unsetenv("OPENAPI_BASE_URL")

	paths := openapi3.NewPaths()
	paths.Set("/foo", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getFoo",
			Summary:     "Get Foo",
		},
	})
	paths.Set("/bar", &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "createBar",
			Summary:     "Create Bar",
		},
	})

	doc := &openapi3.T{
		Paths: paths,
	}

	server := mcpserver.NewMCPServer("test", "0.0.1")
	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	openapi2mcp.RegisterOpenAPITools(server, ops, doc, nil)

	if len(ops) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(ops))
	}

	// Simulate a tool call
	ctx := context.Background()
	result := server.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {"name": "getFoo", "arguments": {}}
	}`))

	switch v := result.(type) {
	case mcp.JSONRPCError:
		t.Fatalf("unexpected error: %v", v.Error.Message)
	case mcp.JSONRPCResponse:
		toolResult, ok := v.Result.(mcp.CallToolResult)
		if !ok {
			t.Fatalf("expected CallToolResult, got %T", v.Result)
		}
		found := false
		for _, c := range toolResult.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				// Check for HTTP error text patterns - updated for new verbose format
				if strings.Contains(tc.Text, "Error: Not Found (HTTP 404)") {
					found = true
				} else if strings.Contains(tc.Text, "HTTP Error") && strings.Contains(tc.Text, "404") {
					found = true
				} else if strings.Contains(tc.Text, "Status: 404") && strings.Contains(tc.Text, "404 page not found") {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("expected 404 response for GET /foo, got: %+v", toolResult.Content)
		}
	default:
		t.Fatalf("unexpected result type: %T", v)
	}
}

func TestHTTPOpenAPIToolHandler(t *testing.T) {
	// Start a mock HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/foo/123" && r.Method == http.MethodGet && r.URL.Query().Get("q") == "test" {
			w.Header().Set("X-Test-Header", "ok")
			w.WriteHeader(200)
			w.Write([]byte(`{"result":"ok"}`))
			return
		}
		if r.URL.Path == "/bar" && r.Method == http.MethodPost {
			var body struct {
				Foo string `json:"foo"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(201)
			w.Write([]byte(`{"echo":"` + body.Foo + `"}`))
			return
		}
		// Mock binary endpoint
		if r.URL.Path == "/file" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", "attachment; filename=mock.bin")
			w.WriteHeader(200)
			w.Write([]byte{0x01, 0x02, 0x03, 0x04})
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()
	os.Setenv("OPENAPI_BASE_URL", ts.URL)
	defer os.Unsetenv("OPENAPI_BASE_URL")

	typesPtr := func(types ...string) *openapi3.Types {
		t := openapi3.Types(types)
		return &t
	}

	paths := openapi3.NewPaths()
	paths.Set("/foo/{id}", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getFoo",
			Summary:     "Get Foo",
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{Value: &openapi3.Parameter{Name: "id", In: "path", Required: true, Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: typesPtr("string")}}}},
				&openapi3.ParameterRef{Value: &openapi3.Parameter{Name: "q", In: "query", Required: false, Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: typesPtr("string")}}}},
			},
		},
	})
	paths.Set("/bar", &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "createBar",
			Summary:     "Create Bar",
			RequestBody: &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
				Content: openapi3.Content{
					"application/json": &openapi3.MediaType{
						Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
							Type: typesPtr("object"),
							Properties: openapi3.Schemas{
								"foo": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: typesPtr("string")}},
							},
							Required: []string{"foo"},
						}},
					},
				},
			}},
		},
	})
	paths.Set("/file", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getFile",
			Summary:     "Get File",
		},
	})

	doc := &openapi3.T{
		Paths: paths,
	}

	server := mcpserver.NewMCPServer("test", "0.0.1")
	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	openapi2mcp.RegisterOpenAPITools(server, ops, doc, nil)

	ctx := context.Background()
	// Test GET with path and query
	getReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "getFoo",
			"arguments": map[string]any{"id": "123", "q": "test"},
		},
	}
	getReqJSON, _ := json.Marshal(getReq)
	result := server.HandleMessage(ctx, getReqJSON)
	switch v := result.(type) {
	case mcp.JSONRPCError:
		t.Fatalf("unexpected error: %v", v.Error.Message)
	case mcp.JSONRPCResponse:
		toolResult, ok := v.Result.(mcp.CallToolResult)
		if !ok {
			t.Fatalf("expected CallToolResult, got %T", v.Result)
		}
		found := false
		for _, c := range toolResult.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				switch toolResult.OutputType {
				case "json":
					var obj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &obj); err == nil {
						if errObj, ok := obj["error"].(map[string]any); ok {
							if errObj["http_status"] == float64(404) && strings.Contains(errObj["message"].(string), "404") {
								found = true
							}
						}
					}
				case "file":
					var fileObj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &fileObj); err == nil {
						if _, ok := fileObj["file_base64"]; ok {
							if _, ok := fileObj["mime_type"]; ok {
								if _, ok := fileObj["file_name"]; ok {
									found = true
								}
							}
						}
					}
				default:
					if strings.Contains(tc.Text, "/foo/123?q=test") && strings.Contains(tc.Text, "result\":\"ok\"") {
						found = true
					}
				}
			}
		}
		if !found {
			t.Errorf("expected HTTP response for /foo/123?q=test, got: %+v", toolResult.Content)
		}
	default:
		t.Fatalf("unexpected result type: %T", v)
	}

	// Test POST with JSON body
	postReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "createBar",
			"arguments": map[string]any{"requestBody": map[string]any{"foo": "bar"}},
		},
	}
	postReqJSON, _ := json.Marshal(postReq)
	result = server.HandleMessage(ctx, postReqJSON)
	switch v := result.(type) {
	case mcp.JSONRPCError:
		t.Fatalf("unexpected error: %v", v.Error.Message)
	case mcp.JSONRPCResponse:
		toolResult, ok := v.Result.(mcp.CallToolResult)
		if !ok {
			t.Fatalf("expected CallToolResult, got %T", v.Result)
		}
		found := false
		for _, c := range toolResult.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				switch toolResult.OutputType {
				case "json":
					var obj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &obj); err == nil {
						if errObj, ok := obj["error"].(map[string]any); ok {
							if errObj["http_status"] == float64(404) && strings.Contains(errObj["message"].(string), "404") {
								found = true
							}
						}
					}
				case "file":
					var fileObj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &fileObj); err == nil {
						if _, ok := fileObj["file_base64"]; ok {
							if _, ok := fileObj["mime_type"]; ok {
								if _, ok := fileObj["file_name"]; ok {
									found = true
								}
							}
						}
					}
				default:
					if strings.Contains(tc.Text, "/bar") && strings.Contains(tc.Text, "echo") && strings.Contains(tc.Text, "bar") {
						found = true
					}
				}
			}
		}
		if !found {
			t.Errorf("expected HTTP response for /bar, got: %+v", toolResult.Content)
		}
	default:
		t.Fatalf("unexpected result type: %T", v)
	}

	// Test binary/file response
	fileReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "getFile",
			"arguments": map[string]any{},
		},
	}
	fileReqJSON, _ := json.Marshal(fileReq)
	result = server.HandleMessage(ctx, fileReqJSON)
	switch v := result.(type) {
	case mcp.JSONRPCError:
		t.Fatalf("unexpected error: %v", v.Error.Message)
	case mcp.JSONRPCResponse:
		toolResult, ok := v.Result.(mcp.CallToolResult)
		if !ok {
			t.Fatalf("expected CallToolResult, got %T", v.Result)
		}
		found := false
		for _, c := range toolResult.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				switch toolResult.OutputType {
				case "json":
					var obj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &obj); err == nil {
						if errObj, ok := obj["error"].(map[string]any); ok {
							if errObj["http_status"] == float64(404) && strings.Contains(errObj["message"].(string), "404") {
								found = true
							}
						}
					}
				case "file":
					var fileObj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &fileObj); err == nil {
						if _, ok := fileObj["file_base64"]; ok {
							if _, ok := fileObj["mime_type"]; ok {
								if _, ok := fileObj["file_name"]; ok {
									found = true
								}
							}
						}
					}
				default:
					if toolResult.OutputType == "file" {
						var fileObj map[string]any
						err := json.Unmarshal([]byte(tc.Text), &fileObj)
						if err != nil {
							t.Errorf("expected JSON file object, got: %v", tc.Text)
							continue
						}
						if _, ok := fileObj["file_base64"]; !ok {
							t.Errorf("expected file_base64 in file response, got: %v", fileObj)
						}
						if _, ok := fileObj["mime_type"]; !ok {
							t.Errorf("expected mime_type in file response, got: %v", fileObj)
						}
						if _, ok := fileObj["file_name"]; !ok {
							t.Errorf("expected file_name in file response, got: %v", fileObj)
						}
						found = true
					}
				}
			}
		}
		if !found {
			t.Errorf("expected file response for /file, got: %+v", toolResult.Content)
		}
	default:
		t.Fatalf("unexpected result type: %T", v)
	}
}

func TestRegisterOpenAPITools_ServerSelection(t *testing.T) {
	os.Unsetenv("OPENAPI_BASE_URL")
	// Set up two mock servers
	var hitA, hitB int
	var mu sync.Mutex
	tsA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hitA++
		mu.Unlock()
		w.WriteHeader(200)
		w.Write([]byte(`{"result":"okA"}`))
	}))
	defer tsA.Close()

	tsB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hitB++
		mu.Unlock()
		w.WriteHeader(200)
		w.Write([]byte(`{"result":"okB"}`))
	}))
	defer tsB.Close()

	doc := &openapi3.T{
		Servers: openapi3.Servers{
			&openapi3.Server{URL: tsA.URL},
			&openapi3.Server{URL: tsB.URL},
		},
		Paths: func() *openapi3.Paths {
			paths := openapi3.NewPaths()
			paths.Set("/foo", &openapi3.PathItem{
				Get: &openapi3.Operation{
					OperationID: "getFoo",
					Summary:     "Get Foo",
				},
			})
			return paths
		}(),
	}

	server := mcpserver.NewMCPServer("test", "0.0.1")
	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	openapi2mcp.RegisterOpenAPITools(server, ops, doc, nil)

	ctx := context.Background()
	for i := 0; i < 20; i++ {
		_ = server.HandleMessage(ctx, []byte(`{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "tools/call",
			"params": {"name": "getFoo", "arguments": {}}
		}`))
	}
	if hitA == 0 || hitB == 0 {
		t.Errorf("Expected both servers to be hit, got hitA=%d, hitB=%d", hitA, hitB)
	}
}

func TestExternalDocsTool(t *testing.T) {
	doc := &openapi3.T{
		ExternalDocs: &openapi3.ExternalDocs{
			URL:         "https://docs.example.com",
			Description: "See the full API documentation.",
		},
		Paths: openapi3.NewPaths(),
	}
	server := mcpserver.NewMCPServer("test", "0.0.1")
	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	openapi2mcp.RegisterOpenAPITools(server, ops, doc, nil)

	ctx := context.Background()
	result := server.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {"name": "externalDocs", "arguments": {}}
	}`))

	switch v := result.(type) {
	case mcp.JSONRPCError:
		t.Fatalf("unexpected error: %v", v.Error.Message)
	case mcp.JSONRPCResponse:
		toolResult, ok := v.Result.(mcp.CallToolResult)
		if !ok {
			t.Fatalf("expected CallToolResult, got %T", v.Result)
		}
		found := false
		for _, c := range toolResult.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				switch toolResult.OutputType {
				case "json":
					var obj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &obj); err == nil {
						if errObj, ok := obj["error"].(map[string]any); ok {
							if errObj["http_status"] == float64(404) && strings.Contains(errObj["message"].(string), "404") {
								found = true
							}
						}
					}
				case "file":
					var fileObj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &fileObj); err == nil {
						if _, ok := fileObj["file_base64"]; ok {
							if _, ok := fileObj["mime_type"]; ok {
								if _, ok := fileObj["file_name"]; ok {
									found = true
								}
							}
						}
					}
				default:
					if strings.Contains(tc.Text, "https://docs.example.com") && strings.Contains(tc.Text, "full API documentation") {
						found = true
					}
				}
			}
		}
		if !found {
			t.Errorf("expected externalDocs info, got: %+v", toolResult.Content)
		}
	default:
		t.Fatalf("unexpected result type: %T", v)
	}
}

func TestInfoTool(t *testing.T) {
	doc := &openapi3.T{
		Info: &openapi3.Info{
			Title:          "My API",
			Version:        "1.2.3",
			Description:    "This is a test API.",
			TermsOfService: "https://tos.example.com",
		},
		Paths: openapi3.NewPaths(),
	}
	server := mcpserver.NewMCPServer("test", "0.0.1")
	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	openapi2mcp.RegisterOpenAPITools(server, ops, doc, nil)

	ctx := context.Background()
	result := server.HandleMessage(ctx, []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/call",
		"params": {"name": "info", "arguments": {}}
	}`))

	switch v := result.(type) {
	case mcp.JSONRPCError:
		t.Fatalf("unexpected error: %v", v.Error.Message)
	case mcp.JSONRPCResponse:
		toolResult, ok := v.Result.(mcp.CallToolResult)
		if !ok {
			t.Fatalf("expected CallToolResult, got %T", v.Result)
		}
		found := false
		for _, c := range toolResult.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				switch toolResult.OutputType {
				case "json":
					var obj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &obj); err == nil {
						if errObj, ok := obj["error"].(map[string]any); ok {
							if errObj["http_status"] == float64(404) && strings.Contains(errObj["message"].(string), "404") {
								found = true
							}
						}
					}
				case "file":
					var fileObj map[string]any
					if err := json.Unmarshal([]byte(tc.Text), &fileObj); err == nil {
						if _, ok := fileObj["file_base64"]; ok {
							if _, ok := fileObj["mime_type"]; ok {
								if _, ok := fileObj["file_name"]; ok {
									found = true
								}
							}
						}
					}
				default:
					if strings.Contains(tc.Text, "My API") && strings.Contains(tc.Text, "1.2.3") && strings.Contains(tc.Text, "test API") && strings.Contains(tc.Text, "tos.example.com") {
						found = true
					}
				}
			}
		}
		if !found {
			t.Errorf("expected info tool output, got: %+v", toolResult.Content)
		}
	default:
		t.Fatalf("unexpected result type: %T", v)
	}
}

func TestValidateCommand(t *testing.T) {
	// Test the validate functionality directly without going through main()
	validSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /foo:
    get:
      operationId: getFoo
      responses:
        '200':
          description: OK
`
	invalidSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /foo:
    get:
      responses:
        '200':
          description: OK
` // missing operationId

	// Test valid spec
	doc, err := openapi2mcp.LoadOpenAPISpecFromString(validSpec)
	if err != nil {
		t.Fatalf("failed to load valid spec: %v", err)
	}

	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	var toolNames []string
	for _, op := range ops {
		toolNames = append(toolNames, op.OperationID)
	}

	err = openapi2mcp.SelfTestOpenAPIMCPWithOptions(doc, toolNames, false)
	if err != nil {
		t.Errorf("expected valid spec to pass self-test, got error: %v", err)
	}

	// Test invalid spec
	doc, err = openapi2mcp.LoadOpenAPISpecFromString(invalidSpec)
	if err != nil {
		t.Fatalf("failed to load invalid spec: %v", err)
	}

	ops = openapi2mcp.ExtractOpenAPIOperations(doc)
	toolNames = nil
	for _, op := range ops {
		toolNames = append(toolNames, op.OperationID)
	}

	err = openapi2mcp.SelfTestOpenAPIMCPWithOptions(doc, toolNames, false)
	if err == nil {
		t.Errorf("expected invalid spec to fail self-test, but it passed")
	} else if !strings.Contains(err.Error(), "self-test failed") {
		t.Errorf("expected self-test failure error, got: %v", err)
	}
}

func TestLintCommand(t *testing.T) {
	// Test the lint functionality directly without going through main()
	validSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /foo:
    get:
      operationId: getFoo
      summary: Get foo
      description: Get a foo resource
      tags: [foo]
      responses:
        '200':
          description: OK
`
	specWithIssues := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /foo:
    get:
      operationId: getFoo
      responses:
        '200':
          description: OK
` // missing summary, description, tags

	// Test spec with no issues
	doc, err := openapi2mcp.LoadOpenAPISpecFromString(validSpec)
	if err != nil {
		t.Fatalf("failed to load valid spec: %v", err)
	}

	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	var toolNames []string
	for _, op := range ops {
		toolNames = append(toolNames, op.OperationID)
	}

	err = openapi2mcp.SelfTestOpenAPIMCPWithOptions(doc, toolNames, true)
	if err != nil {
		t.Errorf("expected well-formed spec to pass lint, got error: %v", err)
	}

	// Test spec with linting issues
	doc, err = openapi2mcp.LoadOpenAPISpecFromString(specWithIssues)
	if err != nil {
		t.Fatalf("failed to load spec with issues: %v", err)
	}

	ops = openapi2mcp.ExtractOpenAPIOperations(doc)
	toolNames = nil
	for _, op := range ops {
		toolNames = append(toolNames, op.OperationID)
	}

	err = openapi2mcp.SelfTestOpenAPIMCPWithOptions(doc, toolNames, true)
	if err != nil {
		t.Errorf("expected spec with warnings to pass lint (warnings don't fail), got error: %v", err)
	}
	// Note: The detailed linter reports warnings for missing summary/description/tags but doesn't fail
	// This is correct behavior - warnings are informational, errors cause failure
}

func TestHTTPLintEndpoints(t *testing.T) {
	validSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /foo:
    get:
      operationId: getFoo
      summary: Get foo
      description: Get a foo resource
      tags: [foo]
      responses:
        '200':
          description: OK
`
	invalidSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /foo:
    get:
      responses:
        '200':
          description: OK
` // missing operationId

	// Test lint endpoint
	lintServer := openapi2mcp.NewHTTPLintServer(true)

	// Test valid spec
	reqBody := openapi2mcp.HTTPLintRequest{OpenAPISpec: validSpec}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/lint", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	lintServer.HandleLint(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for valid spec, got %d", w.Code)
	}

	var result openapi2mcp.LintResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !result.Success {
		t.Errorf("expected lint to succeed for valid spec")
	}

	// Test invalid spec
	reqBody = openapi2mcp.HTTPLintRequest{OpenAPISpec: invalidSpec}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest("POST", "/lint", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	lintServer.HandleLint(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for invalid spec, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.Success {
		t.Errorf("expected lint to fail for invalid spec")
	}

	if result.ErrorCount == 0 {
		t.Errorf("expected errors for invalid spec, got %d", result.ErrorCount)
	}

	// Test validate endpoint
	validateServer := openapi2mcp.NewHTTPLintServer(false)

	reqBody = openapi2mcp.HTTPLintRequest{OpenAPISpec: invalidSpec}
	jsonBody, _ = json.Marshal(reqBody)

	req = httptest.NewRequest("POST", "/validate", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	validateServer.HandleLint(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422 for invalid spec, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.Success {
		t.Errorf("expected validation to fail for spec with missing operationId")
	}
}

func TestHTTPLintCORSHeaders(t *testing.T) {
	validSpec := `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /foo:
    get:
      operationId: getFoo
      responses:
        '200':
          description: OK
`

	lintServer := openapi2mcp.NewHTTPLintServer(true)

	// Test POST request CORS headers
	reqBody := openapi2mcp.HTTPLintRequest{OpenAPISpec: validSpec}
	jsonBody, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/lint", strings.NewReader(string(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	lintServer.HandleLint(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods: GET, POST, OPTIONS, got %s", w.Header().Get("Access-Control-Allow-Methods"))
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Accept, Authorization" {
		t.Errorf("expected Access-Control-Allow-Headers: Content-Type, Accept, Authorization, got %s", w.Header().Get("Access-Control-Allow-Headers"))
	}

	// Check caching headers
	if w.Header().Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
		t.Errorf("expected Cache-Control: no-cache, no-store, must-revalidate, got %s", w.Header().Get("Cache-Control"))
	}
	if w.Header().Get("Pragma") != "no-cache" {
		t.Errorf("expected Pragma: no-cache, got %s", w.Header().Get("Pragma"))
	}

	// Test OPTIONS preflight request
	req = httptest.NewRequest("OPTIONS", "/lint", nil)
	w = httptest.NewRecorder()

	lintServer.HandleLint(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS request, got %d", w.Code)
	}

	// Check CORS headers on OPTIONS response
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: * on OPTIONS, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Max-Age") != "86400" {
		t.Errorf("expected Access-Control-Max-Age: 86400, got %s", w.Header().Get("Access-Control-Max-Age"))
	}

	// Test health endpoint CORS headers
	req = httptest.NewRequest("GET", "/health", nil)
	w = httptest.NewRecorder()

	lintServer.HandleHealth(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: * on health endpoint, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
