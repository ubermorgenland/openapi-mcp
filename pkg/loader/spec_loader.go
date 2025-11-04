package loader

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/auth"
	"github.com/ubermorgenland/openapi-mcp/pkg/models"
	"github.com/ubermorgenland/openapi-mcp/pkg/server"
	"github.com/ubermorgenland/openapi-mcp/pkg/services"
)

// SpecLoader handles loading and management of OpenAPI specifications
type SpecLoader struct {
	specLoaderService  *services.SpecLoaderService
	authStateManager   *auth.StateManager
	loadedSpecs        map[string]*LoadedSpec
	requiredEnvVars    map[string]string
}

// LoadedSpec represents a loaded OpenAPI specification with metadata
type LoadedSpec struct {
	Endpoint    string
	Doc         *openapi3.T
	Spec        *models.OpenAPISpec
	Content     []byte
	LoadedAt    time.Time
}

// NewSpecLoader creates a new specification loader
func NewSpecLoader(specLoaderService *services.SpecLoaderService, authStateManager *auth.StateManager) *SpecLoader {
	return &SpecLoader{
		specLoaderService: specLoaderService,
		authStateManager:  authStateManager,
		loadedSpecs:       make(map[string]*LoadedSpec),
		requiredEnvVars:   make(map[string]string),
	}
}

// LoadFromDatabase loads specifications from the database
func (sl *SpecLoader) LoadFromDatabase(ctx context.Context) ([]*LoadedSpec, error) {
	if sl.specLoaderService == nil {
		return nil, server.NewErrorWithContext(ctx, server.ErrorTypeDatabase, "spec loader service not initialized", "")
	}

	specs, err := sl.specLoaderService.GetAllSpecs()
	if err != nil {
		return nil, server.WrapWithContext(ctx, err, server.ErrorTypeDatabase, "failed to load specs from database")
	}

	var loadedSpecs []*LoadedSpec

	for _, spec := range specs {
		loadedSpec, err := sl.processSpec(ctx, spec.EndpointPath, []byte(spec.SpecContent), spec)
		if err != nil {
			log.Printf("Failed to process spec for endpoint %s: %v", spec.EndpointPath, err)
			continue
		}

		loadedSpecs = append(loadedSpecs, loadedSpec)
		sl.loadedSpecs[spec.EndpointPath] = loadedSpec
	}

	// Update auth state manager
	if sl.authStateManager != nil {
		sl.authStateManager.UpdateSpecs(specs)
	}

	return loadedSpecs, nil
}

// LoadFromFiles loads specifications from file paths
func (sl *SpecLoader) LoadFromFiles(ctx context.Context, filePaths []string) ([]*LoadedSpec, error) {
	var loadedSpecs []*LoadedSpec

	for _, filePath := range filePaths {
		loadedSpec, err := sl.loadFromFile(ctx, filePath)
		if err != nil {
			log.Printf("Failed to load spec from file %s: %v", filePath, err)
			continue
		}

		loadedSpecs = append(loadedSpecs, loadedSpec)
		sl.loadedSpecs[loadedSpec.Endpoint] = loadedSpec
	}

	return loadedSpecs, nil
}

// loadFromFile loads a specification from a single file
func (sl *SpecLoader) loadFromFile(ctx context.Context, filePath string) (*LoadedSpec, error) {
	// Determine if it's a URL or file path
	var content []byte
	var err error

	if strings.HasPrefix(filePath, "http://") || strings.HasPrefix(filePath, "https://") {
		content, err = sl.loadFromURL(ctx, filePath)
	} else {
		content, err = sl.loadFromLocalFile(ctx, filePath)
	}

	if err != nil {
		return nil, err
	}

	// Extract endpoint name from file path
	endpoint := sl.extractEndpointFromPath(filePath)
	
	return sl.processSpec(ctx, endpoint, content, nil)
}

// loadFromURL loads specification from a URL
func (sl *SpecLoader) loadFromURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, server.WrapWithContext(ctx, err, server.ErrorTypeNetwork, "failed to create request")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, server.WrapWithContext(ctx, err, server.ErrorTypeNetwork, "failed to fetch spec from URL")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, server.NewErrorWithContext(ctx, server.ErrorTypeNetwork, 
			fmt.Sprintf("HTTP %d when fetching spec", resp.StatusCode), url)
	}

	return io.ReadAll(resp.Body)
}

// loadFromLocalFile loads specification from a local file
func (sl *SpecLoader) loadFromLocalFile(ctx context.Context, filePath string) ([]byte, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, server.NewErrorWithContext(ctx, server.ErrorTypeNotFound, 
			"spec file not found", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, server.WrapWithContext(ctx, err, server.ErrorTypeInternal, "failed to read spec file")
	}

	return content, nil
}

// processSpec processes raw specification content into a LoadedSpec
func (sl *SpecLoader) processSpec(ctx context.Context, endpoint string, content []byte, spec *models.OpenAPISpec) (*LoadedSpec, error) {
	// Parse OpenAPI document
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(content)
	if err != nil {
		return nil, server.WrapWithContext(ctx, err, server.ErrorTypeValidation, "failed to parse OpenAPI spec")
	}

	// Validate the document
	if err := doc.Validate(ctx); err != nil {
		return nil, server.WrapWithContext(ctx, err, server.ErrorTypeValidation, "OpenAPI spec validation failed")
	}

	// Extract and log authentication information
	sl.extractAuthInfo(endpoint, doc)

	return &LoadedSpec{
		Endpoint: endpoint,
		Doc:      doc,
		Spec:     spec,
		Content:  content,
		LoadedAt: time.Now(),
	}, nil
}

// extractAuthInfo extracts authentication information from the spec
func (sl *SpecLoader) extractAuthInfo(endpoint string, doc *openapi3.T) {
	schemeName, authType, authPath := auth.ExtractAuthSchemeFromSpec(doc)
	if authPath != "" {
		log.Printf("%s API: Found security scheme '%s' with %s authentication: %s", endpoint, schemeName, authType, authPath)
		
		// Add to required environment variables
		switch authType {
		case "bearer":
			sl.requiredEnvVars[strings.ToUpper(endpoint)+"_BEARER_TOKEN"] = "Bearer token for " + doc.Info.Title
		case "apiKey":
			sl.requiredEnvVars[strings.ToUpper(endpoint)+"_API_KEY"] = "API key for " + doc.Info.Title
		case "basic":
			sl.requiredEnvVars[strings.ToUpper(endpoint)+"_BASIC_AUTH"] = "Basic auth for " + doc.Info.Title
		}
	} else {
		log.Printf("%s API: No authentication security scheme found in spec", endpoint)
	}
}

// extractEndpointFromPath extracts an endpoint name from a file path or URL
func (sl *SpecLoader) extractEndpointFromPath(path string) string {
	// For URLs, extract from the last part of the path
	if strings.HasPrefix(path, "http") {
		parts := strings.Split(path, "/")
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			// Remove query parameters
			if idx := strings.Index(filename, "?"); idx != -1 {
				filename = filename[:idx]
			}
			// Remove file extension
			if idx := strings.LastIndex(filename, "."); idx != -1 {
				filename = filename[:idx]
			}
			return strings.ToLower(filename)
		}
	}

	// For file paths, use the base name without extension
	baseName := filepath.Base(path)
	if idx := strings.LastIndex(baseName, "."); idx != -1 {
		baseName = baseName[:idx]
	}
	
	return strings.ToLower(baseName)
}

// GetLoadedSpecs returns all currently loaded specifications
func (sl *SpecLoader) GetLoadedSpecs() map[string]*LoadedSpec {
	return sl.loadedSpecs
}

// GetRequiredEnvVars returns the required environment variables
func (sl *SpecLoader) GetRequiredEnvVars() map[string]string {
	return sl.requiredEnvVars
}

// Reload reloads all specifications
func (sl *SpecLoader) Reload(ctx context.Context) ([]string, error) {
	var reloadedAPIs []string
	
	if sl.specLoaderService != nil {
		// Reload from database
		loadedSpecs, err := sl.LoadFromDatabase(ctx)
		if err != nil {
			return nil, err
		}
		
		for _, spec := range loadedSpecs {
			reloadedAPIs = append(reloadedAPIs, spec.Endpoint)
		}
	}
	
	return reloadedAPIs, nil
}