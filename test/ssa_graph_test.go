package test

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	main "github.com/throwin5tone7/go-call-analysis/cmd/lib"
	"github.com/throwin5tone7/go-call-analysis/internal/analyzer"
)

func TestSSAGraphAnalysis(t *testing.T) {
	// Test configuration
	projectPath := "../test-project"
	outputPath := "../test-output/ssa-graph"
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

	err := main.RunSSAGraph(packagePrefixes, projectPath, outputPath, rootFunction, false)
	if err != nil {
		t.Fatalf("Failed to run SSA graph analysis: %v", err)
	}

	// Define expected CSV files
	expectedFiles := []string{
		"file_version_nodes.csv",
		"belongs_to_edges.csv",
		"value_nodes.csv",
		"instruction_nodes.csv",
		"ordering_edges.csv",
		"control_flow_edges.csv",
		"operand_edges.csv",
		"result_edges.csv",
		"resolved_call_edges.csv",
	}

	// Check if all expected files were generated
	for _, filename := range expectedFiles {
		outputFile := filepath.Join(outputPath, filename)
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not generated", filename)
		}
	}

	// Check no other files were generated
	files, err := os.ReadDir(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output directory: %v", err)
	}
	for _, file := range files {
		if !slices.Contains(expectedFiles, file.Name()) {
			t.Errorf("Unexpected file %s was generated", file.Name())
		}
	}

	// Compare each file with golden files
	for _, filename := range expectedFiles {
		t.Run(fmt.Sprintf("Compare_%s", filename), func(t *testing.T) {
			compareCSVFiles(t, goldenPath, outputPath, filename)
		})
	}
}

func TestSimplifySSA(t *testing.T) {
	// Test configuration
	projectPath := "../test-project"
	outputPath := "../test-output/ssa-simplified"
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

	packagePrefixes, simplificationResult, err := main.RunSSASimplification(rootFunction, projectPath, outputPath, packagePrefixes)
	if err != nil {
		t.Fatalf("Failed to run SSA simplification: %v", err)
	}

	// Generate textual representation of SSA
	ssaText := analyzer.GenerateSSAText(simplificationResult.SSAProgram, packagePrefixes)

	// Write the textual representation to a file for comparison
	outputFile := filepath.Join(outputPath, "simplified_ssa.txt")
	if err := os.WriteFile(outputFile, []byte(ssaText), 0644); err != nil {
		t.Fatalf("Failed to write SSA text output: %v", err)
	}

	// Compare with golden file
	goldenFile := filepath.Join(goldenPath, "simplified_ssa.txt")
	compareTextFiles(t, goldenFile, outputFile, "simplified_ssa.txt")

	// Also compare text files for the unreachable functions - just a simple list of function names that need sorting
	unreachableFunctionsFile := filepath.Join(outputPath, "unreachable_functions.txt")
	if err := os.WriteFile(unreachableFunctionsFile, []byte(strings.Join(simplificationResult.AllUnreachableFunctions(), "\n")), 0644); err != nil {
		t.Fatalf("Failed to write unreachable functions output: %v", err)
	}
	unreachableGoldenFile := filepath.Join(goldenPath, "unreachable_functions.txt")
	compareTextFiles(t, unreachableGoldenFile, unreachableFunctionsFile, "unreachable_functions.txt")
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

// compareTextFiles compares two text files, handling potential differences in line endings
func compareTextFiles(t *testing.T, goldenFile, outputFile, filename string) {
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
	goldenData, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Errorf("Failed to read golden file %s: %v", goldenFile, err)
		return
	}

	// Read output file
	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Errorf("Failed to read output file %s: %v", outputFile, err)
		return
	}

	// Normalize line endings and compare
	goldenText := normalizeText(string(goldenData))
	outputText := normalizeText(string(outputData))

	if goldenText != outputText {
		t.Errorf("Files %s and %s differ", goldenFile, outputFile)

		// Show differences for debugging
		t.Logf("Golden file length: %d characters", len(goldenText))
		t.Logf("Output file length: %d characters", len(outputText))

		// Show first few lines for comparison
		goldenLines := strings.Split(goldenText, "\n")
		outputLines := strings.Split(outputText, "\n")

		maxLines := 10
		if len(goldenLines) < maxLines {
			maxLines = len(goldenLines)
		}
		if len(outputLines) < maxLines {
			maxLines = len(outputLines)
		}

		t.Logf("First %d lines comparison:", maxLines)
		for i := 0; i < maxLines; i++ {
			if i < len(goldenLines) && i < len(outputLines) {
				if goldenLines[i] != outputLines[i] {
					t.Logf("Line %d differs:", i+1)
					t.Logf("  Golden: %s", goldenLines[i])
					t.Logf("  Output: %s", outputLines[i])
				}
			}
		}
	}
}

// normalizeText normalizes text by standardizing line endings and removing trailing whitespace
func normalizeText(text string) string {
	// Replace Windows line endings with Unix line endings
	text = strings.ReplaceAll(text, "\r\n", "\n")

	// Split into lines and trim each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Join back together
	return strings.Join(lines, "\n")
}
