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

type FunctionId struct {
	Package  string
	Function string
}

type CallGraphResult struct {
	CallGraph  *callgraph.Graph
	FileSet    *token.FileSet
	OutputPath string
}

// Analyze performs the analysis on the target project
func Analyze(config *AnalysisConfig) (*CallGraphResult, error) {
	// TODO: Implement analysis logic
	// Load the packages (set your target package here)
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedDeps | packages.NeedSyntax | packages.NeedTypesInfo,
		Dir:   config.ProjectPath,
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

	// Perform RTA (Rapid Type Analysis) to build call graph
	var functions []*ssa.Function
	if config.RootFunction != nil {
		for _, pkg := range ssaPkgs {
			fmt.Printf("Checking package %s\n", pkg.Pkg.Path())
			if pkg.Pkg.Path() == config.RootFunction.Package {
				fmt.Printf("Found package %s\n", pkg.Pkg.Path())
				for _, fn := range pkg.Members {
					if f, ok := fn.(*ssa.Function); ok {
						if f.Name() == config.RootFunction.Function {
							functions = append(functions, f)
						}
					}
				}
			}
		}
	} else {
		for _, pkg := range ssaPkgs {
			for _, fn := range pkg.Members {
				if f, ok := fn.(*ssa.Function); ok {
					functions = append(functions, f)
				}
			}
		}
	}
	fmt.Printf("Found %d functions\n", len(functions))
	if len(functions) == 0 {
		return nil, fmt.Errorf("no functions found for root function %s in package %s", config.RootFunction.Function, config.RootFunction.Package)
	}
	rtaRes := rta.Analyze(functions, true)
	callGraph := rtaRes.CallGraph

	// Print out some call graph edges
	callGraph.DeleteSyntheticNodes() // remove built-in or synthetic calls
	return &CallGraphResult{
		CallGraph:  callGraph,
		FileSet:    prog.Fset,
		OutputPath: config.OutputPath,
	}, nil
}

type PositionInfo struct {
	File   string
	Line   int
	Column int
}

// FunctionNode represents a function in the call graph
type FunctionNode struct {
	PositionInfo
	ID      string
	Name    string
	Package string
}

func (n *FunctionNode) ToMap() map[string]any {
	return map[string]any{
		"id":      n.ID,
		"name":    n.Name,
		"package": n.Package,
		"file":    n.File,
		"line":    n.Line,
		"column":  n.Column,
		"label":   "Function",
	}
}

// CallEdge represents a call relationship between functions
type CallEdge struct {
	FromID   string
	ToID     string
	EdgeText string
	CallSite PositionInfo
}

func (e *CallEdge) ToMap() map[string]any {
	return map[string]any{
		"from_id":          e.FromID,
		"to_id":            e.ToID,
		"type":             "CALLS",
		"call_site_file":   e.CallSite.File,
		"call_site_line":   e.CallSite.Line,
		"call_site_column": e.CallSite.Column,
		"call_site_text":   e.EdgeText,
	}
}

type Mappable interface {
	ToMap() map[string]any
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
			PositionInfo: PositionInfo{
				File:   fileName,
				Line:   sourceLine,
				Column: sourceColumn,
			},
		})

		// Extract edge data
		for _, edge := range node.Out {
			if edge.Callee.Func == nil {
				continue
			}
			edgePos := edge.Pos()
			edgeText := ""
			if edge.Site != nil {
				edgeText = edge.Site.String()
				edgePos = edge.Site.Pos()
			}
			// if edge.Site != nil && edge.Site.Value() != nil {
			// 	edgePos = edge.Site.Value().Call.Pos()
			// 	edgeText = edge.Site.Value().Call.String()
			// }
			pos := result.FileSet.Position(edgePos)
			edges = append(edges, CallEdge{
				FromID:   edge.Caller.Func.String(),
				ToID:     edge.Callee.Func.String(),
				EdgeText: edgeText,
				CallSite: PositionInfo{
					File:   pos.Filename,
					Line:   pos.Line,
					Column: pos.Column,
				},
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
