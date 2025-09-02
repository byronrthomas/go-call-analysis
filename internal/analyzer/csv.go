package analyzer

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
)

const MAX_CSV_LINES = 200_000 // Maximum number of lines per CSV file

// writeCSVToFiles writes data to one or more CSV files, splitting if necessary
func writeCSVToFiles(data [][]string, basePath string, header []string) error {
	totalLines := len(data)
	if totalLines <= MAX_CSV_LINES {
		// Single file case
		file, err := os.Create(basePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", basePath, err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header to %s: %v", basePath, err)
		}
		if err := writer.WriteAll(data); err != nil {
			return fmt.Errorf("failed to write data to %s: %v", basePath, err)
		}
		writer.Flush()
		return nil
	}

	// Multiple files case
	numFiles := (totalLines + MAX_CSV_LINES - 1) / MAX_CSV_LINES
	ext := filepath.Ext(basePath)
	baseName := basePath[:len(basePath)-len(ext)]

	for i := range numFiles {
		start := i * MAX_CSV_LINES
		end := min(start+MAX_CSV_LINES, totalLines)

		// Create filename with index
		filename := fmt.Sprintf("%s-%d%s", baseName, i+1, ext)
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", filename, err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		// Write header to each file
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("failed to write header to %s: %v", filename, err)
		}
		// Write data chunk
		if err := writer.WriteAll(data[start:end]); err != nil {
			return fmt.Errorf("failed to write data to %s: %v", filename, err)
		}
		writer.Flush()
	}
	return nil
}

// getOrderedKeys returns the keys in the specified order for CSV columns
func getOrderedKeys(keys []string) []string {
	// Check if "id" exists - it should be first
	hasId := false
	for _, key := range keys {
		if key == "id" {
			hasId = true
			break
		}
	}

	// Check if "from_id" and "to_id" exist
	hasFromId := false
	hasToId := false
	for _, key := range keys {
		if key == "from_id" {
			hasFromId = true
		}
		if key == "to_id" {
			hasToId = true
		}
	}

	// Create ordered keys
	var orderedKeys []string

	// Add "id" first if it exists
	if hasId {
		orderedKeys = append(orderedKeys, "id")
	}

	// Add "from_id" and "to_id" if they exist and "id" doesn't
	if !hasId && hasFromId && hasToId {
		orderedKeys = append(orderedKeys, "from_id", "to_id")
	}

	// Add remaining keys in sorted order
	var remainingKeys []string
	for _, key := range keys {
		if key == "id" {
			continue // Already added
		}
		if (key == "from_id" || key == "to_id") && !hasId {
			continue // Already added
		}
		remainingKeys = append(remainingKeys, key)
	}
	sort.Strings(remainingKeys)
	orderedKeys = append(orderedKeys, remainingKeys...)

	return orderedKeys
}

// ExportToCSV exports data to CSV files using a generic approach
// dataMap is a map from string (filename) to list of Mappable instances
func ExportToCSV(dataMap map[string][]graphcommon.Mappable, outputPath string) error {
	// Output to files or stdout
	if outputPath == "" {
		// Output to stdout
		for filename, items := range dataMap {
			if len(items) == 0 {
				continue
			}

			fmt.Printf("\n%s:\n", filename)
			writer := csv.NewWriter(os.Stdout)

			// Get the first item to determine keys
			firstItem := items[0].ToMap()
			keys := make([]string, 0, len(firstItem))
			for key := range firstItem {
				keys = append(keys, key)
			}

			// Get ordered keys
			orderedKeys := getOrderedKeys(keys)

			// Write header
			if err := writer.Write(orderedKeys); err != nil {
				return fmt.Errorf("failed to write header for %s: %v", filename, err)
			}

			// Write data rows
			var rows [][]string
			for _, item := range items {
				itemMap := item.ToMap()
				row := make([]string, len(orderedKeys))
				for i, key := range orderedKeys {
					if value, exists := itemMap[key]; exists {
						row[i] = fmt.Sprintf("%v", value)
					} else {
						row[i] = ""
					}
				}
				rows = append(rows, row)
			}

			if err := writer.WriteAll(rows); err != nil {
				return fmt.Errorf("failed to write data for %s: %v", filename, err)
			}
			writer.Flush()
		}
	} else {
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		// Write each dataset to its own file(s)
		for filename, items := range dataMap {
			if len(items) == 0 {
				continue
			}

			// Get the first item to determine keys
			firstItem := items[0].ToMap()
			keys := make([]string, 0, len(firstItem))
			for key := range firstItem {
				keys = append(keys, key)
			}

			// Get ordered keys
			orderedKeys := getOrderedKeys(keys)

			// Convert items to CSV rows
			var rows [][]string
			for _, item := range items {
				itemMap := item.ToMap()
				row := make([]string, len(orderedKeys))
				for i, key := range orderedKeys {
					if value, exists := itemMap[key]; exists {
						row[i] = fmt.Sprintf("%v", value)
					} else {
						row[i] = ""
					}
				}
				rows = append(rows, row)
			}

			// Write to file(s)
			filePath := filepath.Join(outputPath, filename+".csv")
			if err := writeCSVToFiles(rows, filePath, orderedKeys); err != nil {
				return fmt.Errorf("failed to write %s: %v", filename, err)
			}
		}
	}

	return nil
}

// ExportCallGraphToCSV exports the call graph data to CSV files (legacy function for backward compatibility)
func ExportCallGraphToCSV(nodes []FunctionNode, edges []CallEdge, outputPath string) error {
	dataMap := make(map[string][]graphcommon.Mappable)

	// Convert nodes to Mappable interface
	nodeMappables := make([]graphcommon.Mappable, len(nodes))
	for i := range nodes {
		nodeMappables[i] = &nodes[i]
	}
	dataMap["nodes"] = nodeMappables

	// Convert edges to Mappable interface
	edgeMappables := make([]graphcommon.Mappable, len(edges))
	for i := range edges {
		edgeMappables[i] = &edges[i]
	}
	dataMap["edges"] = edgeMappables

	return ExportToCSV(dataMap, outputPath)
}

// ExportSSAGraphToCSV exports the SSA graph data to CSV files
func ExportSSAGraphToCSV(ssaData SSAGraphData, outputPath string) error {
	dataMap := make(map[string][]graphcommon.Mappable)

	// Convert value nodes to Mappable interface
	valueNodeMappables := make([]graphcommon.Mappable, len(ssaData.ValueNodes))
	for i := range ssaData.ValueNodes {
		valueNodeMappables[i] = &ssaData.ValueNodes[i]
	}
	dataMap["value_nodes"] = valueNodeMappables

	// Convert instruction nodes to Mappable interface
	instructionNodeMappables := make([]graphcommon.Mappable, len(ssaData.InstructionNodes))
	for i := range ssaData.InstructionNodes {
		instructionNodeMappables[i] = &ssaData.InstructionNodes[i]
	}
	dataMap["instruction_nodes"] = instructionNodeMappables

	// Convert ordering edges to Mappable interface
	orderingEdgeMappables := make([]graphcommon.Mappable, len(ssaData.OrderingEdges))
	for i := range ssaData.OrderingEdges {
		orderingEdgeMappables[i] = &ssaData.OrderingEdges[i]
	}
	dataMap["ordering_edges"] = orderingEdgeMappables

	// Convert control flow edges to Mappable interface
	controlFlowEdgeMappables := make([]graphcommon.Mappable, len(ssaData.ControlFlowEdges))
	for i := range ssaData.ControlFlowEdges {
		controlFlowEdgeMappables[i] = &ssaData.ControlFlowEdges[i]
	}
	dataMap["control_flow_edges"] = controlFlowEdgeMappables

	// Convert operand edges to Mappable interface
	operandEdgeMappables := make([]graphcommon.Mappable, len(ssaData.OperandEdges))
	for i := range ssaData.OperandEdges {
		operandEdgeMappables[i] = &ssaData.OperandEdges[i]
	}
	dataMap["operand_edges"] = operandEdgeMappables

	// Convert result edges to Mappable interface
	resultEdgeMappables := make([]graphcommon.Mappable, len(ssaData.ResultEdges))
	for i := range ssaData.ResultEdges {
		resultEdgeMappables[i] = &ssaData.ResultEdges[i]
	}
	dataMap["result_edges"] = resultEdgeMappables

	// Convert resolved call edges to Mappable interface
	resolvedCallEdgeMappables := make([]graphcommon.Mappable, len(ssaData.ResolvedCallEdges))
	for i := range ssaData.ResolvedCallEdges {
		resolvedCallEdgeMappables[i] = &ssaData.ResolvedCallEdges[i]
	}
	dataMap["resolved_call_edges"] = resolvedCallEdgeMappables

	return ExportToCSV(dataMap, outputPath)
}
