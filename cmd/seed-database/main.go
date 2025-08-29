package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ubermorgenland/openapi-mcp/pkg/database"
	"github.com/ubermorgenland/openapi-mcp/pkg/services"
	"gopkg.in/yaml.v3"
)

// SpecConfig defines how each spec should be imported
type SpecConfig struct {
	File         string `json:"file" yaml:"file"`
	Name         string `json:"name" yaml:"name"`
	EndpointPath string `json:"endpoint_path" yaml:"endpoint_path"`
	Active       bool   `json:"active" yaml:"active"`
}

// SeedConfig defines the seeding configuration
type SeedConfig struct {
	Specs []SpecConfig `json:"specs" yaml:"specs"`
}

func main() {
	var configFile string
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	// Initialize database connection
	if err := database.InitializeDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	specLoader := services.NewSpecLoaderService(database.DB)

	if configFile != "" {
		// Use config file
		seedFromConfig(specLoader, configFile)
	} else {
		// Auto-discover and seed
		autoSeed(specLoader)
	}
}

func seedFromConfig(specLoader *services.SpecLoaderService, configFile string) {
	// Read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var config SeedConfig
	ext := strings.ToLower(filepath.Ext(configFile))

	if ext == ".json" {
		err = json.Unmarshal(data, &config)
	} else {
		err = yaml.Unmarshal(data, &config)
	}

	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	fmt.Printf("Seeding database with %d specs from config...\n", len(config.Specs))

	imported := 0
	for _, specConfig := range config.Specs {
		// Import the spec
		err := specLoader.ImportSpecFromFile(specConfig.File, specConfig.Name, specConfig.EndpointPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to import %s: %v\n", specConfig.File, err)
			continue
		}

		fmt.Printf("✓ Imported %s as '%s' with endpoint '%s'\n",
			specConfig.File, specConfig.Name, specConfig.EndpointPath)

		// Set active status if specified as inactive
		if !specConfig.Active {
			// We need to get the spec ID first - this is a limitation of the current design
			specs, err := specLoader.GetAllSpecs()
			if err == nil {
				for _, spec := range specs {
					if spec.Name == specConfig.Name {
						specLoader.DeactivateSpec(spec.ID)
						fmt.Printf("  → Deactivated spec '%s'\n", specConfig.Name)
						break
					}
				}
			}
		}

		imported++
	}

	fmt.Printf("\nSeeding completed: %d specs imported successfully\n", imported)
}

func autoSeed(specLoader *services.SpecLoaderService) {
	// Default seeding with predefined configurations
	specs := []SpecConfig{
		{File: "specs/weather.json", Name: "weather", EndpointPath: "/weather", Active: true},
		{File: "specs/twitter.yml", Name: "twitter", EndpointPath: "/twitter", Active: true},
		{File: "specs/google_keywords.yml", Name: "google-keywords", EndpointPath: "/google-keywords", Active: true},
		{File: "specs/perplexity.yml", Name: "perplexity", EndpointPath: "/perplexity", Active: true},
		{File: "specs/alpha_vantage.yaml", Name: "alpha-vantage", EndpointPath: "/alpha-vantage", Active: false},   // Inactive by default
		{File: "specs/google_finance.yml", Name: "google-finance", EndpointPath: "/google-finance", Active: false}, // Inactive by default
		{File: "specs/youtube_transcript.yml", Name: "youtube-transcript", EndpointPath: "/youtube-transcript", Active: true},
	}

	fmt.Printf("Auto-seeding database with %d predefined specs...\n", len(specs))

	imported := 0
	for _, specConfig := range specs {
		// Check if file exists
		if _, err := os.Stat(specConfig.File); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: Spec file not found: %s\n", specConfig.File)
			continue
		}

		// Import the spec
		err := specLoader.ImportSpecFromFile(specConfig.File, specConfig.Name, specConfig.EndpointPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to import %s: %v\n", specConfig.File, err)
			continue
		}

		status := "active"
		if !specConfig.Active {
			// Deactivate the spec
			specs, err := specLoader.GetAllSpecs()
			if err == nil {
				for _, spec := range specs {
					if spec.Name == specConfig.Name {
						specLoader.DeactivateSpec(spec.ID)
						status = "inactive"
						break
					}
				}
			}
		}

		fmt.Printf("✓ Imported %s as '%s' (%s) with endpoint '%s'\n",
			specConfig.File, specConfig.Name, status, specConfig.EndpointPath)
		imported++
	}

	fmt.Printf("\nAuto-seeding completed: %d specs imported successfully\n", imported)

	if imported > 0 {
		fmt.Println("\nTo view imported specs, run:")
		fmt.Println("  ./bin/spec-manager list")
		fmt.Println("\nTo see only active specs, run:")
		fmt.Println("  ./bin/spec-manager active")
		fmt.Println("\nTo start the server with database specs:")
		fmt.Printf("  export DATABASE_URL=\"%s\"\n", os.Getenv("DATABASE_URL"))
		fmt.Println("  ./bin/openapi-mcp")
	}
}
