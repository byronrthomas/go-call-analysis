package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/throwin5tone7/go-call-analysis/internal/analyzer"
)

var rootCmd = &cobra.Command{
	Use:   "gca",
	Short: "Go Call Analysis - A tool for analyzing Go projects",
	Long:  `A command-line tool for analyzing Go projects and generating analysis reports.`,
}

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

var callGraphCmdRunner = func(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("path")
	outputPath, _ := cmd.Flags().GetString("output")
	rootFunction, _ := cmd.Flags().GetString("root-function")
	callGraph, err := buildCallGraph(rootFunction, projectPath, outputPath)
	if err != nil {
		return err
	}

	nodes, edges := analyzer.ExtractCallGraphData(callGraph)

	useNeo4j, _ := cmd.Flags().GetBool("neo4j")

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

var callGraphCmd = &cobra.Command{
	Use:   "call-graph",
	Short: "Analyze a Go project",
	Long:  `Analyze a Go project and generate call graph reports.`,
	RunE:  callGraphCmdRunner,
}

var ssaGraphCmdRunner = func(cmd *cobra.Command, args []string) error {
	packagePrefixes, _ := cmd.Flags().GetStringSlice("package-prefixes")
	projectPath, _ := cmd.Flags().GetString("path")
	outputPath, _ := cmd.Flags().GetString("output")
	rootFunction, _ := cmd.Flags().GetString("root-function")
	useNeo4j, _ := cmd.Flags().GetBool("neo4j")

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
var ssaGraphCmd = &cobra.Command{
	Use:   "ssa-graph",
	Short: "Analyze a Go project using SSA",
	Long:  `Analyze a Go project and generate SSA-based call graph reports.`,
	RunE:  ssaGraphCmdRunner,
}

func init() {
	// Common flags for both commands
	callGraphCmd.Flags().StringP("path", "p", "", "Path to the Go project to analyze")
	callGraphCmd.Flags().StringP("output", "o", "", "Path to write analysis results (for CSV output)")
	callGraphCmd.Flags().StringP("root-function", "r", "", "Root function to analyze")
	callGraphCmd.Flags().Bool("neo4j", false, "Export results to Neo4j instead of CSV")

	ssaGraphCmd.Flags().StringP("path", "p", "", "Path to the Go project to analyze")
	ssaGraphCmd.Flags().StringP("output", "o", "", "Path to write analysis results (for CSV output)")
	ssaGraphCmd.Flags().StringP("root-function", "r", "", "Root function to analyze")
	ssaGraphCmd.Flags().Bool("neo4j", false, "Export results to Neo4j instead of CSV")
	ssaGraphCmd.Flags().StringSlice("package-prefixes", []string{}, "Comma-separated list of package prefixes to include (e.g., 'github.com/user,example.com/project')")

	rootCmd.AddCommand(callGraphCmd)
	rootCmd.AddCommand(ssaGraphCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
