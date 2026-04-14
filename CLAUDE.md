
- When updating the graphs to add/update/delete nodes or edges you will expect the regression tests to fail `make test`
    - When this happens never just commit the golden files directly, instead run `make regenerate-golden-ssa` to regenerate the SSA results and `make regenerate-golden-neo4j` to update the artefacts, and check the diffs match your changes before checking in