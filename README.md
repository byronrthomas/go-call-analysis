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
make build && bin/gca call-graph -p $GCA_PROJECT_PATH -o $CSV_OUTPUT_PATH
```

```bash
make copy-csvs-to-memgraph folder=$CSV_OUTPUT_PATH
```

### Without outputting to CSV:


`make build && bin/gca call-graph --neo4j -p /Users/byron/repos/third-party/sei-protocol/sei-chain-outer/sei-chain`


### SSA first example

`make build && bin/gca ssa-graph -p $GCA_PROJECT_PATH --neo4j -r 'github.com/sei-protocol/sei-chain/cmd/seid:main' --package-prefixes='github.com/sei-protocol/sei-chain/oracle/price-feeder/oracle/client'`


### Test project SSA

`make build && bin/gca ssa-graph -p ./test-project -o ./test-output -r 'github.com/throwin5tone7/go-call-analysis/test-project:main'`

## Command Reference

The `gca` tool provides several subcommands for different types of analysis:

### 1. call-graph
Analyze a Go project and generate call graph reports.

**Basic usage:**
```bash
# Analyze project and output to CSV files
bin/gca call-graph -p /path/to/go/project -o /path/to/output

# Analyze with specific root function
bin/gca call-graph -p /path/to/go/project -r 'package:function' -o /path/to/output

# Export directly to Neo4j (no CSV output)
bin/gca call-graph -p /path/to/go/project --neo4j
```

**Options:**
- `-p, --path`: Path to the Go project to analyze (required)
- `-o, --output`: Path to write analysis results (for CSV output)
- `-r, --root-function`: Root function to analyze (format: 'package:function')
- `--neo4j`: Export results to Neo4j instead of CSV

### 2. ssa-graph
Analyze a Go project using SSA (Static Single Assignment) and generate SSA-based call graph reports.

**Basic usage:**
```bash
# Analyze with SSA and output to CSV
bin/gca ssa-graph -p /path/to/go/project -o /path/to/output

# Analyze specific packages with SSA
bin/gca ssa-graph -p /path/to/go/project --package-prefixes='github.com/user/pkg1,github.com/user/pkg2' -o /path/to/output

# Export SSA analysis to Neo4j
bin/gca ssa-graph -p /path/to/go/project --neo4j -r 'package:function' --package-prefixes='github.com/user/pkg'
```

**Options:**
- `-p, --path`: Path to the Go project to analyze (required)
- `-o, --output`: Path to write analysis results (for CSV output)
- `-r, --root-function`: Root function to analyze (format: 'package:function')
- `--neo4j`: Export results to Neo4j instead of CSV
- `--package-prefixes`: Comma-separated list of package prefixes to include

### 3. dump-packages
Build SSA program and dump package information to stdout.

**Basic usage:**
```bash
# Dump all packages in the project
bin/gca dump-packages -p /path/to/go/project

# Dump packages with verbose output
bin/gca dump-packages -p /path/to/go/project --verbose
```

**Options:**
- `-p, --path`: Path to the Go project to analyze (required)
- `--verbose`: Enable verbose output

### 4. output-SSA
Output SSA program text for matching packages.

**Basic usage:**
```bash
# Output SSA text to stdout
bin/gca output-SSA -p /path/to/go/project

# Output SSA text to file
bin/gca output-SSA -p /path/to/go/project -o /path/to/output.txt

# Output simplified SSA form
bin/gca output-SSA -p /path/to/go/project --simplified

# Output SSA for specific packages
bin/gca output-SSA -p /path/to/go/project --package-prefixes='github.com/user/pkg' -o /path/to/output.txt

# Output simplified SSA with root function
bin/gca output-SSA -p /path/to/go/project -r 'package:function' --simplified -o /path/to/output.txt
```

**Options:**
- `-p, --path`: Path to the Go project to analyze (required)
- `-o, --output`: Path to write SSA output file (outputs to stdout if not specified)
- `-r, --root-function`: Root function to analyze (format: 'package:function')
- `--package-prefixes`: Comma-separated list of package prefixes to include
- `--simplified`: Output simplified SSA form (default: false)