// summary.go
package openapi2mcp

import "fmt"

// PrintToolSummary prints a human-readable summary of OpenAPI operations that will be converted to MCP tools.
//
// This function analyzes the provided operations and outputs:
//   - Total number of tools that will be generated
//   - Breakdown by tags showing operation count per tag
//   - Overall statistics about the OpenAPI specification
//
// This is useful for debugging and understanding what tools will be generated from an OpenAPI specification
// before actually starting the MCP server.
//
// Example usage:
//
//	doc, _ := openapi2mcp.LoadOpenAPISpec("petstore.yaml")
//	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
//	openapi2mcp.PrintToolSummary(ops)
//
// Output example:
//
//	Total tools: 12
//	Tags:
//	  pets: 8
//	  store: 3
//	  user: 1
func PrintToolSummary(ops []OpenAPIOperation) {
	tagCount := map[string]int{}
	for _, op := range ops {
		for _, tag := range op.Tags {
			tagCount[tag]++
		}
	}
	fmt.Printf("Total tools: %d\n", len(ops))
	if len(tagCount) > 0 {
		fmt.Println("Tags:")
		for tag, count := range tagCount {
			fmt.Printf("  %s: %d\n", tag, count)
		}
	}
}
