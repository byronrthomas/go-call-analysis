package test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"testing"

	"github.com/throwin5tone7/go-call-analysis/internal/analyzer"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// buildSSAFixture compiles a Go source string, runs RTA call graph analysis
// with all package-level functions as roots, then runs SSA simplification.
// The returned SSASimplificationResult contains AnnotatedCall / AnnotatedIf
// nodes and is ready for use with analyzer.NewGraphVisitor. The returned
// *ssa.Package can be used to look up individual functions by name.
//
// The source must declare "package p".
func buildSSAFixture(t *testing.T, src string) (*analyzer.SSASimplificationResult, *ssa.Package) {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatalf("buildSSAFixture: parse error: %v", err)
	}
	pkg := types.NewPackage("p", "p")
	ssaPkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset, pkg, []*ast.File{f}, ssa.InstantiateGenerics,
	)
	if err != nil {
		t.Fatalf("buildSSAFixture: build error: %v", err)
	}
	ssaPkg.Build()

	// Use all package-level functions as RTA roots so every function is
	// considered reachable and gets its calls annotated by SimplifySSA.
	var roots []*ssa.Function
	for _, member := range ssaPkg.Members {
		if fn, ok := member.(*ssa.Function); ok {
			roots = append(roots, fn)
		}
	}
	rtaResult := rta.Analyze(roots, true)

	result := analyzer.SimplifySSA(
		&analyzer.CallGraphResult{CallGraph: rtaResult.CallGraph, SSAProgram: ssaPkg.Prog},
		[]string{"p"},
	)
	return result, ssaPkg
}

func TestVisitFunction_BasicFunctions(t *testing.T) {
	result, ssaPkg := buildSSAFixture(t, `
package p
func add(a, b int) int { return a + b }
func f1(a int) int { return add(a, a) }
func f2() int { return add(f1(1), 1) }
`)

	if ssaPkg.Func("f1") == nil {
		t.Fatal("expected f1 to be present in the SSA package")
	}
	if ssaPkg.Func("f2") == nil {
		t.Fatal("expected f2 to be present in the SSA package")
	}

	visitor := analyzer.NewGraphVisitor(result, func(string) bool { return true }, os.TempDir())
	visitor.VisitFunction(ssaPkg.Func("f1"), ssaPkg)
	visitor.VisitFunction(ssaPkg.Func("f2"), ssaPkg)

	if len(visitor.SSAGraphData.FunctionNodes) == 0 {
		t.Fatal("expected FunctionNodes to be populated after VisitFunction")
	}
}
