package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ubermorgenland/openapi-mcp/pkg/database"
	"github.com/ubermorgenland/openapi-mcp/pkg/services"
)

func main() {
	specsDir := "./specs"
	if len(os.Args) > 1 {
		specsDir = os.Args[1]
	}

	// Check if specs directory exists
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		log.Fatalf("Specs directory does not exist: %s", specsDir)
	}

	// Initialize database connection
	if err := database.InitializeDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	specLoader := services.NewSpecLoaderService(database.DB)

	// Read all files in specs directory
	files, err := os.ReadDir(specsDir)
	if err != nil {
		log.Fatalf("Failed to read specs directory: %v", err)
	}

	imported := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		ext := strings.ToLower(filepath.Ext(fileName))

		// Only process YAML and JSON files
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}

		filePath := filepath.Join(specsDir, fileName)

		// Generate name from filename (remove extension)
		name := strings.TrimSuffix(fileName, ext)

		// Generate endpoint path from name
		endpointPath := "/" + strings.ReplaceAll(name, "_", "-")

		// Import the spec
		err := specLoader.ImportSpecFromFile(filePath, name, endpointPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to import %s: %v\n", fileName, err)
			continue
		}

		fmt.Printf("✓ Imported %s as '%s' with endpoint '%s'\n", fileName, name, endpointPath)
		imported++
	}

	fmt.Printf("\nImport completed: %d specs imported successfully\n", imported)

	if imported > 0 {
		fmt.Println("\nTo view imported specs, run:")
		fmt.Println("  spec-manager list")
		fmt.Println("\nTo see only active specs, run:")
		fmt.Println("  spec-manager active")
	}
}
