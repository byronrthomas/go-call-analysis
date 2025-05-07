package analyzer

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

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
