package analyzer

import (
	"fmt"
	"go/token"
	"go/types"
	"log"
	"maps"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

func PackageMatcher(packagePrefixes []string) func(pkgPath string) bool {
	return func(pkgPath string) bool {
		for _, prefix := range packagePrefixes {
			if prefix == "" || strings.HasPrefix(pkgPath, prefix) {
				return true
			}
		}
		return false
	}

}

type SSASimplificationVisitor struct {
	BaseSSAVisitor
	CallGraph            *callgraph.Graph
	unreachableFunctions map[string]bool
}

func (v *SSASimplificationVisitor) VisitFunction(f *ssa.Function, pkg *ssa.Package) {
	// We assume synthetic functions are reachable - they were generated for some reason?
	if v.CallGraph.Nodes[f] == nil && f.Synthetic == "" {
		v.unreachableFunctions[f.String()] = true
		return
	}
	tryFunctionSimplification(f, v.CallGraph)
}

func (v *SSASimplificationVisitor) VisitTypeMethod(_method *types.Func, ssaFunc *ssa.Function, _namedType *types.Named, _pkg *ssa.Package) {
	if v.CallGraph.Nodes[ssaFunc] == nil {
		v.unreachableFunctions[ssaFunc.String()] = true
		return
	}
	tryFunctionSimplification(ssaFunc, v.CallGraph)
}

type SSASimplificationResult struct {
	SSAProgram           *ssa.Program
	unreachableFunctions map[string]bool
}

func (r SSASimplificationResult) AllUnreachableFunctions() []string {
	return slices.Collect(maps.Keys(r.unreachableFunctions))
}

func (r SSASimplificationResult) ShouldVisitFunction(f *ssa.Function) bool {
	return !r.unreachableFunctions[f.String()]
}

func SimplifySSA(input *CallGraphResult, packagePrefixes []string) *SSASimplificationResult {
	program := input.SSAProgram
	visitor := &SSASimplificationVisitor{
		BaseSSAVisitor:       BaseSSAVisitor{},
		CallGraph:            input.CallGraph,
		unreachableFunctions: make(map[string]bool),
	}
	traverser := NewSSATraverser(packagePrefixes)
	traverser.Traverse(program, visitor)
	return &SSASimplificationResult{
		SSAProgram:           program,
		unreachableFunctions: visitor.unreachableFunctions,
	}
}

func formatOperator(op token.Token) string {
	switch op {
	case token.EQL:
		return "=="
	case token.NEQ:
		return "!="
	case token.LSS:
		return "<"
	case token.LEQ:
		return "<="
	case token.GTR:
		return ">"
	case token.GEQ:
		return ">="
	}
	return "unknown_logic_operator"
}

func formatCondition(op token.Token, constValue string) string {
	return fmt.Sprintf("%s %s", formatOperator(op), constValue)
}

var int_match = regexp.MustCompile(`^\d+:int$`)

func tryGetConstantDescription(maybeConst *ssa.Value) string {
	valueName := (*maybeConst).Name()
	if strings.HasPrefix(valueName, "nil:") {
		return "nil"
	}

	if int_match.MatchString(valueName) {
		return valueName[0 : len(valueName)-4]
	}
	return ""
}

// Function looks through the if operand value hierarchy to find it if
// it comes from a binary operation between a const and another value.
// If it does, it returns the instruction that performs the operation (which
// will be removed), a string representing the const condition, and the other value.
// If it doesn't, it returns nil.
func tryGetSimpleCondition(ifOperand *ssa.Value) (*ssa.Instruction, string, *ssa.Value) {
	asInstr, ok := (*ifOperand).(ssa.Instruction)
	if !ok {
		return nil, "", nil
	}
	asBinOp, ok := asInstr.(*ssa.BinOp)
	if !ok {
		return nil, "", nil
	}
	operands := asBinOp.Operands(make([]*ssa.Value, 0))
	if len(operands) != 2 {
		return nil, "", nil
	}
	const0 := tryGetConstantDescription(operands[0])

	if const0 != "" {
		return &asInstr, formatCondition(asBinOp.Op, const0), operands[1]
	}
	const1 := tryGetConstantDescription(operands[1])
	if const1 != "" {
		return &asInstr, formatCondition(asBinOp.Op, const1), operands[0]
	}
	return nil, "", nil
}

type AnnotatedIf struct {
	ssa.Instruction
	ConditionDescription string
	OtherValue           ssa.Value
}

func (a *AnnotatedIf) Operands(rands []*ssa.Value) []*ssa.Value {
	rands = append(rands, &a.OtherValue)
	return rands
}

func (a *AnnotatedIf) String() string {
	// Be robust against malformed CFG.
	tblock, fblock := -1, -1
	if a.Instruction.Block() != nil && len(a.Instruction.Block().Succs) == 2 {
		tblock = a.Instruction.Block().Succs[0].Index
		fblock = a.Instruction.Block().Succs[1].Index
	}
	return fmt.Sprintf("if (%s %s) goto %d else %d", a.OtherValue.Name(), a.ConditionDescription, tblock, fblock)
}

type InstructionAndValue interface {
	ssa.Instruction
	ssa.Value
}

type AnnotatedCall struct {
	InstructionAndValue
	ResolvedTargets []*ssa.Function
	ReturnValues    []ssa.Value
	Args            []*ssa.Value
	DynamicCallee   ssa.Value
}

var _ ssa.Instruction = (*AnnotatedCall)(nil)

var _ ssa.Value = (*AnnotatedCall)(nil)

func liftIfCondition(block *ssa.BasicBlock) {
	instr := block.Instrs[len(block.Instrs)-1]
	_, ok := instr.(*ssa.If)
	if !ok {
		return
	}
	operands := instr.Operands(make([]*ssa.Value, 0))
	if len(operands) != 1 {
		return
	}
	instrToRemove, conditionDescription, otherValue := tryGetSimpleCondition(operands[0])
	if instrToRemove == nil {
		return
	}

	annotatedIf := AnnotatedIf{
		Instruction:          instr,
		ConditionDescription: conditionDescription,
		OtherValue:           *otherValue,
	}
	// removeInstruction(block, instrToRemove)
	block.Instrs[len(block.Instrs)-1] = &annotatedIf
}

func removeInstruction(block *ssa.BasicBlock, instrToRemove *ssa.Instruction) {
	foundInstrToRemove := false
	blocksVisited := make(map[*ssa.BasicBlock]bool)
	blocksToVisit := make([]*ssa.BasicBlock, 0)
	blocksToVisit = append(blocksToVisit, block)
	for len(blocksToVisit) > 0 {
		block := blocksToVisit[0]
		blocksToVisit = blocksToVisit[1:]
		blocksVisited[block] = true
		for i, instr := range block.Instrs {
			if instr == *instrToRemove {
				block.Instrs = slices.Delete(block.Instrs, i, i+1)
				foundInstrToRemove = true
				break
			}
		}
		if foundInstrToRemove {
			break
		}
		for _, pred := range block.Preds {
			if !blocksVisited[pred] {
				blocksToVisit = append(blocksToVisit, pred)
			}
		}
	}
	if !foundInstrToRemove {
		log.Fatalf("Failed to find instruction to remove: %v", instrToRemove)
	}
}

type FuncReturnPlaceholder struct {
	FromCall *ssa.Call
	Index    int
	myType   types.Type
}

var _ ssa.Value = (*FuncReturnPlaceholder)(nil)

func (f *FuncReturnPlaceholder) String() string {
	return fmt.Sprintf("func_return_placeholder_%d_%s", f.Index, f.FromCall.Name())
}

func (f *FuncReturnPlaceholder) Name() string {
	return f.String()
}

func (f *FuncReturnPlaceholder) Type() types.Type {
	return f.myType
}

func (f *FuncReturnPlaceholder) Parent() *ssa.Function {
	return f.FromCall.Parent()
}

func (f *FuncReturnPlaceholder) Pos() token.Pos {
	return token.NoPos
}

func (f *FuncReturnPlaceholder) Referrers() *[]ssa.Instruction {
	return nil
}

func liftCallReferrers(annotatedCall *AnnotatedCall, block *ssa.BasicBlock, ogReferrers *[]ssa.Instruction, ogValue ssa.Value) {
	if len(*ogReferrers) == 0 {
		return
	}
	if len(annotatedCall.ReturnValues) == 0 {
		log.Fatalf("Annotated call has no return values but %d referrers", len(*ogReferrers))
	}
	// There are two expected modes for this:
	// 1. There is only a single return value - all referrers should not be extract instructions,
	// and the return value should just be the original call result value (which everything else will
	// then correctly be linked to)
	// 2. There are multiple return values - all referrers should be extract instructions and we can
	// replace the return values with the extract instructions to just give the results of those instructions
	if len(annotatedCall.ReturnValues) == 1 {
		for _, referrer := range *ogReferrers {
			if asExtract, ok := referrer.(*ssa.Extract); ok {
				log.Fatalf("Extract instruction found in referrers of annotated call: %v", asExtract)
			}
		}
		annotatedCall.ReturnValues[0] = ogValue
	} else {
		instrsToRemove := make([]ssa.Instruction, 0)
		for _, referrer := range *ogReferrers {
			if asExtract, ok := referrer.(*ssa.Extract); ok {
				instrsToRemove = append(instrsToRemove, asExtract)
				annotatedCall.ReturnValues[asExtract.Index] = asExtract
			} else {
				log.Fatalf("Non-extract instruction found in referrers of annotated call: %v", referrer)
			}
		}
		for _, instrToRemove := range instrsToRemove {
			removeInstruction(block, &instrToRemove)
		}
	}
}

func findAllTargets(call *ssa.Call, callGraph *callgraph.Graph) []*ssa.Function {
	callSiteNode := callGraph.Nodes[call.Parent()]
	if callSiteNode == nil {
		if call.Parent().Synthetic == "" {
			log.Fatalf("Non-synthetic site node has no function: %v", call.Parent())
		}
		if call.Call.StaticCallee() != nil {
			return []*ssa.Function{call.Call.StaticCallee()}
		}
		// Otherwise, it is some kind of dynamic value, no targets
		return nil
	}
	allTargets := make([]*ssa.Function, 0)
	for _, edge := range callSiteNode.Out {
		if edge.Site != nil && edge.Site.Value() == call {
			allTargets = append(allTargets, edge.Callee.Func)
		}
	}
	return allTargets
}

// annotateCall just processes the type of the call result and
// generates n values that represent the return values of the call
// if the call doesn't return a tuple, n = 1, if it does, n is the length of the tuple
func annotateCall(call *ssa.Call, callGraph *callgraph.Graph) *AnnotatedCall {
	callType := call.Type()
	n := 1
	// NOTE: a void return type is a tuple with 0 elements
	// which means we handle it correctly here - no values being produced
	// if we want to be able to detect it easily, should add to the graph a
	// special value node for void returns and link it up, or add an attrib
	// on the call itself
	allTargets := findAllTargets(call, callGraph)
	dynamicCallee := (ssa.Value)(nil)
	if len(allTargets) == 0 {
		// When we fail to resolve targets, we just assume it is some
		// dynamically-called function value (or builtin)
		// We leave it to the graph to hook this in appropriately
		dynamicCallee = call.Call.Value
	}
	if asTuple, ok := callType.(*types.Tuple); ok {
		n = asTuple.Len()
		if n == 1 {
			log.Fatalf("Single return value call type for tuple?: %v", callType)
		}
		returnValues := make([]ssa.Value, n)
		for i := 0; i < n; i++ {
			retVal := FuncReturnPlaceholder{
				FromCall: call,
				Index:    i,
				myType:   asTuple.At(i).Type(),
			}
			returnValues[i] = &retVal
		}
		annotatedCall := AnnotatedCall{InstructionAndValue: call, ReturnValues: returnValues, ResolvedTargets: allTargets, DynamicCallee: dynamicCallee, Args: makeArgs(call)}
		return &annotatedCall
	}

	return &AnnotatedCall{InstructionAndValue: call, ReturnValues: []ssa.Value{call}, ResolvedTargets: allTargets, DynamicCallee: dynamicCallee, Args: makeArgs(call)}
}

func makeArgs(call *ssa.Call) []*ssa.Value {
	// NOTE: this is based on the description in https://pkg.go.dev/golang.org/x/tools@v0.36.0/go/ssa#Call
	//
	// Here our code attempts to say "keep a receiver value from invoke mode",
	// although if the resolved callee is a method and we're not in invoke mode, then args[0] will be the
	// receiver.

	// Did this because we want to drop function values from the args:
	// * this was done to avoid statically-resolved values like main.check or fmt.Println
	// being included in as an operand in the graph (they are not parameters of the receiving
	// function, and so it makes it hard to match call operands to parameter values)
	// * however we want to keep receiver values on method calls in the args (because these
	// are parameters of the receiving function)
	//
	// NOTE - when the value is a dynamic function value, we'll just have to figure out another way to include them.
	// If we were to include true dynamic callees, we'd need to put them in the graph in a
	// more sensible place.
	//
	args := make([]*ssa.Value, 0)
	if call.Common().Method != nil {
		args = append(args, &call.Common().Value)
	}

	for _, arg := range call.Common().Args {
		args = append(args, &arg)
	}
	return args
}

func tryFunctionSimplification(f *ssa.Function, callGraph *callgraph.Graph) {
	for _, block := range f.Blocks {
		liftIfCondition(block)
		for instrInd, instr := range block.Instrs {
			if asCall, ok := instr.(*ssa.Call); ok {
				annotatedCall := annotateCall(asCall, callGraph)
				block.Instrs[instrInd] = annotatedCall
				// Also need to lift referrers of the original call result
				liftCallReferrers(annotatedCall, block, asCall.Referrers(), asCall)
			}
		}
	}
}
