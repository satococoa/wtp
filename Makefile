.PHONY: build test lint clean install dev release-test

# Build variables
BINARY_NAME = wtp
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT = $(shell git rev-parse HEAD)
DATE = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# Build flags
LDFLAGS = -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/git-wtp

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/git-wtp
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/git-wtp
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/git-wtp
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/git-wtp
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/git-wtp

# Run tests
test:
	@echo "Running tests..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	go tool golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf dist/

# Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	go install -ldflags="$(LDFLAGS)" ./cmd/git-wtp

# Development helpers
dev: clean lint test build
	@echo "Development build completed"

# Test GoReleaser configuration
release-test:
	@echo "Testing GoReleaser configuration..."
	goreleaser check
	goreleaser build --snapshot --clean

# Release with GoReleaser (requires tag)
release:
	@echo "Creating release..."
	goreleaser release --clean

# Update dependencies
deps:
	@echo "Updating dependencies..."
	go mod tidy
	go mod download

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	go tool goimports -w .

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	go mod verify

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all platforms"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  lint         - Run linter"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install to GOPATH/bin"
	@echo "  dev          - Run full development build (lint, test, build)"
	@echo "  release-test - Test GoReleaser configuration"
	@echo "  release      - Create release with GoReleaser"
	@echo "  deps         - Update dependencies"
	@echo "  fmt          - Format code"
	@echo "  verify       - Verify dependencies"
	@echo "  help         - Show this help"