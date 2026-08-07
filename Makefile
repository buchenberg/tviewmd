# Makefile for tviewmd

.PHONY: all test lint build clean coverage fuzz release

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=tviewmd

# Linter
GOLANGCI_LINT?=golangci-lint

all: test lint build

test: 
	$(GOTEST) -v -race ./...
	test -v -race ./...

# Run tests with coverage
coverage:
	$(GOTEST) -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -func=coverage.out -o=coverage.txt
	@echo "Coverage report generated: coverage.out, coverage.txt"

# Run fuzz tests (60s fuzz time, 10s minimize time)
fuzz:
	$(GOTEST) -fuzz=FuzzParse -fuzztime=60s -fuzzminimizetime=10s ./...

# Lint code
lint:
	$(GOLANGCI_LINT) run --timeout 5m

# Build
build:
	$(GOBUILD) -v ./...

# Tidy go.mod
mod-tidy:
	$(GOMOD) tidy

# Clean
clean:
	rm -f coverage.out coverage.txt
	rm -f $(BINARY_NAME)

# Install linter
install-linter:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell $(GOCMD) env GOPATH)/bin v1.61.0

# Run all checks (test, lint, build)
check: test lint build

# Help
help:
	@echo "Available targets:"
	@echo "  all        - Run test, lint, and build"
	@echo "  test       - Run all tests"
	@echo "  coverage   - Run tests with coverage"
	@echo "  fuzz       - Run fuzz tests"
	@echo "  lint       - Run golangci-lint"
	@echo "  build      - Build the project"
	@echo "  mod-tidy   - Tidy go.mod"
	@echo "  clean      - Clean build artifacts"
	@echo "  check      - Run test, lint, and build"
	@echo "  install-linter - Install golangci-lint"
	@echo "  help       - Show this help"