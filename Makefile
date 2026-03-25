.PHONY: build test clean lint build-transform build-all test-ssa test-transform regenerate-golden-ssa regenerate-golden-transform

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

# Run transform-json-nodes tests specifically
test-transform:
	go test -v ./test -run TestTransformJSONNodes

# Regenerate golden files for SSA graph tests
regenerate-golden-ssa:
	bin/gca ssa-graph -p ./test-project -o ./test/resources/golden/ssa -r 'github.com/throwin5tone7/go-call-analysis/test-project:main' --package-prefixes='github.com/throwin5tone7/go-call-analysis'

# Regenerate golden files for transform-json-nodes tests
regenerate-golden-transform:
	@echo "Regenerating golden files for transform-json-nodes..."
	rm -rf ./test/resources/golden/transform-json-nodes
	bin/transform-json-nodes -input ./test/resources/transform-json-nodes/sample_input.jsonl -root '/Users/byron/repos/third-party/injective/injective-core' -output ./test/resources/golden/transform-json-nodes -annotation 'to check'

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

# Run all checks
check: lint test 