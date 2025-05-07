package analyzer

import (
	"fmt"
	"go/token"
	"log"

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

// FunctionNode represents a function in the call graph
type FunctionNode struct {
	ID      string
	Name    string
	Package string
	File    string
	Line    int
	Column  int
}

// CallEdge represents a call relationship between functions
type CallEdge struct {
	FromID string
	ToID   string
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
			})
		}
	}

	return nodes, edges
}

// ExportCallGraph exports the call graph to CSV files
func ExportCallGraph(result *CallGraphResult) error {
	nodes, edges := ExtractCallGraphData(result)
	return ExportCallGraphToCSV(nodes, edges, result.OutputPath)
}
