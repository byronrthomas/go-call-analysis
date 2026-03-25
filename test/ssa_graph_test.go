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
	"github.com/throwin5tone7/go-call-analysis/internal/analyzer/mock"
)

// updateGolden returns true when the UPDATE_GOLDEN env var is set to "1".
// In that mode comparison functions write normalised output to the golden
// directory instead of comparing against it.
func updateGolden() bool {
	return os.Getenv("UPDATE_GOLDEN") == "1"
}

// absPath resolves a path relative to the test directory to an absolute path.
func absPath(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("failed to resolve path %s: %v", rel, err)
	}
	return abs
}

// stripPrefix removes every occurrence of absPrefix from s so that
// machine-specific absolute path roots are stripped while the remainder
// (e.g. "test-project/main.go") is preserved for readability.
func stripPrefix(s, absPrefix string) string {
	if absPrefix == "" {
		return s
	}
	return strings.ReplaceAll(s, absPrefix, "")
}

// normaliseCSVData applies stripPrefix to every cell in a CSV data slice.
func normaliseCSVData(data [][]string, absPrefix string) [][]string {
	result := make([][]string, len(data))
	for i, row := range data {
		result[i] = make([]string, len(row))
		for j, cell := range row {
			result[i][j] = stripPrefix(cell, absPrefix)
		}
	}
	return result
}

func TestSSAGraphAnalysis(t *testing.T) {
	// Test configuration
	projectPath := "../test-project"
	outputPath := "../test-output/ssa-graph"
	goldenPath := "resources/golden/ssa"
	rootFunction := "github.com/throwin5tone7/go-call-analysis/test-project:main"
	packagePrefixes := []string{"github.com/throwin5tone7/go-call-analysis"}
	// Strip everything up to (but not including) "test-project/" so that
	// golden files contain portable paths like "test-project/main.go".
	absPathPrefix := filepath.Dir(absPath(t, projectPath)) + string(filepath.Separator)

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
		"function_nodes.csv",
		"ordering_edges.csv",
		"control_flow_edges.csv",
		"operand_edges.csv",
		"result_edges.csv",
		"resolved_call_edges.csv",
		"function_entry_edges.csv",
		"has_parameter_edges.csv",
		"return_point_edges.csv",
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
			compareCSVFiles(t, goldenPath, outputPath, filename, absPathPrefix)
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
	compareTextFiles(t, goldenFile, outputFile, "simplified_ssa.txt", "")

	// Also compare text files for the unreachable functions - just a simple list of function names that need sorting
	unreachableFunctionsFile := filepath.Join(outputPath, "unreachable_functions.txt")
	if err := os.WriteFile(unreachableFunctionsFile, []byte(strings.Join(simplificationResult.AllUnreachableFunctions(), "\n")), 0644); err != nil {
		t.Fatalf("Failed to write unreachable functions output: %v", err)
	}
	unreachableGoldenFile := filepath.Join(goldenPath, "unreachable_functions.txt")
	compareTextFiles(t, unreachableGoldenFile, unreachableFunctionsFile, "unreachable_functions.txt", "")
}

func compareCSVFiles(t *testing.T, goldenPath, outputPath, filename, absProjectPath string) {
	t.Helper()
	goldenFile := filepath.Join(goldenPath, filename)
	outputFile := filepath.Join(outputPath, filename)

	// Check if output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Output file %s does not exist", outputFile)
		return
	}

	// Read and normalise output
	outputData, err := readCSVFile(outputFile)
	if err != nil {
		t.Errorf("Failed to read output file %s: %v", outputFile, err)
		return
	}
	normalisedOutput := normaliseCSVData(outputData, absProjectPath)
	sortedOutput := sortCSVRows(normalisedOutput)

	if updateGolden() {
		if err := os.MkdirAll(goldenPath, 0755); err != nil {
			t.Fatalf("Failed to create golden directory: %v", err)
		}
		if err := writeCSVFile(goldenFile, sortedOutput); err != nil {
			t.Errorf("Failed to write golden file %s: %v", goldenFile, err)
		}
		return
	}

	// Check if golden file exists
	if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
		t.Errorf("Golden file %s does not exist (run with UPDATE_GOLDEN=1 to create it)", goldenFile)
		return
	}

	// Read golden (already normalised)
	goldenData, err := readCSVFile(goldenFile)
	if err != nil {
		t.Errorf("Failed to read golden file %s: %v", goldenFile, err)
		return
	}
	sortedGolden := sortCSVRows(goldenData)

	// Write sorted output back for easier golden updates
	if err := writeCSVFile(outputFile, sortedOutput); err != nil {
		t.Errorf("Failed to write sorted output data to %s: %v", outputFile, err)
		return
	}

	if !reflect.DeepEqual(sortedGolden, sortedOutput) {
		t.Errorf("Files %s and %s differ", goldenFile, outputFile)
		t.Logf("Golden file has %d rows", len(goldenData))
		t.Logf("Output file has %d rows", len(outputData))

		maxRows := len(sortedGolden)
		if len(sortedOutput) < maxRows {
			maxRows = len(sortedOutput)
		}

		for i := 0; i < maxRows && i < 5; i++ {
			if !reflect.DeepEqual(sortedGolden[i], sortedOutput[i]) {
				t.Logf("Row %d differs:", i)
				t.Logf("  Golden: %v", sortedGolden[i])
				t.Logf("  Output: %v", sortedOutput[i])
			}
		}
	}
}

func writeCSVFile(filepath string, data [][]string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if err := writer.WriteAll(data); err != nil {
		return err
	}
	writer.Flush()
	return nil
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
	for i := 1; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			// NOTE: This relies on the fact that the first column is the ID
			// or the first two columns are from_id and to_id
			rowI := sorted[i][0]
			rowJ := sorted[j][0]
			if rowI == rowJ {
				rowI = sorted[i][1]
				rowJ = sorted[j][1]
			}
			if rowI > rowJ {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// compareTextFiles compares two text files, handling potential differences in line endings.
// absProjectPath is stripped from both files before comparison; pass "" to skip stripping.
func compareTextFiles(t *testing.T, goldenFile, outputFile, filename, absProjectPath string) {
	t.Helper()

	// Check if output file exists
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("Output file %s does not exist", outputFile)
		return
	}

	// Read and normalise output
	outputData, err := os.ReadFile(outputFile)
	if err != nil {
		t.Errorf("Failed to read output file %s: %v", outputFile, err)
		return
	}
	outputText := normalizeText(stripPrefix(string(outputData), absProjectPath))

	if updateGolden() {
		if err := os.MkdirAll(filepath.Dir(goldenFile), 0755); err != nil {
			t.Fatalf("Failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenFile, []byte(outputText), 0644); err != nil {
			t.Errorf("Failed to write golden file %s: %v", goldenFile, err)
		}
		return
	}

	// Check if golden file exists
	if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
		t.Errorf("Golden file %s does not exist (run with UPDATE_GOLDEN=1 to create it)", goldenFile)
		return
	}

	// Read golden (already normalised)
	goldenData, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Errorf("Failed to read golden file %s: %v", goldenFile, err)
		return
	}
	goldenText := normalizeText(string(goldenData))

	if goldenText != outputText {
		t.Errorf("Files %s and %s differ", goldenFile, outputFile)

		t.Logf("Golden file length: %d characters", len(goldenText))
		t.Logf("Output file length: %d characters", len(outputText))

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

func TestNeo4jSSAGraphExport(t *testing.T) {

	// Setup mock mode
	originalMockMode := analyzer.InMockMode
	analyzer.InMockMode = true
	defer func() {
		analyzer.InMockMode = originalMockMode
		analyzer.MockSession = nil
	}()

	// Create test data
	projectPath := "../test-project"
	outputPath := "../test-output/neo4j-ssagraph"
	goldenPath := "resources/golden/neo4j"
	rootFunction := "github.com/throwin5tone7/go-call-analysis/test-project:main"
	packagePrefixes := []string{"github.com/throwin5tone7/go-call-analysis"}
	absPathPrefix := filepath.Dir(absPath(t, projectPath)) + string(filepath.Separator)

	// Clean up previous test output
	if err := os.RemoveAll(outputPath); err != nil {
		t.Fatalf("Failed to clean up test output directory: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	err := main.RunSSAGraph(packagePrefixes, projectPath, outputPath, rootFunction, true)
	if err != nil {
		t.Fatalf("Failed to run SSA graph analysis for Neo4j: %v", err)
	}

	// Capture the mock session results
	if analyzer.MockSession == nil {
		t.Fatalf("MockSession was not created")
	}

	var sb strings.Builder
	mockSession := analyzer.MockSession.(*mock.MockSession)
	mockSession.FormatCapturedQueries(&sb)

	// Write captured queries to output file
	outputFile := filepath.Join(outputPath, "captured_queries.txt")
	if err := os.WriteFile(outputFile, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("Failed to write captured queries: %v", err)
	}

	// Compare with golden file
	goldenFile := filepath.Join(goldenPath, "ssagraph_queries.txt")
	compareTextFiles(t, goldenFile, outputFile, "ssagraph_queries.txt", absPathPrefix)
}
