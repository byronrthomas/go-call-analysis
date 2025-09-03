package lib

import (
	"fmt"
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

	nodes, edges := analyzer.ExtractCallGraphData(callGraph)
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

func RunSSAGraph(packagePrefixes []string, projectPath string, outputPath string, rootFunction string, useNeo4j bool) error {
	packagePrefixes, ssaProgram, err := RunSSASimplification(rootFunction, projectPath, outputPath, packagePrefixes)
	if err != nil {
		return err
	}
	ssaResult := analyzer.ExtractSSAGraphData(ssaProgram, packagePrefixes)

	if useNeo4j {

		config := analyzer.Neo4jConfig{
			URI:      "bolt://localhost:7687",
			Username: "",
			Password: "",
			Database: "",
		}
		return analyzer.ExportSSAGraphToNeo4j(ssaResult, config)
	} else {

		return analyzer.ExportSSAGraphToCSV(ssaResult, outputPath)
	}
}

func RunSSASimplification(rootFunction string, projectPath string, outputPath string, packagePrefixes []string) ([]string, *ssa.Program, error) {
	callGraph, err := buildCallGraph(rootFunction, projectPath, outputPath)
	if err != nil {
		return nil, nil, err
	}

	if len(packagePrefixes) == 0 {
		packagePrefixes = []string{""}
	}
	ssaProgram := analyzer.SimplifySSA(callGraph, packagePrefixes)
	return packagePrefixes, ssaProgram, nil
}

// RunDumpPackages builds an SSA program and dumps package information
func RunDumpPackages(projectPath string) error {
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
	analyzer.DumpPackages(ssaProgram.AllPackages())

	return nil
}
