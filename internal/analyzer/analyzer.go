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

// Analyze performs the analysis on the target project
func (a *Analyzer) Analyze() (*callgraph.Graph, error) {
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
	mainPkg := ssaPkgs[0]
	var functions []*ssa.Function
	for _, fn := range mainPkg.Members {
		if f, ok := fn.(*ssa.Function); ok {
			functions = append(functions, f)
		}
	}
	rtaRes := rta.Analyze(functions, true)
	callGraph := rtaRes.CallGraph

	// Print out some call graph edges
	callGraph.DeleteSyntheticNodes() // remove built-in or synthetic calls
	return callGraph, nil
}

// ExportCallGraph exports the call graph to CSV files
func ExportCallGraph(a *Analyzer, callGraph *callgraph.Graph) error {
	// Prepare nodes and edges data
	var nodes [][]string
	var edges [][]string

	// Add header rows
	nodes = append(nodes, []string{"id", "name", "package", "label"})
	edges = append(edges, []string{"id_from", "id_to", "type"})

	// Process nodes and edges
	for _, node := range callGraph.Nodes {
		if node.Func == nil {
			continue
		}
		// Add node
		packageName := "unknown-package"
		if node.Func.Pkg != nil {
			packageName = node.Func.Pkg.Pkg.Name()
		}
		nodes = append(nodes, []string{
			node.Func.String(),
			node.Func.Name(),
			packageName,
			"function",
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
	if a.outputPath == "" {
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
		if err := os.MkdirAll(a.outputPath, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}

		// Write nodes to file
		nodesFile, err := os.Create(filepath.Join(a.outputPath, "nodes.csv"))
		if err != nil {
			return fmt.Errorf("failed to create nodes file: %v", err)
		}
		defer nodesFile.Close()

		writer := csv.NewWriter(nodesFile)
		if err := writer.WriteAll(nodes); err != nil {
			return fmt.Errorf("failed to write nodes: %v", err)
		}
		writer.Flush()

		// Write edges to file
		edgesFile, err := os.Create(filepath.Join(a.outputPath, "edges.csv"))
		if err != nil {
			return fmt.Errorf("failed to create edges file: %v", err)
		}
		defer edgesFile.Close()

		writer = csv.NewWriter(edgesFile)
		if err := writer.WriteAll(edges); err != nil {
			return fmt.Errorf("failed to write edges: %v", err)
		}
		writer.Flush()
	}

	return nil
}
