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

func RunPropagationQueries() error {
	return analyzer.RunPropagationQueries(defaultNeoConfig, fixedWidthPropagationQueries)
}

func RunTwoVaryingPropagationQueries() error {
	return analyzer.RunPropagationQueries(defaultNeoConfig, twoVaryingPropagationQueries)
}

const manualFunctionMarkingQuery = `
MATCH
(ftgt:Function {id: "__ID__"})
WHERE
NOT coalesce(ftgt.func_returns_fixed_width, false)
SET ftgt.func_returns_fixed_width = true, ftgt.annotation = "MANUALLY INSPECTED AND KNOWN GOOD"
`

func RunManualFunctionMarkingQuery(functionId string) error {
	err := analyzer.RunSingleUpdateQuery(defaultNeoConfig, strings.Replace(manualFunctionMarkingQuery, "__ID__", functionId, 1))
	if err != nil {
		return err
	}
	fmt.Printf("Function %s marked as known to return fixed width\n", functionId)
	fmt.Printf("\nIMPORTANT: You should re-run fixed width propagation queries after running this query, and reset any second-phase markings if needed.")
	return nil
}
