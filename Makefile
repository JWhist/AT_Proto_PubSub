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
	@echo "  test                - Run tests"
	@echo "  example             - Run the WebSocket test client"
	@echo "  clean               - Clean build artifacts"
	@echo "  fmt                 - Format Go code"
	@echo "  lint                - Run linter"
	@echo "  swagger             - Generate Swagger API documentation"
	@echo "  swagger-install     - Install Swagger tools"
	@echo "  docker-build        - Build Docker image"
	@echo "  docker-run          - Build and run with Docker (foreground)"
	@echo "  docker-run-detached - Build and run with Docker (background)"
	@echo "  docker-stop         - Stop Docker container"
	@echo "  docker-clean        - Stop and remove Docker container and image"
	@echo "  docker-logs         - Show Docker container logs"
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