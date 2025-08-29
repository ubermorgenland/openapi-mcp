package models

import (
	"time"
)

// OpenAPISpec represents the openapi_specs table structure
type OpenAPISpec struct {
	ID           int        `json:"id" db:"id"`
	Name         string     `json:"name" db:"name"`
	Title        *string    `json:"title,omitempty" db:"title"`
	Version      *string    `json:"version,omitempty" db:"version"`
	SpecContent  string     `json:"spec_content" db:"spec_content"`
	EndpointPath string     `json:"endpoint_path" db:"endpoint_path"`
	FileFormat   *string    `json:"file_format,omitempty" db:"file_format"`
	FileSize     *int       `json:"file_size,omitempty" db:"file_size"`
	ApiKeyToken  *string    `json:"api_key_token,omitempty" db:"api_key_token"`
	IsActive     *bool      `json:"is_active,omitempty" db:"is_active"`
	CreatedAt    *time.Time `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

// TableName returns the table name for the OpenAPISpec model
func (OpenAPISpec) TableName() string {
	return "openapi_specs"
}

// NewOpenAPISpec creates a new OpenAPISpec instance with default values
func NewOpenAPISpec(name, specContent, endpointPath string) *OpenAPISpec {
	now := time.Now()
	active := true
	format := "yaml"

	return &OpenAPISpec{
		Name:         name,
		SpecContent:  specContent,
		EndpointPath: endpointPath,
		FileFormat:   &format,
		IsActive:     &active,
		CreatedAt:    &now,
		UpdatedAt:    &now,
	}
}
