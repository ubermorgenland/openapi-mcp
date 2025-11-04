package server

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

// Config holds server configuration
type Config struct {
	DatabaseMode bool
	HTTPMode     bool
	HTTPAddr     string
	DatabaseURL  string
	Port         int
	SpecFiles    []string
	RequiredEnvVars map[string]string
}

// LoadConfig loads configuration from environment variables and command line arguments
func LoadConfig(args []string) (*Config, error) {
	config := &Config{
		RequiredEnvVars: make(map[string]string),
	}

	// Check for database mode
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.DatabaseMode = true
		config.DatabaseURL = dbURL
		log.Println("Database mode enabled")
	}

	// Check for HTTP mode
	httpAddr := ""
	for i, arg := range args {
		if arg == "--http" && i+1 < len(args) {
			httpAddr = args[i+1]
			break
		}
	}
	if httpAddr != "" {
		config.HTTPMode = true
		config.HTTPAddr = httpAddr
		log.Printf("HTTP mode enabled on %s", httpAddr)
	}

	// Parse port for HTTP mode
	if config.HTTPMode && config.HTTPAddr != "" {
		if config.HTTPAddr[0] == ':' {
			if port, err := strconv.Atoi(config.HTTPAddr[1:]); err == nil {
				config.Port = port
			}
		}
	}

	// In file mode, collect spec files from arguments
	if !config.DatabaseMode {
		for _, arg := range args {
			if arg != "--http" && !IsHTTPAddress(arg) {
				config.SpecFiles = append(config.SpecFiles, arg)
			}
		}
	}

	return config, nil
}

// IsHTTPAddress checks if a string looks like an HTTP address
func IsHTTPAddress(s string) bool {
	return s[0] == ':' || (len(s) > 0 && s[0] >= '0' && s[0] <= '9')
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.DatabaseMode && c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required for database mode")
	}

	if !c.DatabaseMode && len(c.SpecFiles) == 0 {
		return fmt.Errorf("no OpenAPI spec files provided")
	}

	return nil
}

// LogConfiguration logs the current configuration
func (c *Config) LogConfiguration() {
	if c.DatabaseMode {
		log.Println("Running in database mode")
		log.Printf("Database URL: %s", maskSensitive(c.DatabaseURL))
	} else {
		log.Printf("Running in file mode with %d spec files", len(c.SpecFiles))
	}

	if c.HTTPMode {
		log.Printf("HTTP server will start on %s", c.HTTPAddr)
	}
}

// maskSensitive masks sensitive parts of URLs for logging
func maskSensitive(url string) string {
	if len(url) > 20 {
		return url[:8] + "***" + url[len(url)-8:]
	}
	return "***"
}