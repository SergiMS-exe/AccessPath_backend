.PHONY: build run dev clean deps migrate seed

# Build the application
build:
	go build -o bin/server ./cmd/server

# Run the built binary
run: build
	./bin/server

# Run with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

# Clean build artifacts
clean:
	rm -rf bin/ tmp/

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run DDL migrations
migrate:
	psql $(DATABASE_URL) -f db/ddl.sql

# Run DML seed data
seed:
	psql $(DATABASE_URL) -f db/dml.sql

# Run both DDL and DML
setup-db: migrate seed
