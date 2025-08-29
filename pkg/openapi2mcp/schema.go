// schema.go
package openapi2mcp

import (
	"fmt"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// escapeParameterName converts parameter names with brackets to MCP-compatible names.
// For example: "filter[created_at]" becomes "filter_created_at_"
// The trailing underscore distinguishes escaped names from naturally occurring names.
func escapeParameterName(name string) string {
	if !strings.Contains(name, "[") && !strings.Contains(name, "]") {
		return name // No escaping needed
	}

	// Replace brackets with underscores and add trailing underscore
	escaped := strings.ReplaceAll(name, "[", "_")
	escaped = strings.ReplaceAll(escaped, "]", "_")

	// Add trailing underscore if not already present to mark as escaped
	if !strings.HasSuffix(escaped, "_") {
		escaped += "_"
	}

	return escaped
}

// isMessageArrayPattern checks if the oneOf schema represents a common message array pattern
// used in chat APIs where messages can be system, user, or assistant messages
func isMessageArrayPattern(oneOf []*openapi3.SchemaRef) bool {
	if len(oneOf) < 2 {
		return false
	}

	// Check if we have SystemMessage and UserMessage patterns
	hasSystemMessage := false
	hasUserMessage := false

	for _, schemaRef := range oneOf {
		if schemaRef == nil {
			continue
		}

		// First check the schema reference name
		if schemaRef.Ref != "" {
			refName := schemaRef.Ref
			if strings.Contains(refName, "SystemMessage") || strings.Contains(refName, "system") {
				hasSystemMessage = true
			}
			if strings.Contains(refName, "UserMessage") || strings.Contains(refName, "user") {
				hasUserMessage = true
			}
		}

		// Then check the actual schema value if available
		if schemaRef.Value != nil {
			// Check if this schema has a role property with system or user enum
			if schemaRef.Value.Properties != nil {
				if roleProp, exists := schemaRef.Value.Properties["role"]; exists && roleProp.Value != nil {
					if roleProp.Value.Enum != nil {
						for _, enumVal := range roleProp.Value.Enum {
							if str, ok := enumVal.(string); ok {
								if str == "system" {
									hasSystemMessage = true
								} else if str == "user" {
									hasUserMessage = true
								}
							}
						}
					}
				}
			}
		}
	}

	// If we found the pattern by reference names, that's sufficient
	if hasSystemMessage && hasUserMessage {
		return true
	}

	// Additional check: if we have exactly 2 schemas and they both have role properties,
	// it's likely a message pattern even if we can't detect the specific roles
	if len(oneOf) == 2 {
		bothHaveRole := true
		for _, schemaRef := range oneOf {
			if schemaRef == nil || schemaRef.Value == nil {
				bothHaveRole = false
				break
			}
			if schemaRef.Value.Properties == nil {
				bothHaveRole = false
				break
			}
			if _, exists := schemaRef.Value.Properties["role"]; !exists {
				bothHaveRole = false
				break
			}
		}
		if bothHaveRole {
			return true
		}
	}

	return false
}

// resolveSchemaRef resolves a schema reference to get the actual schema
func resolveSchemaRef(schemaRef *openapi3.SchemaRef, doc *openapi3.T) *openapi3.Schema {
	if schemaRef == nil {
		return nil
	}

	// If it's a reference, try to resolve it from the document
	if schemaRef.Ref != "" && doc != nil && doc.Components != nil && doc.Components.Schemas != nil {
		// Extract the schema name from the reference
		refPath := strings.TrimPrefix(schemaRef.Ref, "#/components/schemas/")
		if resolvedRef, exists := doc.Components.Schemas[refPath]; exists && resolvedRef.Value != nil {
			return resolvedRef.Value
		}
	}

	// Return the value if available
	return schemaRef.Value
}

// mergeOneOfSchemas creates a unified schema that accepts any of the oneOf variants
// This provides better MCP compatibility by creating a single schema with all possible properties
func mergeOneOfSchemas(oneOf []*openapi3.SchemaRef, doc *openapi3.T) map[string]any {
	merged := map[string]any{
		"type": "object",
	}

	allProperties := make(map[string]map[string]any)
	var allRequired []string
	requiredCount := make(map[string]int)
	totalSchemas := 0

	// Process each schema in oneOf
	for _, schemaRef := range oneOf {
		schema := resolveSchemaRef(schemaRef, doc)
		if schema == nil {
			continue
		}

		totalSchemas++

		// Extract properties from this schema
		if schema.Properties != nil {
			for propName, propSchemaRef := range schema.Properties {
				if propSchema := extractPropertyWithContext(propSchemaRef, doc); propSchema != nil {
					allProperties[propName] = propSchema
				}
			}
		}

		// Track required fields
		for _, req := range schema.Required {
			requiredCount[req]++
		}
	}

	// Set properties in merged schema
	if len(allProperties) > 0 {
		merged["properties"] = allProperties
	}

	// A field is required only if it's required in ALL schemas
	for field, count := range requiredCount {
		if count == totalSchemas {
			allRequired = append(allRequired, field)
		}
	}

	if len(allRequired) > 0 {
		merged["required"] = allRequired
	}

	// Add a description explaining this is a oneOf merge
	merged["description"] = fmt.Sprintf("Accepts any of %d possible schema variants (oneOf)", totalSchemas)

	return merged
}

// unescapeParameterName converts escaped parameter names back to their original form.
// This maintains a mapping from escaped names to original names for parameter lookup.
func unescapeParameterName(escaped string, originalNames map[string]string) string {
	if original, exists := originalNames[escaped]; exists {
		return original
	}
	return escaped // Return as-is if not found in mapping
}

// buildParameterNameMapping creates a mapping from escaped parameter names to original names.
// This is used to reverse the escaping when looking up parameter values.
func buildParameterNameMapping(params openapi3.Parameters) map[string]string {
	mapping := make(map[string]string)
	for _, paramRef := range params {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}
		p := paramRef.Value
		escaped := escapeParameterName(p.Name)
		if escaped != p.Name {
			mapping[escaped] = p.Name
		}
	}
	return mapping
}

// extractProperty recursively extracts a property schema from an OpenAPI SchemaRef.
// Handles allOf, oneOf, anyOf, discriminator, default, example, and basic OpenAPI 3.1 features.
func extractProperty(s *openapi3.SchemaRef) map[string]any {
	return extractPropertyWithContext(s, nil)
}

// extractPropertyWithContext recursively extracts a property schema from an OpenAPI SchemaRef with document context.
// Handles allOf, oneOf, anyOf, discriminator, default, example, and basic OpenAPI 3.1 features.
func extractPropertyWithContext(s *openapi3.SchemaRef, doc *openapi3.T) map[string]any {
	if s == nil || s.Value == nil {
		return nil
	}

	val := s.Value
	prop := map[string]any{}
	// Handle allOf (merge all subschemas)
	if len(val.AllOf) > 0 {
		merged := map[string]any{}
		for _, sub := range val.AllOf {
			subProp := extractPropertyWithContext(sub, doc)
			for k, v := range subProp {
				merged[k] = v
			}
		}
		for k, v := range merged {
			prop[k] = v
		}
	}
	// Handle oneOf with full support including schema reference resolution
	if len(val.OneOf) > 0 {
		// Check if this is a message array pattern (common in chat APIs)
		if isMessageArrayPattern(val.OneOf) {
			// Create a union type that accepts any of the message types
			unionSchema := map[string]any{
				"type": "object",
				"properties": map[string]any{
					"role": map[string]any{
						"type": "string",
						"enum": []string{"system", "user", "assistant"},
					},
					"content": map[string]any{
						"type": "string",
					},
				},
				"required": []string{"role", "content"},
			}
			return unionSchema
		} else {
			// Use enhanced oneOf handling that merges schemas for better MCP compatibility
			return mergeOneOfSchemas(val.OneOf, doc)
		}
	}
	if len(val.AnyOf) > 0 {
		fmt.Fprintf(os.Stderr, "[WARN] anyOf used in schema at %p. Only basic support is provided.\n", val)
		anyOf := []any{}
		for _, sub := range val.AnyOf {
			anyOf = append(anyOf, extractPropertyWithContext(sub, doc))
		}
		prop["anyOf"] = anyOf
	}
	// Handle discriminator (OpenAPI 3.0/3.1)
	if val.Discriminator != nil {
		fmt.Fprintf(os.Stderr, "[WARN] discriminator used in schema at %p. Only basic support is provided.\n", val)
		prop["discriminator"] = val.Discriminator
	}
	// Type, format, description, enum, default, example
	if val.Type != nil && len(*val.Type) > 0 {
		// Use the first type if multiple types are specified
		prop["type"] = (*val.Type)[0]
	}
	if val.Format != "" {
		prop["format"] = val.Format
	}
	if val.Description != "" {
		prop["description"] = val.Description
	}
	if len(val.Enum) > 0 {
		prop["enum"] = val.Enum
	}
	if val.Default != nil {
		prop["default"] = val.Default
	}
	if val.Example != nil {
		prop["example"] = val.Example
	}
	// Object properties
	if val.Type != nil && val.Type.Is("object") && val.Properties != nil {
		objProps := map[string]any{}
		for name, sub := range val.Properties {
			objProps[name] = extractPropertyWithContext(sub, doc)
		}
		prop["properties"] = objProps
		if len(val.Required) > 0 {
			prop["required"] = val.Required
		}
	}
	// Array items
	if val.Type != nil && val.Type.Is("array") && val.Items != nil {
		prop["items"] = extractPropertyWithContext(val.Items, doc)
	}
	return prop
}

// BuildInputSchema converts OpenAPI parameters and request body schema to a single JSON Schema object for MCP tool input validation.
// Returns a JSON Schema as a map[string]any.
// Example usage for BuildInputSchema:
//
//	params := ... // openapi3.Parameters from an operation
//	reqBody := ... // *openapi3.RequestBodyRef from an operation
//	schema := openapi2mcp.BuildInputSchema(params, reqBody)
//	// schema is a map[string]any representing the JSON schema for tool input
func BuildInputSchema(params openapi3.Parameters, requestBody *openapi3.RequestBodyRef) map[string]any {
	return BuildInputSchemaWithContext(params, requestBody, nil)
}

// BuildInputSchemaWithContext converts OpenAPI parameters and request body schema to a single JSON Schema object for MCP tool input validation with document context.
// Returns a JSON Schema as a map[string]any.
func BuildInputSchemaWithContext(params openapi3.Parameters, requestBody *openapi3.RequestBodyRef, doc *openapi3.T) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	properties := schema["properties"].(map[string]any)
	var required []string

	// Parameters (query, path, header, cookie)
	for _, paramRef := range params {
		if paramRef == nil || paramRef.Value == nil {
			continue
		}
		p := paramRef.Value
		if p.Schema != nil && p.Schema.Value != nil {
			if p.Schema.Value.Type != nil && p.Schema.Value.Type.Is("string") && p.Schema.Value.Format == "binary" {
				fmt.Fprintf(os.Stderr, "[WARN] Parameter '%s' uses 'string' with 'binary' format. Non-JSON body types are not fully supported.\n", p.Name)
			}
			prop := extractPropertyWithContext(p.Schema, doc)
			if p.Description != "" {
				prop["description"] = p.Description
			}
			// Use escaped parameter name for MCP schema compatibility
			escapedName := escapeParameterName(p.Name)
			properties[escapedName] = prop
			if p.Required {
				required = append(required, escapedName)
			}
		}
		// Warn about unsupported parameter locations
		if p.In != "query" && p.In != "path" && p.In != "header" && p.In != "cookie" {
			fmt.Fprintf(os.Stderr, "[WARN] Parameter '%s' uses unsupported location '%s'.\n", p.Name, p.In)
		}
	}

	// Request body (application/json and application/vnd.api+json)
	if requestBody != nil && requestBody.Value != nil {
		for mtName := range requestBody.Value.Content {
			// Check base content type without parameters
			baseMT := mtName
			if idx := strings.IndexByte(mtName, ';'); idx > 0 {
				baseMT = strings.TrimSpace(mtName[:idx])
			}
			if baseMT != "application/json" && baseMT != "application/vnd.api+json" {
				fmt.Fprintf(os.Stderr, "[WARN] Request body uses media type '%s'. Only 'application/json' and 'application/vnd.api+json' are fully supported.\n", mtName)
			}
		}
		// Try application/json first, then application/vnd.api+json (including with parameters)
		mt := getContentByType(requestBody.Value.Content, "application/json")
		if mt == nil {
			mt = getContentByType(requestBody.Value.Content, "application/vnd.api+json")
		}
		if mt != nil && mt.Schema != nil && mt.Schema.Value != nil {
			bodyProp := extractPropertyWithContext(mt.Schema, doc)
			bodyProp["description"] = "The JSON request body."
			properties["requestBody"] = bodyProp
			if requestBody.Value.Required {
				required = append(required, "requestBody")
			}
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
