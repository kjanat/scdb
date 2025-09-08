# SCDB Downloader Makefile
# Provides testing, building, and CI integration targets

.PHONY: all build test test-unit test-integration test-e2e test-coverage clean fmt vet lint install-deps help

# Build configuration
BINARY_NAME=scdb-downloader
MAIN_PACKAGE=.
BUILD_DIR=./bin
COVERAGE_DIR=./coverage

# Go configuration
GO_VERSION=1.19
GOFLAGS=-v
LDFLAGS=-w -s

# Default target
all: clean fmt vet test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run all tests
test: test-unit test-integration test-e2e

# Run unit tests (fast, no network)
test-unit:
	@echo "Running unit tests..."
	@go test $(GOFLAGS) -short -race ./...

# Run integration tests (with mocking)
test-integration:
	@echo "Running integration tests..."
	@go test $(GOFLAGS) -run "TestSCDBDownloader|TestMock" ./...

# Run end-to-end tests (comprehensive scenarios)
test-e2e:
	@echo "Running end-to-end tests..."
	@go test $(GOFLAGS) -run "TestE2E" ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@go test $(GOFLAGS) -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v -race ./...

# Run benchmark tests
test-bench:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run golangci-lint (if available)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping..."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Install development dependencies
install-deps:
	@echo "Installing development dependencies..."
	@go mod download
	@go mod tidy
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@go clean -cache -testcache

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install $(GOFLAGS) -ldflags="$(LDFLAGS)" $(MAIN_PACKAGE)

# Run development checks (format, vet, test)
check: fmt vet test-unit

# Quick development cycle (format, vet, short tests, build)
dev: fmt vet test-unit build
	@echo "Development build complete"

# CI pipeline simulation
ci: clean install-deps fmt vet lint test-coverage build
	@echo "CI pipeline complete"

# Release build with optimizations
release: clean
	@echo "Building release binary..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build \
		-ldflags="-w -s -extldflags=-static" \
		-a -installsuffix cgo \
		-o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Release build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Cross-platform builds
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "Cross-platform builds complete"

# Show test results summary
test-summary: test-coverage
	@echo ""
	@echo "=== Test Summary ==="
	@echo "Unit tests: PASS"
	@echo "Integration tests: PASS" 
	@echo "E2E tests: PASS"
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"
	@echo ""

# Help target
help:
	@echo "SCDB Downloader - Available Make Targets:"
	@echo ""
	@echo "Building:"
	@echo "  build         Build the binary"
	@echo "  build-all     Cross-platform builds"
	@echo "  release       Optimized release build"
	@echo "  install       Install binary to GOPATH/bin"
	@echo ""
	@echo "Testing:"
	@echo "  test          Run all tests"
	@echo "  test-unit     Run unit tests only"
	@echo "  test-integration Run integration tests"
	@echo "  test-e2e      Run end-to-end tests"
	@echo "  test-coverage Run tests with coverage"
	@echo "  test-verbose  Run tests with verbose output"
	@echo "  test-bench    Run benchmark tests"
	@echo "  test-summary  Show test results summary"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt           Format code"
	@echo "  vet           Run go vet"
	@echo "  lint          Run golangci-lint"
	@echo "  check         Quick development checks"
	@echo ""
	@echo "Development:"
	@echo "  dev           Development build cycle"
	@echo "  clean         Clean build artifacts"
	@echo "  install-deps  Install development dependencies"
	@echo ""
	@echo "CI/CD:"
	@echo "  ci            Full CI pipeline simulation"
	@echo ""
	@echo "Usage Examples:"
	@echo "  make dev                  # Quick development cycle"
	@echo "  make test-coverage        # Run tests with coverage"
	@echo "  make ci                   # Full CI pipeline"
	@echo "  make build-all            # Cross-platform builds"