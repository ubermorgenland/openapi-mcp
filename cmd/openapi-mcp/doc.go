// doc.go
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
)

// handleDocMode handles the --doc mode, generating Markdown documentation for all tools.
func handleDocMode(flags *cliFlags, ops []openapi2mcp.OpenAPIOperation, doc *openapi3.T) {
	toolSummaries := make([]map[string]any, 0, len(ops))
	for _, op := range ops {
		name := op.OperationID
		if flags.toolNameFormat != "" {
			name = formatToolName(flags.toolNameFormat, name)
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
	jsonBytes, _ := json.MarshalIndent(toolSummaries, "", "  ")
	if flags.postHookCmd != "" {
		out, err := processWithPostHook(jsonBytes, flags.postHookCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running post-hook-cmd: %v\n", err)
			os.Exit(1)
		}
		jsonBytes = out
	}
	if flags.docFormat == "markdown" {
		// Parse the possibly post-processed JSON back to []map[string]any
		var processed []map[string]any
		if err := json.Unmarshal(jsonBytes, &processed); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing post-processed JSON: %v\n", err)
			os.Exit(1)
		}
		if err := writeMarkdownDocFromSummaries(flags.docFile, processed, doc); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing Markdown doc: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote Markdown documentation to %s\n", flags.docFile)
		os.Exit(0)
	} else if flags.docFormat == "html" {
		fmt.Fprintf(os.Stderr, "HTML documentation output is not yet implemented.\n")
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "Unknown doc format: %s\n", flags.docFormat)
		os.Exit(1)
	}
}

// writeMarkdownDocFromSummaries writes Markdown documentation from a []map[string]any (post-processed summaries).
func writeMarkdownDocFromSummaries(path string, summaries []map[string]any, doc *openapi3.T) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	f.WriteString("# MCP Tools Documentation\n\n")
	if doc.Info != nil {
		f.WriteString(fmt.Sprintf("**API Title:** %s\n\n", doc.Info.Title))
		f.WriteString(fmt.Sprintf("**Version:** %s\n\n", doc.Info.Version))
		if doc.Info.Description != "" {
			f.WriteString(doc.Info.Description + "\n\n")
		}
	}
	for _, m := range summaries {
		name, _ := m["name"].(string)
		desc, _ := m["description"].(string)
		tags, _ := m["tags"].([]any)
		inputSchema, _ := m["inputSchema"].(map[string]any)
		f.WriteString(fmt.Sprintf("## %s\n\n", name))
		if desc != "" {
			f.WriteString(desc + "\n\n")
		}
		if len(tags) > 0 {
			tagStrs := make([]string, len(tags))
			for i, t := range tags {
				tagStrs[i], _ = t.(string)
			}
			f.WriteString(fmt.Sprintf("**Tags:** %s\n\n", strings.Join(tagStrs, ", ")))
		}
		// Arguments
		props, _ := inputSchema["properties"].(map[string]any)
		if len(props) > 0 {
			f.WriteString("**Arguments:**\n\n")
			f.WriteString("| Name | Type | Description |\n|------|------|-------------|\n")
			for name, v := range props {
				vmap, _ := v.(map[string]any)
				typeStr, _ := vmap["type"].(string)
				desc, _ := vmap["description"].(string)
				f.WriteString(fmt.Sprintf("| %s | %s | %s |\n", name, typeStr, desc))
			}
			f.WriteString("\n")
		}
		// Example call (best effort)
		example := map[string]any{}
		for name, v := range props {
			vmap, _ := v.(map[string]any)
			typeStr, _ := vmap["type"].(string)
			descStr, _ := vmap["description"].(string)
			if typeStr == "string" && strings.Contains(strings.ToLower(descStr), "integer") {
				example[name] = "123"
				continue
			}
			switch typeStr {
			case "string":
				example[name] = "example"
			case "number":
				example[name] = 123.45
			case "integer":
				example[name] = 123
			case "boolean":
				example[name] = true
			default:
				example[name] = "..."
			}
		}
		if len(example) > 0 {
			exampleJSON, _ := json.MarshalIndent(example, "", "  ")
			f.WriteString("**Example call:**\n\n")
			f.WriteString("```json\n" + fmt.Sprintf("call %s %s\n", name, string(exampleJSON)) + "```\n\n")
		}
	}
	return nil
}

// processWithPostHook pipes JSON through an external command and returns the output.
func processWithPostHook(jsonBytes []byte, postHookCmd string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", postHookCmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	errPipe, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	stdin.Write(jsonBytes)
	stdin.Close()
	out, _ := io.ReadAll(stdout)
	errBytes, _ := io.ReadAll(errPipe)
	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("post-hook-cmd failed: %v\n%s", err, string(errBytes))
	}
	return out, nil
}

// formatToolName applies the requested tool name formatting.
func formatToolName(format, name string) string {
	switch format {
	case "lower":
		return strings.ToLower(name)
	case "upper":
		return strings.ToUpper(name)
	case "snake":
		return toSnakeCase(name)
	case "camel":
		return toCamelCase(name)
	default:
		return name
	}
}

// toSnakeCase converts a string to snake_case.
func toSnakeCase(s string) string {
	var out []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			out = append(out, '_')
		}
		out = append(out, r)
	}
	return strings.ToLower(string(out))
}

// toCamelCase converts a string to camelCase.
func toCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		return s
	}
	out := strings.ToLower(parts[0])
	for _, p := range parts[1:] {
		if len(p) > 0 {
			out += strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return out
}
