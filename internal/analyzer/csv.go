package analyzer

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
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

// ExportCallGraphToCSV exports the call graph data to CSV files
func ExportCallGraphToCSV(nodes []FunctionNode, edges []CallEdge, outputPath string) error {
	// Convert nodes to CSV format
	var nodeRows [][]string
	nodeHeader := []string{"id", "name", "package", "label", "file", "line", "char"}
	for _, node := range nodes {
		nodeRows = append(nodeRows, []string{
			node.ID,
			node.Name,
			node.Package,
			"Function", // Constant label for all function nodes
			node.File,
			fmt.Sprintf("%d", node.Line),
			fmt.Sprintf("%d", node.Column),
		})
	}

	// Convert edges to CSV format
	var edgeRows [][]string
	edgeHeader := []string{"id_from", "id_to", "type"}
	for _, edge := range edges {
		edgeRows = append(edgeRows, []string{
			edge.FromID,
			edge.ToID,
			"CALLS", // Constant type for all call edges
		})
	}

	// Output to files or stdout
	if outputPath == "" {
		// Output nodes to stdout
		fmt.Println("Nodes:")
		writer := csv.NewWriter(os.Stdout)
		writer.WriteAll(nodeRows)
		writer.Flush()

		// Output edges to stdout
		fmt.Println("\nEdges:")
		writer = csv.NewWriter(os.Stdout)
		writer.WriteAll(edgeRows)
		writer.Flush()
	} else {
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		// Write nodes to file(s)
		nodesPath := filepath.Join(outputPath, "nodes.csv")
		if err := writeCSVToFiles(nodeRows, nodesPath, nodeHeader); err != nil {
			return fmt.Errorf("failed to write nodes: %v", err)
		}

		// Write edges to file(s)
		edgesPath := filepath.Join(outputPath, "edges.csv")
		if err := writeCSVToFiles(edgeRows, edgesPath, edgeHeader); err != nil {
			return fmt.Errorf("failed to write edges: %v", err)
		}
	}

	return nil
}
