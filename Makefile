# Binaries will be built into the ./bin directory
.PHONY: all mcp-client openapi-mcp spec-manager import-specs seed-database spec-api-server clean

all: bin/mcp-client bin/openapi-mcp bin/spec-manager bin/import-specs bin/seed-database bin/spec-api-server

bin/mcp-client: $(shell find pkg -type f -name '*.go') $(shell find cmd/mcp-client -type f -name '*.go')
	@mkdir -p bin
	go build -o bin/mcp-client ./cmd/mcp-client

bin/openapi-mcp: $(shell find pkg -type f -name '*.go') $(shell find cmd/openapi-mcp -type f -name '*.go')
	@mkdir -p bin
	go build -o bin/openapi-mcp ./cmd/openapi-mcp

bin/spec-manager: $(shell find pkg -type f -name '*.go') $(shell find cmd/spec-manager -type f -name '*.go')
	@mkdir -p bin
	go build -o bin/spec-manager ./cmd/spec-manager

bin/import-specs: $(shell find pkg -type f -name '*.go') scripts/import_specs.go
	@mkdir -p bin
	go build -o bin/import-specs ./scripts/import_specs.go

bin/seed-database: $(shell find pkg -type f -name '*.go') scripts/seed_database.go
	@mkdir -p bin
	go build -o bin/seed-database ./scripts/seed_database.go

bin/spec-api-server: $(shell find pkg -type f -name '*.go') $(shell find cmd/spec-api-server -type f -name '*.go') spec-api-swagger.json
	@mkdir -p bin
	go build -o bin/spec-api-server ./cmd/spec-api-server
	@cp spec-api-swagger.json bin/

test:
	go test ./...

import-specs-from-files:
	@echo "Importing specs from ./specs directory..."
	DATABASE_URL="${DATABASE_URL}" ./bin/import-specs

seed-database:
	@echo "Seeding database with predefined spec configuration..."
	DATABASE_URL="${DATABASE_URL}" ./bin/seed-database

seed-from-config:
	@echo "Seeding database from seed_config.yaml..."
	DATABASE_URL="${DATABASE_URL}" ./bin/seed-database seed_config.yaml

clean:
	rm -f bin/mcp-client bin/openapi-mcp bin/spec-manager bin/import-specs bin/seed-database bin/spec-api-server
