package analyzer

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/ssa"
)

// generateSSAText generates a textual representation of the SSA program
// by processing each package that matches the package prefixes
func GenerateSSAText(prog *ssa.Program, packagePrefixes []string) string {
	var result strings.Builder

	// Helper function to check if a package path matches any of the prefixes
	matchesPrefix := func(pkgPath string) bool {
		for _, prefix := range packagePrefixes {
			if prefix == "" || strings.HasPrefix(pkgPath, prefix) {
				return true
			}
		}
		return false
	}

	// Process each package
	for _, pkg := range prog.AllPackages() {
		// Check if the package path matches any of the provided prefixes
		if matchesPrefix(pkg.Pkg.Path()) {
			result.WriteString(fmt.Sprintf("Package: %s\n", pkg.Pkg.Path()))
			result.WriteString(strings.Repeat("-", 50) + "\n")

			// Collect members to sort them for deterministic output
			var functions []*ssa.Function
			var values []ssa.Value
			var namedTypes []*ssa.Type

			for _, mem := range pkg.Members {
				if f, ok := mem.(*ssa.Function); ok {
					functions = append(functions, f)
				} else if v, ok := mem.(ssa.Value); ok {
					values = append(values, v)
				} else if t, ok := mem.(*ssa.Type); ok {
					if _, ok := t.Type().(*types.Named); ok {
						namedTypes = append(namedTypes, t)
					}
				}
			}

			// Sort functions by name for deterministic output
			sort.Slice(functions, func(i, j int) bool {
				return functions[i].Name() < functions[j].Name()
			})

			// Sort values by name for deterministic output
			sort.Slice(values, func(i, j int) bool {
				return values[i].Name() < values[j].Name()
			})

			// Sort types by name for deterministic output
			sort.Slice(namedTypes, func(i, j int) bool {
				return namedTypes[i].Name() < namedTypes[j].Name()
			})

			// Process sorted functions
			for _, f := range functions {
				result.WriteString(fmt.Sprintf("Function: %s\n", f.Name()))

				// Process each basic block
				textualizeFunction(f, &result)
			}

			// Process sorted values
			for _, v := range values {
				outputValue(&result, "Value: ", "\n  Type: ", v)
				for _, referrer := range extractReferrerStrings(v.Referrers()) {
					result.WriteString("  <- Referenced by: ")
					result.WriteString(referrer)
					result.WriteString("\n")
				}
			}
			result.WriteString("\n")

			// Process sorted types
			for _, t := range namedTypes {
				methods := make([]*types.Func, 0)
				for meth := range t.Type().(*types.Named).Methods() {
					methods = append(methods, meth)
				}
				sort.Slice(methods, func(i, j int) bool {
					return methods[i].Name() < methods[j].Name()
				})
				for _, meth := range methods {
					result.WriteString(fmt.Sprintf("Method: %s.%s\n", t.Name(), meth.Name()))
					textualizeFunction(prog.FuncValue(meth), &result)
				}
			}

			result.WriteString("\n")
		}
	}

	return result.String()
}

func textualizeFunction(f *ssa.Function, result *strings.Builder) {
	for blockIndex, block := range f.Blocks {
		printBlockInfo(result, blockIndex, block)

		// Process each instruction in the block
		for instrIndex, instr := range block.Instrs {
			result.WriteString(fmt.Sprintf("    %d: %s\n", instrIndex, instr.String()))

			if asAnnotatedCall, ok := instr.(*AnnotatedCall); ok {
				for retInd, returnValue := range asAnnotatedCall.ReturnValues {
					outputValue(result, fmt.Sprintf("      Return %d: ", retInd), "  Type: ", returnValue)
					for _, referrer := range extractReferrerStrings(returnValue.Referrers()) {
						result.WriteString("        <- Referenced by: ")
						result.WriteString(referrer)
						result.WriteString("\n")
					}
				}
			} else if val, ok := instr.(ssa.Value); ok {
				if val.Name() != "" {
					outputValue(result, "      As value: ", "  Type: ", val)
					for _, referrer := range extractReferrerStrings(val.Referrers()) {
						result.WriteString("        <- Referenced by: ")
						result.WriteString(referrer)
						result.WriteString("\n")
					}
				}
			}

			// Show operands
			for i, op := range instr.Operands(make([]*ssa.Value, 0)) {
				if *op == nil {
					continue
				}
				outputValue(result, fmt.Sprintf("      Operand %d: ", i), "  Type: ", *op)
			}
		}

		result.WriteString("\n")
	}
	result.WriteString("\n")
}

func extractReferrerStrings(referrers *[]ssa.Instruction) []string {
	if referrers == nil {
		return []string{}
	}
	referrerStrings := make([]string, len(*referrers))
	for i, referrer := range *referrers {
		referrerStrings[i] = referrer.String()
	}
	sort.Strings(referrerStrings)
	return referrerStrings
}

func printBlockInfo(result *strings.Builder, blockIndex int, block *ssa.BasicBlock) {
	fmt.Fprintf(result, "  Block %d -", blockIndex)
	// Show predecessors, sorted by index, on one line
	preds := make([]int, len(block.Preds))
	for i, pred := range block.Preds {
		preds[i] = pred.Index
	}
	sort.Ints(preds)
	fmt.Fprintf(result, "   Predecessors [")
	for i, pred := range preds {
		if i != len(preds)-1 {
			fmt.Fprintf(result, "%d, ", pred)
		} else {
			fmt.Fprintf(result, "%d", pred)
		}
	}
	fmt.Fprintf(result, "]")
	// Show successors, sorted by index, on one line
	succs := make([]int, len(block.Succs))
	for i, succ := range block.Succs {
		succs[i] = succ.Index
	}
	sort.Ints(succs)
	fmt.Fprintf(result, "   Successors [")
	for i, succ := range succs {
		if i != len(succs)-1 {
			fmt.Fprintf(result, "%d, ", succ)
		} else {
			fmt.Fprintf(result, "%d", succ)
		}
	}
	fmt.Fprintf(result, "]:\n")
}

func outputValue(result *strings.Builder, valuePrefix string, typePrefix string, v ssa.Value) {
	fmt.Fprint(result, valuePrefix)
	fmt.Fprint(result, v.Name())
	if v.Type() != nil {
		fmt.Fprint(result, typePrefix)
		fmt.Fprint(result, v.Type().String())
	}
	fmt.Fprint(result, "\n")
}
