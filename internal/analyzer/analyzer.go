package analyzer

import (
	"encoding/csv"
	"fmt"
	"go/token"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

const MAX_CSV_LINES = 200_000 // Maximum number of lines per CSV file

// Analyzer represents the main analysis engine
type Analyzer struct {
	projectPath string
	outputPath  string
}

// NewAnalyzer creates a new analyzer instance
func NewAnalyzer(projectPath, outputPath string) (*Analyzer, error) {
	if projectPath == "" {
		return nil, fmt.Errorf("project path is required")
	}

	return &Analyzer{
		projectPath: projectPath,
		outputPath:  outputPath,
	}, nil
}

type CallGraphResult struct {
	CallGraph  *callgraph.Graph
	FileSet    *token.FileSet
	OutputPath string
}

// Analyze performs the analysis on the target project
func (a *Analyzer) Analyze() (*CallGraphResult, error) {
	// TODO: Implement analysis logic
	// Load the packages (set your target package here)
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedDeps | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   a.projectPath,
		Fset:  token.NewFileSet(),
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatalf("failed to load packages: %v", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		log.Fatal("package loading failed due to errors")
	}
	fmt.Printf("Loaded %d packages\n", len(pkgs))

	// Create SSA packages for well-typed packages and their dependencies.
	prog, ssaPkgs := ssautil.AllPackages(pkgs, ssa.PrintPackages|ssa.InstantiateGenerics)
	_ = ssaPkgs

	// Build SSA code for the whole program.
	prog.Build()
	// prog := ssa.NewProgram(cfg.Fset, ssa.SanityCheckFunctions)
	// var ssaPkgs []*ssa.Package
	// for _, pkg := range pkgs {
	// 	if pkg.Types != nil && pkg.Syntax != nil && pkg.TypesInfo != nil {
	// 		ssaPkg := prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
	// 		ssaPkgs = append(ssaPkgs, ssaPkg)
	// 	} else {
	// 		noTypes := pkg.Types == nil
	// 		noSyntax := pkg.Syntax == nil
	// 		noTypesInfo := pkg.TypesInfo == nil
	// 		fmt.Printf("Skipping package %s: NoTypes: %v, NoSyntax: %v, NoTypesInfo: %v\n", pkg.PkgPath, noTypes, noSyntax, noTypesInfo)
	// 	}
	// }
	// prog.Build()

	// Perform RTA (Rapid Type Analysis) to build call graph
	var functions []*ssa.Function
	for _, pkg := range ssaPkgs {
		for _, fn := range pkg.Members {
			if f, ok := fn.(*ssa.Function); ok {
				functions = append(functions, f)
			}
		}
	}
	rtaRes := rta.Analyze(functions, true)
	callGraph := rtaRes.CallGraph

	// Print out some call graph edges
	callGraph.DeleteSyntheticNodes() // remove built-in or synthetic calls
	return &CallGraphResult{
		CallGraph:  callGraph,
		FileSet:    prog.Fset,
		OutputPath: a.outputPath,
	}, nil
}

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

// FunctionNode represents a function in the call graph
type FunctionNode struct {
	ID      string
	Name    string
	Package string
	Label   string
	File    string
	Line    int
	Column  int
}

// CallEdge represents a call relationship between functions
type CallEdge struct {
	FromID string
	ToID   string
	Type   string
}

// ExtractCallGraphData extracts nodes and edges from the call graph result
func ExtractCallGraphData(result *CallGraphResult) ([]FunctionNode, []CallEdge) {
	var nodes []FunctionNode
	var edges []CallEdge

	for _, node := range result.CallGraph.Nodes {
		if node.Func == nil {
			continue
		}

		// Extract node data
		packageName := "unknown-package"
		fileName := "unknown-file"
		sourceLine := 0
		sourceColumn := 0
		if node.Func.Pkg != nil {
			packageName = node.Func.Pkg.Pkg.Path()
		}
		if node.Func.Pos() != token.NoPos {
			pos := node.Func.Pos()
			position := result.FileSet.Position(pos)
			fileName = position.Filename
			sourceLine = position.Line
			sourceColumn = position.Column
		}

		nodes = append(nodes, FunctionNode{
			ID:      node.Func.String(),
			Name:    node.Func.Name(),
			Package: packageName,
			Label:   "Function",
			File:    fileName,
			Line:    sourceLine,
			Column:  sourceColumn,
		})

		// Extract edge data
		for _, edge := range node.Out {
			if edge.Callee.Func == nil {
				continue
			}
			edges = append(edges, CallEdge{
				FromID: edge.Caller.Func.String(),
				ToID:   edge.Callee.Func.String(),
				Type:   "CALLS",
			})
		}
	}

	return nodes, edges
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
			node.Label,
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
			edge.Type,
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

// ExportCallGraph exports the call graph to CSV files
func ExportCallGraph(result *CallGraphResult) error {
	nodes, edges := ExtractCallGraphData(result)
	return ExportCallGraphToCSV(nodes, edges, result.OutputPath)
}
