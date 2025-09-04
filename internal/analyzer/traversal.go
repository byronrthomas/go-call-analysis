package analyzer

import (
	"go/types"
	"log"

	"golang.org/x/tools/go/ssa"
)

// SSAVisitor defines the interface for visiting SSA elements during traversal
type SSAVisitor interface {
	// VisitFunction is called for each ssa.Function found in matching packages
	VisitFunction(f *ssa.Function, pkg *ssa.Package)

	// VisitValue is called for each ssa.Value found in matching packages
	// (only called for package-level values, not function-local values)
	VisitValue(v ssa.Value, pkg *ssa.Package)

	// VisitType is called for each ssa.Type found in matching packages
	VisitType(t *ssa.Type, pkg *ssa.Package)

	// VisitTypeMethod is called for each method found on named types
	// This is called after VisitType for types that have methods
	VisitTypeMethod(method *types.Func, ssaFunc *ssa.Function, namedType *types.Named, pkg *ssa.Package)
}

// SSATraverser handles the traversal of SSA packages with prefix filtering
type SSATraverser struct {
	packagePrefixes []string
	matchesPrefix   func(string) bool
}

// NewSSATraverser creates a new traverser with the given package prefixes
func NewSSATraverser(packagePrefixes []string) *SSATraverser {
	return &SSATraverser{
		packagePrefixes: packagePrefixes,
		matchesPrefix:   PackageMatcher(packagePrefixes),
	}
}

// Traverse walks through the SSA program, visiting elements that match the package prefixes
func (t *SSATraverser) Traverse(ssaProgram *ssa.Program, visitor SSAVisitor) {
	for _, pkg := range ssaProgram.AllPackages() {
		// Check if the package path matches any of the provided prefixes
		if t.matchesPrefix(pkg.Pkg.Path()) {
			t.traversePackage(ssaProgram, pkg, visitor)
		}
	}
}

// traversePackage handles the traversal of a single package
func (t *SSATraverser) traversePackage(ssaProgram *ssa.Program, pkg *ssa.Package, visitor SSAVisitor) {
	for _, mem := range pkg.Members {
		switch m := mem.(type) {
		case *ssa.Function:
			visitor.VisitFunction(m, pkg)

		case ssa.Value:
			visitor.VisitValue(m, pkg)

		case *ssa.Type:
			visitor.VisitType(m, pkg)

			// Handle methods on named types
			if namedType, ok := m.Type().(*types.Named); ok {
				t.traverseTypeMethods(ssaProgram, namedType, pkg, visitor)
			}
		case *ssa.NamedConst:
			visitor.VisitValue(m.Value, pkg)
		default:
			log.Fatalf("WARN: Unexpected member of package %s: %T", pkg.Pkg.Path(), m)
		}
	}
}

// traverseTypeMethods handles the traversal of methods on a named type
func (t *SSATraverser) traverseTypeMethods(ssaProgram *ssa.Program, namedType *types.Named, pkg *ssa.Package, visitor SSAVisitor) {
	for i := 0; i < namedType.NumMethods(); i++ {
		method := namedType.Method(i)
		ssaFunc := ssaProgram.FuncValue(method)
		visitor.VisitTypeMethod(method, ssaFunc, namedType, pkg)
	}
}

// BaseSSAVisitor provides a default implementation of SSAVisitor
// Embed this in your visitor implementations to only override the methods you need
type BaseSSAVisitor struct{}

func (b *BaseSSAVisitor) VisitFunction(f *ssa.Function, pkg *ssa.Package) {
	// Default: do nothing
}

func (b *BaseSSAVisitor) VisitValue(v ssa.Value, pkg *ssa.Package) {
	// Default: do nothing
}

func (b *BaseSSAVisitor) VisitType(t *ssa.Type, pkg *ssa.Package) {
	// Default: do nothing
}

func (b *BaseSSAVisitor) VisitTypeMethod(method *types.Func, ssaFunc *ssa.Function, namedType *types.Named, pkg *ssa.Package) {
	// Default: do nothing
}
