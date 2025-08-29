package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/database"
	"github.com/ubermorgenland/openapi-mcp/pkg/mcp/server"
	"github.com/ubermorgenland/openapi-mcp/pkg/models"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
	"github.com/ubermorgenland/openapi-mcp/pkg/services"
)

// Spec management request/response types
type ImportSpecRequest struct {
	Name         string `json:"name"`
	EndpointPath string `json:"endpoint_path"`
	SpecContent  string `json:"spec_content"`
	FileFormat   string `json:"file_format,omitempty"`   // "json" or "yaml", auto-detected if not provided
	ApiKeyToken  string `json:"api_key_token,omitempty"` // API key for this specific spec
	Active       *bool  `json:"active,omitempty"`        // defaults to true if not provided
}

type UpdateSpecRequest struct {
	Name         string `json:"name,omitempty"`
	EndpointPath string `json:"endpoint_path,omitempty"`
	SpecContent  string `json:"spec_content,omitempty"`
	FileFormat   string `json:"file_format,omitempty"`
	ApiKeyToken  string `json:"api_key_token,omitempty"`
	Active       *bool  `json:"active,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// extractAuthSchemeFromSpec extracts authentication scheme information from the OpenAPI spec
func extractAuthSchemeFromSpec(doc *openapi3.T) (string, string, string) {
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		return "", "", ""
	}

	// Look for the first security scheme
	for schemeName, schemeRef := range doc.Components.SecuritySchemes {
		if schemeRef.Value != nil {
			switch schemeRef.Value.Type {
			case "apiKey":
				// Return the scheme name and the location/name info
				location := "header"
				if schemeRef.Value.In == "query" {
					location = "query"
				}
				return schemeName, "apiKey", location + ":" + schemeRef.Value.Name
			case "http":
				switch schemeRef.Value.Scheme {
				case "bearer":
					return schemeName, "bearer", "header:Authorization"
				case "basic":
					return schemeName, "basic", "header:Authorization"
				}
			}
		}
	}
	return "", "", ""
}

// Global variables for dynamic reloading
var (
	// specLookup is a global map to store endpoint -> spec mapping for API key token lookup
	specLookup map[string]*models.OpenAPISpec

	// Dynamic reloading state
	globalMux      *http.ServeMux
	reloadMux      sync.RWMutex
	lastSpecHash   string
	pollingEnabled bool
	specLoader     *services.SpecLoaderService
)

// SpecReloadResponse represents the response from reload endpoint
type SpecReloadResponse struct {
	Success      bool     `json:"success"`
	Message      string   `json:"message"`
	ReloadedAPIs []string `json:"reloaded_apis,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// customAuthContextFunc extends the default authContextFunc to handle API key headers dynamically
// Now uses database spec api_key_token when available
func customAuthContextFunc(ctx context.Context, r *http.Request, doc *openapi3.T, spec *models.OpenAPISpec) context.Context {
	// Determine the endpoint dynamically from the request path
	endpoint := ""
	path := strings.Trim(r.URL.Path, "/")
	if path != "" {
		// Split by "/" and take the first part as the endpoint
		parts := strings.Split(path, "/")
		if len(parts) > 0 && parts[0] != "" {
			// Convert to uppercase for environment variable naming convention
			endpoint = strings.ToUpper(parts[0])
		}
	}

	// Set up global authentication environment variables dynamically for this request
	setupGlobalAuth(endpoint)

	// Extract authentication scheme from the OpenAPI spec
	_, authType, authPath := extractAuthSchemeFromSpec(doc)

	// Save original environment values to restore them later
	origAPIKey := os.Getenv("API_KEY")
	origBearerToken := os.Getenv("BEARER_TOKEN")
	origBasicAuth := os.Getenv("BASIC_AUTH")

	// Save endpoint-specific API key if it exists
	var origEndpointAPIKey string
	if endpoint != "" {
		origEndpointAPIKey = os.Getenv(strings.ToUpper(endpoint) + "_API_KEY")
	}

	// PRIORITY 1: Use database spec api_key_token if available
	if spec != nil && spec.ApiKeyToken != nil && *spec.ApiKeyToken != "" {
		// Determine authentication type from OpenAPI spec
		_, authType, _ := extractAuthSchemeFromSpec(doc)

		switch authType {
		case "bearer":
			// Use api_key_token as Bearer token
			os.Setenv("BEARER_TOKEN", *spec.ApiKeyToken)
			log.Printf("Using database token as Bearer token for %s: %s", endpoint, (*spec.ApiKeyToken)[:10]+"...")
		case "apiKey":
			// Use api_key_token as API key
			os.Setenv("API_KEY", *spec.ApiKeyToken)
			log.Printf("Using database token as API key for %s: %s", endpoint, (*spec.ApiKeyToken)[:10]+"...")
		case "basic":
			// Use api_key_token as Basic auth
			os.Setenv("BASIC_AUTH", *spec.ApiKeyToken)
			log.Printf("Using database token as Basic auth for %s: %s", endpoint, (*spec.ApiKeyToken)[:10]+"...")
		default:
			// Default to API key for unknown types
			os.Setenv("API_KEY", *spec.ApiKeyToken)
			log.Printf("Using database token as API key (default) for %s: %s", endpoint, (*spec.ApiKeyToken)[:10]+"...")
		}
	} else {
		// PRIORITY 2: Load from environment variables as fallback
		if endpoint != "" {
			if envAPIKey := os.Getenv(strings.ToUpper(endpoint) + "_API_KEY"); envAPIKey != "" {
				os.Setenv("API_KEY", envAPIKey)
			}
			if envBearerToken := os.Getenv(strings.ToUpper(endpoint) + "_BEARER_TOKEN"); envBearerToken != "" {
				os.Setenv("BEARER_TOKEN", envBearerToken)
			}
		}

		// PRIORITY 3: Check for general environment variables as final fallback
		if os.Getenv("API_KEY") == "" {
			if generalAPIKey := os.Getenv("GENERAL_API_KEY"); generalAPIKey != "" {
				os.Setenv("API_KEY", generalAPIKey)
			}
		}
		if os.Getenv("BEARER_TOKEN") == "" {
			if generalBearerToken := os.Getenv("GENERAL_BEARER_TOKEN"); generalBearerToken != "" {
				os.Setenv("BEARER_TOKEN", generalBearerToken)
			}
		}
	}

	// Bearer token handling is now done in the priority section above

	// Extract authentication from HTTP headers based on the auth scheme from spec
	if authPath != "" {
		parts := strings.Split(authPath, ":")
		if len(parts) == 2 {
			location := parts[0]
			keyName := parts[1]

			var authValue string
			switch location {
			case "header":
				authValue = r.Header.Get(keyName)
			case "query":
				authValue = r.URL.Query().Get(keyName)
			}

			if authValue != "" {
				switch authType {
				case "apiKey":
					if endpoint != "" {
						os.Setenv(strings.ToUpper(endpoint)+"_API_KEY", authValue)
					}
					os.Setenv("API_KEY", authValue)
				case "bearer":
					// Extract bearer token from Authorization header
					if len(authValue) >= 7 && authValue[:7] == "Bearer " {
						bearerToken := authValue[7:]
						if endpoint != "" {
							os.Setenv(strings.ToUpper(endpoint)+"_BEARER_TOKEN", bearerToken)
						}
						os.Setenv("BEARER_TOKEN", bearerToken)
					}
				case "basic":
					// Extract basic auth from Authorization header
					if len(authValue) > 6 && authValue[:6] == "Basic " {
						basicAuth := authValue[6:]
						if endpoint != "" {
							os.Setenv(strings.ToUpper(endpoint)+"_BASIC_AUTH", basicAuth)
						}
						os.Setenv("BASIC_AUTH", basicAuth)
					}
				}
			}
		}
	} else {
		// Fallback to common API key header patterns if spec doesn't define them
		switch {
		case r.Header.Get("X-API-Key") != "":
			apiKey := r.Header.Get("X-API-Key")
			if endpoint != "" {
				os.Setenv(endpoint+"_API_KEY", apiKey)
			}
			os.Setenv("API_KEY", apiKey)
		case r.Header.Get("Api-Key") != "":
			apiKey := r.Header.Get("Api-Key")
			if endpoint != "" {
				os.Setenv(strings.ToUpper(endpoint)+"_API_KEY", apiKey)
			}
			os.Setenv("API_KEY", apiKey)
		case r.Header.Get("x-rapidapi-key") != "":
			// Handle Twitter API's specific header
			apiKey := r.Header.Get("x-rapidapi-key")
			if endpoint != "" {
				os.Setenv(strings.ToUpper(endpoint)+"_API_KEY", apiKey)
			}
			os.Setenv("API_KEY", apiKey)
		}
	}

	// Check for existing environment variables as defaults
	if os.Getenv("BEARER_TOKEN") == "" {
		if generalBearerToken := os.Getenv("GENERAL_BEARER_TOKEN"); generalBearerToken != "" {
			os.Setenv("BEARER_TOKEN", generalBearerToken)
		}
	}
	if os.Getenv("BASIC_AUTH") == "" {
		if generalBasicAuth := os.Getenv("GENERAL_BASIC_AUTH"); generalBasicAuth != "" {
			os.Setenv("BASIC_AUTH", generalBasicAuth)
		}
	}

	// Handle Authorization header (only override if not already set from env)
	if bearerToken := r.Header.Get("Authorization"); bearerToken != "" {
		switch {
		case len(bearerToken) >= 7 && bearerToken[:7] == "Bearer ":
			if os.Getenv("BEARER_TOKEN") == "" {
				os.Setenv("BEARER_TOKEN", bearerToken[7:])
			}
		case len(bearerToken) >= 6 && bearerToken[:6] == "Basic ":
			if os.Getenv("BASIC_AUTH") == "" {
				os.Setenv("BASIC_AUTH", bearerToken[6:])
			}
		}
	}

	// Create a context that restores the original environment when done
	return &authContext{
		Context:            ctx,
		origAPIKey:         origAPIKey,
		origBearerToken:    origBearerToken,
		origBasicAuth:      origBasicAuth,
		endpoint:           endpoint,
		origEndpointAPIKey: origEndpointAPIKey,
	}
}

// authContext wraps a context and restores original environment variables when done
type authContext struct {
	context.Context
	origAPIKey         string
	origBearerToken    string
	origBasicAuth      string
	endpoint           string
	origEndpointAPIKey string
}

// Done restores the original environment variables when the context is done
func (c *authContext) Done() <-chan struct{} {
	done := c.Context.Done()
	if done != nil {
		go func() {
			<-done
			c.restoreEnv()
		}()
	}
	return done
}

func (c *authContext) restoreEnv() {
	if c.origAPIKey != "" {
		os.Setenv("API_KEY", c.origAPIKey)
	} else {
		os.Unsetenv("API_KEY")
	}
	if c.origBearerToken != "" {
		os.Setenv("BEARER_TOKEN", c.origBearerToken)
	} else {
		os.Unsetenv("BEARER_TOKEN")
	}
	if c.origBasicAuth != "" {
		os.Setenv("BASIC_AUTH", c.origBasicAuth)
	} else {
		os.Unsetenv("BASIC_AUTH")
	}
	if c.endpoint != "" {
		if c.origEndpointAPIKey != "" {
			os.Setenv(strings.ToUpper(c.endpoint)+"_API_KEY", c.origEndpointAPIKey)
		} else {
			os.Unsetenv(strings.ToUpper(c.endpoint) + "_API_KEY")
		}
	}
}

// setupGlobalAuthAtStartup sets up global environment variables for authentication at server startup
func setupGlobalAuthAtStartup() {
	log.Printf("Setting up global authentication environment variables at startup...")

	// Dynamically check for any endpoint bearer tokens and set the first one found globally
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]

			// Look for any *_BEARER_TOKEN environment variable (except GENERAL_BEARER_TOKEN)
			if strings.HasSuffix(key, "_BEARER_TOKEN") && key != "GENERAL_BEARER_TOKEN" && value != "" {
				os.Setenv("BEARER_TOKEN", value)
				log.Printf("Set global BEARER_TOKEN from %s: %s", key, value[:10]+"...")
				break // Use the first one found
			}
		}
	}

	// Check for general bearer token and set it globally if no specific one was found
	if os.Getenv("BEARER_TOKEN") == "" {
		if generalToken := os.Getenv("GENERAL_BEARER_TOKEN"); generalToken != "" {
			os.Setenv("BEARER_TOKEN", generalToken)
			log.Printf("Set global BEARER_TOKEN from GENERAL_BEARER_TOKEN: %s", generalToken[:10]+"...")
		}
	}

	// Dynamically check for any endpoint API keys and set the first one found globally
	if os.Getenv("API_KEY") == "" {
		for _, envVar := range os.Environ() {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]

				// Look for any *_API_KEY environment variable (except GENERAL_API_KEY)
				if strings.HasSuffix(key, "_API_KEY") && key != "GENERAL_API_KEY" && value != "" {
					os.Setenv("API_KEY", value)
					log.Printf("Set global API_KEY from %s: %s", key, value[:10]+"...")
					break // Use the first one found
				}
			}
		}
	}

	// Check for general API key and set it globally if no specific one was found
	if os.Getenv("API_KEY") == "" {
		if generalAPIKey := os.Getenv("GENERAL_API_KEY"); generalAPIKey != "" {
			os.Setenv("API_KEY", generalAPIKey)
			log.Printf("Set global API_KEY from GENERAL_API_KEY: %s", generalAPIKey[:10]+"...")
		}
	}

	// Log final state
	if bearerToken := os.Getenv("BEARER_TOKEN"); bearerToken != "" {
		log.Printf("Final BEARER_TOKEN set: %s", bearerToken[:10]+"...")
	} else {
		log.Printf("WARNING: No BEARER_TOKEN set at startup")
	}

	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		log.Printf("Final API_KEY set: %s", apiKey[:10]+"...")
	} else {
		log.Printf("WARNING: No API_KEY set at startup")
	}
}

// setupGlobalAuth sets up global environment variables for authentication dynamically
func setupGlobalAuth(endpoint string) {
	// Set endpoint-specific bearer token if available (ALWAYS override)
	if endpoint != "" {
		if endpointToken := os.Getenv(strings.ToUpper(endpoint) + "_BEARER_TOKEN"); endpointToken != "" {
			os.Setenv("BEARER_TOKEN", endpointToken)
			log.Printf("Set global BEARER_TOKEN from %s_BEARER_TOKEN: %s", strings.ToUpper(endpoint), endpointToken[:10]+"...")
		}

		if endpointAPIKey := os.Getenv(strings.ToUpper(endpoint) + "_API_KEY"); endpointAPIKey != "" {
			os.Setenv("API_KEY", endpointAPIKey)
			log.Printf("Set global API_KEY from %s_API_KEY: %s", strings.ToUpper(endpoint), endpointAPIKey[:10]+"...")
		}
	}

	// Set general bearer token if no endpoint-specific one was set
	if os.Getenv("BEARER_TOKEN") == "" {
		if generalToken := os.Getenv("GENERAL_BEARER_TOKEN"); generalToken != "" {
			os.Setenv("BEARER_TOKEN", generalToken)
			log.Printf("Set global BEARER_TOKEN from GENERAL_BEARER_TOKEN: %s", generalToken[:10]+"...")
		}
	}

	// Set general API key if no endpoint-specific one was set
	if os.Getenv("API_KEY") == "" {
		if generalAPIKey := os.Getenv("GENERAL_API_KEY"); generalAPIKey != "" {
			os.Setenv("API_KEY", generalAPIKey)
			log.Printf("Set global API_KEY from GENERAL_API_KEY: %s", generalAPIKey[:10]+"...")
		}
	}

	// Log current state for debugging
	bearerToken := os.Getenv("BEARER_TOKEN")
	apiKey := os.Getenv("API_KEY")

	bearerDisplay := "NOT_SET"
	if len(bearerToken) > 0 {
		if len(bearerToken) > 10 {
			bearerDisplay = bearerToken[:10] + "..."
		} else {
			bearerDisplay = bearerToken
		}
	}

	apiKeyDisplay := "NOT_SET"
	if len(apiKey) > 0 {
		if len(apiKey) > 10 {
			apiKeyDisplay = apiKey[:10] + "..."
		} else {
			apiKeyDisplay = apiKey
		}
	}

	log.Printf("Current environment: BEARER_TOKEN=%s, API_KEY=%s", bearerDisplay, apiKeyDisplay)
}

// getEndpointFromFilename converts a filename to an endpoint URL path
func getEndpointFromFilename(filename string) string {
	// Remove file extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	// Replace underscores with hyphens
	return strings.ReplaceAll(name, "_", "-")
}

// loadSpecsFromDatabase loads specs from database and returns them with a hash for change detection
func loadSpecsFromDatabase() ([]*models.OpenAPISpec, string, error) {
	if specLoader == nil {
		return nil, "", fmt.Errorf("spec loader not initialized")
	}

	specs, err := specLoader.GetActiveSpecs()
	if err != nil {
		return nil, "", err
	}

	// Create hash of specs for change detection
	hash := fmt.Sprintf("%d", len(specs))
	for _, spec := range specs {
		hash += fmt.Sprintf("-%d-%s", spec.ID, spec.Name)
		if spec.ApiKeyToken != nil {
			hash += fmt.Sprintf("-%d", len(*spec.ApiKeyToken))
		}
	}

	return specs, hash, nil
}

// createSpecEndpoints creates HTTP endpoints for the given specs
func createSpecEndpoints(specs []*models.OpenAPISpec) ([]string, error) {
	reloadMux.Lock()
	defer reloadMux.Unlock()

	// Clear existing spec lookup
	specLookup = make(map[string]*models.OpenAPISpec)

	// Create new mux
	newMux := http.NewServeMux()

	// Add health endpoint
	newMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Add reload endpoint
	newMux.HandleFunc("/reload", handleReload)

	// Add swagger endpoint
	newMux.HandleFunc("/swagger", handleSwagger)

	// Set up CORS middleware
	corsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next(w, r)
		}
	}

	// Add spec management endpoints
	newMux.HandleFunc("/specs", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			handleGetSpecs(w, r)
		case "POST":
			handleCreateSpec(w, r)
		default:
			writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	newMux.HandleFunc("/specs/active", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleGetActiveSpecs(w, r)
	}))

	newMux.HandleFunc("/specs/", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		// Extract ID from path
		path := strings.TrimPrefix(r.URL.Path, "/specs/")
		if path == "" {
			writeErrorResponse(w, "Spec ID required", http.StatusBadRequest)
			return
		}

		// Handle /specs/{id}/activate, /specs/{id}/deactivate, and /specs/{id}/token
		parts := strings.Split(path, "/")
		if len(parts) == 2 {
			id, err := strconv.Atoi(parts[0])
			if err != nil {
				writeErrorResponse(w, "Invalid spec ID", http.StatusBadRequest)
				return
			}

			switch parts[1] {
			case "activate":
				if r.Method != "POST" {
					writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				handleActivateSpec(w, r, id)
				return
			case "deactivate":
				if r.Method != "POST" {
					writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				handleDeactivateSpec(w, r, id)
				return
			case "token":
				if r.Method != "PUT" {
					writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				handleUpdateApiKeyToken(w, r, id)
				return
			}
		}

		// Handle /specs/{id} operations
		id, err := strconv.Atoi(parts[0])
		if err != nil {
			writeErrorResponse(w, "Invalid spec ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case "GET":
			handleGetSpec(w, r, id)
		case "PUT":
			handleUpdateSpec(w, r, id)
		case "DELETE":
			handleDeleteSpec(w, r, id)
		default:
			writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	var mountedAPIs []string

	// Process each database spec
	for _, spec := range specs {
		endpoint := strings.TrimPrefix(spec.EndpointPath, "/")

		// Store spec in lookup for auth context
		specLookup[endpoint] = spec

		log.Printf("Loading database spec: %s -> endpoint: /%s", spec.Name, endpoint)

		// Parse spec content to get OpenAPI doc
		loader := openapi3.NewLoader()
		doc, err := loader.LoadFromData([]byte(spec.SpecContent))
		if err != nil {
			log.Printf("Failed to parse spec content for %s: %v", spec.Name, err)
			continue
		}

		// Log the authentication info
		schemeName, authType, authPath := extractAuthSchemeFromSpec(doc)
		if authPath != "" {
			log.Printf("%s API: Found security scheme '%s' with %s authentication: %s", endpoint, schemeName, authType, authPath)
			// Show database token status and how it will be used
			if spec.ApiKeyToken != nil && *spec.ApiKeyToken != "" {
				switch authType {
				case "bearer":
					log.Printf("%s API: Will use database token as BEARER TOKEN (length: %d)", endpoint, len(*spec.ApiKeyToken))
				case "apiKey":
					log.Printf("%s API: Will use database token as API KEY (length: %d)", endpoint, len(*spec.ApiKeyToken))
				case "basic":
					log.Printf("%s API: Will use database token as BASIC AUTH (length: %d)", endpoint, len(*spec.ApiKeyToken))
				default:
					log.Printf("%s API: Will use database token as API KEY - default (length: %d)", endpoint, len(*spec.ApiKeyToken))
				}
			} else {
				log.Printf("%s API: No token in database, will use environment variables for %s auth", endpoint, authType)
			}
		}

		// Create MCP server - don't set auth env vars here, let the context function handle it
		srv := openapi2mcp.NewServer(doc.Info.Title, doc.Info.Version, doc)

		// Create a custom StreamableHTTPServer with database spec-aware auth function
		streamableServer := server.NewStreamableHTTPServer(srv,
			server.WithEndpointPath("/"+endpoint),
			server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
				return customAuthContextFunc(ctx, r, doc, spec)
			}),
		)

		// Mount the server at the endpoint path
		newMux.Handle("/"+endpoint, streamableServer)
		newMux.Handle("/"+endpoint+"/", streamableServer)

		log.Printf("Mounted %s API at /%s", doc.Info.Title, endpoint)
		mountedAPIs = append(mountedAPIs, endpoint)
	}

	// Replace global mux
	globalMux = newMux

	return mountedAPIs, nil
}

// handleSwagger serves the OpenAPI specification for this server
func handleSwagger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Look for swagger file
	swaggerPaths := []string{
		"spec-api-swagger.json",
		"bin/spec-api-swagger.json",
		"./spec-api-swagger.json",
	}

	var swaggerContent []byte
	var swaggerFound bool

	for _, path := range swaggerPaths {
		if content, err := os.ReadFile(path); err == nil {
			swaggerContent = content
			swaggerFound = true
			break
		}
	}

	if !swaggerFound {
		// Create a basic swagger spec for the dynamic reloading server
		basicSwagger := map[string]interface{}{
			"openapi": "3.0.0",
			"info": map[string]interface{}{
				"title":       "OpenAPI MCP Dynamic Server",
				"version":     "1.0.0",
				"description": "Dynamic OpenAPI MCP server with database-driven spec loading and intelligent authentication",
			},
			"servers": []map[string]interface{}{
				{"url": "http://localhost:8080", "description": "Local development server"},
			},
			"paths": map[string]interface{}{
				"/health": map[string]interface{}{
					"get": map[string]interface{}{
						"summary":     "Health check",
						"description": "Returns OK if server is running",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Server is healthy",
								"content": map[string]interface{}{
									"text/plain": map[string]interface{}{
										"schema": map[string]interface{}{
											"type":    "string",
											"example": "OK",
										},
									},
								},
							},
						},
					},
				},
				"/reload": map[string]interface{}{
					"post": map[string]interface{}{
						"summary":     "Reload OpenAPI specs",
						"description": "Manually trigger reload of OpenAPI specs from database",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Reload status",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]interface{}{
											"type": "object",
											"properties": map[string]interface{}{
												"success": map[string]interface{}{
													"type": "boolean",
												},
												"message": map[string]interface{}{
													"type": "string",
												},
												"reloaded_apis": map[string]interface{}{
													"type": "array",
													"items": map[string]interface{}{
														"type": "string",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				"/swagger": map[string]interface{}{
					"get": map[string]interface{}{
						"summary":     "Get OpenAPI specification",
						"description": "Returns the OpenAPI specification for this server",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "OpenAPI specification",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]interface{}{
											"type": "object",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(basicSwagger)
		return
	}

	var swaggerSpec map[string]interface{}
	if err := json.Unmarshal(swaggerContent, &swaggerSpec); err != nil {
		http.Error(w, "Invalid swagger specification format", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(swaggerSpec)
}

// handleReload handles HTTP reload requests
func handleReload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != "POST" {
		response := SpecReloadResponse{
			Success: false,
			Error:   "Method not allowed. Use POST.",
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Reload requested via HTTP endpoint")

	// Load specs from database
	specs, newHash, err := loadSpecsFromDatabase()
	if err != nil {
		response := SpecReloadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to load specs from database: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if specs have changed
	if newHash == lastSpecHash {
		response := SpecReloadResponse{
			Success: true,
			Message: "No changes detected in database specs",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Reload endpoints
	mountedAPIs, err := createSpecEndpoints(specs)
	if err != nil {
		response := SpecReloadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create spec endpoints: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	lastSpecHash = newHash

	response := SpecReloadResponse{
		Success:      true,
		Message:      fmt.Sprintf("Successfully reloaded %d API specs", len(mountedAPIs)),
		ReloadedAPIs: mountedAPIs,
	}

	log.Printf("Successfully reloaded %d API specs: %v", len(mountedAPIs), mountedAPIs)
	json.NewEncoder(w).Encode(response)
}

// Spec management handler functions
func writeErrorResponse(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}

func writeSuccessResponse(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func handleGetSpecs(w http.ResponseWriter, r *http.Request) {
	if specLoader == nil {
		writeErrorResponse(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	specs, err := specLoader.GetAllSpecs()
	if err != nil {
		writeErrorResponse(w, "Failed to get specs", http.StatusInternalServerError)
		return
	}

	writeSuccessResponse(w, "Specs retrieved successfully", specs)
}

func handleGetActiveSpecs(w http.ResponseWriter, r *http.Request) {
	if specLoader == nil {
		writeErrorResponse(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	specs, err := specLoader.GetActiveSpecs()
	if err != nil {
		writeErrorResponse(w, "Failed to get active specs", http.StatusInternalServerError)
		return
	}

	writeSuccessResponse(w, "Active specs retrieved successfully", specs)
}

func handleCreateSpec(w http.ResponseWriter, r *http.Request) {
	if specLoader == nil {
		writeErrorResponse(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	var req ImportSpecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeErrorResponse(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.EndpointPath == "" {
		writeErrorResponse(w, "Endpoint path is required", http.StatusBadRequest)
		return
	}
	if req.SpecContent == "" {
		writeErrorResponse(w, "Spec content is required", http.StatusBadRequest)
		return
	}

	// Auto-detect format if not provided
	if req.FileFormat == "" {
		if strings.HasPrefix(strings.TrimSpace(req.SpecContent), "{") {
			req.FileFormat = "json"
		} else {
			req.FileFormat = "yaml"
		}
	}

	// Set default active status
	if req.Active == nil {
		active := true
		req.Active = &active
	}

	// Convert API key token
	var apiKeyToken *string
	if req.ApiKeyToken != "" {
		apiKeyToken = &req.ApiKeyToken
	}

	// Create spec directly from content
	if err := specLoader.CreateSpecFromContent(req.Name, req.EndpointPath, req.SpecContent, req.FileFormat, apiKeyToken); err != nil {
		writeErrorResponse(w, fmt.Sprintf("Failed to create spec: %v", err), http.StatusBadRequest)
		return
	}

	// If requested as inactive, deactivate it
	if !*req.Active {
		specs, err := specLoader.GetAllSpecs()
		if err == nil {
			for _, spec := range specs {
				if spec.Name == req.Name {
					specLoader.DeactivateSpec(spec.ID)
					break
				}
			}
		}
	}

	writeSuccessResponse(w, "Spec imported successfully", map[string]interface{}{
		"name":          req.Name,
		"endpoint_path": req.EndpointPath,
		"active":        *req.Active,
		"has_api_token": apiKeyToken != nil,
	})
}

func handleGetSpec(w http.ResponseWriter, r *http.Request, id int) {
	writeErrorResponse(w, "Get spec by ID not implemented yet", http.StatusNotImplemented)
}

func handleUpdateSpec(w http.ResponseWriter, r *http.Request, id int) {
	writeErrorResponse(w, "Update spec not implemented yet", http.StatusNotImplemented)
}

func handleDeleteSpec(w http.ResponseWriter, r *http.Request, id int) {
	if specLoader == nil {
		writeErrorResponse(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	if err := specLoader.DeleteSpec(id); err != nil {
		writeErrorResponse(w, fmt.Sprintf("Failed to delete spec: %v", err), http.StatusBadRequest)
		return
	}

	writeSuccessResponse(w, "Spec deleted successfully", map[string]int{"id": id})
}

func handleActivateSpec(w http.ResponseWriter, r *http.Request, id int) {
	if specLoader == nil {
		writeErrorResponse(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	if err := specLoader.ActivateSpec(id); err != nil {
		writeErrorResponse(w, fmt.Sprintf("Failed to activate spec: %v", err), http.StatusBadRequest)
		return
	}

	writeSuccessResponse(w, "Spec activated successfully", map[string]int{"id": id})
}

func handleDeactivateSpec(w http.ResponseWriter, r *http.Request, id int) {
	if specLoader == nil {
		writeErrorResponse(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	if err := specLoader.DeactivateSpec(id); err != nil {
		writeErrorResponse(w, fmt.Sprintf("Failed to deactivate spec: %v", err), http.StatusBadRequest)
		return
	}

	writeSuccessResponse(w, "Spec deactivated successfully", map[string]int{"id": id})
}

func handleUpdateApiKeyToken(w http.ResponseWriter, r *http.Request, id int) {
	if specLoader == nil {
		writeErrorResponse(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ApiKeyToken *string `json:"api_key_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := specLoader.UpdateApiKeyToken(id, req.ApiKeyToken); err != nil {
		writeErrorResponse(w, fmt.Sprintf("Failed to update API key token: %v", err), http.StatusBadRequest)
		return
	}

	writeSuccessResponse(w, "API key token updated successfully", map[string]interface{}{
		"id":                    id,
		"api_key_token_updated": true,
	})
}

// startDatabasePolling starts a goroutine that polls the database for changes
func startDatabasePolling(intervalSeconds int) {
	if !pollingEnabled {
		log.Printf("Database polling disabled")
		return
	}

	log.Printf("Starting database polling every %d seconds", intervalSeconds)

	go func() {
		ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Load specs from database
			specs, newHash, err := loadSpecsFromDatabase()
			if err != nil {
				log.Printf("Database polling error: %v", err)
				continue
			}

			// Check if specs have changed
			if newHash != lastSpecHash {
				log.Printf("Database changes detected, reloading specs...")

				// Reload endpoints
				mountedAPIs, err := createSpecEndpoints(specs)
				if err != nil {
					log.Printf("Failed to reload specs during polling: %v", err)
					continue
				}

				lastSpecHash = newHash
				log.Printf("Automatically reloaded %d API specs: %v", len(mountedAPIs), mountedAPIs)
			}
		}
	}()
}

func main() {
	// Set up global environment variables for authentication at startup
	setupGlobalAuthAtStartup()

	// Initialize specLookup map
	specLookup = make(map[string]*models.OpenAPISpec)

	// Check for configuration environment variables
	pollingInterval := 30 // Default 30 seconds
	if intervalStr := os.Getenv("POLLING_INTERVAL"); intervalStr != "" {
		if interval, err := fmt.Sscanf(intervalStr, "%d", &pollingInterval); err == nil && interval == 1 && pollingInterval > 0 {
			// Use provided interval
		} else {
			log.Printf("Invalid POLLING_INTERVAL '%s', using default %d seconds", intervalStr, pollingInterval)
		}
	}

	// Enable polling by default if DATABASE_URL is set
	pollingEnabled = os.Getenv("DATABASE_URL") != ""
	if os.Getenv("DISABLE_POLLING") == "true" {
		pollingEnabled = false
	}

	// Track required environment variables
	requiredEnvVars := make(map[string]string)

	// Try to load from database first
	if os.Getenv("DATABASE_URL") != "" {
		log.Printf("DATABASE_URL found, attempting to load specs from database...")

		if err := database.InitializeDatabase(); err != nil {
			log.Printf("Failed to initialize database: %v, falling back to file loading", err)
		} else {
			specLoader = services.NewSpecLoaderService(database.DB)
			specs, hash, err := loadSpecsFromDatabase()
			if err != nil {
				log.Printf("Failed to get active specs from database: %v, falling back to file loading", err)
			} else if len(specs) > 0 {
				log.Printf("Successfully loaded %d active specs from database", len(specs))

				// Create initial endpoints
				mountedAPIs, err := createSpecEndpoints(specs)
				if err != nil {
					log.Fatalf("Failed to create spec endpoints: %v", err)
				}

				lastSpecHash = hash
				log.Printf("Initial load complete. Mounted APIs: %v", mountedAPIs)

				// Start database polling for automatic reload
				startDatabasePolling(pollingInterval)

				// Create HTTP server with dynamic handler
				srv := &http.Server{
					Addr: ":8080",
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						reloadMux.RLock()
						mux := globalMux
						reloadMux.RUnlock()

						if mux != nil {
							mux.ServeHTTP(w, r)
						} else {
							http.Error(w, "Server not ready", http.StatusServiceUnavailable)
						}
					}),
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 60 * time.Second,
				}

				log.Printf("Starting dynamic database-driven server on %s", srv.Addr)
				log.Printf("Available endpoints:")
				log.Printf("  POST   /reload                  - Reload specs from database")
				log.Printf("  GET    /health                  - Health check")
				log.Printf("  GET    /swagger                 - OpenAPI specification")
				log.Printf("  GET    /specs                   - List all specs")
				log.Printf("  POST   /specs                   - Create new spec")
				log.Printf("  GET    /specs/active            - List active specs")
				log.Printf("  GET    /specs/{id}              - Get spec by ID")
				log.Printf("  PUT    /specs/{id}              - Update spec")
				log.Printf("  DELETE /specs/{id}              - Delete spec")
				log.Printf("  POST   /specs/{id}/activate     - Activate spec")
				log.Printf("  POST   /specs/{id}/deactivate   - Deactivate spec")
				log.Printf("  PUT    /specs/{id}/token        - Update API key token")
				for _, api := range mountedAPIs {
					log.Printf("  *      /%s                   - %s API", api, api)
				}
				if pollingEnabled {
					log.Printf("🔄 Database polling enabled (every %d seconds)", pollingInterval)
					log.Printf("   Set DISABLE_POLLING=true to disable automatic polling")
				} else {
					log.Printf("📋 Database polling disabled")
					log.Printf("   Use POST /reload to manually reload specs")
				}

				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Fatalf("HTTP server error: %v", err)
				}
				return
			}
		}
	}

	log.Printf("No DATABASE_URL or no database specs found, falling back to file loading...")

	specsDir := "./specs"
	mux := http.NewServeMux()

	// Get all spec files from the specs directory
	specFiles, err := filepath.Glob(filepath.Join(specsDir, "*"))
	if err != nil {
		log.Fatalf("Failed to read specs directory: %v", err)
	}

	if len(specFiles) == 0 {
		log.Fatalf("No spec files found in %s", specsDir)
	}

	// Process each spec file (fallback mode)
	for _, specFile := range specFiles {
		// Skip directories
		if info, err := os.Stat(specFile); err != nil || info.IsDir() {
			continue
		}

		// Get the filename for endpoint creation
		filename := filepath.Base(specFile)
		endpoint := getEndpointFromFilename(filename)

		log.Printf("Loading spec: %s -> endpoint: /%s", filename, endpoint)

		// Load OpenAPI spec
		doc, err := openapi2mcp.LoadOpenAPISpec(specFile)
		if err != nil {
			log.Printf("Failed to load spec %s: %v", filename, err)
			continue
		}

		// Log the authentication scheme extracted from spec
		schemeName, authType, authPath := extractAuthSchemeFromSpec(doc)
		if authPath != "" {
			log.Printf("%s API: Found security scheme '%s' with %s authentication: %s", endpoint, schemeName, authType, authPath)
			// Add to required environment variables
			switch authType {
			case "apiKey":
				requiredEnvVars[strings.ToUpper(endpoint)+"_API_KEY"] = "API key for " + doc.Info.Title
			case "bearer":
				requiredEnvVars[strings.ToUpper(endpoint)+"_BEARER_TOKEN"] = "Bearer token for " + doc.Info.Title
			case "basic":
				requiredEnvVars[strings.ToUpper(endpoint)+"_BASIC_AUTH"] = "Basic auth for " + doc.Info.Title
			}
		} else {
			log.Printf("%s API: No authentication security scheme found in spec", endpoint)
		}

		// Create MCP server
		bearerToken := os.Getenv("BEARER_TOKEN")
		apiKey := os.Getenv("API_KEY")

		log.Printf("Creating MCP server for %s with BEARER_TOKEN=%s, API_KEY=%s",
			endpoint,
			func() string {
				if bearerToken != "" {
					return bearerToken[:10] + "..."
				}
				return "NOT_SET"
			}(),
			func() string {
				if apiKey != "" {
					return apiKey[:10] + "..."
				}
				return "NOT_SET"
			}())
		srv := openapi2mcp.NewServer(doc.Info.Title, doc.Info.Version, doc)

		// Create a custom StreamableHTTPServer with the package's built-in auth function
		// For file-based loading, pass nil spec to use environment variables
		streamableServer := server.NewStreamableHTTPServer(srv,
			server.WithEndpointPath("/"+endpoint),
			server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
				return customAuthContextFunc(ctx, r, doc, nil)
			}),
		)

		// Mount the server at the endpoint path
		mux.Handle("/"+endpoint, streamableServer)
		mux.Handle("/"+endpoint+"/", streamableServer)

		log.Printf("Mounted %s API at /%s", doc.Info.Title, endpoint)
	}

	// Log required environment variables
	log.Printf("=== REQUIRED ENVIRONMENT VARIABLES ===")
	if len(requiredEnvVars) == 0 {
		log.Printf("No authentication environment variables required")
	} else {
		log.Printf("The following environment variables should be set:")
		for envVar, description := range requiredEnvVars {
			log.Printf("  %s: %s", envVar, description)
		}
		log.Printf("")
		log.Printf("Example usage:")
		for envVar := range requiredEnvVars {
			if strings.Contains(envVar, "API_KEY") {
				log.Printf("  export %s=\"your_API_KEY_here\"", envVar)
			} else if strings.Contains(envVar, "BEARER_TOKEN") {
				log.Printf("  export %s=\"your_BEARER_TOKEN_here\"", envVar)
			} else if strings.Contains(envVar, "BASIC_AUTH") {
				log.Printf("  export %s=\"your_BASIC_AUTH_here\"", envVar)
			}
		}
		log.Printf("")
		log.Printf("You can also set general defaults that will be used if endpoint-specific ones are not set:")
		log.Printf("  export GENERAL_API_KEY=\"your_default_API_KEY_here\"")
		log.Printf("  export GENERAL_BEARER_TOKEN=\"your_default_BEARER_TOKEN_here\"")
		log.Printf("  export GENERAL_BASIC_AUTH=\"your_default_BASIC_AUTH_here\"")
	}
	log.Printf("=====================================")

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	log.Printf("Starting server on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}
