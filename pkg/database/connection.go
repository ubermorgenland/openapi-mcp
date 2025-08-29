package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// DB is the global database connection
var DB *sql.DB

// Connect establishes a connection to the PostgreSQL database
func Connect() (*sql.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	// Basic validation of PostgreSQL URL format
	if !strings.HasPrefix(databaseURL, "postgresql://") {
		return nil, fmt.Errorf("DATABASE_URL must be a valid PostgreSQL connection string starting with 'postgresql://'")
	}

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	log.Printf("Database connected successfully: %s", strings.Split(databaseURL, "@")[0]+"@[HIDDEN]")

	// Set global DB variable
	DB = db
	return db, nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// Ping checks if the database connection is still alive
func Ping() error {
	if DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}
	return DB.Ping()
}

// InitializeDatabase connects to the database and runs migrations
func InitializeDatabase() error {
	db, err := Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	return nil
}
