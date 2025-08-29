package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/chzyer/readline"
)

// cliFlags holds all parsed CLI flags and arguments for mcp-client.
type cliFlags struct {
	showHelp bool
	quiet    bool
	machine  bool
	args     []string
}

// parseFlags parses all CLI flags and returns a cliFlags struct.
func parseFlags() *cliFlags {
	var flags cliFlags
	flag.BoolVar(&flags.showHelp, "h", false, "Show help")
	flag.BoolVar(&flags.showHelp, "help", false, "Show help")
	flag.BoolVar(&flags.quiet, "quiet", false, "Suppress banners and non-essential output")
	flag.BoolVar(&flags.machine, "machine", false, "Minimal output: only print raw result")
	flag.Parse()
	flags.args = flag.Args()
	return &flags
}

// printHelp prints the CLI help message for mcp-client.
func printHelp() {
	fmt.Print(`mcp-client: Simple MCP client for openapi-to-mcp

Usage:
  mcp-client <server-command> [args...]

Flags:
  --quiet              Suppress banners and non-essential output
  --machine            Minimal output: only print raw result
  --help, -h           Show help

By default, output is human-friendly. Use --machine or --quiet for minimal/agent output.
`)
	os.Exit(0)
}

// main is the entrypoint for the mcp-client CLI.
func main() {
	flags := parseFlags()
	showHelp := flags.showHelp
	quiet := flags.quiet
	machine := flags.machine

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: mcp-client <server-command> [args...]")
		os.Exit(1)
	}

	// Start the MCP server subprocess
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	serverIn, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get server stdin:", err)
		os.Exit(1)
	}
	serverOut, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get server stdout:", err)
		os.Exit(1)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to start server:", err)
		os.Exit(1)
	}

	serverReader := bufio.NewReader(serverOut)
	id := 1

	// Tool info cache
	var (
		toolNames   []string
		toolSchemas = make(map[string]map[string]any)
	)

	// Fetch tool list and schema at startup only
	msg := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/list",
		"params":  map[string]any{},
	}
	id++
	_ = json.NewEncoder(serverIn).Encode(msg)
	for {
		line, err := serverReader.ReadString('\n')
		if err != nil {
			break
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err == nil {
			if result, ok := obj["result"]; ok {
				if tools, ok := result.(map[string]any)["tools"]; ok {
					if arr, ok := tools.([]any); ok {
						for _, t := range arr {
							if tmap, ok := t.(map[string]any); ok {
								if name, ok := tmap["name"].(string); ok {
									toolNames = append(toolNames, name)
									if schema, ok := tmap["inputSchema"].(map[string]any); ok {
										toolSchemas[name] = schema
									}
								}
							}
						}
					}
				}
			}
			break
		}
	}

	// Set up readline for prompt/history and autocompletion
	makeCompleter := func() *readline.PrefixCompleter {
		callItems := []readline.PrefixCompleterInterface{}
		for _, name := range toolNames {
			// Argument completion for each tool
			argItems := []readline.PrefixCompleterInterface{}
			if schema, ok := toolSchemas[name]; ok {
				if props, ok := schema["properties"].(map[string]any); ok {
					for arg, v := range props {
						// Suggest possible values for enums or booleans
						if vmap, ok := v.(map[string]any); ok {
							if enumVals, ok := vmap["enum"].([]any); ok && len(enumVals) > 0 {
								valItems := []readline.PrefixCompleterInterface{}
								for _, ev := range enumVals {
									valItems = append(valItems, readline.PcItem(fmt.Sprintf("%v", ev)))
								}
								argItems = append(argItems, readline.PcItem(arg, valItems...))
								continue
							}
							if t, ok := vmap["type"].(string); ok && t == "boolean" {
								argItems = append(argItems, readline.PcItem(arg, readline.PcItem("true"), readline.PcItem("false")))
								continue
							}
						}
						argItems = append(argItems, readline.PcItem(arg))
					}
				}
			}
			callItems = append(callItems, readline.PcItem(name, argItems...))
		}
		schemaItems := []readline.PrefixCompleterInterface{}
		for _, name := range toolNames {
			schemaItems = append(schemaItems, readline.PcItem(name))
		}
		return readline.NewPrefixCompleter(
			readline.PcItem("list"),
			readline.PcItem("help"),
			readline.PcItem("exit"),
			readline.PcItem("quit"),
			readline.PcItem("call", callItems...),
			readline.PcItem("schema", schemaItems...),
			readline.PcItem("history"),
		)
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "mcp> ",
		HistoryFile:     os.ExpandEnv("$HOME/.mcp_client_history"),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		AutoComplete:    makeCompleter(),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to initialize readline:", err)
		os.Exit(1)
	}
	defer rl.Close()

	if !quiet && !machine {
		fmt.Println("Welcome to mcp-client! Type 'help' for available commands.")
	}

	// Goroutine to print server responses
	go func() {
		for {
			line, err := serverReader.ReadString('\n')
			if err != nil {
				fmt.Fprintln(os.Stderr, "[server closed]", err)
				os.Exit(0)
			}
			var obj map[string]any
			if err := json.Unmarshal([]byte(line), &obj); err == nil {
				if method, ok := obj["method"].(string); ok && method == "tools/call" {
					// Notification, ignore
				} else if result, ok := obj["result"]; ok {
					if quiet || machine {
						if pretty, err := json.MarshalIndent(result, "", "  "); err == nil {
							fmt.Println(string(pretty))
						} else {
							fmt.Println(result)
						}
					} else {
						if resultMap, ok := result.(map[string]any); ok {
							if contentArr, ok := resultMap["content"].([]any); ok {
								for _, c := range contentArr {
									if cMap, ok := c.(map[string]any); ok {
										if txt, ok := cMap["text"].(string); ok {
											if idx := strings.Index(txt, "Response:\n"); idx != -1 {
												prefix := txt[:idx+len("Response:\n")]
												jsonPart := strings.TrimSpace(txt[idx+len("Response:\n"):])
												if len(jsonPart) > 0 && (jsonPart[0] == '{' || jsonPart[0] == '[') {
													var prettyBuf bytes.Buffer
													if err := json.Indent(&prettyBuf, []byte(jsonPart), "", "  "); err == nil {
														fmt.Fprintf(os.Stdout, "%s%s\n", prefix, prettyBuf.String())
														continue
													}
												}
											}
											fmt.Fprintln(os.Stdout, txt)
										}
									}
								}
							} else {
								prettyResult, _ := json.MarshalIndent(result, "", "  ")
								fmt.Fprintf(os.Stdout, "[tool response] %s\n", prettyResult)
							}
						} else {
							prettyResult, _ := json.MarshalIndent(result, "", "  ")
							fmt.Fprintf(os.Stdout, "[server result] %s\n", prettyResult)
						}
					}
				} else if errObj, ok := obj["error"]; ok {
					if quiet || machine {
						if pretty, err := json.MarshalIndent(errObj, "", "  "); err == nil {
							fmt.Fprintln(os.Stderr, string(pretty))
						} else {
							fmt.Fprintln(os.Stderr, errObj)
						}
					} else {
						prettyErr, _ := json.MarshalIndent(errObj, "", "  ")
						fmt.Fprintf(os.Stderr, "[server error] %s\n", prettyErr)
					}
				} else {
					pretty, _ := json.MarshalIndent(obj, "", "  ")
					fmt.Fprintf(os.Stderr, "[server] %s\n", pretty)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[server] %s", line)
			}
			rl.Refresh()
		}
	}()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				cmd.Process.Kill()
				return
			}
			continue
		} else if err == io.EOF {
			cmd.Process.Kill()
			return
		}
		line = strings.TrimSpace(line)
		if line == "exit" || line == "quit" {
			cmd.Process.Kill()
			return
		}
		if line == "help" {
			fmt.Print(`Available commands:

  help        Show this help message
  exit        Exit the client
  schema      Show the schema for a tool
  call        Call a tool with arguments
  list        List available tools
  version     Show version info
  history     View command history
`)
			continue
		}
		if line == "list" {
			msg := map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"method":  "tools/list",
				"params":  map[string]any{},
			}
			id++
			json.NewEncoder(serverIn).Encode(msg)
			continue
		}
		if strings.HasPrefix(line, "schema ") {
			name := strings.TrimSpace(line[len("schema "):])
			if schema, ok := toolSchemas[name]; ok {
				pretty, _ := json.MarshalIndent(schema, "", "  ")
				fmt.Printf("Schema for %s:\n%s\n", name, pretty)

				// Generate example call command
				if props, ok := schema["properties"].(map[string]any); ok {
					example := map[string]any{}
					for k, v := range props {
						if vmap, ok := v.(map[string]any); ok {
							typeStr, _ := vmap["type"].(string)
							descStr, _ := vmap["description"].(string)
							if typeStr == "string" && strings.Contains(strings.ToLower(descStr), "integer") {
								example[k] = "123"
								continue
							}
							switch typeStr {
							case "string":
								example[k] = "example"
							case "number":
								example[k] = 123.45
							case "integer":
								example[k] = 123
							case "boolean":
								example[k] = true
							case "array":
								example[k] = []any{"item1", "item2"}
							case "object":
								example[k] = map[string]any{"key": "value"}
							default:
								example[k] = nil
							}
						} else {
							example[k] = nil
						}
					}
					exampleJSON, _ := json.Marshal(example)
					fmt.Printf("Example: call %s %s\n", name, exampleJSON)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[error] No schema found for tool '%s'. Try 'refresh' if the tool was just added.\n", name)
			}
			continue
		}
		if len(line) > 5 && line[:5] == "call " {
			rest := line[5:]
			space := -1
			for i, c := range rest {
				if c == ' ' {
					space = i
					break
				}
			}
			if space == -1 {
				fmt.Fprintln(os.Stderr, "Usage: call <tool> <json-args>")
				continue
			}
			tool := rest[:space]
			args := rest[space+1:]
			var argObj map[string]any
			if err := json.Unmarshal([]byte(args), &argObj); err != nil {
				fmt.Fprintln(os.Stderr, "Invalid JSON for args:", err)
				if schema, ok := toolSchemas[tool]; ok {
					pretty, _ := json.MarshalIndent(schema, "", "  ")
					fmt.Fprintf(os.Stderr, "Expected schema for %s:\n%s\n", tool, pretty)
				}
				continue
			}
			msg := map[string]any{
				"jsonrpc": "2.0",
				"id":      id,
				"method":  "tools/call",
				"params": map[string]any{
					"name":      tool,
					"arguments": argObj,
				},
			}
			id++
			json.NewEncoder(serverIn).Encode(msg)
			continue
		}
		if line == "history" || strings.HasPrefix(line, "history ") {
			query := strings.TrimSpace(line[len("history"):])
			historyFile := os.ExpandEnv("$HOME/.mcp_client_history")
			file, err := os.Open(historyFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[error] Could not open history file: %v\n", err)
				continue
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			if query == "" {
				for i, l := range lines {
					fmt.Printf("%4d: %s\n", i+1, l)
				}
			} else {
				for i, l := range lines {
					if strings.Contains(l, query) {
						fmt.Printf("%4d: %s\n", i+1, l)
					}
				}
			}
			continue
		}
		if line == "" {
			continue
		}
		fmt.Fprintln(os.Stderr, "[error] Unknown command. Type 'help' for available commands.")
	}
}
