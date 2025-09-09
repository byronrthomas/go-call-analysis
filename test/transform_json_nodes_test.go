package test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestTransformJSONNodes(t *testing.T) {
	// Test configuration
	inputFile := "resources/transform-json-nodes/sample_input.jsonl"
	outputPath := "../test-output/transform-json-nodes"
	goldenPath := "resources/golden/transform-json-nodes"
	relativeRoot := "/Users/byron/repos/third-party/injective/injective-core"

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
