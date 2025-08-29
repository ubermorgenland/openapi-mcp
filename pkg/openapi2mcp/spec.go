// spec.go
package openapi2mcp

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// generateAIOpenAPILoadError creates comprehensive, AI-optimized error responses for OpenAPI loading failures
func generateAIOpenAPILoadError(operation, path string, originalErr error) error {
	var response strings.Builder

	response.WriteString(fmt.Sprintf("OPENAPI LOADING ERROR: %s failed\n\n", operation))

	if path != "" {
		response.WriteString(fmt.Sprintf("FILE: %s\n\n", path))
	}

	response.WriteString("ORIGINAL ERROR:\n")
	response.WriteString(originalErr.Error())
	response.WriteString("\n\n")

	// Analyze error and provide specific guidance
	errStr := strings.ToLower(originalErr.Error())

	if strings.Contains(errStr, "no such file") || strings.Contains(errStr, "cannot find") {
		response.WriteString("ISSUE: File not found\n\n")
		response.WriteString("TROUBLESHOOTING STEPS:\n")
		response.WriteString("1. Verify the file path is correct\n")
		response.WriteString("2. Check that the file exists: ls -la " + path + "\n")
		response.WriteString("3. Ensure you have read permissions on the file\n")
		response.WriteString("4. Try using an absolute path instead of relative path\n")
		response.WriteString("5. Check current working directory: pwd\n")
	} else if strings.Contains(errStr, "permission denied") {
		response.WriteString("ISSUE: Permission denied\n\n")
		response.WriteString("TROUBLESHOOTING STEPS:\n")
		response.WriteString("1. Check file permissions: ls -la " + path + "\n")
		response.WriteString("2. Ensure you have read access to the file\n")
		response.WriteString("3. Try running with appropriate permissions\n")
		response.WriteString("4. Contact your system administrator if needed\n")
	} else if strings.Contains(errStr, "yaml") || strings.Contains(errStr, "unmarshal") {
		response.WriteString("ISSUE: YAML/JSON parsing error\n\n")
		response.WriteString("TROUBLESHOOTING STEPS:\n")
		response.WriteString("1. Validate your YAML/JSON syntax using an online validator\n")
		response.WriteString("2. Check for common issues:\n")
		response.WriteString("   - Missing quotes around strings\n")
		response.WriteString("   - Incorrect indentation (YAML is sensitive to spaces)\n")
		response.WriteString("   - Missing commas in JSON\n")
		response.WriteString("   - Unclosed brackets or braces\n")
		response.WriteString("3. Use a YAML/JSON linter or formatter\n")
		response.WriteString("4. Verify the file encoding is UTF-8\n")
	} else if strings.Contains(errStr, "validation") || strings.Contains(errStr, "invalid") {
		response.WriteString("ISSUE: OpenAPI specification validation failed\n\n")
		response.WriteString("TROUBLESHOOTING STEPS:\n")
		response.WriteString("1. Ensure your spec follows OpenAPI 3.0+ format\n")
		response.WriteString("2. Required fields to check:\n")
		response.WriteString("   - 'openapi' version field (e.g., openapi: 3.0.0)\n")
		response.WriteString("   - 'info' section with title and version\n")
		response.WriteString("   - 'paths' section with at least one endpoint\n")
		response.WriteString("3. Validate using OpenAPI tools:\n")
		response.WriteString("   - Swagger Editor: https://editor.swagger.io/\n")
		response.WriteString("   - OpenAPI Generator validation\n")
		response.WriteString("   - Use this tool's validation: openapi-mcp validate " + path + "\n")
		response.WriteString("4. Common validation issues:\n")
		response.WriteString("   - Missing operationId for operations\n")
		response.WriteString("   - Invalid parameter definitions\n")
		response.WriteString("   - Incorrect schema references\n")
		response.WriteString("   - Missing required properties in schemas\n")
	} else if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "network") {
		response.WriteString("ISSUE: Network or timeout error\n\n")
		response.WriteString("TROUBLESHOOTING STEPS:\n")
		response.WriteString("1. Check your internet connection\n")
		response.WriteString("2. Verify any referenced external URLs are accessible\n")
		response.WriteString("3. Try downloading the spec locally if it's remote\n")
		response.WriteString("4. Check firewall settings\n")
	} else {
		response.WriteString("GENERAL TROUBLESHOOTING STEPS:\n")
		response.WriteString("1. Verify the OpenAPI spec file format (YAML or JSON)\n")
		response.WriteString("2. Check the OpenAPI version (should be 3.0+)\n")
		response.WriteString("3. Validate the spec using: openapi-mcp validate " + path + "\n")
		response.WriteString("4. Try using a minimal OpenAPI spec to test\n")
		response.WriteString("5. Check the documentation: https://spec.openapis.org/oas/v3.0.3/\n")
	}

	response.WriteString("\nEXAMPLE MINIMAL OPENAPI SPEC:\n")
	response.WriteString("```yaml\n")
	response.WriteString("openapi: 3.0.0\n")
	response.WriteString("info:\n")
	response.WriteString("  title: My API\n")
	response.WriteString("  version: 1.0.0\n")
	response.WriteString("paths:\n")
	response.WriteString("  /health:\n")
	response.WriteString("    get:\n")
	response.WriteString("      operationId: getHealth\n")
	response.WriteString("      summary: Health check\n")
	response.WriteString("      responses:\n")
	response.WriteString("        '200':\n")
	response.WriteString("          description: OK\n")
	response.WriteString("```\n")

	return fmt.Errorf(response.String())
}

// LoadOpenAPISpec loads and parses an OpenAPI YAML or JSON file from the given path.
// Returns the parsed OpenAPI document or an error.
// Example usage for LoadOpenAPISpec:
//
//	doc, err := openapi2mcp.LoadOpenAPISpec("petstore.yaml")
//	if err != nil { log.Fatal(err) }
//	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
func LoadOpenAPISpec(path string) (*openapi3.T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, generateAIOpenAPILoadError("File reading", path, err)
	}
	doc, err := LoadOpenAPISpecFromBytes(data)
	if err != nil {
		return nil, generateAIOpenAPILoadError("Spec parsing", path, err)
	}
	return doc, nil
}

// LoadOpenAPISpecFromString loads and parses an OpenAPI YAML or JSON spec from a string.
// Returns the parsed OpenAPI document or an error.
func LoadOpenAPISpecFromString(data string) (*openapi3.T, error) {
	return LoadOpenAPISpecFromBytes([]byte(data))
}

// LoadOpenAPISpecFromBytes loads and parses an OpenAPI YAML or JSON spec from a byte slice.
// Returns the parsed OpenAPI document or an error.
func LoadOpenAPISpecFromBytes(data []byte) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	if err != nil {
		return nil, generateAIOpenAPILoadError("Spec parsing", "", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, generateAIOpenAPILoadError("Spec validation", "", err)
	}
	return doc, nil
}

// ExtractOpenAPIOperations extracts all operations from the OpenAPI spec, merging path-level and operation-level parameters.
// Returns a slice of OpenAPIOperation describing each operation.
// Example usage for ExtractOpenAPIOperations:
//
//	doc, err := openapi2mcp.LoadOpenAPISpec("petstore.yaml")
//	if err != nil { log.Fatal(err) }
//	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
func ExtractOpenAPIOperations(doc *openapi3.T) []OpenAPIOperation {
	var ops []OpenAPIOperation
	for path, pathItem := range doc.Paths.Map() {
		for method, op := range pathItem.Operations() {
			id := op.OperationID
			if id == "" {
				id = fmt.Sprintf("%s_%s", method, path)
			}
			desc := op.Description

			// Merge path-level and operation-level parameters
			mergedParams := openapi3.Parameters{}
			if pathItem.Parameters != nil {
				mergedParams = append(mergedParams, pathItem.Parameters...)
			}
			if op.Parameters != nil {
				mergedParams = append(mergedParams, op.Parameters...)
			}

			tags := op.Tags
			var security openapi3.SecurityRequirements
			if op.Security != nil {
				security = *op.Security
			} else {
				security = doc.Security
			}
			ops = append(ops, OpenAPIOperation{
				OperationID: id,
				Summary:     op.Summary,
				Description: desc,
				Path:        path,
				Method:      method,
				Parameters:  mergedParams,
				RequestBody: op.RequestBody,
				Tags:        tags,
				Security:    security,
			})
		}
	}
	return ops
}

// ExtractFilteredOpenAPIOperations returns only those operations whose description matches includeRegex (if not nil) and does not match excludeRegex (if not nil).
// Returns a filtered slice of OpenAPIOperation.
// Example usage for ExtractFilteredOpenAPIOperations:
//
//	include := regexp.MustCompile("pets")
//	filtered := openapi2mcp.ExtractFilteredOpenAPIOperations(doc, include, nil)
func ExtractFilteredOpenAPIOperations(doc *openapi3.T, includeRegex, excludeRegex *regexp.Regexp) []OpenAPIOperation {
	all := ExtractOpenAPIOperations(doc)
	var filtered []OpenAPIOperation
	for _, op := range all {
		desc := op.Description
		if desc == "" {
			desc = op.Summary
		}
		if includeRegex != nil && !includeRegex.MatchString(desc) {
			continue
		}
		if excludeRegex != nil && excludeRegex.MatchString(desc) {
			continue
		}
		filtered = append(filtered, op)
	}
	return filtered
}
