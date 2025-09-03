package analyzer

import (
	"fmt"
	"go/token"
	"log"

	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
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
	SSAProgram *ssa.Program
	OutputPath string
}

// Analyze performs the analysis on the target project
func CallGraphAnalysis(config *AnalysisConfig) (*CallGraphResult, error) {
	// TODO: Implement analysis logic
	// Load the packages (set your target package here)
	prog := BuildSSAProgram(config)
	ssaPkgs := prog.AllPackages()

	// Perform RTA (Rapid Type Analysis) to build call graph
	var functions []*ssa.Function
	if config.RootFunction != nil {
		for _, pkg := range ssaPkgs {
			if pkg.Pkg.Path() == config.RootFunction.Package {
				for _, fn := range pkg.Members {
					if f, ok := fn.(*ssa.Function); ok {
						if f.Name() == config.RootFunction.Function {
							fmt.Printf("Found root function %s in package %s\n", f.Name(), pkg.Pkg.Path())
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
	//callGraph.DeleteSyntheticNodes() // remove built-in or synthetic calls
	return &CallGraphResult{
		CallGraph:  callGraph,
		SSAProgram: prog,
		OutputPath: config.OutputPath,
	}, nil
}

func DumpPackages(ssaPkgs []*ssa.Package) {
	fmt.Printf("\nFound %d packages\n", len(ssaPkgs))
	for _, pkg := range ssaPkgs {
		fmt.Printf("Package %s\n", pkg.Pkg.Path())
		for _, mem := range pkg.Members {
			fmt.Printf("\tMember %T\n", mem)
		}
		for _, nm := range pkg.Pkg.Scope().Names() {
			fmt.Printf("\tIn Scope Name %s\n", nm)
		}
	}
	fmt.Printf("\n")
}

func BuildSSAProgram(config *AnalysisConfig) *ssa.Program {
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
	prog, _ := ssautil.AllPackages(pkgs, ssa.InstantiateGenerics)

	// Build SSA code for the whole program.
	prog.Build()
	return prog
}

// FunctionNode represents a function in the call graph
type FunctionNode struct {
	graphcommon.NodeCommon
}

func (n *FunctionNode) ToMap() map[string]any {
	nodeCommonMap := graphcommon.NodeCommonAsMap(n.NodeCommon)
	nodeCommonMap["label"] = "Function"
	return nodeCommonMap
}

// CallEdge represents a call relationship between functions
type CallEdge struct {
	graphcommon.EdgeCommon
	EdgeText string
	CallSite graphcommon.PositionInfo
}

func (e *CallEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "CALLS"
	edgeCommonMap["call_site_file"] = e.CallSite.File
	edgeCommonMap["call_site_line"] = e.CallSite.Line
	edgeCommonMap["call_site_column"] = e.CallSite.Column
	edgeCommonMap["call_site_text"] = e.EdgeText
	return edgeCommonMap
}

// ExtractCallGraphData extracts nodes and edges from the call graph result
func ExtractCallGraphData(result *CallGraphResult) ([]FunctionNode, []CallEdge) {
	var nodes []FunctionNode
	var edges []CallEdge
	fileSet := result.SSAProgram.Fset

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
			position := fileSet.Position(pos)
			fileName = position.Filename
			sourceLine = position.Line
			sourceColumn = position.Column
		}

		nodes = append(nodes, FunctionNode{
			NodeCommon: graphcommon.NodeCommon{
				ID:      node.Func.String(),
				Name:    node.Func.Name(),
				Package: packageName,
				PositionInfo: graphcommon.PositionInfo{
					File:   fileName,
					Line:   sourceLine,
					Column: sourceColumn,
				},
			}})

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
			pos := fileSet.Position(edgePos)
			edges = append(edges, CallEdge{
				EdgeCommon: graphcommon.EdgeCommon{
					FromID: edge.Caller.Func.String(),
					ToID:   edge.Callee.Func.String(),
				},
				EdgeText: edgeText,
				CallSite: graphcommon.PositionInfo{
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
