# Go Call Analysis

A command-line tool for analyzing Go projects and generating analysis reports.

## Features

- Analyze Go project structure
- Generate call graphs
- Static analysis of Go code
- (More features to be added)

## Installation

```bash
go install github.com/throwin5tone7/go-call-analysis@latest
```

## Development Setup

1. Clone the repository:
```bash
git clone https://github.com/throwin5tone7/go-call-analysis.git
cd go-call-analysis
```

2. Install dependencies:
```bash
go mod download
```

3. Run tests:
```bash
go test ./...
```

## Project Structure

```
go-call-analysis/
├── cmd/                    # Command-line interface
│   └── gca/               # Main application
├── internal/              # Private application code
│   ├── analyzer/         # Analysis logic
│   └── parser/           # Code parsing utilities
├── pkg/                   # Public library code
├── test/                 # Test files
└── tools/                # Development tools
```

## License

MIT License 

## Running examples

```bash
export GCA_PROJECT_PATH=/Users/byron/repos/third-party/sei-protocol/sei-chain-outer/sei-chain
export CSV_OUTPUT_PATH=/Users/byron/projects/bugging/sei-protocol/sei-chain-call-graph
make clear-old-csvs folder=$CSV_OUTPUT_PATH; \
make build && bin/gca analyze -p $GCA_PROJECT_PATH -o $CSV_OUTPUT_PATH
```

```bash
make copy-csvs-to-memgraph folder=$CSV_OUTPUT_PATH
```

### Without outputting to CSV:


`make build && bin/gca analyze --neo4j -p /Users/byron/repos/third-party/sei-protocol/sei-chain-outer/sei-chain`