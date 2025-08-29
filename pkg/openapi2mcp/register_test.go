package openapi2mcp

import (
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/mcp/server"
)

func stringPtr(s string) *string {
	return &s
}

func typesPtr(types ...string) *openapi3.Types {
	t := openapi3.Types(types)
	return &t
}

func minimalOpenAPIDoc() *openapi3.T {
	paths := openapi3.NewPaths()
	paths.Set("/foo", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getFoo",
			Summary:     "Get Foo",
			Parameters:  openapi3.Parameters{},
		},
	})

	return &openapi3.T{
		Info:  &openapi3.Info{Title: "Test API", Version: "1.0.0"},
		Paths: paths,
	}
}

func toolSetEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ma := map[string]struct{}{}
	mb := map[string]struct{}{}
	for _, x := range a {
		ma[x] = struct{}{}
	}
	for _, x := range b {
		mb[x] = struct{}{}
	}
	for k := range ma {
		if _, ok := mb[k]; !ok {
			return false
		}
	}
	return true
}

func TestRegisterOpenAPITools_Basic(t *testing.T) {
	doc := minimalOpenAPIDoc()
	srv := server.NewMCPServer("test", "1.0.0")
	ops := ExtractOpenAPIOperations(doc)
	opts := &ToolGenOptions{}
	names := RegisterOpenAPITools(srv, ops, doc, opts)
	expected := []string{"getFoo", "info", "describe"}
	if !toolSetEqual(names, expected) {
		t.Fatalf("expected tools %v, got: %v", expected, names)
	}
}

func TestRegisterOpenAPITools_TagFilter(t *testing.T) {
	doc := minimalOpenAPIDoc()
	pathItem := doc.Paths.Value("/foo")
	if pathItem != nil && pathItem.Get != nil {
		pathItem.Get.Tags = []string{"bar"}
	}
	srv := server.NewMCPServer("test", "1.0.0")
	ops := ExtractOpenAPIOperations(doc)
	opts := &ToolGenOptions{
		TagFilter: []string{"baz"}, // should filter out
	}
	names := RegisterOpenAPITools(srv, ops, doc, opts)
	expected := []string{"info", "describe"}
	if !toolSetEqual(names, expected) {
		t.Fatalf("expected only meta tools %v, got: %v", expected, names)
	}
}

func TestSelfTestOpenAPIMCP_Pass(t *testing.T) {
	doc := minimalOpenAPIDoc()
	srv := server.NewMCPServer("test", "1.0.0")
	ops := ExtractOpenAPIOperations(doc)
	opts := &ToolGenOptions{}
	RegisterOpenAPITools(srv, ops, doc, opts)
	toolNames := make([]string, 0)
	for _, tool := range srv.ListTools() {
		toolNames = append(toolNames, tool.Name)
	}
	err := SelfTestOpenAPIMCP(doc, toolNames)
	if err != nil {
		t.Fatalf("expected selftest to pass, got: %v", err)
	}
}

func TestSelfTestOpenAPIMCP_MissingTool(t *testing.T) {
	doc := minimalOpenAPIDoc()
	err := SelfTestOpenAPIMCP(doc, []string{})
	if err == nil {
		t.Fatalf("expected selftest to fail due to missing tool")
	}
}

func TestNumberVsIntegerTypes(t *testing.T) {
	// Create a spec with both number and integer types
	paths := openapi3.NewPaths()

	responses := openapi3.NewResponses()
	responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{Description: stringPtr("OK")},
	})

	paths.Set("/test", &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "testNumbers",
			Summary:     "Test number types",
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Required: true,
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: typesPtr("object"),
									Properties: openapi3.Schemas{
										"integerField": &openapi3.SchemaRef{
											Value: &openapi3.Schema{Type: typesPtr("integer")},
										},
										"numberField": &openapi3.SchemaRef{
											Value: &openapi3.Schema{Type: typesPtr("number")},
										},
									},
									Required: []string{"integerField", "numberField"},
								},
							},
						},
					},
				},
			},
			Responses: responses,
		},
	})

	doc := &openapi3.T{
		Info:  &openapi3.Info{Title: "Number Test API", Version: "1.0.0"},
		Paths: paths,
	}

	ops := ExtractOpenAPIOperations(doc)
	if len(ops) == 0 {
		t.Fatal("No operations extracted")
	}

	op := ops[0]
	if op.OperationID != "testNumbers" {
		t.Fatalf("Expected operation ID 'testNumbers', got '%s'", op.OperationID)
	}

	// Build the input schema and check that it handles number vs integer correctly
	inputSchema := BuildInputSchema(op.Parameters, op.RequestBody)

	// The schema should be valid and not cause any errors when processed
	if inputSchema == nil {
		t.Fatal("Input schema is nil")
	}

	// Verify that the schema contains the expected properties
	props, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema properties not found")
	}

	// Should have requestBody property
	requestBodyProp, ok := props["requestBody"].(map[string]any)
	if !ok {
		t.Fatal("requestBody property not found")
	}

	// Check that requestBody has the correct nested properties
	requestBodyProps, ok := requestBodyProp["properties"].(map[string]any)
	if !ok {
		t.Fatal("requestBody properties not found")
	}

	// Verify integerField has type integer
	if intField, ok := requestBodyProps["integerField"].(map[string]any); ok {
		if fieldType, ok := intField["type"].(string); !ok || fieldType != "integer" {
			t.Errorf("Expected integerField to have type 'integer', got '%v'", fieldType)
		}
	} else {
		t.Error("integerField not found in schema")
	}

	// Verify numberField has type number
	if numField, ok := requestBodyProps["numberField"].(map[string]any); ok {
		if fieldType, ok := numField["type"].(string); !ok || fieldType != "number" {
			t.Errorf("Expected numberField to have type 'number', got '%v'", fieldType)
		}
	} else {
		t.Error("numberField not found in schema")
	}
}

func TestFormatPreservation(t *testing.T) {
	// Create a spec with various format specifiers
	paths := openapi3.NewPaths()

	responses := openapi3.NewResponses()
	responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{Description: stringPtr("OK")},
	})

	paths.Set("/test", &openapi3.PathItem{
		Post: &openapi3.Operation{
			OperationID: "testFormats",
			Summary:     "Test format preservation",
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Required: true,
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: typesPtr("object"),
									Properties: openapi3.Schemas{
										"int32Field": &openapi3.SchemaRef{
											Value: &openapi3.Schema{Type: typesPtr("integer"), Format: "int32"},
										},
										"floatField": &openapi3.SchemaRef{
											Value: &openapi3.Schema{Type: typesPtr("number"), Format: "float"},
										},
										"dateField": &openapi3.SchemaRef{
											Value: &openapi3.Schema{Type: typesPtr("string"), Format: "date"},
										},
									},
								},
							},
						},
					},
				},
			},
			Responses: responses,
		},
	})

	doc := &openapi3.T{
		Info:  &openapi3.Info{Title: "Format Test API", Version: "1.0.0"},
		Paths: paths,
	}

	ops := ExtractOpenAPIOperations(doc)
	if len(ops) == 0 {
		t.Fatal("No operations extracted")
	}

	op := ops[0]
	inputSchema := BuildInputSchema(op.Parameters, op.RequestBody)

	// Navigate to request body properties
	props, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema properties not found")
	}

	requestBodyProp, ok := props["requestBody"].(map[string]any)
	if !ok {
		t.Fatal("requestBody property not found")
	}

	requestBodyProps, ok := requestBodyProp["properties"].(map[string]any)
	if !ok {
		t.Fatal("requestBody properties not found")
	}

	// Verify format preservation for int32Field
	if int32Field, ok := requestBodyProps["int32Field"].(map[string]any); ok {
		if format, ok := int32Field["format"].(string); !ok || format != "int32" {
			t.Errorf("Expected int32Field to have format 'int32', got '%v'", format)
		}
		if fieldType, ok := int32Field["type"].(string); !ok || fieldType != "integer" {
			t.Errorf("Expected int32Field to have type 'integer', got '%v'", fieldType)
		}
	} else {
		t.Error("int32Field not found in schema")
	}

	// Verify format preservation for floatField
	if floatField, ok := requestBodyProps["floatField"].(map[string]any); ok {
		if format, ok := floatField["format"].(string); !ok || format != "float" {
			t.Errorf("Expected floatField to have format 'float', got '%v'", format)
		}
		if fieldType, ok := floatField["type"].(string); !ok || fieldType != "number" {
			t.Errorf("Expected floatField to have type 'number', got '%v'", fieldType)
		}
	} else {
		t.Error("floatField not found in schema")
	}

	// Verify format preservation for dateField
	if dateField, ok := requestBodyProps["dateField"].(map[string]any); ok {
		if format, ok := dateField["format"].(string); !ok || format != "date" {
			t.Errorf("Expected dateField to have format 'date', got '%v'", format)
		}
		if fieldType, ok := dateField["type"].(string); !ok || fieldType != "string" {
			t.Errorf("Expected dateField to have type 'string', got '%v'", fieldType)
		}
	} else {
		t.Error("dateField not found in schema")
	}
}

func TestComprehensiveValidation(t *testing.T) {
	// Create a spec with multiple validation issues to test comprehensive collection
	spec := `openapi: 3.0.0
info:
  title: Multi-Issue Test API
  version: 1.0.0
paths:
  /test:
    get:
      # Missing operationId (error)
      # Missing summary, description, tags (warnings)
      parameters:
        - name: param1
          in: query
          required: true
          schema:
            type: string
            # Missing enum, default, example (warnings)
        - name: param2
          in: query
          required: true
          schema:
            type: integer
            # Missing enum, default, example (warnings)
      responses:
        '200':
          description: OK`

	doc, err := LoadOpenAPISpecFromString(spec)
	if err != nil {
		t.Fatalf("Failed to parse spec: %v", err)
	}

	// Test comprehensive linting - should collect all issues, not stop at first
	result := LintOpenAPISpec(doc, true)

	// Should have at least 1 error (missing operationId) and multiple warnings
	if result.ErrorCount < 1 {
		t.Errorf("Expected at least 1 error, got %d", result.ErrorCount)
	}

	if result.WarningCount < 5 {
		t.Errorf("Expected at least 5 warnings, got %d", result.WarningCount)
	}

	if len(result.Issues) < 6 {
		t.Errorf("Expected comprehensive collection of issues, got %d total issues", len(result.Issues))
	}

	// Should not succeed due to errors
	if result.Success {
		t.Error("Expected validation to fail due to errors")
	}

	// Verify we have issues for different types of problems
	hasOperationIdError := false
	hasSummaryWarning := false
	hasParameterWarning := false

	for _, issue := range result.Issues {
		if issue.Type == "error" && strings.Contains(issue.Message, "operationId") {
			hasOperationIdError = true
		}
		if issue.Type == "warning" && strings.Contains(issue.Message, "summary") {
			hasSummaryWarning = true
		}
		if issue.Type == "warning" && strings.Contains(issue.Message, "enum") {
			hasParameterWarning = true
		}
	}

	if !hasOperationIdError {
		t.Error("Expected to find operationId error")
	}
	if !hasSummaryWarning {
		t.Error("Expected to find summary warning")
	}
	if !hasParameterWarning {
		t.Error("Expected to find parameter warnings")
	}
}

func TestEscapeParameterName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal_param", "normal_param"},
		{"filter[created_at]", "filter_created_at_"},
		{"page[number]", "page_number_"},
		{"filter[user][name]", "filter_user__name_"},
		{"already_escaped_", "already_escaped_"},
		{"param[with][multiple][brackets]", "param_with__multiple__brackets_"},
		{"", ""},
	}

	for _, test := range tests {
		result := escapeParameterName(test.input)
		if result != test.expected {
			t.Errorf("escapeParameterName(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestBracketParameterHandling(t *testing.T) {
	// Create a spec with bracket parameters like filter[created_at]
	paths := openapi3.NewPaths()

	responses := openapi3.NewResponses()
	responses.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{Description: stringPtr("OK")},
	})

	paths.Set("/events", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "listEvents",
			Summary:     "List Events",
			Parameters: openapi3.Parameters{
				&openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        "filter[created_at]",
						In:          "query",
						Required:    false,
						Description: "Filter by creation date",
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{Type: typesPtr("string")},
						},
					},
				},
				&openapi3.ParameterRef{
					Value: &openapi3.Parameter{
						Name:        "page[number]",
						In:          "query",
						Required:    false,
						Description: "Page number",
						Schema: &openapi3.SchemaRef{
							Value: &openapi3.Schema{Type: typesPtr("integer")},
						},
					},
				},
			},
			Responses: responses,
		},
	})

	doc := &openapi3.T{
		Info:  &openapi3.Info{Title: "Bracket Test API", Version: "1.0.0"},
		Paths: paths,
	}

	ops := ExtractOpenAPIOperations(doc)
	if len(ops) == 0 {
		t.Fatal("No operations extracted")
	}

	op := ops[0]
	if op.OperationID != "listEvents" {
		t.Fatalf("Expected operation ID 'listEvents', got '%s'", op.OperationID)
	}

	// Build the input schema and verify bracket parameters are escaped
	inputSchema := BuildInputSchema(op.Parameters, op.RequestBody)

	// The schema should be valid and not cause any errors when processed
	if inputSchema == nil {
		t.Fatal("Input schema is nil")
	}

	// Verify that the schema contains the escaped parameter names
	props, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema properties not found")
	}

	// Check that bracket parameters are properly escaped
	if _, ok := props["filter_created_at_"]; !ok {
		t.Error("Expected escaped parameter 'filter_created_at_' not found in schema")
	}

	if _, ok := props["page_number_"]; !ok {
		t.Error("Expected escaped parameter 'page_number_' not found in schema")
	}

	// Verify original bracket names are NOT in the schema
	if _, ok := props["filter[created_at]"]; ok {
		t.Error("Original bracket parameter 'filter[created_at]' should not be in schema")
	}

	if _, ok := props["page[number]"]; ok {
		t.Error("Original bracket parameter 'page[number]' should not be in schema")
	}
}

func TestParameterNameMapping(t *testing.T) {
	// Create parameters with brackets
	params := openapi3.Parameters{
		&openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name: "filter[created_at]",
				In:   "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: typesPtr("string")},
				},
			},
		},
		&openapi3.ParameterRef{
			Value: &openapi3.Parameter{
				Name: "normal_param",
				In:   "query",
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: typesPtr("string")},
				},
			},
		},
	}

	mapping := buildParameterNameMapping(params)

	// Should contain mapping for bracket parameter
	if original, exists := mapping["filter_created_at_"]; !exists || original != "filter[created_at]" {
		t.Errorf("Expected mapping 'filter_created_at_' -> 'filter[created_at]', got: %v", mapping)
	}

	// Should NOT contain mapping for normal parameter
	if _, exists := mapping["normal_param"]; exists {
		t.Error("Normal parameter should not be in mapping")
	}
}

func TestGetParameterValue(t *testing.T) {
	mapping := map[string]string{
		"filter_created_at_": "filter[created_at]",
	}

	// Test with escaped parameter name in args
	args := map[string]any{
		"filter_created_at_": "2024-01-01",
		"normal_param":       "value",
	}

	// Should find value using escaped name
	val, ok := getParameterValue(args, "filter[created_at]", mapping)
	if !ok || val != "2024-01-01" {
		t.Errorf("Expected to find value '2024-01-01', got: %v (found: %v)", val, ok)
	}

	// Should find normal parameter
	val, ok = getParameterValue(args, "normal_param", mapping)
	if !ok || val != "value" {
		t.Errorf("Expected to find value 'value', got: %v (found: %v)", val, ok)
	}

	// Should not find non-existent parameter
	val, ok = getParameterValue(args, "non_existent", mapping)
	if ok {
		t.Errorf("Expected to not find non-existent parameter, but found: %v", val)
	}
}
