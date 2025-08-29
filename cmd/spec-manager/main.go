package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ubermorgenland/openapi-mcp/pkg/database"
	"github.com/ubermorgenland/openapi-mcp/pkg/services"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	// Initialize database connection
	if err := database.InitializeDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	specLoader := services.NewSpecLoaderService(database.DB)

	switch command {
	case "list":
		handleList(specLoader)
	case "import":
		handleImport(specLoader)
	case "activate":
		handleActivate(specLoader)
	case "deactivate":
		handleDeactivate(specLoader)
	case "delete":
		handleDelete(specLoader)
	case "active":
		handleActiveList(specLoader)
	case "set-token":
		handleSetToken(specLoader)
	case "help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("OpenAPI Spec Manager")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  list                           List all specs in the database")
	fmt.Println("  active                         List only active specs")
	fmt.Println("  import <file> <name> <endpoint> Import a spec file into the database")
	fmt.Println("  activate <id>                  Activate a spec by ID")
	fmt.Println("  deactivate <id>                Deactivate a spec by ID")
	fmt.Println("  delete <id>                    Delete a spec by ID")
	fmt.Println("  set-token <id> <token>         Set API key token for a spec")
	fmt.Println("  help                           Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  spec-manager import weather.yaml weather /weather")
	fmt.Println("  spec-manager list")
	fmt.Println("  spec-manager activate 1")
	fmt.Println("  spec-manager deactivate 1")
	fmt.Println("  spec-manager set-token 1 \"your_api_token_here\"")
	fmt.Println("")
	fmt.Println("Environment Variables:")
	fmt.Println("  DATABASE_URL                   PostgreSQL connection string")
}

func handleList(specLoader *services.SpecLoaderService) {
	specs, err := specLoader.GetAllSpecs()
	if err != nil {
		log.Fatalf("Failed to get specs: %v", err)
	}

	if len(specs) == 0 {
		fmt.Println("No specs found in the database.")
		return
	}

	fmt.Printf("%-4s %-20s %-30s %-10s %-8s %-10s %-12s %s\n", "ID", "Name", "Title", "Version", "Active", "Format", "Has Token", "Endpoint")
	fmt.Println(strings.Repeat("-", 115))

	for _, spec := range specs {
		title := ""
		if spec.Title != nil {
			title = *spec.Title
			if len(title) > 28 {
				title = title[:28] + "..."
			}
		}

		version := ""
		if spec.Version != nil {
			version = *spec.Version
			if len(version) > 8 {
				version = version[:8] + "..."
			}
		}

		active := "false"
		if spec.IsActive != nil && *spec.IsActive {
			active = "true"
		}

		format := ""
		if spec.FileFormat != nil {
			format = *spec.FileFormat
		}

		name := spec.Name
		if len(name) > 18 {
			name = name[:18] + "..."
		}

		hasToken := "No"
		if spec.ApiKeyToken != nil && *spec.ApiKeyToken != "" {
			hasToken = "Yes"
		}

		fmt.Printf("%-4d %-20s %-30s %-10s %-8s %-10s %-12s %s\n",
			spec.ID, name, title, version, active, format, hasToken, spec.EndpointPath)
	}
}

func handleActiveList(specLoader *services.SpecLoaderService) {
	specs, err := specLoader.GetActiveSpecs()
	if err != nil {
		log.Fatalf("Failed to get active specs: %v", err)
	}

	if len(specs) == 0 {
		fmt.Println("No active specs found in the database.")
		return
	}

	fmt.Printf("%-4s %-20s %-30s %-10s %-10s %-12s %s\n", "ID", "Name", "Title", "Version", "Format", "Has Token", "Endpoint")
	fmt.Println(strings.Repeat("-", 105))

	for _, spec := range specs {
		title := ""
		if spec.Title != nil {
			title = *spec.Title
			if len(title) > 28 {
				title = title[:28] + "..."
			}
		}

		version := ""
		if spec.Version != nil {
			version = *spec.Version
			if len(version) > 8 {
				version = version[:8] + "..."
			}
		}

		format := ""
		if spec.FileFormat != nil {
			format = *spec.FileFormat
		}

		name := spec.Name
		if len(name) > 18 {
			name = name[:18] + "..."
		}

		hasToken := "No"
		if spec.ApiKeyToken != nil && *spec.ApiKeyToken != "" {
			hasToken = "Yes"
		}

		fmt.Printf("%-4d %-20s %-30s %-10s %-10s %-12s %s\n",
			spec.ID, name, title, version, format, hasToken, spec.EndpointPath)
	}
}

func handleImport(specLoader *services.SpecLoaderService) {
	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: spec-manager import <file-path> <name> <endpoint-path>\n")
		os.Exit(1)
	}

	filePath := os.Args[2]
	name := os.Args[3]
	endpointPath := os.Args[4]

	err := specLoader.ImportSpecFromFile(filePath, name, endpointPath)
	if err != nil {
		log.Fatalf("Failed to import spec: %v", err)
	}

	fmt.Printf("Successfully imported spec '%s' from '%s' with endpoint '%s'\n", name, filePath, endpointPath)
}

func handleActivate(specLoader *services.SpecLoaderService) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: spec-manager activate <id>\n")
		os.Exit(1)
	}

	id, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid ID: %v", err)
	}

	err = specLoader.ActivateSpec(id)
	if err != nil {
		log.Fatalf("Failed to activate spec: %v", err)
	}

	fmt.Printf("Successfully activated spec with ID %d\n", id)
}

func handleDeactivate(specLoader *services.SpecLoaderService) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: spec-manager deactivate <id>\n")
		os.Exit(1)
	}

	id, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid ID: %v", err)
	}

	err = specLoader.DeactivateSpec(id)
	if err != nil {
		log.Fatalf("Failed to deactivate spec: %v", err)
	}

	fmt.Printf("Successfully deactivated spec with ID %d\n", id)
}

func handleDelete(specLoader *services.SpecLoaderService) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: spec-manager delete <id>\n")
		os.Exit(1)
	}

	id, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid ID: %v", err)
	}

	err = specLoader.DeleteSpec(id)
	if err != nil {
		log.Fatalf("Failed to delete spec: %v", err)
	}

	fmt.Printf("Successfully deleted spec with ID %d\n", id)
}

func handleSetToken(specLoader *services.SpecLoaderService) {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: spec-manager set-token <id> <token>\n")
		fmt.Fprintf(os.Stderr, "       spec-manager set-token <id> \"\"  (to clear token)\n")
		os.Exit(1)
	}

	id, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid ID: %v", err)
	}

	token := os.Args[3]
	var tokenPtr *string
	if token == "" {
		tokenPtr = nil
	} else {
		tokenPtr = &token
	}

	err = specLoader.UpdateApiKeyToken(id, tokenPtr)
	if err != nil {
		log.Fatalf("Failed to update API key token: %v", err)
	}

	if tokenPtr == nil {
		fmt.Printf("Successfully cleared API key token for spec with ID %d\n", id)
	} else {
		fmt.Printf("Successfully set API key token for spec with ID %d\n", id)
	}
}
