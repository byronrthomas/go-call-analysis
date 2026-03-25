# go-call-analysis

[![CI](https://github.com/byronrthomas/go-call-analysis/actions/workflows/ci.yml/badge.svg)](https://github.com/byronrthomas/go-call-analysis/actions/workflows/ci.yml)
[![Go 1.24](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

A static analysis toolkit for Go code that builds SSA (Static Single Assignment) graphs and loads them into a Neo4j-compatible graph database for interactive exploration and annotation-driven analysis.

The primary workflow is: analyze a Go project → load the graph into Memgraph → explore with Cypher queries → annotate nodes → propagate annotations across the graph.

## Motivation

This tool simplifies the SSA graph produced by Go's analysis packages and maps it into a rich graph
schema that enables effective value tracing across a codebase. The resulting analysis graph captures:

* What variables are set to the return value of functionX, at every call site
* Which variables are read to provide the arguments of functionX, from every call site
* The resolved function calls (e.g. interfaceI.call can be resolved to the 2 concrete implementations of interfaceI.call), wherever statically determinable
* The types of various values in the graph
* Along with all of the natively available SSA relationships: 
  * The ordering of instructions within blocks
  * The ordering of blocks and conditional edges between them
  * The value dependencies linking writes of variable to reads in operations

With such information, a wide range of interesting analyses can be performed. It is trivial for example to see when error values are dropped without being checked:

```cypher
MATCH p=(func)<-[:Resolved_Call]-(cs :Instruction)-[:Produces_Result]->(verr),
// Below is a common pattern to locate an Instruction within it's code file - beware that some ops are synthetic and so hard to correctly disambiguate
(fv:FileVersion)<-[:Belongs_To]-(:Function)-[:Function_Entry|:And_Then|:Control_Flow*]->(cs)
WHERE verr.is_error_type
AND NOT EXISTS ((verr)<-[:Uses_Operand]-(:Instruction))
RETURN fv.id as call_site_filename, cs.line as call_site_line, func.id as called_function limit 20
```

Because Memgraph and Neo4j are schema-less, you can tag additional properties onto nodes and edges
using update queries to annotate them, then reference those annotations in later queries. For example:

```cypher
MATCH
(i1:Instruction {instruction_type: "Store"})-[:Uses_Operand {index: 0}]->
(v:Value)<-[:Uses_Operand {index: 0}]-(i2:Instruction {instruction_type: "Store"})
WHERE i1.id != i2.id
SET v.has_multiple_assignments = true
```

This will mark all values that are actually assigned multiple times (only possible for 
package-level vars and other global-type values due to the SSA transform). That then allows
you to query for things that are only set once, or for example, that are only set once
inside a package initializer which means they are constants during execution.


## Use cases

The SSA graph is general-purpose and can support many types of analysis. The current built-in annotation propagation is specialized for **Cosmos blockchain key collision analysis** — tracking whether `[]byte` values (store keys, prefixes, etc.) are of fixed width or composed of two varying-width components, which is the prerequisite for detecting key collision vulnerabilities.

The underlying graph structure is not specific to this use case; you can build your own Cypher-based analyses on top of it.

## Prerequisites

- Go 1.21+
- [Memgraph](https://memgraph.com/) (recommended) or any Neo4j-compatible graph database running locally

### Starting Memgraph

The quickest way is with Docker or Podman:

```bash
docker run -p 7687:7687 -p 7444:7444 memgraph/memgraph-mage
```

Memgraph Lab (browser UI) is available at `http://localhost:3000` if you use the `memgraph-platform` image instead.

## Build

```bash
git clone https://github.com/byronrthomas/go-call-analysis.git
cd go-call-analysis
make build        # builds bin/gca
make build-all    # also builds bin/transform-json-nodes
```

## Workflow

### 1. Double check what packages you want to analyse, and find the correct entry point

Initially it's a good idea to check what packages are in the project you're analysing by dumping
them:

```bash
bin/gca dump-packages -p /path/to/your/go/project
```

You then want to find two things:

1. A list of package prefixes that will filter to only the code you wish to analyze and not include third-party dependencies that are out of scope
  * This is important to limit the size of the graph - an SSA graph of a typical codebase can easily be a million edges or more
2. A root function (entry point) to analyse from - the correctly namespaced main function of your codebase
  * If you don't use this, then you can end up including test code and other code that's actually unreachable from the executed codepath, giving you false positives to eliminate

### 2. Load a Go project into the graph database

Use `ssa-graph` to analyze a project and write the results directly to the database:

```bash
bin/gca ssa-graph \
  -p /path/to/your/go/project \
  -r 'github.com/your/module/cmd/app:main' \
  --package-prefixes='github.com/your/module'
```

- `-r` sets the root function (entry point) for call graph traversal — format is `package/path:FunctionName`
- `--package-prefixes` filters which packages are included in the SSA graph; use your module path to exclude stdlib and third-party noise
- Without `-r`, the entire program is analyzed

### 3. Explore the graph

Connect to your database's query interface (e.g. Memgraph Lab, `mgconsole`, or `cypher-shell`) and start exploring with Cypher.

**Graph schema overview:**

| Node label | Key properties |
|---|---|
| `Function` | `id`, `name`, `package`, `file`, `line`, `func_returns_fixed_width`, `func_returns_two_comp_varying` |
| `Instruction` | `id`, `instruction_type`, `file`, `line` |
| `Value` | `id`, `type_name`, `fixed_width_value_kind`, `fixed_width_string_kind`, `known_two_component_varying` |
| `FileVersion` | `id`, `file`, `last_file_revision` |

| Relationship type | Meaning |
|---|---|
| `CALLS` | function calls another function |
| `Resolved_Call` | call site instruction resolves to a function |
| `Uses_Operand` | instruction uses a value as operand (indexed) |
| `Produces_Result` | instruction produces a value as result (indexed) |
| `Has_Parameter` | function has a parameter value |
| `Has_Return_Point` | function has a return instruction |
| `Ordering` | sequential ordering between instructions |
| `Belongs_To` | instruction/value belongs to a function |

**Example queries:**

```cypher
// Find all functions in a package
MATCH (f:Function)
WHERE f.package STARTS WITH 'github.com/your/module/store'
RETURN f.name, f.file, f.line
ORDER BY f.name;

// Inspect how a value flows through instructions
MATCH (v:Value {id: 'some-value-id'})<-[:Produces_Result]-(i:Instruction)
RETURN i.instruction_type, i.file, i.line;

// Find all []byte values not yet annotated
MATCH (v:Value)
WHERE v.type_name = '[]byte'
AND v.fixed_width_value_kind IS NULL
RETURN v.id, v.file, v.line
LIMIT 50;
```

### 4. Seed initial annotations

Once you identify values or functions of interest through exploration, seed the initial annotations. The propagation commands (step 5) extend these through the graph automatically, but they need a starting point.

**Manually mark a function as returning fixed-width bytes:**

```bash
bin/gca mark-function-known-fixed -f 'github.com/your/module/types:GetPrefix'
```

You can also set properties directly in Cypher if you want to annotate individual `Value` nodes:

```cypher
MATCH (v:Value {id: 'your-value-id'})
SET v.fixed_width_value_kind = 'KNOWN_CONSTANT';
```

### 5. Propagate annotations

Once seed annotations are in place, run the propagation commands to extend them transitively through the graph.

**Phase 1 — fixed-width propagation** (marks values and functions that provably produce fixed-width `[]byte`):

```bash
bin/gca known-fixed-propagation
```

This propagates through pointer dereferences, type conversions from fixed-width strings, `append` of two fixed-width values, and function return values. It runs iteratively until no new nodes can be marked.

**Phase 2 — two-varying propagation** (marks values that are `append` of two varying-width components):

```bash
bin/gca known-two-varying-propagation
```

Run phase 1 before phase 2. If you manually mark additional functions with `mark-function-known-fixed`, re-run phase 1 (and reset any phase 2 results if needed).

### 6. Query results

After propagation, query the annotated graph to find your points of interest:

```cypher
// Functions that return fixed-width bytes
MATCH (f:Function {func_returns_fixed_width: true})
RETURN f.id, f.file, f.line;

// Call sites where two-component varying bytes are produced
MATCH (v:Value)
WHERE v.known_two_component_varying IS NOT NULL
RETURN v.id, v.file, v.line;
```

Export query results to JSONL using Memgraph's export functionality for further processing.

## Configuration

By default, `gca` connects to `bolt://localhost:7687` with no credentials (Memgraph's default). Override with environment variables:

| Variable | Default | Description |
|---|---|---|
| `NEO4J_URI` | `bolt://localhost:7687` | Bolt URI of your database |
| `NEO4J_USERNAME` | _(empty)_ | Database username |
| `NEO4J_PASSWORD` | _(empty)_ | Database password |
| `NEO4J_DATABASE` | _(empty)_ | Database name (leave empty for default) |

```bash
export NEO4J_URI=bolt://myhost:7687
export NEO4J_USERNAME=neo4j
export NEO4J_PASSWORD=secret
bin/gca ssa-graph -p /path/to/project ...
```

## Command reference

### `gca ssa-graph` — recommended analysis command

Builds an SSA call graph and loads it into the database. This is the primary command for loading analysis data.

```
-p, --path              Path to the Go project (required)
-r, --root-function     Entry point — format: 'package/path:FunctionName'
    --package-prefixes  Comma-separated package prefixes to include
-o, --output            Write CSV files to this directory instead of writing to the database
```

### `gca known-fixed-propagation`

Runs fixed-width annotation propagation queries against the database. Iterates until convergence.

### `gca known-two-varying-propagation`

Runs two-component-varying annotation propagation queries. Run after `known-fixed-propagation`.

### `gca mark-function-known-fixed`

Manually marks a function node as known to return fixed-width `[]byte`. Use when automatic propagation can't reach a function (e.g. it has multiple return points, or the seed value isn't in the graph).

```
-f, --function-id   Function ID to mark (use the id property from the graph)
```

After running this, re-run `known-fixed-propagation`.

### `gca call-graph` — lightweight call graph

A simpler call graph analysis (no SSA). Useful for quick exploration of call structure without loading the full SSA graph.

```
-p, --path            Path to the Go project (required)
-r, --root-function   Entry point
-o, --output          Write CSV files to this directory instead of writing to the database
```

### `gca dump-packages`

Lists all packages found in the project. Useful for discovering the right values for `--package-prefixes`.

```
-p, --path      Path to the Go project (required)
-v, --verbose   Detailed package information
```

### `gca output-SSA`

Outputs the SSA representation of matching packages as text. Useful for understanding what the SSA graph will contain before loading it.

```
-p, --path              Path to the Go project (required)
-r, --root-function     Entry point
    --package-prefixes  Comma-separated package prefixes to include
    --simplified        Output simplified SSA form
-o, --output            Output file (defaults to stdout)
```

## Development

```bash
make test           # run all tests
make test-ssa       # run SSA graph tests only
make check          # lint + test
```

Golden files for tests live in `test/resources/golden/`. To regenerate after intentional changes:

```bash
make regenerate-golden-ssa
```

## Integration: transform-json-nodes

`bin/transform-json-nodes` is a helper that converts JSONL output from Memgraph queries into [code-notator](https://github.com/byronrthomas/code-notator) annotation files. Most users won't need this — it's a bridge to a specific annotation workflow built on top of the graph query results.

```bash
bin/transform-json-nodes \
  -input  query_results.jsonl \
  -root   /path/to/your/go/project \
  -output ./annotations \
  -annotation 'to check'
```
