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

// ExportCallGraph exports the call graph to CSV files
func ExportCallGraph(result *CallGraphResult) error {
	// Prepare nodes and edges data
	var nodes [][]string
	var edges [][]string

	// Add header rows
	nodeHeader := []string{"id", "name", "package", "label", "file", "line", "char"}
	edgeHeader := []string{"id_from", "id_to", "type"}

	// Process nodes and edges
	for _, node := range result.CallGraph.Nodes {
		if node.Func == nil {
			continue
		}
		// Add node
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
		nodes = append(nodes, []string{
			node.Func.String(),
			node.Func.Name(),
			packageName,
			"Function",
			fileName,
			fmt.Sprintf("%d", sourceLine),
			fmt.Sprintf("%d", sourceColumn),
		})

		// Add edges
		for _, edge := range node.Out {
			if edge.Callee.Func == nil {
				continue
			}
			edges = append(edges, []string{
				edge.Caller.Func.String(),
				edge.Callee.Func.String(),
				"CALLS",
			})
		}
	}

	// Output to files or stdout
	if result.OutputPath == "" {
		// Output nodes to stdout
		fmt.Println("Nodes:")
		writer := csv.NewWriter(os.Stdout)
		writer.WriteAll(nodes)
		writer.Flush()

		// Output edges to stdout
		fmt.Println("\nEdges:")
		writer = csv.NewWriter(os.Stdout)
		writer.WriteAll(edges)
		writer.Flush()
	} else {
		// Create output directory if it doesn't exist
		if err := os.MkdirAll(result.OutputPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		// Write nodes to file(s)
		nodesPath := filepath.Join(result.OutputPath, "nodes.csv")
		if err := writeCSVToFiles(nodes, nodesPath, nodeHeader); err != nil {
			return fmt.Errorf("failed to write nodes: %v", err)
		}

		// Write edges to file(s)
		edgesPath := filepath.Join(result.OutputPath, "edges.csv")
		if err := writeCSVToFiles(edges, edgesPath, edgeHeader); err != nil {
			return fmt.Errorf("failed to write edges: %v", err)
		}
	}

	return nil
}
