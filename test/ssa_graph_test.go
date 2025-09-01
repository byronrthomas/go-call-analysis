package test

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/throwin5tone7/go-call-analysis/internal/analyzer"
	"golang.org/x/tools/go/ssa"
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

func TestSimplifySSA(t *testing.T) {
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

	// Call SimplifySSA function
	simplifiedSSA := analyzer.SimplifySSA(callGraph, packagePrefixes)
	if simplifiedSSA == nil {
		t.Fatalf("SimplifySSA returned nil")
	}

	// Generate textual representation of SSA
	ssaText := generateSSAText(simplifiedSSA, packagePrefixes)

	// Write the textual representation to a file for comparison
	outputFile := filepath.Join(outputPath, "simplified_ssa.txt")
	if err := os.WriteFile(outputFile, []byte(ssaText), 0644); err != nil {
		t.Fatalf("Failed to write SSA text output: %v", err)
	}

	// Compare with golden file
	goldenFile := filepath.Join(goldenPath, "simplified_ssa.txt")
	compareTextFiles(t, goldenFile, outputFile, "simplified_ssa.txt")
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

// generateSSAText generates a textual representation of the SSA program
// by processing each package that matches the package prefixes
func generateSSAText(prog *ssa.Program, packagePrefixes []string) string {
	var result strings.Builder

	// Helper function to check if a package path matches any of the prefixes
	matchesPrefix := func(pkgPath string) bool {
		for _, prefix := range packagePrefixes {
			if prefix == "" || strings.HasPrefix(pkgPath, prefix) {
				return true
			}
		}
		return false
	}

	// Process each package
	for _, pkg := range prog.AllPackages() {
		// Check if the package path matches any of the provided prefixes
		if matchesPrefix(pkg.Pkg.Path()) {
			result.WriteString(fmt.Sprintf("Package: %s\n", pkg.Pkg.Path()))
			result.WriteString(strings.Repeat("-", 50) + "\n")

			// Collect members to sort them for deterministic output
			var functions []*ssa.Function
			var values []ssa.Value

			for _, mem := range pkg.Members {
				if f, ok := mem.(*ssa.Function); ok {
					functions = append(functions, f)
				} else if v, ok := mem.(ssa.Value); ok {
					values = append(values, v)
				}
			}

			// Sort functions by name for deterministic output
			sort.Slice(functions, func(i, j int) bool {
				return functions[i].Name() < functions[j].Name()
			})

			// Sort values by name for deterministic output
			sort.Slice(values, func(i, j int) bool {
				return values[i].Name() < values[j].Name()
			})

			// Process sorted functions
			for _, f := range functions {
				result.WriteString(fmt.Sprintf("Function: %s\n", f.Name()))

				// Process each basic block
				for blockIndex, block := range f.Blocks {
					printBlockInfo(&result, blockIndex, block)

					// Process each instruction in the block
					for instrIndex, instr := range block.Instrs {
						result.WriteString(fmt.Sprintf("    %d: %s\n", instrIndex, instr.String()))

						// If the instruction is also a value, show its name
						if val, ok := instr.(ssa.Value); ok {
							if val.Name() != "" {
								outputValue(&result, "      As value: ", "  Type: ", val)
							}
						}

						// Show operands
						for i, op := range instr.Operands(make([]*ssa.Value, 0)) {
							if *op == nil {
								continue
							}
							outputValue(&result, fmt.Sprintf("      Operand %d: ", i), "  Type: ", *op)
						}
					}

					result.WriteString("\n")
				}
				result.WriteString("\n")
			}

			// Process sorted values
			for _, v := range values {
				outputValue(&result, "Value: ", "\n  Type: ", v)
			}
			result.WriteString("\n")
		}
	}

	return result.String()
}

func printBlockInfo(result *strings.Builder, blockIndex int, block *ssa.BasicBlock) {
	fmt.Fprintf(result, "  Block %d -", blockIndex)
	// Show predecessors, sorted by index, on one line
	preds := make([]int, len(block.Preds))
	for i, pred := range block.Preds {
		preds[i] = pred.Index
	}
	sort.Ints(preds)
	fmt.Fprintf(result, "   Predecessors [")
	for i, pred := range preds {
		if i != len(preds)-1 {
			fmt.Fprintf(result, "%d, ", pred)
		} else {
			fmt.Fprintf(result, "%d", pred)
		}
	}
	fmt.Fprintf(result, "]")
	// Show successors, sorted by index, on one line
	succs := make([]int, len(block.Succs))
	for i, succ := range block.Succs {
		succs[i] = succ.Index
	}
	sort.Ints(succs)
	fmt.Fprintf(result, "   Successors [")
	for i, succ := range succs {
		if i != len(succs)-1 {
			fmt.Fprintf(result, "%d, ", succ)
		} else {
			fmt.Fprintf(result, "%d", succ)
		}
	}
	fmt.Fprintf(result, "]:\n")
}

func outputValue(result *strings.Builder, valuePrefix string, typePrefix string, v ssa.Value) {
	fmt.Fprint(result, valuePrefix)
	fmt.Fprint(result, v.Name())
	if v.Type() != nil {
		fmt.Fprint(result, typePrefix)
		fmt.Fprint(result, v.Type().String())
	}
	fmt.Fprint(result, "\n")
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
