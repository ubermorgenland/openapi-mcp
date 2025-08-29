// selftest.go
package openapi2mcp

import (
	"fmt"
	"os"
	"regexp"

	"github.com/getkin/kin-openapi/openapi3"
)

// SelfTestOpenAPIMCP checks that the generated MCP server matches the OpenAPI contract (basic: all required tools and arguments are present).
// Returns an error if any required tools or arguments are missing.
func SelfTestOpenAPIMCP(doc *openapi3.T, toolNames []string) error {
	ops := ExtractOpenAPIOperations(doc)
	failures := 0
	warnings := 0
	toolMap := map[string]struct{}{}
	for _, name := range toolNames {
		toolMap[name] = struct{}{}
	}
	recommendedTypes := map[string]bool{"string": true, "integer": true, "boolean": true, "number": true, "array": true, "object": true}
	recommendedLocations := map[string]bool{"path": true, "query": true, "header": true, "cookie": true}

	// Check for missing operationIds in the original spec
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation.OperationID == "" {
				fmt.Fprintf(os.Stderr, "[ERROR] Operation for path '%s' and method '%s' is missing an operationId.\n", path, method)
				fmt.Fprintf(os.Stderr, "  Suggestion: Add an 'operationId' field, e.g.\n    %s:\n      %s:\n        operationId: <uniqueOperationId>\n", path, method)
				failures++
			}
		}
	}

	for _, op := range ops {
		if _, ok := toolMap[op.OperationID]; !ok && op.OperationID != "" {
			fmt.Fprintf(os.Stderr, "[ERROR] Tool '%s' (operationId) is missing from MCP server.\n", op.OperationID)
			fmt.Fprintf(os.Stderr, "  Suggestion: Ensure the operationId '%s' is unique and present in the OpenAPI spec.\n", op.OperationID)
			failures++
		}
		if op.Summary == "" {
			fmt.Fprintf(os.Stderr, "[WARN] Operation '%s' (path: '%s', method: '%s') is missing a summary.\n", op.OperationID, op.Path, op.Method)
			fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'summary' field to describe the operation's purpose.\n")
			warnings++
		}
		if op.Description == "" {
			fmt.Fprintf(os.Stderr, "[WARN] Operation '%s' (path: '%s', method: '%s') is missing a description.\n", op.OperationID, op.Path, op.Method)
			fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'description' field for more detail.\n")
			warnings++
		}
		if len(op.Tags) == 0 {
			fmt.Fprintf(os.Stderr, "[WARN] Operation '%s' (path: '%s', method: '%s') has no tags.\n", op.OperationID, op.Path, op.Method)
			fmt.Fprintf(os.Stderr, "  Suggestion: Add tags to group related operations.\n")
			warnings++
		}
		// Parameter checks
		for _, paramRef := range op.Parameters {
			if paramRef == nil || paramRef.Value == nil {
				continue
			}
			p := paramRef.Value
			if p.Name == "" {
				fmt.Fprintf(os.Stderr, "[ERROR] Operation '%s' has a parameter with no name.\n", op.OperationID)
				fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'name' field to the parameter.\n")
				failures++
			}
			if !recommendedLocations[p.In] {
				fmt.Fprintf(os.Stderr, "[WARN] Parameter '%s' in operation '%s' uses non-standard location '%s'.\n", p.Name, op.OperationID, p.In)
				fmt.Fprintf(os.Stderr, "  Suggestion: Use one of: path, query, header, cookie.\n")
				warnings++
			}
			var schema *openapi3.Schema
			var typeStr string

			if p.Schema == nil || p.Schema.Value == nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Parameter '%s' in operation '%s' is missing a schema/type.\n", p.Name, op.OperationID)
				fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'schema' with a 'type', e.g.\n    - name: %s\n      in: %s\n      schema:\n        type: string\n", p.Name, p.In)
				failures++
				// Don't continue - we can still check other parameter properties
			} else {
				schema = p.Schema.Value
				if schema.Type != nil && len(*schema.Type) > 0 {
					typeStr = (*schema.Type)[0]
				} else {
					typeStr = ""
				}
				if typeStr == "" {
					fmt.Fprintf(os.Stderr, "[ERROR] Parameter '%s' in operation '%s' is missing a type in its schema.\n", p.Name, op.OperationID)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'type' to the schema, e.g. type: string\n")
					failures++
				} else if !recommendedTypes[typeStr] {
					fmt.Fprintf(os.Stderr, "[WARN] Parameter '%s' in operation '%s' uses uncommon type '%s'.\n", p.Name, op.OperationID, typeStr)
					fmt.Fprintf(os.Stderr, "  Suggestion: Use one of: string, integer, boolean, number, array, object.\n")
					warnings++
				}
			}
			// Enum/default/example suggestions (only if schema exists)
			if schema != nil && (typeStr == "string" || typeStr == "integer" || typeStr == "boolean") {
				if len(schema.Enum) == 0 {
					fmt.Fprintf(os.Stderr, "[INFO] Parameter '%s' in operation '%s' has no enum.\n", p.Name, op.OperationID)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add an 'enum' if the parameter has a fixed set of values.\n")
					warnings++
				}
				if schema.Default == nil {
					fmt.Fprintf(os.Stderr, "[INFO] Parameter '%s' in operation '%s' has no default value.\n", p.Name, op.OperationID)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'default' value for better UX.\n")
					warnings++
				}
				if schema.Example == nil {
					fmt.Fprintf(os.Stderr, "[INFO] Parameter '%s' in operation '%s' has no example.\n", p.Name, op.OperationID)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add an 'example' for documentation and testing.\n")
					warnings++
				}
				// Enum/default consistency
				if len(schema.Enum) > 0 && schema.Default != nil {
					found := false
					for _, v := range schema.Enum {
						if v == schema.Default {
							found = true
							break
						}
					}
					if !found {
						fmt.Fprintf(os.Stderr, "[WARN] Parameter '%s' in operation '%s' has a default value not in its enum list.\n", p.Name, op.OperationID)
						fmt.Fprintf(os.Stderr, "  Suggestion: Ensure the default value is one of the enum values.\n")
						warnings++
					}
				}
			}
		}
		// Request body checks
		if op.RequestBody != nil && op.RequestBody.Value != nil {
			for mtName, mt := range op.RequestBody.Value.Content {
				var schema *openapi3.Schema
				var typeStr string

				if mt.Schema == nil || mt.Schema.Value == nil {
					fmt.Fprintf(os.Stderr, "[ERROR] Request body for operation '%s' (media type: '%s') is missing a schema/type.\n", op.OperationID, mtName)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'schema' with a 'type', e.g. type: object\n")
					failures++
					// Don't continue - we can still check other media types and properties
				} else {
					schema = mt.Schema.Value
					if schema.Type != nil && len(*schema.Type) > 0 {
						typeStr = (*schema.Type)[0]
					} else {
						typeStr = ""
					}
					if typeStr == "" {
						fmt.Fprintf(os.Stderr, "[ERROR] Request body for operation '%s' (media type: '%s') is missing a type in its schema.\n", op.OperationID, mtName)
						fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'type' to the schema, e.g. type: object\n")
						failures++
					} else if !recommendedTypes[typeStr] {
						fmt.Fprintf(os.Stderr, "[WARN] Request body for operation '%s' uses uncommon type '%s'.\n", op.OperationID, typeStr)
						fmt.Fprintf(os.Stderr, "  Suggestion: Use one of: string, integer, boolean, number, array, object.\n")
						warnings++
					}
				}
				// Enum/default/example suggestions for request body properties (only if schema exists)
				if schema != nil && typeStr == "object" && schema.Properties != nil {
					for propName, propRef := range schema.Properties {
						if propRef == nil || propRef.Value == nil {
							continue
						}
						prop := propRef.Value
						var ptype string
						if prop.Type != nil && len(*prop.Type) > 0 {
							ptype = (*prop.Type)[0]
						}
						if ptype == "string" || ptype == "integer" || ptype == "boolean" {
							if len(prop.Enum) == 0 {
								fmt.Fprintf(os.Stderr, "[INFO] Request body property '%s' in operation '%s' has no enum.\n", propName, op.OperationID)
								fmt.Fprintf(os.Stderr, "  Suggestion: Add an 'enum' if the property has a fixed set of values.\n")
								warnings++
							}
							if prop.Default == nil {
								fmt.Fprintf(os.Stderr, "[INFO] Request body property '%s' in operation '%s' has no default value.\n", propName, op.OperationID)
								fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'default' value for better UX.\n")
								warnings++
							}
							if prop.Example == nil {
								fmt.Fprintf(os.Stderr, "[INFO] Request body property '%s' in operation '%s' has no example.\n", propName, op.OperationID)
								fmt.Fprintf(os.Stderr, "  Suggestion: Add an 'example' for documentation and testing.\n")
								warnings++
							}
							if len(prop.Enum) > 0 && prop.Default != nil {
								found := false
								for _, v := range prop.Enum {
									if v == prop.Default {
										found = true
										break
									}
								}
								if !found {
									fmt.Fprintf(os.Stderr, "[WARN] Request body property '%s' in operation '%s' has a default value not in its enum list.\n", propName, op.OperationID)
									fmt.Fprintf(os.Stderr, "  Suggestion: Ensure the default value is one of the enum values.\n")
									warnings++
								}
							}
						}
					}
				}
			}
		}
		// Schema required argument checks
		inputSchema := BuildInputSchema(op.Parameters, op.RequestBody)
		props, _ := inputSchema["properties"].(map[string]any)
		if reqList, ok := inputSchema["required"].([]string); ok {
			for _, req := range reqList {
				if _, ok := props[req]; !ok {
					fmt.Fprintf(os.Stderr, "[ERROR] Tool '%s' is missing required argument '%s' in schema.\n", op.OperationID, req)
					// Try to suggest the type if possible
					typeHint := "string"
					if param, found := findParamByName(op.Parameters, req); found && param.Schema != nil && param.Schema.Value != nil && param.Schema.Value.Type != nil && len(*param.Schema.Value.Type) > 0 {
						typeHint = (*param.Schema.Value.Type)[0]
					}
					fmt.Fprintf(os.Stderr, "  Suggestion: Add the required argument '%s' (type: %s) to the schema for tool '%s' (path: '%s', method: '%s').\n", req, typeHint, op.OperationID, op.Path, op.Method)
					fmt.Fprintf(os.Stderr, "    Example property: %s: { type: %q }\n", req, typeHint)
					failures++
				}
				// Warn if required field is missing an example
				if param, found := findParamByName(op.Parameters, req); found && param.Schema != nil && param.Schema.Value != nil && (param.Schema.Value.Example == nil) {
					fmt.Fprintf(os.Stderr, "[INFO] Required parameter '%s' in operation '%s' has no example.\n", req, op.OperationID)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add an 'example' for this required parameter.\n")
					warnings++
				}
			}
		}
		// Cross-field consistency: check if required parameters are mentioned in summary or description
		if reqList, ok := inputSchema["required"].([]string); ok {
			for _, req := range reqList {
				mentioned := false
				if op.Summary != "" && containsWord(op.Summary, req) {
					mentioned = true
				}
				if op.Description != "" && containsWord(op.Description, req) {
					mentioned = true
				}
				if !mentioned {
					fmt.Fprintf(os.Stderr, "[INFO] Required parameter '%s' in operation '%s' is not mentioned in summary or description.\n", req, op.OperationID)
					fmt.Fprintf(os.Stderr, "  Suggestion: Document required parameters in the summary or description for clarity.\n")
					warnings++
				}
			}
		}
	}
	if failures > 0 || warnings > 0 {
		fmt.Fprintf(os.Stderr, "[INFO] See the suggestions above to fix the reported issues.\n")
	}
	if failures > 0 {
		return fmt.Errorf("self-test failed: %d errors, %d warnings. See errors and suggestions above.", failures, warnings)
	}
	if warnings > 0 {
		fmt.Fprintf(os.Stderr, "[INFO] Self-test passed with %d warnings.\n", warnings)
	} else {
		fmt.Fprintf(os.Stderr, "[INFO] Self-test passed: all tools and required arguments are present.\n")
	}
	return nil
}

// findParamByName returns the parameter with the given name, if present.
func findParamByName(params openapi3.Parameters, name string) (*openapi3.Parameter, bool) {
	for _, paramRef := range params {
		if paramRef != nil && paramRef.Value != nil && paramRef.Value.Name == name {
			return paramRef.Value, true
		}
	}
	return nil, false
}

// containsWord checks if a word is present in a string (case-insensitive, word boundary).
func containsWord(s, word string) bool {
	if len(word) == 0 || len(s) == 0 {
		return false
	}
	return regexp.MustCompile(`(?i)\\b` + regexp.QuoteMeta(word) + `\\b`).MatchString(s)
}

// SelfTestOpenAPIMCPWithOptions runs the self-test with or without detailed suggestions.
func SelfTestOpenAPIMCPWithOptions(doc *openapi3.T, toolNames []string, detailedSuggestions bool) error {
	if detailedSuggestions {
		return SelfTestOpenAPIMCP(doc, toolNames)
	}
	// Basic actionable errors only
	ops := ExtractOpenAPIOperations(doc)
	failures := 0
	toolMap := map[string]struct{}{}
	for _, name := range toolNames {
		toolMap[name] = struct{}{}
	}

	// Check for missing operationIds in the original spec
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation.OperationID == "" {
				fmt.Fprintf(os.Stderr, "[ERROR] Operation for path '%s' and method '%s' is missing an operationId.\n", path, method)
				fmt.Fprintf(os.Stderr, "  Suggestion: Add an 'operationId' field, e.g.\n    %s:\n      %s:\n        operationId: <uniqueOperationId>\n", path, method)
				failures++
			}
		}
	}

	for _, op := range ops {
		if _, ok := toolMap[op.OperationID]; !ok && op.OperationID != "" {
			fmt.Fprintf(os.Stderr, "[ERROR] Tool '%s' (operationId) is missing from MCP server.\n", op.OperationID)
			fmt.Fprintf(os.Stderr, "  Suggestion: Ensure the operationId '%s' is unique and present in the OpenAPI spec.\n", op.OperationID)
			failures++
		}
		for _, paramRef := range op.Parameters {
			if paramRef == nil || paramRef.Value == nil {
				continue
			}
			p := paramRef.Value
			if p.Name == "" {
				fmt.Fprintf(os.Stderr, "[ERROR] Operation '%s' has a parameter with no name.\n", op.OperationID)
				fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'name' field to the parameter.\n")
				failures++
			}
			if p.Schema == nil || p.Schema.Value == nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Parameter '%s' in operation '%s' is missing a schema/type.\n", p.Name, op.OperationID)
				fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'schema' with a 'type', e.g.\n    - name: %s\n      in: %s\n      schema:\n        type: string\n", p.Name, p.In)
				failures++
				// Don't continue - we can still check other parameters
			} else {
				var typeStr string
				if p.Schema.Value.Type != nil && len(*p.Schema.Value.Type) > 0 {
					typeStr = (*p.Schema.Value.Type)[0]
				}
				if typeStr == "" {
					fmt.Fprintf(os.Stderr, "[ERROR] Parameter '%s' in operation '%s' is missing a type in its schema.\n", p.Name, op.OperationID)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'type' to the schema, e.g. type: string\n")
					failures++
				}
			}
		}
		// Request body checks
		if op.RequestBody != nil && op.RequestBody.Value != nil {
			for mtName, mt := range op.RequestBody.Value.Content {
				if mt.Schema == nil || mt.Schema.Value == nil {
					fmt.Fprintf(os.Stderr, "[ERROR] Request body for operation '%s' (media type: '%s') is missing a schema/type.\n", op.OperationID, mtName)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'schema' with a 'type', e.g. type: object\n")
					failures++
					// Don't continue - we can still check other media types
				} else {
					var typeStr string
					if mt.Schema.Value.Type != nil && len(*mt.Schema.Value.Type) > 0 {
						typeStr = (*mt.Schema.Value.Type)[0]
					}
					if typeStr == "" {
						fmt.Fprintf(os.Stderr, "[ERROR] Request body for operation '%s' (media type: '%s') is missing a type in its schema.\n", op.OperationID, mtName)
						fmt.Fprintf(os.Stderr, "  Suggestion: Add a 'type' to the schema, e.g. type: object\n")
						failures++
					}
				}
			}
		}
		// Schema required argument checks
		inputSchema := BuildInputSchema(op.Parameters, op.RequestBody)
		props, _ := inputSchema["properties"].(map[string]any)
		if reqList, ok := inputSchema["required"].([]string); ok {
			for _, req := range reqList {
				if _, ok := props[req]; !ok {
					fmt.Fprintf(os.Stderr, "[ERROR] Tool '%s' is missing required argument '%s' in schema.\n", op.OperationID, req)
					fmt.Fprintf(os.Stderr, "  Suggestion: Add the required argument '%s' to the schema for tool '%s' (path: '%s', method: '%s').\n", req, op.OperationID, op.Path, op.Method)
					fmt.Fprintf(os.Stderr, "    Example property: %s: { type: \"string\" }\n", req)
					failures++
				}
			}
		}
	}
	if failures > 0 {
		fmt.Fprintf(os.Stderr, "[INFO] See the suggestions above to fix the reported issues.\n")
		return fmt.Errorf("self-test failed: %d issues found. See errors and suggestions above.", failures)
	}
	fmt.Fprintf(os.Stderr, "[INFO] Self-test passed: all tools and required arguments are present.\n")
	return nil
}

// Example usage for SelfTestOpenAPIMCP:
//
//   doc, _ := openapi2mcp.LoadOpenAPISpec("petstore.yaml")
//   ops := openapi2mcp.ExtractOpenAPIOperations(doc)
//   srv := openapi2mcp.NewServer("petstore", doc.Info.Version, doc)
//   toolNames := srv.ListToolNames()
//   if err := openapi2mcp.SelfTestOpenAPIMCP(doc, toolNames); err != nil {
//       log.Fatal(err)
//   }

// LintOpenAPISpec performs comprehensive linting and returns structured results
func LintOpenAPISpec(doc *openapi3.T, detailedSuggestions bool) *LintResult {
	ops := ExtractOpenAPIOperations(doc)
	var toolNames []string
	for _, op := range ops {
		toolNames = append(toolNames, op.OperationID)
	}

	result := &LintResult{
		Issues: []LintIssue{},
	}

	// Capture linting issues
	issues := captureLintIssues(doc, toolNames, detailedSuggestions)
	result.Issues = issues

	// Count errors and warnings
	for _, issue := range issues {
		if issue.Type == "error" {
			result.ErrorCount++
		} else if issue.Type == "warning" {
			result.WarningCount++
		}
	}

	// Determine success status
	result.Success = result.ErrorCount == 0

	// Generate summary
	if result.ErrorCount == 0 && result.WarningCount == 0 {
		if detailedSuggestions {
			result.Summary = "OpenAPI linting passed: spec follows all best practices."
		} else {
			result.Summary = "MCP validation passed: all tools and required arguments are present."
		}
	} else if result.ErrorCount > 0 {
		if detailedSuggestions {
			result.Summary = fmt.Sprintf("OpenAPI linting completed with issues: %d errors, %d warnings.", result.ErrorCount, result.WarningCount)
		} else {
			result.Summary = fmt.Sprintf("MCP validation failed: %d errors, %d warnings.", result.ErrorCount, result.WarningCount)
		}
	} else {
		result.Summary = fmt.Sprintf("OpenAPI linting passed with %d warnings.", result.WarningCount)
	}

	return result
}

// captureLintIssues captures linting issues without printing to stderr
func captureLintIssues(doc *openapi3.T, toolNames []string, detailedSuggestions bool) []LintIssue {
	var issues []LintIssue
	ops := ExtractOpenAPIOperations(doc)
	toolMap := map[string]struct{}{}
	for _, name := range toolNames {
		toolMap[name] = struct{}{}
	}

	// Check for missing operationIds in the original spec
	for path, pathItem := range doc.Paths.Map() {
		for method, operation := range pathItem.Operations() {
			if operation.OperationID == "" {
				issues = append(issues, LintIssue{
					Type:       "error",
					Message:    fmt.Sprintf("Operation for path '%s' and method '%s' is missing an operationId.", path, method),
					Suggestion: fmt.Sprintf("Add an 'operationId' field, e.g.\n    %s:\n      %s:\n        operationId: <uniqueOperationId>", path, method),
					Path:       path,
					Method:     method,
				})
			}
		}
	}

	if !detailedSuggestions {
		// Basic validation only - check tool presence
		for _, op := range ops {
			if _, ok := toolMap[op.OperationID]; !ok && op.OperationID != "" {
				issues = append(issues, LintIssue{
					Type:       "error",
					Message:    fmt.Sprintf("Tool '%s' (operationId) is missing from MCP server.", op.OperationID),
					Suggestion: fmt.Sprintf("Ensure the operationId '%s' is unique and present in the OpenAPI spec.", op.OperationID),
					Operation:  op.OperationID,
				})
			}

			// Basic parameter checks
			for _, paramRef := range op.Parameters {
				if paramRef == nil || paramRef.Value == nil {
					continue
				}
				p := paramRef.Value
				if p.Name == "" {
					issues = append(issues, LintIssue{
						Type:       "error",
						Message:    fmt.Sprintf("Operation '%s' has a parameter with no name.", op.OperationID),
						Suggestion: "Add a 'name' field to the parameter.",
						Operation:  op.OperationID,
					})
				}
				if p.Schema == nil || p.Schema.Value == nil {
					issues = append(issues, LintIssue{
						Type:       "error",
						Message:    fmt.Sprintf("Parameter '%s' in operation '%s' is missing a schema/type.", p.Name, op.OperationID),
						Suggestion: fmt.Sprintf("Add a 'schema' with a 'type', e.g.\n    - name: %s\n      in: %s\n      schema:\n        type: string", p.Name, p.In),
						Operation:  op.OperationID,
						Parameter:  p.Name,
					})
				}
			}
		}
		return issues
	}

	// Detailed linting with comprehensive suggestions
	recommendedTypes := map[string]bool{"string": true, "integer": true, "boolean": true, "number": true, "array": true, "object": true}
	recommendedLocations := map[string]bool{"path": true, "query": true, "header": true, "cookie": true}

	for _, op := range ops {
		if _, ok := toolMap[op.OperationID]; !ok && op.OperationID != "" {
			issues = append(issues, LintIssue{
				Type:       "error",
				Message:    fmt.Sprintf("Tool '%s' (operationId) is missing from MCP server.", op.OperationID),
				Suggestion: fmt.Sprintf("Ensure the operationId '%s' is unique and present in the OpenAPI spec.", op.OperationID),
				Operation:  op.OperationID,
			})
		}

		// Check for missing summary, description, tags
		if op.Summary == "" {
			issues = append(issues, LintIssue{
				Type:       "warning",
				Message:    fmt.Sprintf("Operation '%s' (path: '%s', method: '%s') is missing a summary.", op.OperationID, op.Path, op.Method),
				Suggestion: "Add a 'summary' field to describe the operation's purpose.",
				Operation:  op.OperationID,
				Path:       op.Path,
				Method:     op.Method,
			})
		}
		if op.Description == "" {
			issues = append(issues, LintIssue{
				Type:       "warning",
				Message:    fmt.Sprintf("Operation '%s' (path: '%s', method: '%s') is missing a description.", op.OperationID, op.Path, op.Method),
				Suggestion: "Add a 'description' field for more detail.",
				Operation:  op.OperationID,
				Path:       op.Path,
				Method:     op.Method,
			})
		}
		if len(op.Tags) == 0 {
			issues = append(issues, LintIssue{
				Type:       "warning",
				Message:    fmt.Sprintf("Operation '%s' (path: '%s', method: '%s') has no tags.", op.OperationID, op.Path, op.Method),
				Suggestion: "Add tags to group related operations.",
				Operation:  op.OperationID,
				Path:       op.Path,
				Method:     op.Method,
			})
		}

		// Parameter checks with detailed suggestions
		for _, paramRef := range op.Parameters {
			if paramRef == nil || paramRef.Value == nil {
				continue
			}
			p := paramRef.Value
			if p.Name == "" {
				issues = append(issues, LintIssue{
					Type:       "error",
					Message:    fmt.Sprintf("Operation '%s' has a parameter with no name.", op.OperationID),
					Suggestion: "Add a 'name' field to the parameter.",
					Operation:  op.OperationID,
				})
				// Don't continue - we can still check schema and other properties
			}

			var schema *openapi3.Schema
			var typeStr string

			if p.Schema == nil || p.Schema.Value == nil {
				issues = append(issues, LintIssue{
					Type:       "error",
					Message:    fmt.Sprintf("Parameter '%s' in operation '%s' is missing a schema/type.", p.Name, op.OperationID),
					Suggestion: fmt.Sprintf("Add a 'schema' with a 'type', e.g.\n    - name: %s\n      in: %s\n      schema:\n        type: string", p.Name, p.In),
					Operation:  op.OperationID,
					Parameter:  p.Name,
				})
				// Don't continue - we can still check other parameter properties
			} else {
				schema = p.Schema.Value
				if schema.Type != nil && len(*schema.Type) > 0 {
					typeStr = (*schema.Type)[0]
				} else {
					typeStr = ""
				}
			}

			// Check type recommendations and other schema properties (only if schema exists)
			if schema != nil && typeStr != "" && !recommendedTypes[typeStr] {
				issues = append(issues, LintIssue{
					Type:       "warning",
					Message:    fmt.Sprintf("Parameter '%s' in operation '%s' has type '%s' which may not be well-supported.", p.Name, op.OperationID, typeStr),
					Suggestion: "Consider using standard types: string, integer, boolean, number, array, object.",
					Operation:  op.OperationID,
					Parameter:  p.Name,
				})
			}
			if p.In != "" && !recommendedLocations[p.In] {
				issues = append(issues, LintIssue{
					Type:       "warning",
					Message:    fmt.Sprintf("Parameter '%s' in operation '%s' is in location '%s' which may not be well-supported.", p.Name, op.OperationID, p.In),
					Suggestion: "Consider using standard locations: path, query, header, cookie.",
					Operation:  op.OperationID,
					Parameter:  p.Name,
				})
			}

			// Additional detailed checks (only if schema exists)
			if schema != nil {
				if len(schema.Enum) == 0 && (typeStr == "string" || typeStr == "integer") {
					issues = append(issues, LintIssue{
						Type:       "warning",
						Message:    fmt.Sprintf("Parameter '%s' in operation '%s' has no enum.", p.Name, op.OperationID),
						Suggestion: "Add an 'enum' if the parameter has a fixed set of values.",
						Operation:  op.OperationID,
						Parameter:  p.Name,
					})
				}
				if schema.Default == nil {
					issues = append(issues, LintIssue{
						Type:       "warning",
						Message:    fmt.Sprintf("Parameter '%s' in operation '%s' has no default value.", p.Name, op.OperationID),
						Suggestion: "Add a 'default' value for better UX.",
						Operation:  op.OperationID,
						Parameter:  p.Name,
					})
				}
				if schema.Example == nil {
					issues = append(issues, LintIssue{
						Type:       "warning",
						Message:    fmt.Sprintf("Parameter '%s' in operation '%s' has no example.", p.Name, op.OperationID),
						Suggestion: "Add an 'example' for documentation and testing.",
						Operation:  op.OperationID,
						Parameter:  p.Name,
					})
				}
			}
		}
	}

	return issues
}
