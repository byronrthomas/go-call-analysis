package lib

import (
	"fmt"
	"strings"

	"github.com/throwin5tone7/go-call-analysis/internal/analyzer"
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
	callGraph, err := buildCallGraph(rootFunction, projectPath, outputPath)
	if err != nil {
		return err
	}

	if len(packagePrefixes) == 0 {
		packagePrefixes = []string{""}
	}
	ssaProgram := analyzer.SimplifySSA(callGraph, packagePrefixes)
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
