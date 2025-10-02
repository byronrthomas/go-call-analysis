package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/throwin5tone7/go-call-analysis/internal/analyzer"
	"golang.org/x/tools/go/ssa"
)

// buildCallGraph is a shared function that builds the call graph for both commands
func buildCallGraph(rootFunction string, projectPath string, outputPath string) (*analyzer.CallGraphResult, error) {
	var rootFunctionId *analyzer.FunctionId
	if rootFunction != "" {
		rootFunctionId = &analyzer.FunctionId{
			Package:  strings.Split(rootFunction, ":")[0],
			Function: strings.Split(rootFunction, ":")[1],
		}
	}

	if projectPath == "" {
		return nil, fmt.Errorf("project path is required")
	}

	fmt.Printf("Analyzing project at: %s\n", projectPath)
	config, err := analyzer.NewAnalysisConfig(projectPath, outputPath, rootFunctionId)
	if err != nil {
		return nil, err
	}

	return analyzer.CallGraphAnalysis(config)
}

func RunCallGraph(rootFunction string, projectPath string, outputPath string, useNeo4j bool) error {
	callGraph, err := buildCallGraph(rootFunction, projectPath, outputPath)
	if err != nil {
		return err
	}

	nodes, edges := analyzer.ExtractCallGraphData(callGraph, projectPath)
	if useNeo4j {

		config := analyzer.Neo4jConfig{
			URI:      "bolt://localhost:7687",
			Username: "",
			Password: "",
			Database: "",
		}
		return analyzer.ExportCallGraphToNeo4j(nodes, edges, config)
	} else {

		return analyzer.ExportCallGraphToCSV(nodes, edges, outputPath)
	}
}

var defaultNeoConfig = analyzer.Neo4jConfig{
	URI:      "bolt://localhost:7687",
	Username: "",
	Password: "",
	Database: "",
}

func RunSSAGraph(packagePrefixes []string, projectPath string, outputPath string, rootFunction string, useNeo4j bool) error {
	packagePrefixes, simplificationResult, err := RunSSASimplification(rootFunction, projectPath, outputPath, packagePrefixes)
	if err != nil {
		return err
	}
	ssaResult := analyzer.ExtractSSAGraphData(simplificationResult, packagePrefixes, projectPath)

	if useNeo4j {

		return analyzer.ExportSSAGraphToNeo4j(ssaResult, defaultNeoConfig)
	} else {

		return analyzer.ExportSSAGraphToCSV(ssaResult, outputPath)
	}
}

func RunSSASimplification(rootFunction string, projectPath string, outputPath string, packagePrefixes []string) ([]string, *analyzer.SSASimplificationResult, error) {
	callGraph, err := buildCallGraph(rootFunction, projectPath, outputPath)
	if err != nil {
		return nil, nil, err
	}

	if len(packagePrefixes) == 0 {
		packagePrefixes = []string{""}
	}
	simplificationResult := analyzer.SimplifySSA(callGraph, packagePrefixes)
	return packagePrefixes, simplificationResult, nil
}

// RunDumpPackages builds an SSA program and dumps package information
func RunDumpPackages(projectPath string, verbose bool) error {
	if projectPath == "" {
		return fmt.Errorf("project path is required")
	}

	fmt.Printf("Building SSA program for project at: %s\n", projectPath)

	// Create a minimal config for BuildSSAProgram
	config := &analyzer.AnalysisConfig{
		ProjectPath:  projectPath,
		OutputPath:   "",  // Not needed for this command
		RootFunction: nil, // Not needed for this command
	}

	// Call BuildSSAProgram
	ssaProgram := analyzer.BuildSSAProgram(config)

	// Call DumpPackages with the SSA packages
	analyzer.DumpPackages(ssaProgram.AllPackages(), verbose)

	return nil
}

// RunOutputSSA builds an SSA program and outputs the textual representation
func RunOutputSSA(packagePrefixes []string, projectPath string, outputPath string, rootFunction string, simplified bool) error {
	if projectPath == "" {
		return fmt.Errorf("project path is required")
	}

	if len(packagePrefixes) == 0 {
		packagePrefixes = []string{""}
	}

	var ssaProgram *ssa.Program

	if simplified {
		// Use simplified SSA form
		_, simplificationResult, err := RunSSASimplification(rootFunction, projectPath, "", packagePrefixes)
		if err != nil {
			return err
		}
		ssaProgram = simplificationResult.SSAProgram
	} else {
		// Use regular SSA form
		var rootFunctionId *analyzer.FunctionId
		if rootFunction != "" {
			rootFunctionId = &analyzer.FunctionId{
				Package:  strings.Split(rootFunction, ":")[0],
				Function: strings.Split(rootFunction, ":")[1],
			}
		}

		config := &analyzer.AnalysisConfig{
			ProjectPath:  projectPath,
			OutputPath:   "", // Not needed for this command
			RootFunction: rootFunctionId,
		}

		ssaProgram = analyzer.BuildSSAProgram(config)
	}

	// Generate the SSA text
	ssaText := analyzer.GenerateSSAText(ssaProgram, packagePrefixes)

	// Output to file or stdout
	if outputPath != "" {
		// Create output directory if it doesn't exist
		dir := filepath.Dir(outputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		// Write to file
		if err := os.WriteFile(outputPath, []byte(ssaText), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %v", err)
		}
		fmt.Printf("SSA output written to: %s\n", outputPath)
	} else {
		// Output to stdout
		fmt.Print(ssaText)
	}

	return nil
}

const derefPropagationQueryPrefix = `
MATCH (vIn:Value)<-[:Uses_Operand {index: 0}]-(deref:Instruction {instruction_type: "UnOp(*)"})
-[:Produces_Result {index: 0}]->(vOut:Value)
WHERE vIn.fixed_width_value_kind IS NOT NULL
AND vOut.fixed_width_value_kind IS NULL
`

const derefPropagationQueryCount = derefPropagationQueryPrefix + `
RETURN count(vOut)
`

const derefPropagationQueryUpdate = `
` + derefPropagationQueryPrefix + `
SET vOut.fixed_width_value_kind = "deref(" + vIn.fixed_width_value_kind + ")"
`

var derefPropagationQuery = analyzer.PropagationQuery{
	CountQuery:     derefPropagationQueryCount,
	UpdateQuery:    derefPropagationQueryUpdate,
	CountFieldName: "count(vOut)",
	QueryName:      "Deref",
}

const appendFixedQueryPrefix = `
MATCH 
(i:Function {id: "^builtin^append"})
<-[:Resolved_Call]-(cs)
-[:Produces_Result]->(v),
(cs)-[:Uses_Operand {index: 0}]->(appArg1),
(cs)-[:Uses_Operand {index: 1}]->(appArg2)
WHERE
appArg1.fixed_width_value_kind IS NOT NULL
AND appArg2.fixed_width_value_kind IS NOT NULL
AND v.fixed_width_value_kind IS NULL`

const appendFixedQueryCount = appendFixedQueryPrefix + `
RETURN count(DISTINCT v)
`

const appendFixedQueryUpdate = `
` + appendFixedQueryPrefix + `
SET v.fixed_width_value_kind = "append(fixed, fixed)"
`

var appendFixedQuery = analyzer.PropagationQuery{
	CountQuery:     appendFixedQueryCount,
	UpdateQuery:    appendFixedQueryUpdate,
	CountFieldName: "count(DISTINCT v)",
	QueryName:      "append(fixed, fixed)",
}

const funcSingleReturnFixedQueryPrefix = `
MATCH
(ftgt:Function {num_return_points: 1})
-[:Has_Return_Point]->(ri:Instruction)
-[:Uses_Operand {index: 0}]->(v)
WHERE
v.fixed_width_value_kind IS NOT NULL
AND v.type_name = "[]byte"
AND NOT coalesce(ftgt.func_returns_fixed_width, false)
`

const funcSingleReturnFixedQueryCount = funcSingleReturnFixedQueryPrefix + `
RETURN count(ftgt)
`

const funcSingleReturnFixedQueryUpdate = funcSingleReturnFixedQueryPrefix + `
SET ftgt.func_returns_fixed_width = true
`

var funcSingleReturnFixedQuery = analyzer.PropagationQuery{
	CountQuery:     funcSingleReturnFixedQueryCount,
	UpdateQuery:    funcSingleReturnFixedQueryUpdate,
	CountFieldName: "count(ftgt)",
	QueryName:      "func has single return fixed",
}

const labelFuncToRetValPrefix = `
MATCH
(v:Value)<-[:Produces_Result {index: 0}]-
(cs:Instruction)-[:Resolved_Call]->(ftgt {func_returns_fixed_width: true})
WHERE v.fixed_width_value_kind IS NULL
`

const labelFuncToRetValQueryCount = labelFuncToRetValPrefix + `
RETURN count(distinct v)
`

const labelFuncToRetValQueryUpdate = labelFuncToRetValPrefix + `
SET v.fixed_width_value_kind = "fixed width func result"
`

var labelFuncToRetValQuery = analyzer.PropagationQuery{
	CountQuery:     labelFuncToRetValQueryCount,
	UpdateQuery:    labelFuncToRetValQueryUpdate,
	CountFieldName: "count(distinct v)",
	QueryName:      "label func to ret val",
}

func RunPropagationQueries() error {
	return analyzer.RunPropagationQueries(defaultNeoConfig, []analyzer.PropagationQuery{derefPropagationQuery, appendFixedQuery, funcSingleReturnFixedQuery, labelFuncToRetValQuery})
}

const manualFunctionMarkingQuery = `
MATCH
(ftgt:Function {id: "__ID__"})
WHERE
NOT coalesce(ftgt.func_returns_fixed_width, false)
SET ftgt.func_returns_fixed_width = true, ftgt.annotation = "MANUALLY INSPECTED AND KNOWN GOOD"
`

func RunManualFunctionMarkingQuery(functionId string) error {
	return analyzer.RunSingleUpdateQuery(defaultNeoConfig, strings.Replace(manualFunctionMarkingQuery, "__ID__", functionId, 1))
}
