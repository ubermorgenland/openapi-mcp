package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/auth"
	"github.com/ubermorgenland/openapi-mcp/pkg/models"
)

// ReloadResponse represents the response from a reload operation
type ReloadResponse struct {
	Success      bool     `json:"success"`
	ReloadedAPIs []string `json:"reloaded_apis,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// HTTPContextFunc defines the function signature for HTTP context modification
type HTTPContextFunc func(context.Context, *http.Request, *openapi3.T, *models.OpenAPISpec) context.Context

// SecureAuthContextFunc creates a secure, request-scoped authentication context without global state mutation
func SecureAuthContextFunc(authStateManager *auth.StateManager) HTTPContextFunc {
	return func(ctx context.Context, r *http.Request, doc *openapi3.T, spec *models.OpenAPISpec) context.Context {
		// Create authentication context for this request
		authCtx := auth.CreateAuthContext(r, doc, spec)

		// If no spec provided, try to get it from state manager
		if spec == nil && authStateManager != nil {
			endpoint := strings.ToLower(strings.Trim(r.URL.Path, "/"))
			if strings.Contains(endpoint, "/") {
				endpoint = strings.Split(endpoint, "/")[0]
			}
			if foundSpec, exists := authStateManager.GetSpec(endpoint); exists {
				// Recreate auth context with the found spec
				authCtx = auth.CreateAuthContext(r, doc, foundSpec)
			}
		}

		// Add auth context to request context - this is secure and thread-safe
		ctx = auth.WithAuthContext(ctx, authCtx)

		return ctx
	}
}

// HandleReload handles the /reload endpoint for reloading API specifications
func HandleReload(reloadFunc func() ([]string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		reloadedAPIs, err := reloadFunc()
		
		response := ReloadResponse{
			Success:      err == nil,
			ReloadedAPIs: reloadedAPIs,
		}
		
		if err != nil {
			response.Error = err.Error()
			log.Printf("Reload failed: %v", err)
		} else {
			log.Printf("Successfully reloaded %d APIs: %v", len(reloadedAPIs), reloadedAPIs)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode reload response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// HandleHealth handles the /health endpoint for health checks
func HandleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		response := map[string]interface{}{
			"status": "healthy",
			"service": "openapi-mcp",
		}
		
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode health response: %v", err)
		}
	}
}

// HandleAPIList handles listing available APIs
func HandleAPIList(listFunc func() ([]map[string]interface{}, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apis, err := listFunc()
		if err != nil {
			log.Printf("Failed to list APIs: %v", err)
			http.Error(w, fmt.Sprintf("Failed to list APIs: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(apis); err != nil {
			log.Printf("Failed to encode API list: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}