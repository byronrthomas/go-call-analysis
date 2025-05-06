package analyzer

import (
	"fmt"
	"go/token"
	"log"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
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
		Mode:  packages.LoadAllSyntax,
		Dir:   ".", // current directory
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

	// Create SSA program
	prog := ssa.NewProgram(cfg.Fset, ssa.SanityCheckFunctions)
	var ssaPkgs []*ssa.Package
	for _, pkg := range pkgs {
		if pkg.Types != nil && pkg.Syntax != nil {
			ssaPkg := prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
			ssaPkgs = append(ssaPkgs, ssaPkg)
		}
	}
	prog.Build()

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
