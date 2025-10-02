package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/throwin5tone7/go-call-analysis/cmd/lib"
)

var rootCmd = &cobra.Command{
	Use:   "gca",
	Short: "Go Call Analysis - A tool for analyzing Go projects",
	Long:  `A command-line tool for analyzing Go projects and generating analysis reports.`,
}

var callGraphCmdRunner = func(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("path")
	outputPath, _ := cmd.Flags().GetString("output")
	rootFunction, _ := cmd.Flags().GetString("root-function")
	useNeo4j, _ := cmd.Flags().GetBool("neo4j")
	return lib.RunCallGraph(rootFunction, projectPath, outputPath, useNeo4j)
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

	return lib.RunSSAGraph(packagePrefixes, projectPath, outputPath, rootFunction, useNeo4j)
}
var ssaGraphCmd = &cobra.Command{
	Use:   "ssa-graph",
	Short: "Analyze a Go project using SSA",
	Long:  `Analyze a Go project and generate SSA-based call graph reports.`,
	RunE:  ssaGraphCmdRunner,
}

var dumpPackagesCmdRunner = func(cmd *cobra.Command, args []string) error {
	projectPath, _ := cmd.Flags().GetString("path")
	verbose, _ := cmd.Flags().GetBool("verbose")
	return lib.RunDumpPackages(projectPath, verbose)
}

var dumpPackagesCmd = &cobra.Command{
	Use:   "dump-packages",
	Short: "Build SSA program and dump package information",
	Long:  `Build an SSA program from a Go project and dump detailed package information to stdout.`,
	RunE:  dumpPackagesCmdRunner,
}

var outputSSACmdRunner = func(cmd *cobra.Command, args []string) error {
	packagePrefixes, _ := cmd.Flags().GetStringSlice("package-prefixes")
	projectPath, _ := cmd.Flags().GetString("path")
	outputPath, _ := cmd.Flags().GetString("output")
	rootFunction, _ := cmd.Flags().GetString("root-function")
	simplified, _ := cmd.Flags().GetBool("simplified")
	return lib.RunOutputSSA(packagePrefixes, projectPath, outputPath, rootFunction, simplified)
}

var outputSSACmd = &cobra.Command{
	Use:   "output-SSA",
	Short: "Output SSA program text for matching packages",
	Long:  `Build an SSA program from a Go project and output the textual representation of packages matching the specified prefixes.`,
	RunE:  outputSSACmdRunner,
}

var runKnownFixedWidthPropagationQueriesCmdRunner = func(cmd *cobra.Command, args []string) error {
	return lib.RunPropagationQueries()
}
var runKnownFixedWidthPropagationQueriesCmd = &cobra.Command{
	Use:   "known-fixed-propagation",
	Short: "Run known fixed width propagation queries on Neo4j",
	Long:  `Run known fixed width propagation queries on Neo4j.`,
	RunE:  runKnownFixedWidthPropagationQueriesCmdRunner,
}

var runKnownTwoVaryingPropagationQueriesCmdRunner = func(cmd *cobra.Command, args []string) error {
	return lib.RunTwoVaryingPropagationQueries()
}
var runKnownTwoVaryingPropagationQueriesCmd = &cobra.Command{

	Use:   "known-two-varying-propagation",
	Short: "Run known two varying width propagation queries on Neo4j",
	Long:  `Run known two varying width propagation queries on Neo4j.`,
	RunE:  runKnownTwoVaryingPropagationQueriesCmdRunner,
}

var runManualFunctionMarkingQueryCmdRunner = func(cmd *cobra.Command, args []string) error {
	functionId, _ := cmd.Flags().GetString("function-id")
	return lib.RunManualFunctionMarkingQuery(functionId)
}
var runManualFunctionMarkingQueryCmd = &cobra.Command{

	Use:   "mark-function-known-fixed",
	Short: "Mark a function as known to return fixed width",
	Long:  `Mark a function as known to return fixed width.`,
	RunE:  runManualFunctionMarkingQueryCmdRunner,
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

	dumpPackagesCmd.Flags().StringP("path", "p", "", "Path to the Go project to analyze")
	dumpPackagesCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output with detailed package information")

	outputSSACmd.Flags().StringP("path", "p", "", "Path to the Go project to analyze")
	outputSSACmd.Flags().StringP("output", "o", "", "Path to write SSA output file (outputs to stdout if not specified)")
	outputSSACmd.Flags().StringP("root-function", "r", "", "Root function to analyze")
	outputSSACmd.Flags().StringSlice("package-prefixes", []string{}, "Comma-separated list of package prefixes to include (e.g., 'github.com/user,example.com/project')")
	outputSSACmd.Flags().Bool("simplified", false, "Output simplified SSA form (default: false)")

	runManualFunctionMarkingQueryCmd.Flags().StringP("function-id", "f", "", "Function ID to mark as known to return fixed width")

	rootCmd.AddCommand(callGraphCmd)
	rootCmd.AddCommand(ssaGraphCmd)
	rootCmd.AddCommand(dumpPackagesCmd)
	rootCmd.AddCommand(outputSSACmd)
	rootCmd.AddCommand(runKnownFixedWidthPropagationQueriesCmd)
	rootCmd.AddCommand(runKnownTwoVaryingPropagationQueriesCmd)
	rootCmd.AddCommand(runManualFunctionMarkingQueryCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
