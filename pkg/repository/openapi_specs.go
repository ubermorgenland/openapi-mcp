package repository

import (
	"database/sql"
	"fmt"

	"github.com/ubermorgenland/openapi-mcp/pkg/models"
)

// OpenAPISpecRepository handles database operations for OpenAPI specs
type OpenAPISpecRepository struct {
	db *sql.DB
}

// NewOpenAPISpecRepository creates a new repository instance
func NewOpenAPISpecRepository(db *sql.DB) *OpenAPISpecRepository {
	return &OpenAPISpecRepository{db: db}
}

// Create inserts a new OpenAPI spec into the database
func (r *OpenAPISpecRepository) Create(spec *models.OpenAPISpec) (*models.OpenAPISpec, error) {
	query := `
		INSERT INTO openapi_specs (name, title, version, spec_content, endpoint_path, file_format, file_size, api_key_token, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		spec.Name,
		spec.Title,
		spec.Version,
		spec.SpecContent,
		spec.EndpointPath,
		spec.FileFormat,
		spec.FileSize,
		spec.ApiKeyToken,
		spec.IsActive,
	).Scan(&spec.ID, &spec.CreatedAt, &spec.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create openapi spec: %v", err)
	}

	return spec, nil
}

// GetByID retrieves an OpenAPI spec by its ID
func (r *OpenAPISpecRepository) GetByID(id int) (*models.OpenAPISpec, error) {
	query := `
		SELECT id, name, title, version, spec_content, endpoint_path, file_format, file_size, api_key_token, is_active, created_at, updated_at
		FROM openapi_specs
		WHERE id = $1
	`

	spec := &models.OpenAPISpec{}
	err := r.db.QueryRow(query, id).Scan(
		&spec.ID,
		&spec.Name,
		&spec.Title,
		&spec.Version,
		&spec.SpecContent,
		&spec.EndpointPath,
		&spec.FileFormat,
		&spec.FileSize,
		&spec.ApiKeyToken,
		&spec.IsActive,
		&spec.CreatedAt,
		&spec.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("openapi spec with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get openapi spec: %v", err)
	}

	return spec, nil
}

// GetByName retrieves an OpenAPI spec by its name
func (r *OpenAPISpecRepository) GetByName(name string) (*models.OpenAPISpec, error) {
	query := `
		SELECT id, name, title, version, spec_content, endpoint_path, file_format, file_size, api_key_token, is_active, created_at, updated_at
		FROM openapi_specs
		WHERE name = $1
	`

	spec := &models.OpenAPISpec{}
	err := r.db.QueryRow(query, name).Scan(
		&spec.ID,
		&spec.Name,
		&spec.Title,
		&spec.Version,
		&spec.SpecContent,
		&spec.EndpointPath,
		&spec.FileFormat,
		&spec.FileSize,
		&spec.ApiKeyToken,
		&spec.IsActive,
		&spec.CreatedAt,
		&spec.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("openapi spec with name %s not found", name)
		}
		return nil, fmt.Errorf("failed to get openapi spec: %v", err)
	}

	return spec, nil
}

// GetByEndpointPath retrieves an OpenAPI spec by its endpoint path
func (r *OpenAPISpecRepository) GetByEndpointPath(path string) (*models.OpenAPISpec, error) {
	query := `
		SELECT id, name, title, version, spec_content, endpoint_path, file_format, file_size, api_key_token, is_active, created_at, updated_at
		FROM openapi_specs
		WHERE endpoint_path = $1
	`

	spec := &models.OpenAPISpec{}
	err := r.db.QueryRow(query, path).Scan(
		&spec.ID,
		&spec.Name,
		&spec.Title,
		&spec.Version,
		&spec.SpecContent,
		&spec.EndpointPath,
		&spec.FileFormat,
		&spec.FileSize,
		&spec.ApiKeyToken,
		&spec.IsActive,
		&spec.CreatedAt,
		&spec.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("openapi spec with endpoint path %s not found", path)
		}
		return nil, fmt.Errorf("failed to get openapi spec: %v", err)
	}

	return spec, nil
}

// GetAll retrieves all OpenAPI specs
func (r *OpenAPISpecRepository) GetAll() ([]*models.OpenAPISpec, error) {
	query := `
		SELECT id, name, title, version, spec_content, endpoint_path, file_format, file_size, api_key_token, is_active, created_at, updated_at
		FROM openapi_specs
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all openapi specs: %v", err)
	}
	defer rows.Close()

	var specs []*models.OpenAPISpec
	for rows.Next() {
		spec := &models.OpenAPISpec{}
		err := rows.Scan(
			&spec.ID,
			&spec.Name,
			&spec.Title,
			&spec.Version,
			&spec.SpecContent,
			&spec.EndpointPath,
			&spec.FileFormat,
			&spec.FileSize,
			&spec.ApiKeyToken,
			&spec.IsActive,
			&spec.CreatedAt,
			&spec.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan openapi spec: %v", err)
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

// GetActive retrieves all active OpenAPI specs
func (r *OpenAPISpecRepository) GetActive() ([]*models.OpenAPISpec, error) {
	query := `
		SELECT id, name, title, version, spec_content, endpoint_path, file_format, file_size, api_key_token, is_active, created_at, updated_at
		FROM openapi_specs
		WHERE is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active openapi specs: %v", err)
	}
	defer rows.Close()

	var specs []*models.OpenAPISpec
	for rows.Next() {
		spec := &models.OpenAPISpec{}
		err := rows.Scan(
			&spec.ID,
			&spec.Name,
			&spec.Title,
			&spec.Version,
			&spec.SpecContent,
			&spec.EndpointPath,
			&spec.FileFormat,
			&spec.FileSize,
			&spec.ApiKeyToken,
			&spec.IsActive,
			&spec.CreatedAt,
			&spec.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan openapi spec: %v", err)
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

// Update modifies an existing OpenAPI spec
func (r *OpenAPISpecRepository) Update(spec *models.OpenAPISpec) (*models.OpenAPISpec, error) {
	query := `
		UPDATE openapi_specs
		SET name = $2, title = $3, version = $4, spec_content = $5, endpoint_path = $6, 
		    file_format = $7, file_size = $8, api_key_token = $9, is_active = $10, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		spec.ID,
		spec.Name,
		spec.Title,
		spec.Version,
		spec.SpecContent,
		spec.EndpointPath,
		spec.FileFormat,
		spec.FileSize,
		spec.ApiKeyToken,
		spec.IsActive,
	).Scan(&spec.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update openapi spec: %v", err)
	}

	return spec, nil
}

// Delete removes an OpenAPI spec from the database
func (r *OpenAPISpecRepository) Delete(id int) error {
	query := `DELETE FROM openapi_specs WHERE id = $1`

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete openapi spec: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("openapi spec with id %d not found", id)
	}

	return nil
}

// SetActive sets the is_active status of an OpenAPI spec
func (r *OpenAPISpecRepository) SetActive(id int, active bool) error {
	query := `UPDATE openapi_specs SET is_active = $2, updated_at = NOW() WHERE id = $1`

	result, err := r.db.Exec(query, id, active)
	if err != nil {
		return fmt.Errorf("failed to set active status: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("openapi spec with id %d not found", id)
	}

	return nil
}

// UpdateApiKeyToken updates the API key token for an OpenAPI spec
func (r *OpenAPISpecRepository) UpdateApiKeyToken(id int, apiKeyToken *string) error {
	query := `UPDATE openapi_specs SET api_key_token = $2, updated_at = NOW() WHERE id = $1`

	result, err := r.db.Exec(query, id, apiKeyToken)
	if err != nil {
		return fmt.Errorf("failed to update API key token: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("openapi spec with id %d not found", id)
	}

	return nil
}
