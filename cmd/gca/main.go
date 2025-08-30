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

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze a Go project",
	Long:  `Analyze a Go project and generate analysis reports.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectPath, _ := cmd.Flags().GetString("path")
		outputPath, _ := cmd.Flags().GetString("output")
		useNeo4j, _ := cmd.Flags().GetBool("neo4j")
		rootFunction, _ := cmd.Flags().GetString("root-function")
		var rootFunctionId *analyzer.FunctionId
		if rootFunction != "" {
			rootFunctionId = &analyzer.FunctionId{
				Package:  strings.Split(rootFunction, ":")[0],
				Function: strings.Split(rootFunction, ":")[1],
			}
		}
		if projectPath == "" {
			return fmt.Errorf("project path is required")
		}

		fmt.Printf("Analyzing project at: %s\n", projectPath)
		config, err := analyzer.NewAnalysisConfig(projectPath, outputPath, rootFunctionId)
		if err != nil {
			return err
		}
		callGraph, err := analyzer.CallGraphAnalysis(config)
		if err != nil {
			return err
		}

		// Extract call graph data
		nodes, edges := analyzer.ExtractCallGraphData(callGraph)

		// Handle output based on flags
		if useNeo4j {
			// Use hardcoded Neo4j connection details
			config := analyzer.Neo4jConfig{
				URI:      "bolt://localhost:7687",
				Username: "",
				Password: "",
				Database: "",
			}
			return analyzer.ExportCallGraphToNeo4j(nodes, edges, config)
		} else {
			// Use CSV output (default behavior)
			return analyzer.ExportCallGraphToCSV(nodes, edges, outputPath)
		}
	},
}

func init() {
	analyzeCmd.Flags().StringP("path", "p", "", "Path to the Go project to analyze")
	analyzeCmd.Flags().StringP("output", "o", "", "Path to write analysis results (for CSV output)")
	analyzeCmd.Flags().StringP("root-function", "r", "", "Root function to analyze")
	analyzeCmd.Flags().Bool("neo4j", false, "Export results to Neo4j instead of CSV")
	rootCmd.AddCommand(analyzeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
