.PHONY: help build test lint precommit clean install-tools

# Default target
help:
	@echo "Available targets:"
	@echo "  make build       - Build the binary"
	@echo "  make test        - Run tests"
	@echo "  make lint        - Run linter"
	@echo "  make precommit   - Run lint and test (use before committing)"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make install-tools - Install development tools"

# Build the binary
build:
	@echo "Building..."
	go build -o ionos-cloud-watchdog ./cmd/ionos-cloud-watchdog

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Precommit checks (lint + test)
precommit: lint test
	@echo ""
	@echo "✓ All checks passed! Ready to commit."

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f ionos-cloud-watchdog
	rm -rf dist/

# Install development tools
install-tools:
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Tools installed"
