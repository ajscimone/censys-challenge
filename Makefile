.PHONY: help proto sqlc migrate-up migrate-down migrate-create generate build run clean

# Default target - show help
help:
	@echo "Available targets:"
	@echo "  make proto        - Generate Go code from protobuf definitions"
	@echo "  make sqlc         - Generate Go code from SQL queries"
	@echo "  make generate     - Run both proto and sqlc generation"
	@echo "  make migrate-up   - Run database migrations (up)"
	@echo "  make migrate-down - Rollback last database migration"
	@echo "  make migrate-create NAME=<name> - Create new migration files"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run the application"
	@echo "  make clean        - Clean generated files"

# Generate Go code from proto files using buf
proto:
	@echo "Generating protobuf code..."
	buf generate

# Generate Go code from SQL queries using sqlc
sqlc:
	@echo "Generating SQL code..."
	sqlc generate

# Run both generators
generate: proto sqlc
	@echo "Code generation complete!"

# Database migration commands
DATABASE_URL ?= postgres://admin:password1@localhost:5432/censys-challenge?sslmode=disable

migrate-up:
	@echo "Running migrations up..."
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	@echo "Rolling back last migration..."
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Error: NAME is required. Usage: make migrate-create NAME=my_migration"; \
		exit 1; \
	fi
	@echo "Creating migration: $(NAME)"
	migrate create -ext sql -dir db/migrations -seq $(NAME)

# Build the application
build:
	@echo "Building application..."
	go build -o bin/censys-challenge ./cmd/server

# Run the application
run:
	@echo "Running application..."
	go run ./cmd/server

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf gen/
	rm -f bin/censys-challenge
