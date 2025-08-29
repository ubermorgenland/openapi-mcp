package services

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ubermorgenland/openapi-mcp/pkg/database"
	"github.com/ubermorgenland/openapi-mcp/pkg/models"
	"github.com/ubermorgenland/openapi-mcp/pkg/openapi2mcp"
	"github.com/ubermorgenland/openapi-mcp/pkg/repository"
)

// SpecLoaderService handles loading OpenAPI specs from database or files
type SpecLoaderService struct {
	specRepo *repository.OpenAPISpecRepository
	db       *sql.DB
}

// NewSpecLoaderService creates a new spec loader service
func NewSpecLoaderService(db *sql.DB) *SpecLoaderService {
	return &SpecLoaderService{
		specRepo: repository.NewOpenAPISpecRepository(db),
		db:       db,
	}
}

// LoadFromDatabase loads all active OpenAPI specs from the database
func (s *SpecLoaderService) LoadFromDatabase() ([]openapi2mcp.OpenAPIOperation, []*openapi3.T, error) {
	specs, err := s.specRepo.GetActive()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load specs from database: %v", err)
	}

	if len(specs) == 0 {
		return nil, nil, fmt.Errorf("no active specs found in database")
	}

	var allOps []openapi2mcp.OpenAPIOperation
	var allDocs []*openapi3.T

	for _, spec := range specs {
		doc, err := s.parseSpecContent(spec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse spec '%s': %v\n", spec.Name, err)
			continue
		}

		ops := openapi2mcp.ExtractOpenAPIOperations(doc)
		allOps = append(allOps, ops...)
		allDocs = append(allDocs, doc)

		fmt.Fprintf(os.Stderr, "Loaded spec '%s' with %d operations from database\n", spec.Name, len(ops))
	}

	return allOps, allDocs, nil
}

// parseSpecContent parses the spec content based on its format
func (s *SpecLoaderService) parseSpecContent(spec *models.OpenAPISpec) (*openapi3.T, error) {
	loader := openapi3.NewLoader()

	// Determine if content is JSON or YAML based on format or content
	format := "yaml"
	if spec.FileFormat != nil {
		format = *spec.FileFormat
	}

	var doc *openapi3.T
	var err error

	if format == "json" || strings.HasPrefix(strings.TrimSpace(spec.SpecContent), "{") {
		// Parse as JSON
		doc, err = loader.LoadFromData([]byte(spec.SpecContent))
	} else {
		// Parse as YAML
		doc, err = loader.LoadFromData([]byte(spec.SpecContent))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec content: %v", err)
	}

	return doc, nil
}

// LoadSpecByName loads a specific spec by name from the database
func (s *SpecLoaderService) LoadSpecByName(name string) ([]openapi2mcp.OpenAPIOperation, *openapi3.T, error) {
	spec, err := s.specRepo.GetByName(name)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load spec by name: %v", err)
	}

	if spec.IsActive != nil && !*spec.IsActive {
		return nil, nil, fmt.Errorf("spec '%s' is not active", name)
	}

	doc, err := s.parseSpecContent(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse spec content: %v", err)
	}

	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	return ops, doc, nil
}

// LoadSpecByEndpoint loads a specific spec by endpoint path from the database
func (s *SpecLoaderService) LoadSpecByEndpoint(endpointPath string) ([]openapi2mcp.OpenAPIOperation, *openapi3.T, error) {
	spec, err := s.specRepo.GetByEndpointPath(endpointPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load spec by endpoint: %v", err)
	}

	if spec.IsActive != nil && !*spec.IsActive {
		return nil, nil, fmt.Errorf("spec at endpoint '%s' is not active", endpointPath)
	}

	doc, err := s.parseSpecContent(spec)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse spec content: %v", err)
	}

	ops := openapi2mcp.ExtractOpenAPIOperations(doc)
	return ops, doc, nil
}

// ImportSpecFromFile imports a spec from a file into the database
func (s *SpecLoaderService) ImportSpecFromFile(filePath, name, endpointPath string) error {
	return s.ImportSpecFromFileWithToken(filePath, name, endpointPath, nil)
}

// ImportSpecFromFileWithToken imports a spec from a file into the database with an API key token
func (s *SpecLoaderService) ImportSpecFromFileWithToken(filePath, name, endpointPath string, apiKeyToken *string) error {
	// Check if database is connected
	if database.DB == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read spec file: %v", err)
	}

	// Determine file format
	format := "yaml"
	if strings.HasSuffix(strings.ToLower(filePath), ".json") {
		format = "json"
	}

	// Parse the spec to extract title and version
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(content)
	if err != nil {
		return fmt.Errorf("failed to parse OpenAPI spec: %v", err)
	}

	var title, version *string
	if doc.Info != nil {
		if doc.Info.Title != "" {
			title = &doc.Info.Title
		}
		if doc.Info.Version != "" {
			version = &doc.Info.Version
		}
	}

	// Create new spec model
	spec := models.NewOpenAPISpec(name, string(content), endpointPath)
	spec.Title = title
	spec.Version = version
	spec.FileFormat = &format
	spec.ApiKeyToken = apiKeyToken
	fileSize := len(content)
	spec.FileSize = &fileSize

	// Save to database
	_, err = s.specRepo.Create(spec)
	if err != nil {
		return fmt.Errorf("failed to save spec to database: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully imported spec '%s' to database\n", name)
	return nil
}

// GetAllSpecs returns all specs from the database
func (s *SpecLoaderService) GetAllSpecs() ([]*models.OpenAPISpec, error) {
	return s.specRepo.GetAll()
}

// GetActiveSpecs returns all active specs from the database
func (s *SpecLoaderService) GetActiveSpecs() ([]*models.OpenAPISpec, error) {
	return s.specRepo.GetActive()
}

// ActivateSpec activates a spec by ID
func (s *SpecLoaderService) ActivateSpec(id int) error {
	return s.specRepo.SetActive(id, true)
}

// DeactivateSpec deactivates a spec by ID
func (s *SpecLoaderService) DeactivateSpec(id int) error {
	return s.specRepo.SetActive(id, false)
}

// DeleteSpec deletes a spec by ID
func (s *SpecLoaderService) DeleteSpec(id int) error {
	return s.specRepo.Delete(id)
}

// UpdateApiKeyToken updates the API key token for a spec by ID
func (s *SpecLoaderService) UpdateApiKeyToken(id int, apiKeyToken *string) error {
	return s.specRepo.UpdateApiKeyToken(id, apiKeyToken)
}

// CreateSpecFromContent creates a new spec directly from content
func (s *SpecLoaderService) CreateSpecFromContent(name, endpointPath, specContent, fileFormat string, apiKeyToken *string) error {
	// Check if database is connected
	if database.DB == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// Parse the spec to extract title and version
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(specContent))
	if err != nil {
		return fmt.Errorf("failed to parse OpenAPI spec: %v", err)
	}

	var title, version *string
	if doc.Info != nil {
		if doc.Info.Title != "" {
			title = &doc.Info.Title
		}
		if doc.Info.Version != "" {
			version = &doc.Info.Version
		}
	}

	// Create new spec model
	spec := models.NewOpenAPISpec(name, specContent, endpointPath)
	spec.Title = title
	spec.Version = version
	spec.FileFormat = &fileFormat
	spec.ApiKeyToken = apiKeyToken
	fileSize := len(specContent)
	spec.FileSize = &fileSize

	// Save to database
	_, err = s.specRepo.Create(spec)
	if err != nil {
		return fmt.Errorf("failed to save spec to database: %v", err)
	}

	return nil
}
