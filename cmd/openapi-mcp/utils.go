// utils.go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
)

// handleDryRunMode handles the --dry-run mode, printing tool schemas and summaries.
func handleDryRunMode(flags *cliFlags, ops []openapi2mcp.OpenAPIOperation, doc *openapi3.T) {
	opts := &openapi2mcp.ToolGenOptions{
		NameFormat:              nil, // Not used for dry-run output
		TagFilter:               flags.tagFlags,
		DryRun:                  true,
		PrettyPrint:             true,
		Version:                 doc.Info.Version,
		ConfirmDangerousActions: !flags.noConfirmDangerous,
	}
	openapi2mcp.RegisterOpenAPITools(nil, ops, doc, opts)
	if flags.summary {
		openapi2mcp.PrintToolSummary(ops)
	}
	if flags.diffFile != "" {
		compareWithDiffFile(opts, doc, ops, flags.diffFile)
	}
	os.Exit(0)
}

// compareWithDiffFile compares the generated output to a previous run (file path).
func compareWithDiffFile(opts *openapi2mcp.ToolGenOptions, doc *openapi3.T, ops []openapi2mcp.OpenAPIOperation, diffFile string) {
	// Generate current output
	var toolSummaries []map[string]any
	for _, op := range ops {
		if len(opts.TagFilter) > 0 {
			found := false
			for _, tag := range op.Tags {
				for _, want := range opts.TagFilter {
					if tag == want {
						found = true
						break
					}
				}
			}
			if !found {
				continue
			}
		}
		name := op.OperationID
		if opts.NameFormat != nil {
			name = opts.NameFormat(name)
		}
		desc := op.Description
		if desc == "" {
			desc = op.Summary
		}
		inputSchema := openapi2mcp.BuildInputSchemaWithContext(op.Parameters, op.RequestBody, doc)
		toolSummaries = append(toolSummaries, map[string]any{
			"name":        name,
			"description": desc,
			"tags":        op.Tags,
			"inputSchema": inputSchema,
		})
	}
	curBytes, _ := json.MarshalIndent(toolSummaries, "", "  ")
	_, err := os.ReadFile(diffFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not read diff file: %v\n", err)
		return
	}
	tmpFile, err := os.CreateTemp("", "openapi2mcp-diff-*.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not create temp file for diff: %v\n", err)
		return
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Write(curBytes)
	tmpFile.Close()
	cmd := exec.Command("diff", "-u", diffFile, tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil && err.Error() != "exit status 1" {
		fmt.Fprintf(os.Stderr, "Error running diff: %v\n", err)
	}
}
