package database

import (
	"database/sql"
	"fmt"
	"log"
)

// CreateOpenAPISpecsTable creates the openapi_specs table with all constraints and indexes
func CreateOpenAPISpecsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS openapi_specs (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) UNIQUE NOT NULL,
		title VARCHAR(500),
		version VARCHAR(100),
		spec_content TEXT NOT NULL,
		endpoint_path VARCHAR(255) UNIQUE NOT NULL,
		file_format VARCHAR(10) DEFAULT 'yaml',
		file_size INTEGER,
		api_key_token VARCHAR(500),
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMP(6) DEFAULT NOW(),
		updated_at TIMESTAMP(6) DEFAULT NOW()
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_openapi_specs_endpoint_path ON openapi_specs(endpoint_path);
	CREATE INDEX IF NOT EXISTS idx_openapi_specs_is_active ON openapi_specs(is_active);
	CREATE INDEX IF NOT EXISTS idx_openapi_specs_name ON openapi_specs(name);

	-- Create updated_at trigger
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	DROP TRIGGER IF EXISTS update_openapi_specs_updated_at ON openapi_specs;
	CREATE TRIGGER update_openapi_specs_updated_at
		BEFORE UPDATE ON openapi_specs
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create openapi_specs table: %v", err)
	}

	log.Println("Successfully created openapi_specs table with indexes and triggers")
	return nil
}

// DropOpenAPISpecsTable drops the openapi_specs table (useful for testing)
func DropOpenAPISpecsTable(db *sql.DB) error {
	query := `
	DROP TRIGGER IF EXISTS update_openapi_specs_updated_at ON openapi_specs;
	DROP FUNCTION IF EXISTS update_updated_at_column();
	DROP TABLE IF EXISTS openapi_specs CASCADE;
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to drop openapi_specs table: %v", err)
	}

	log.Println("Successfully dropped openapi_specs table")
	return nil
}

// RunMigrations runs all database migrations
func RunMigrations(db *sql.DB) error {
	log.Println("Running database migrations...")

	if err := CreateOpenAPISpecsTable(db); err != nil {
		return fmt.Errorf("migration failed: %v", err)
	}

	log.Println("All migrations completed successfully")
	return nil
}
