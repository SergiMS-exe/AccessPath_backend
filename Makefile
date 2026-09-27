.PHONY: build run dev clean deps migrate migrate-up migrate-down seed setup-db test ci

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

# Run DML seed data (asume que el schema ya esta aplicado).
seed:
	psql $(DATABASE_URL) -f db/dml.sql

# Aplica las migraciones incrementales desde db/migrations/ hacia la BD.
# Necesita DATABASE_URL en el entorno (o .env cargado).
migrate-up:
	@command -v migrate >/dev/null 2>&1 || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path db/migrations -database "postgres://$(DATABASE_URL)" up

# Revierte la ultima migracion aplicada.
migrate-down:
	@command -v migrate >/dev/null 2>&1 || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	migrate -path db/migrations -database "postgres://$(DATABASE_URL)" down 1

# Setup inicial: schema (migrations) + seed (dml).
setup-db: migrate-up seed

# Tests con race detector (mismo comando que usa CI).
test:
	go test ./... -race -count=1

# atajo local para lo que corre CI: format + vet + test + lint.
ci:
	gofmt -l . && go vet ./... && go test ./... -race -count=1 && golangci-lint run
