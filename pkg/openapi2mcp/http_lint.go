// http_lint.go
package openapi2mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HTTPLintServer provides HTTP endpoints for OpenAPI validation and linting
type HTTPLintServer struct {
	detailedSuggestions bool
}

// NewHTTPLintServer creates a new HTTP lint server
func NewHTTPLintServer(detailedSuggestions bool) *HTTPLintServer {
	return &HTTPLintServer{
		detailedSuggestions: detailedSuggestions,
	}
}

// setCORSAndCacheHeaders sets CORS and caching headers for API responses
func setCORSAndCacheHeaders(w http.ResponseWriter) {
	// CORS headers - allow access from any origin
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours for preflight cache

	// Caching headers - prevent caching of API responses since they depend on request body
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// HandleLint handles POST requests to lint OpenAPI specs
func (s *HTTPLintServer) HandleLint(w http.ResponseWriter, r *http.Request) {
	// Set CORS and caching headers for all responses
	setCORSAndCacheHeaders(w)

	// Handle preflight OPTIONS requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var req HTTPLintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	if req.OpenAPISpec == "" {
		http.Error(w, "Missing openapi_spec field", http.StatusBadRequest)
		return
	}

	// Parse the OpenAPI spec
	doc, err := LoadOpenAPISpecFromString(req.OpenAPISpec)
	if err != nil {
		result := &LintResult{
			Success:      false,
			ErrorCount:   1,
			WarningCount: 0,
			Issues: []LintIssue{{
				Type:       "error",
				Message:    fmt.Sprintf("Failed to parse OpenAPI spec: %v", err),
				Suggestion: "Ensure the OpenAPI spec is valid YAML or JSON and follows OpenAPI 3.x format.",
			}},
			Summary: "OpenAPI spec parsing failed.",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(result)
		return
	}

	// Perform linting
	result := LintOpenAPISpec(doc, s.detailedSuggestions)

	// Set appropriate HTTP status code
	if result.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	json.NewEncoder(w).Encode(result)
}

// HandleHealth handles GET requests for health checks
func (s *HTTPLintServer) HandleHealth(w http.ResponseWriter, r *http.Request) {
	// Set CORS and caching headers
	setCORSAndCacheHeaders(w)

	// Handle preflight OPTIONS requests
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "openapi-lint",
		"detailed":  s.detailedSuggestions,
	}
	json.NewEncoder(w).Encode(response)
}

// ServeHTTPLint starts an HTTP server for linting OpenAPI specs
func ServeHTTPLint(addr string, detailedSuggestions bool) error {
	server := NewHTTPLintServer(detailedSuggestions)

	mux := http.NewServeMux()
	// Always register both endpoints with different behaviors
	validateServer := NewHTTPLintServer(false) // Basic validation
	lintServer := NewHTTPLintServer(true)      // Detailed linting

	mux.HandleFunc("/validate", validateServer.HandleLint)
	mux.HandleFunc("/lint", lintServer.HandleLint)
	mux.HandleFunc("/health", server.HandleHealth)

	// Add a root handler that shows available endpoints
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Set CORS and caching headers
		setCORSAndCacheHeaders(w)

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		endpoints := map[string]interface{}{
			"service":   "openapi-lint",
			"endpoints": map[string]interface{}{},
			"usage": map[string]interface{}{
				"request_body": map[string]string{
					"openapi_spec": "OpenAPI spec as YAML or JSON string",
				},
				"response": map[string]interface{}{
					"success":       "boolean - whether linting passed",
					"error_count":   "number - count of errors found",
					"warning_count": "number - count of warnings found",
					"issues":        "array - list of issues with details",
					"summary":       "string - summary message",
				},
			},
		}

		endpointsMap := endpoints["endpoints"].(map[string]interface{})
		// Both endpoints are always available
		endpointsMap["POST /validate"] = "Basic OpenAPI validation for critical issues"
		endpointsMap["POST /lint"] = "Comprehensive OpenAPI linting with detailed suggestions"
		endpointsMap["GET /health"] = "Health check endpoint"

		json.NewEncoder(w).Encode(endpoints)
	})

	log.Printf("Starting OpenAPI validation/linting HTTP server on %s (validate & lint endpoints available)", addr)

	return http.ListenAndServe(addr, mux)
}
