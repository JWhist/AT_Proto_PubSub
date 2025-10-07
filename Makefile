# Makefile for AT Proto PubSub

# Build variables
BINARY_NAME=at-proto-pubsub
MAIN_PATH=./cmd/atprotopubsub
BUILD_DIR=./bin
DOCKER_IMAGE=at-proto-pubsub
DOCKER_TAG=latest
CONTAINER_NAME=at-proto-pubsub-container

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

# Build for production (with optimizations)
.PHONY: build-prod
build-prod:
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application with Docker
.PHONY: run
run: docker-build
	docker run --rm -p 8080:8080 --name $(CONTAINER_NAME) $(DOCKER_IMAGE):$(DOCKER_TAG)

# Run the application locally (without Docker)
.PHONY: run-local
run-local:
	go run $(MAIN_PATH)

# Run development environment
.PHONY: dev
dev: compose-dev-up

# Run production environment
.PHONY: prod
prod: compose-prod-up

# Run tests
.PHONY: test
test:
	go test --race ./... -v

# Generate Swagger documentation
.PHONY: swagger
swagger:
	swag init -g internal/api/handlers.go -o docs/

# Install Swagger tools
.PHONY: swagger-install
swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest

# Docker commands
.PHONY: docker-build
docker-build:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

.PHONY: docker-run
docker-run: docker-build
	docker run --rm -p 8080:8080 --name $(CONTAINER_NAME) $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-run-detached
docker-run-detached: docker-build
	docker run -d -p 8080:8080 --name $(CONTAINER_NAME) $(DOCKER_IMAGE):$(DOCKER_TAG)

.PHONY: docker-stop
docker-stop:
	docker stop $(CONTAINER_NAME) || true

.PHONY: docker-clean
docker-clean:
	docker stop $(CONTAINER_NAME) || true
	docker rm $(CONTAINER_NAME) || true
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) || true

.PHONY: docker-logs
docker-logs:
	docker logs -f $(CONTAINER_NAME)

# Docker Compose commands for development
.PHONY: compose-dev-up
compose-dev-up:
	docker compose -f docker-compose.dev.yml up --build

.PHONY: compose-dev-up-detached
compose-dev-up-detached:
	docker compose -f docker-compose.dev.yml up --build -d

.PHONY: compose-dev-down
compose-dev-down:
	docker compose -f docker-compose.dev.yml down

.PHONY: compose-dev-logs
compose-dev-logs:
	docker compose -f docker-compose.dev.yml logs -f

.PHONY: compose-dev-restart
compose-dev-restart:
	docker compose -f docker-compose.dev.yml restart

# Docker Compose commands for production
.PHONY: compose-prod-up
compose-prod-up:
	docker compose -f docker-compose.prod.yml up --build

.PHONY: compose-prod-up-detached
compose-prod-up-detached:
	docker compose -f docker-compose.prod.yml up --build -d

.PHONY: compose-prod-down
compose-prod-down:
	docker compose -f docker-compose.prod.yml down

.PHONY: compose-prod-logs
compose-prod-logs:
	docker compose -f docker-compose.prod.yml logs -f

.PHONY: compose-prod-restart
compose-prod-restart:
	docker compose -f docker-compose.prod.yml restart

# Run the example
.PHONY: example
example:
	go run ./test_websocket_client.go

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	golangci-lint run

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build               - Build the application"
	@echo "  build-prod          - Build optimized binary for production"
	@echo "  run                 - Run the application with Docker"
	@echo "  run-local           - Run the application locally (without Docker)"
	@echo "  dev                 - Start development environment (shortcut for compose-dev-up)"
	@echo "  prod                - Start production environment (shortcut for compose-prod-up)"
	@echo "  test                - Run tests"
	@echo "  example             - Run the WebSocket test client"
	@echo "  clean               - Clean build artifacts"
	@echo "  fmt                 - Format Go code"
	@echo "  lint                - Run linter"
	@echo "  swagger             - Generate Swagger API documentation"
	@echo "  swagger-install     - Install Swagger tools"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-build        - Build Docker image"
	@echo "  docker-run          - Build and run with Docker (foreground)"
	@echo "  docker-run-detached - Build and run with Docker (background)"
	@echo "  docker-stop         - Stop Docker container"
	@echo "  docker-clean        - Stop and remove Docker container and image"
	@echo "  docker-logs         - Show Docker container logs"
	@echo ""
	@echo "Docker Compose (Development):"
	@echo "  compose-dev-up      - Start development environment (foreground)"
	@echo "  compose-dev-up-detached - Start development environment (background)"
	@echo "  compose-dev-down    - Stop development environment"
	@echo "  compose-dev-logs    - Show development logs"
	@echo "  compose-dev-restart - Restart development services"
	@echo ""
	@echo "Docker Compose (Production):"
	@echo "  compose-prod-up     - Start production environment (foreground)"
	@echo "  compose-prod-up-detached - Start production environment (background)"
	@echo "  compose-prod-down   - Stop production environment"
	@echo "  compose-prod-logs   - Show production logs"
	@echo "  compose-prod-restart - Restart production services"
	@echo ""
	@echo "Cross-compilation:"
	@echo "  build-linux         - Build for Linux"
	@echo "  build-windows       - Build for Windows"
	@echo "  build-mac           - Build for macOS"
	@echo "  build-all           - Build binaries for all platforms"
	@echo "  help                - Show this help"

# Cross-compilation targets
.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux $(MAIN_PATH)

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME).exe $(MAIN_PATH)

.PHONY: build-mac
build-mac:
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-mac $(MAIN_PATH)

.PHONY: build-all
build-all: build-linux build-windows build-mac
	@echo "Built binaries for all platforms"