package analyzer

import (
	"golang.org/x/tools/go/ssa"
)

func SimplifySSA(input *CallGraphResult, packagePrefixes []string) *ssa.Program {
	return input.SSAProgram
}
