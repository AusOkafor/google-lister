.PHONY: help build run test clean docker-up docker-down migrate generate

# Default target
help:
	@echo "Available commands:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the API server"
	@echo "  run-worker   - Run the worker"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-up    - Start Docker services"
	@echo "  docker-down  - Stop Docker services"
	@echo "  migrate      - Run database migrations"
	@echo "  generate     - Generate Prisma client"

# Build the application
build:
	go build -o bin/api cmd/api/main.go
	go build -o bin/worker cmd/worker/main.go

# Run the API server
run:
	go run cmd/api/main.go

# Run the worker
run-worker:
	go run cmd/worker/main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Start Docker services
docker-up:
	docker-compose up -d

# Stop Docker services
docker-down:
	docker-compose down

# Run database migrations
migrate:
	# TODO: Implement database migrations
	@echo "Database migrations not implemented yet"

# Generate Prisma client
generate:
	cd prisma && npx prisma generate

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Run all services
dev: docker-up
	@echo "Starting development environment..."
	@echo "API will be available at http://localhost:8080"
	@echo "Adminer (DB admin) at http://localhost:8080"
	@echo "Redis Commander at http://localhost:8081"
