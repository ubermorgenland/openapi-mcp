# Database-Driven OpenAPI Spec Loading

This document explains how to use the database-driven spec loading feature that allows you to store and manage OpenAPI specifications in a PostgreSQL database instead of loading them from files.

## Overview

The system now supports dynamic loading of OpenAPI specs from a PostgreSQL database. When `DATABASE_URL` is set, the application will:

1. Try to load active specs from the database first
2. Fall back to file-based loading if database is unavailable or contains no active specs
3. Combine all active specs into a single MCP server instance

## Database Setup

### 1. Database Connection

Set your PostgreSQL connection string:

```bash
export DATABASE_URL="postgresql://username:password@host:port/database"
```

Example:
```bash
export DATABASE_URL="postgresql://your_username:your_password@localhost:5432/your_database_name"
```

### 2. Build Tools

Build all required tools:

```bash
make all
```

This creates:
- `bin/openapi-mcp` - Main MCP server (now supports database loading)
- `bin/spec-manager` - CLI tool for managing specs in the database
- `bin/import-specs` - Utility to import specs from files to database

## Managing Specs

### Import Specs from Files

Import all specs from the `specs/` directory:

```bash
make import-specs-from-files
```

Or import manually:

```bash
./bin/spec-manager import specs/weather.json weather /weather
./bin/spec-manager import specs/twitter.yml twitter /twitter
```

### List All Specs

```bash
./bin/spec-manager list
```

Example output:
```
ID   Name                 Title                          Version    Active   Format     Endpoint
----------------------------------------------------------------------------------------------------
1    weather             OpenWeatherMap API             2.5        true     json       /weather
2    twitter             Twitter API                    2.0        false    yaml       /twitter
3    google-keywords     Google Keywords API            1.0        true     yml        /google-keywords
```

### List Only Active Specs

```bash
./bin/spec-manager active
```

### Activate/Deactivate Specs

```bash
# Activate a spec
./bin/spec-manager activate 2

# Deactivate a spec  
./bin/spec-manager deactivate 2
```

### Delete a Spec

```bash
./bin/spec-manager delete 2
```

## Running the Server

### With Database (Recommended)

When `DATABASE_URL` is set, the server automatically loads from database:

```bash
export DATABASE_URL="postgresql://..."
./bin/openapi-mcp
```

The server will:
- Load all active specs from database
- Combine operations from all specs
- Start MCP server with combined operations

### HTTP Mode with Database

```bash
export DATABASE_URL="postgresql://..."
./bin/openapi-mcp --http :8080
```

### Fallback to Files

If database is unavailable, provide a file path as fallback:

```bash
export DATABASE_URL="postgresql://..."
./bin/openapi-mcp specs/weather.json
```

## Database Schema

The `openapi_specs` table structure:

```sql
CREATE TABLE openapi_specs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(500),
    version VARCHAR(100), 
    spec_content TEXT NOT NULL,
    endpoint_path VARCHAR(255) UNIQUE NOT NULL,
    file_format VARCHAR(10) DEFAULT 'yaml',
    file_size INTEGER,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP(6) DEFAULT NOW(),
    updated_at TIMESTAMP(6) DEFAULT NOW()
);
```

Key fields:
- `name`: Unique identifier for the spec
- `spec_content`: Full OpenAPI specification (JSON/YAML)
- `endpoint_path`: Unique path for API endpoint routing
- `is_active`: Whether the spec should be loaded by the server
- `file_format`: Format hint ('json', 'yaml', 'yml')

## Example Workflow

1. **Import existing specs**:
   ```bash
   make import-specs-from-files
   ```

2. **Verify imports**:
   ```bash
   ./bin/spec-manager list
   ```

3. **Activate desired specs**:
   ```bash
   ./bin/spec-manager activate 1
   ./bin/spec-manager activate 3
   ```

4. **Start server with database specs**:
   ```bash
   export DATABASE_URL="postgresql://..."
   ./bin/openapi-mcp --http :8080
   ```

5. **Server loads and combines active specs automatically**

## Benefits

- **Dynamic Management**: Add/remove specs without restarting server
- **Centralized Storage**: All specs in database, not scattered files
- **Version Control**: Track spec versions and changes
- **Easy Deployment**: No need to manage spec files in deployments
- **Selective Loading**: Enable/disable specs as needed
- **Fallback Support**: Graceful degradation to file-based loading

## Troubleshooting

### Database Connection Issues

- Verify `DATABASE_URL` format and credentials
- Ensure PostgreSQL server is running and accessible
- Check firewall/network connectivity

### Migration Issues

The system auto-creates required tables. If issues occur:

```bash
# Check database logs
# Verify user has CREATE TABLE permissions
# Ensure PostgreSQL version compatibility (9.5+)
```

### No Active Specs

If server shows "no active specs found":

```bash
# Check what specs exist
./bin/spec-manager list

# Activate specs as needed
./bin/spec-manager activate <id>
```

### Import Failures

Common import issues:
- Invalid OpenAPI specification format
- Duplicate names or endpoint paths
- File not found or permission issues
- Database connection problems

Use `spec-manager import` for detailed error messages.