# About these resources

In order to show the tool on some non-trivial live data, we ingest an SSA graph
of the github CLI tool at a specific tag. This was chosen as a well-known, trusted,
good quality and medium-size codebase to analyse. We fix to a tag so we can
get repeated results.

## Rough instructions to ingest the graph into a running memgraph


```
export GH_CLI_DIR='/Some_local_path/`
# For me it's export GH_CLI_DIR="/Users/byron/repos/third-party/github-cli/cli"
git clone `https://github.com/cli/cli` $GH_CLI_DIR
cd $GH_CLI_DIR
git switch v2.89.0

cd -
bin/gca dump-packages -p $GH_CLI_DIR

bin/gca ssa-graph -p $GH_CLI_DIR -r 'github.com/cli/cli/v2/cmd/gh:main' --package-prefixes='github.com/cli/cli/v2'
```

Get the package dependencies, e.g. in memgraph lab which allows downloading JSON results

```cypher
MATCH p=(p1:Package)-[:DepGraph_Depends]->(p2:Package)
RETURN p
```