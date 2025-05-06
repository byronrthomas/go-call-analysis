package main

import (
	"fmt"
	"os"

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

		if projectPath == "" {
			return fmt.Errorf("project path is required")
		}

		fmt.Printf("Analyzing project at: %s\n", projectPath)
		analyzer, err := analyzer.NewAnalyzer(projectPath, outputPath)
		if err != nil {
			return err
		}
		callGraph, err := analyzer.Analyze()
		if err != nil {
			return err
		}
		fmt.Println(callGraph)
		return nil
	},
}

func init() {
	analyzeCmd.Flags().StringP("path", "p", "", "Path to the Go project to analyze")
	analyzeCmd.Flags().StringP("output", "o", "", "Path to write analysis results")
	rootCmd.AddCommand(analyzeCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
