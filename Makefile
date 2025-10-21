# Go Quality Gate Makefile

# Variables
BINARY_NAME=quality-gate
MAIN_PACKAGE=./cmd/quality-gate
VERSION?=1.2.0
BUILD_DATE?=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Default target
.DEFAULT_GOAL := build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BINARY_NAME)"

# Build for development (without version info)
.PHONY: dev
dev:
	@echo "Building $(BINARY_NAME) for development..."
	go build -o $(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Development build complete: $(BINARY_NAME)"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test ./... -v

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install the binary to $GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to $(shell go env GOPATH)/bin..."
	cp $(BINARY_NAME) $(shell go env GOPATH)/bin/
	@echo "Installation complete"

# Show version information
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Git Commit: $(GIT_COMMIT)"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Format complete"

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run ./...
	@echo "Lint complete"

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Multi-platform build complete"

# Initialize a new project with quality-gate
.PHONY: init-project
init-project: build
	./$(BINARY_NAME) --init
	./$(BINARY_NAME) --install

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          Build the binary with version information"
	@echo "  dev            Build the binary for development"
	@echo "  test           Run all tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  clean          Remove build artifacts"
	@echo "  install        Install binary to GOPATH/bin"
	@echo "  version        Show version information"
	@echo "  fmt            Format source code"
	@echo "  lint           Run linter"
	@echo "  build-all      Build for multiple platforms"
	@echo "  init-project   Initialize quality-gate in current project"
	@echo "  help           Show this help message"