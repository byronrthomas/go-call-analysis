package test

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
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

func TestTransformJSONNodes(t *testing.T) {
	// Test configuration
	inputFile := "resources/transform-json-nodes/sample_input.jsonl"
	outputPath := "../test-output/transform-json-nodes"
	goldenPath := "resources/golden/transform-json-nodes"
	relativeRoot := "/Users/byron/repos/third-party/injective"

	// Build the transform tool if it doesn't exist
	toolPath := "../bin/transform-json-nodes"
	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		t.Log("Building transform-json-nodes tool...")
		cmd := exec.Command("make", "build-transform")
		cmd.Dir = ".."
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build transform-json-nodes tool: %v\nOutput: %s", err, output)
		}
	}

	// Clean up previous test output
	if err := os.RemoveAll(outputPath); err != nil {
		t.Fatalf("Failed to clean up test output directory: %v", err)
	}

	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Run the transform-json-nodes tool
	cmd := exec.Command(toolPath,
		"-input", inputFile,
		"-root", relativeRoot,
		"-output", outputPath)
	cmd.Dir = "."

	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to run transform-json-nodes tool: %v\nOutput: %s", err, output)
	}

	// Compare output with golden files
	compareDirectories(t, goldenPath, outputPath)
}

// compareDirectories recursively compares two directory structures and their contents
func compareDirectories(t *testing.T, goldenPath, outputPath string) {
	// Check if golden directory exists
	if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
		t.Errorf("Golden directory %s does not exist", goldenPath)
		return
	}

	// Check if output directory exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output directory %s does not exist", outputPath)
		return
	}

	// Walk through golden directory and compare each file
	err := filepath.Walk(goldenPath, func(goldenFilePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from golden directory
		relPath, err := filepath.Rel(goldenPath, goldenFilePath)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Construct corresponding output path
		outputFilePath := filepath.Join(outputPath, relPath)

		if info.IsDir() {
			// Check if directory exists in output
			if _, err := os.Stat(outputFilePath); os.IsNotExist(err) {
				t.Errorf("Expected directory %s does not exist in output", relPath)
			}
		} else {
			// Compare file contents
			t.Run(fmt.Sprintf("Compare_%s", strings.ReplaceAll(relPath, string(filepath.Separator), "_")), func(t *testing.T) {
				compareJSONFiles(t, goldenFilePath, outputFilePath, relPath)
			})
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking golden directory: %v", err)
	}

	// Also check for unexpected files in output directory
	err = filepath.Walk(outputPath, func(outputFilePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from output directory
		relPath, err := filepath.Rel(outputPath, outputFilePath)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Construct corresponding golden path
		goldenFilePath := filepath.Join(goldenPath, relPath)

		if !info.IsDir() {
			// Check if file exists in golden directory
			if _, err := os.Stat(goldenFilePath); os.IsNotExist(err) {
				t.Errorf("Unexpected file %s found in output", relPath)
			}
		}

		return nil
	})

	if err != nil {
		t.Errorf("Error walking output directory: %v", err)
	}
}

// compareJSONFiles compares two JSON files containing arrays of objects
func compareJSONFiles(t *testing.T, goldenFile, outputFile, filename string) {
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
	goldenData, err := readJSONArrayFile(goldenFile)
	if err != nil {
		t.Errorf("Failed to read golden file %s: %v", goldenFile, err)
		return
	}

	// Read output file
	outputData, err := readJSONArrayFile(outputFile)
	if err != nil {
		t.Errorf("Failed to read output file %s: %v", outputFile, err)
		return
	}

	// Compare data (sort arrays first to handle non-deterministic order)
	sortedGoldenData := sortJSONArray(goldenData)
	sortedOutputData := sortJSONArray(outputData)

	if !reflect.DeepEqual(sortedGoldenData, sortedOutputData) {
		t.Errorf("Files %s and %s differ", goldenFile, outputFile)
		t.Logf("Golden file has %d entries", len(goldenData))
		t.Logf("Output file has %d entries", len(outputData))

		// Show first few differences for debugging
		maxEntries := len(sortedGoldenData)
		if len(sortedOutputData) < maxEntries {
			maxEntries = len(sortedOutputData)
		}

		for i := 0; i < maxEntries && i < 3; i++ {
			if !reflect.DeepEqual(sortedGoldenData[i], sortedOutputData[i]) {
				t.Logf("Entry %d differs:", i)
				goldenJSON, _ := json.MarshalIndent(sortedGoldenData[i], "    ", "  ")
				outputJSON, _ := json.MarshalIndent(sortedOutputData[i], "    ", "  ")
				t.Logf("  Golden: %s", string(goldenJSON))
				t.Logf("  Output: %s", string(outputJSON))
			}
		}
	}
}

// readJSONArrayFile reads a JSON file containing an array of objects
func readJSONArrayFile(filepath string) ([]interface{}, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var jsonArray []interface{}
	if err := json.Unmarshal(data, &jsonArray); err != nil {
		return nil, err
	}

	return jsonArray, nil
}

// sortJSONArray sorts an array of JSON objects by their string representation for consistent comparison
func sortJSONArray(data []interface{}) []interface{} {
	if len(data) <= 1 {
		return data
	}

	// Create a copy to avoid modifying the original
	sorted := make([]interface{}, len(data))
	copy(sorted, data)

	// Sort by converting each object to a string and comparing
	sort.Slice(sorted, func(i, j int) bool {
		jsonI, _ := json.Marshal(sorted[i])
		jsonJ, _ := json.Marshal(sorted[j])
		return string(jsonI) < string(jsonJ)
	})

	return sorted
}
