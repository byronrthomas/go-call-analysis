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
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

// buildSSAPackage compiles a Go source string into a built *ssa.Package.
// The source must declare "package p". Use ssaPkg.Func(name) to retrieve
// individual *ssa.Function values for passing to visitor methods.
func buildSSAPackage(t *testing.T, src string) *ssa.Package {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "fixture.go", src, 0)
	if err != nil {
		t.Fatalf("buildSSAPackage: parse error: %v", err)
	}
	pkg := types.NewPackage("p", "p")
	ssaPkg, _, err := ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset, pkg, []*ast.File{f}, ssa.InstantiateGenerics,
	)
	if err != nil {
		t.Fatalf("buildSSAPackage: build error: %v", err)
	}
	ssaPkg.Build()
	return ssaPkg
}

func TestVisitFunction_BasicFunctions(t *testing.T) {
	ssaPkg := buildSSAPackage(t, `
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

	visitor := analyzer.NewGraphVisitor(ssaPkg.Prog, func(string) bool { return true }, os.TempDir())
	visitor.VisitFunction(ssaPkg.Func("f1"), ssaPkg)
	visitor.VisitFunction(ssaPkg.Func("f2"), ssaPkg)

	if len(visitor.SSAGraphData.FunctionNodes) == 0 {
		t.Fatal("expected FunctionNodes to be populated after VisitFunction")
	}
}
