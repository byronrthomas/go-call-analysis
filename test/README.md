# SSA Graph Tests

This directory contains tests for the SSA (Static Single Assignment) graph analysis functionality.

## Test Overview

The `TestSSAGraphAnalysis` test:

1. **Runs the SSA graph analysis directly** - Calls the internal functions instead of going through the command line
2. **Compares results with golden files** - Validates that the output matches expected results in `resources/golden/ssa/`
3. **Detects missing files** - Ensures all expected CSV files are generated
4. **Handles non-deterministic ordering** - Sorts rows before comparison to handle variations in output order

## Expected CSV Files

The test expects the following CSV files to be generated:
- `value_nodes.csv` - SSA value nodes
- `instruction_nodes.csv` - SSA instruction nodes  
- `refer_edges.csv` - Reference edges between nodes
- `ordering_edges.csv` - Ordering relationships between instructions
- `ssa_ordering_edges.csv` - SSA-specific ordering edges
- `operand_edges.csv` - Operand usage relationships
- `result_edges.csv` - Result production relationships

## Running the Tests

### Quick Test
```bash
make test-ssa
```

### Full Test Suite
```bash
make test
```

### Manual Test
```bash
go test -v ./test -run TestSSAGraphAnalysis
```

## Regenerating Golden Files

If you need to update the golden files (e.g., after making changes to the SSA analysis):

```bash
make regenerate-golden-ssa
```

This will regenerate the golden files in `resources/golden/ssa/` using the current implementation.

## Test Configuration

The test uses the following configuration:
- **Project Path**: `../test-project`
- **Output Path**: `../test-output`
- **Root Function**: `github.com/throwin5tone7/go-call-analysis/test-project:main`
- **Package Prefixes**: `github.com/throwin5tone7/go-call-analysis`

## Troubleshooting

If tests fail due to differences in output:

1. **Check if the differences are meaningful** - Some variations in ordering or internal IDs may be acceptable
2. **Regenerate golden files** - Use `make regenerate-golden-ssa` if the new output is correct
3. **Review the test output** - The test provides detailed information about differences for debugging
