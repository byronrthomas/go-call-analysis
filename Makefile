.PHONY: build test clean lint copy-csvs-to-memgraph

# Build the application
build:
	go build -o bin/gca ./cmd/gca

# Run tests
test:
	go test -v ./...

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
	find $(folder) -name "*.csv" -exec docker cp {} memgraph-mage:/tmp/ \;

clear-old-csvs:
	@if [ -z "$(folder)" ]; then \
		echo "Error: folder argument is required. Usage: make clear-old-csvs folder=/path/to/folder"; \
		exit 1; \
	fi
	rm -rf $(folder)/*.csv

# Run all checks
check: lint test 