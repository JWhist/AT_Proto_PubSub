# Makefile for atp-test

# Build variables
BINARY_NAME=atp-test
MAIN_PATH=./cmd/atp-test
BUILD_DIR=./bin

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

# Run the application
.PHONY: run
run:
	go run $(MAIN_PATH)

# Run tests
.PHONY: test
test:
	go test ./...

# Run the example
.PHONY: example
example:
	go run ./examples/filter-demo.go

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
	@echo "  build      - Build the application"
	@echo "  build-prod - Build optimized binary for production"
	@echo "  run        - Run the application"
	@echo "  test       - Run tests"
	@echo "  example    - Run the filter demo example"
	@echo "  clean      - Clean build artifacts"
	@echo "  fmt        - Format Go code"
	@echo "  lint       - Run linter"
	@echo "  help       - Show this help"

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