package openapi2mcp

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestSchemaBasic(t *testing.T) {
	// TODO: Add tests for schema parsing and validation
	t.Run("dummy", func(t *testing.T) {
		t.Log("basic schema test placeholder")
	})
}

func TestBuildInputSchema_Basic(t *testing.T) {
	params := openapi3.Parameters{
		&openapi3.ParameterRef{Value: &openapi3.Parameter{
			Name:     "foo",
			In:       "query",
			Required: true,
			Schema:   &openapi3.SchemaRef{Value: &openapi3.Schema{Type: typesPtr("string")}},
		}},
	}
	schema := BuildInputSchema(params, nil)
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["foo"]; !ok {
		t.Fatalf("expected property 'foo' in schema")
	}
	if req, ok := schema["required"].([]string); !ok || len(req) != 1 || req[0] != "foo" {
		t.Fatalf("expected 'foo' to be required, got: %v", schema["required"])
	}
}

func TestBuildInputSchema_Empty(t *testing.T) {
	schema := BuildInputSchema(nil, nil)
	if props, ok := schema["properties"].(map[string]any); !ok || len(props) != 0 {
		t.Fatalf("expected empty properties, got: %v", props)
	}
}

func TestBuildInputSchema_Malformed(t *testing.T) {
	params := openapi3.Parameters{
		&openapi3.ParameterRef{Value: nil}, // malformed
	}
	schema := BuildInputSchema(params, nil)
	if props, ok := schema["properties"].(map[string]any); !ok || len(props) != 0 {
		t.Fatalf("expected empty properties for malformed param, got: %v", props)
	}
}

func TestBuildInputSchema_RequiredFromBody(t *testing.T) {
	body := &openapi3.RequestBodyRef{Value: &openapi3.RequestBody{
		Required: true,
		Content: openapi3.Content{
			"application/json": &openapi3.MediaType{
				Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{
					Type: typesPtr("object"),
					Properties: map[string]*openapi3.SchemaRef{
						"bar": {Value: &openapi3.Schema{Type: typesPtr("integer")}},
					},
					Required: []string{"bar"},
				}},
			},
		},
	}}
	schema := BuildInputSchema(nil, body)
	props, _ := schema["properties"].(map[string]any)
	reqBody, ok := props["requestBody"].(map[string]any)
	if !ok {
		t.Fatalf("expected property 'requestBody' in schema")
	}
	reqBodyProps, _ := reqBody["properties"].(map[string]any)
	if _, ok := reqBodyProps["bar"]; !ok {
		t.Fatalf("expected property 'bar' in requestBody schema")
	}
	if req, ok := reqBody["required"].([]string); !ok || len(req) != 1 || req[0] != "bar" {
		t.Fatalf("expected 'bar' to be required in requestBody, got: %v", reqBody["required"])
	}
	if req, ok := schema["required"].([]string); !ok || len(req) != 1 || req[0] != "requestBody" {
		t.Fatalf("expected 'requestBody' to be required, got: %v", schema["required"])
	}
}
