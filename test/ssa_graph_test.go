package test

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/throwin5tone7/go-call-analysis/internal/analyzer"
)

func TestSSAGraphAnalysis(t *testing.T) {
	// Test configuration
	projectPath := "../test-project"
	outputPath := "../test-output"
	goldenPath := "resources/golden/ssa"
	rootFunction := "github.com/throwin5tone7/go-call-analysis/test-project:main"
	packagePrefixes := []string{"github.com/throwin5tone7/go-call-analysis"}

	// Clean up previous test output
	if err := os.RemoveAll(outputPath); err != nil {
		t.Fatalf("Failed to clean up test output directory: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Parse root function
	rootFunctionId := &analyzer.FunctionId{
		Package:  strings.Split(rootFunction, ":")[0],
		Function: strings.Split(rootFunction, ":")[1],
	}

	// Create analysis config
	config, err := analyzer.NewAnalysisConfig(projectPath, outputPath, rootFunctionId)
	if err != nil {
		t.Fatalf("Failed to create analysis config: %v", err)
	}

	// Run call graph analysis
	callGraph, err := analyzer.CallGraphAnalysis(config)
	if err != nil {
		t.Fatalf("Failed to run call graph analysis: %v", err)
	}

	// Extract SSA graph data
	ssaResult := analyzer.ExtractSSAGraphData(callGraph, packagePrefixes)

	// Export to CSV
	if err := analyzer.ExportSSAGraphToCSV(ssaResult, outputPath); err != nil {
		t.Fatalf("Failed to export SSA graph to CSV: %v", err)
	}

	// Define expected CSV files
	expectedFiles := []string{
		"value_nodes.csv",
		"instruction_nodes.csv",
		"refer_edges.csv",
		"ordering_edges.csv",
		"ssa_ordering_edges.csv",
		"operand_edges.csv",
		"result_edges.csv",
	}

	// Check if all expected files were generated
	for _, filename := range expectedFiles {
		outputFile := filepath.Join(outputPath, filename)
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not generated", filename)
		}
	}

	// Compare each file with golden files
	for _, filename := range expectedFiles {
		t.Run(fmt.Sprintf("Compare_%s", filename), func(t *testing.T) {
			compareCSVFiles(t, goldenPath, outputPath, filename)
		})
	}
}

func compareCSVFiles(t *testing.T, goldenPath, outputPath, filename string) {
	goldenFile := filepath.Join(goldenPath, filename)
	outputFile := filepath.Join(outputPath, filename)

	// Check if golden file exists
	if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
		t.Errorf("Golden file %s does not exist", goldenFile)
		return
	}

	// Check if output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Output file %s does not exist", outputFile)
		return
	}

	// Read golden file
	goldenData, err := readCSVFile(goldenFile)
	if err != nil {
		t.Errorf("Failed to read golden file %s: %v", goldenFile, err)
		return
	}

	// Read output file
	outputData, err := readCSVFile(outputFile)
	if err != nil {
		t.Errorf("Failed to read output file %s: %v", outputFile, err)
		return
	}

	// Compare data (sort rows first to handle non-deterministic order)
	sortedGoldenData := sortCSVRows(goldenData)
	sortedOutputData := sortCSVRows(outputData)

	if !reflect.DeepEqual(sortedGoldenData, sortedOutputData) {
		t.Errorf("Files %s and %s differ", goldenFile, outputFile)
		t.Logf("Golden file has %d rows", len(goldenData))
		t.Logf("Output file has %d rows", len(outputData))

		// Show first few differences for debugging
		maxRows := len(sortedGoldenData)
		if len(sortedOutputData) < maxRows {
			maxRows = len(sortedOutputData)
		}

		for i := 0; i < maxRows && i < 5; i++ {
			if !reflect.DeepEqual(sortedGoldenData[i], sortedOutputData[i]) {
				t.Logf("Row %d differs:", i)
				t.Logf("  Golden: %v", sortedGoldenData[i])
				t.Logf("  Output: %v", sortedOutputData[i])
			}
		}
	}
}

func readCSVFile(filepath string) ([][]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var records [][]string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func sortCSVRows(rows [][]string) [][]string {
	if len(rows) <= 1 {
		return rows
	}

	// Create a copy to avoid modifying the original
	sorted := make([][]string, len(rows))
	copy(sorted, rows)

	// Sort by converting each row to a string and comparing
	// This ensures consistent ordering regardless of the original order
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			rowI := strings.Join(sorted[i], ",")
			rowJ := strings.Join(sorted[j], ",")
			if rowI > rowJ {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}
