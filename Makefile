.PHONY: build test clean lint copy-csvs-to-memgraph build-transform build-all

# Build the application
build:
	go build -o bin/gca ./cmd/gca

# Build the transform-json-nodes tool
build-transform:
	go build -o bin/transform-json-nodes ./cmd/transform-json-nodes

# Build all tools
build-all: build build-transform

# Run tests
test:
	go test -v ./...

# Run SSA graph tests specifically
test-ssa:
	go test -v ./test -run TestSSAGraphAnalysis

# Regenerate golden files for SSA graph tests
regenerate-golden-ssa:
	bin/gca ssa-graph -p ./test-project -o ./test/resources/golden/ssa -r 'github.com/throwin5tone7/go-call-analysis/test-project:main' --package-prefixes='github.com/throwin5tone7/go-call-analysis'

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Run linter
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Copy CSV files to Memgraph container
copy-csvs-to-memgraph:
	@if [ -z "$(folder)" ]; then \
		echo "Error: folder argument is required. Usage: make copy-csvs-to-memgraph folder=/path/to/folder"; \
		exit 1; \
	fi
	find $(folder) -name "*.csv" -exec podman cp {} memgraph-mage:/tmp/ \;

clear-old-csvs:
	@if [ -z "$(folder)" ]; then \
		echo "Error: folder argument is required. Usage: make clear-old-csvs folder=/path/to/folder"; \
		exit 1; \
	fi
	rm -rf $(folder)/*.csv

# Run all checks
check: lint test 