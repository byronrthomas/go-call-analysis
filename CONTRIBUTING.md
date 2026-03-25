# Contributing

## Prerequisites

- Go 1.24+
- [Memgraph](https://memgraph.com/) (for integration testing against a live database)

## Building

```bash
make build      # builds bin/gca
make build-all  # also builds bin/transform-json-nodes
```

## Running the tests

```bash
make test        # full test suite
make test-ssa    # SSA graph tests only
```

Tests use golden files in `test/resources/golden/`. If you make an intentional change to the graph output, regenerate them with:

```bash
make build && make regenerate-golden-ssa
```

Then review the diff and commit the updated golden files alongside your code change.

## Project layout

```
cmd/
  gca/                  # main CLI entrypoint
  lib/                  # shared command logic and propagation query definitions
  transform-json-nodes/ # standalone helper tool
internal/
  analyzer/             # SSA analysis, graph extraction, Neo4j export
  graphcommon/          # shared graph types
test/                   # integration tests and golden files
test-project/           # minimal Go project used as test fixture
```

## Submitting changes

1. Fork the repository and create a branch from `main`
2. Make your changes, ensuring `make test` passes
3. Open a pull request with a clear description of what the change does and why

## Reporting issues

Open an issue on GitHub with enough context to reproduce the problem — the Go project being analysed, the command you ran, and the output or error you saw.
