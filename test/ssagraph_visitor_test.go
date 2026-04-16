package test

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/throwin5tone7/go-call-analysis/internal/analyzer"
	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
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

const simple3FuncsCode = `
package p
func add(a, b int) int { return a + b }
func f1(a int) int { return add(a, a) }
func f2() int { return add(f1(1), 1) }
`

func TestVisitFunction_FunctionNodes(t *testing.T) {

	result, ssaPkg := buildSSAFixture(t, simple3FuncsCode)

	if ssaPkg.Func("f1") == nil {
		t.Fatal("expected f1 to be present in the SSA package")
	}
	if ssaPkg.Func("f2") == nil {
		t.Fatal("expected f2 to be present in the SSA package")
	}

	visitor := analyzer.NewGraphVisitor(result, func(string) bool { return true }, os.TempDir())
	visitor.VisitFunction(ssaPkg.Func("f1"), ssaPkg)
	visitor.VisitFunction(ssaPkg.Func("f2"), ssaPkg)

	expectedNodeIds := []string{"p.add", "p.f1", "p.f2"}
	actualNodeIds := make([]string, 0)
	for _, node := range visitor.SSAGraphData.FunctionNodes {
		actualNodeIds = append(actualNodeIds, node.ID)
		t.Logf("FunctionNode ID: %s", node.ID)
	}
	slices.Sort(expectedNodeIds)
	slices.Sort(actualNodeIds)
	t.Logf("Expected len %d, Actual len %d", len(expectedNodeIds), len(actualNodeIds))
	if !reflect.DeepEqual(expectedNodeIds, actualNodeIds) {
		t.Fatalf("expected FunctionNodes to have IDs %v, got %v", expectedNodeIds, actualNodeIds)
	}
}

func formatEdgeCommon(edge *graphcommon.EdgeCommon) string {
	return fmt.Sprintf("%s -> %s", edge.FromID, edge.ToID)
}

func TestVisitFunction_CallGraph_Calls_Edges(t *testing.T) {

	result, ssaPkg := buildSSAFixture(t, simple3FuncsCode)

	if ssaPkg.Func("f1") == nil {
		t.Fatal("expected f1 to be present in the SSA package")
	}
	if ssaPkg.Func("f2") == nil {
		t.Fatal("expected f2 to be present in the SSA package")
	}

	visitor := analyzer.NewGraphVisitor(result, func(string) bool { return true }, os.TempDir())
	visitor.VisitFunction(ssaPkg.Func("f1"), ssaPkg)
	visitor.VisitFunction(ssaPkg.Func("f2"), ssaPkg)

	if len(visitor.SSAGraphData.CallGraphCallsEdges) != 3 {
		t.Fatal("expected 3 CallGraphCallsEdges to be populated after VisitFunction")
	}
	expectedEdges := []string{
		formatEdgeCommon(&graphcommon.EdgeCommon{FromID: "p.f1", ToID: "p.add"}),
		formatEdgeCommon(&graphcommon.EdgeCommon{FromID: "p.f2", ToID: "p.add"}),
		formatEdgeCommon(&graphcommon.EdgeCommon{FromID: "p.f2", ToID: "p.f1"}),
	}
	actualEdges := make([]string, len(visitor.SSAGraphData.CallGraphCallsEdges))
	for _, edge := range visitor.SSAGraphData.CallGraphCallsEdges {
		actualEdges = append(actualEdges, formatEdgeCommon(&edge.EdgeCommon))
	}
	slices.Sort(actualEdges)
	if !reflect.DeepEqual(expectedEdges, actualEdges) {
		t.Fatalf("expected CallGraphCallsEdges to be %v, got %v", expectedEdges, actualEdges)
	}

}
