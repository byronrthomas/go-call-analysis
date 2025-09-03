package analyzer

import (
	"fmt"
	"go/token"
	"go/types"
	"log"

	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
	"golang.org/x/tools/go/ssa"
)

type SSAGraphData struct {
	ValueNodes        []ValueNode
	InstructionNodes  []InstructionNode
	OrderingEdges     []OrderingEdge
	OperandEdges      []OperandEdge
	ControlFlowEdges  []ControlFlowEdge
	ResultEdges       []ResultEdge
	ResolvedCallEdges []ResolvedCallEdge
}

type ValueNode struct {
	graphcommon.NodeCommon
	ValueType   string
	TypeName    string
	IsErrorType bool
}

type InstructionNode struct {
	graphcommon.NodeCommon
	InstructionType string
}

type OrderingEdge struct {
	graphcommon.EdgeCommon
}

type ControlFlowEdge struct {
	graphcommon.EdgeCommon
	Condition string
}

type OperandEdge struct {
	graphcommon.EdgeCommon
}

type ResolvedCallEdge struct {
	graphcommon.EdgeCommon
	EdgeCardinality int
}

type ResultEdge struct {
	graphcommon.EdgeCommon
}

func (n *ValueNode) ToMap() map[string]any {
	nodeCommonMap := graphcommon.NodeCommonAsMap(n.NodeCommon)
	nodeCommonMap["label"] = "Value"
	nodeCommonMap["value_type"] = n.ValueType
	nodeCommonMap["type_name"] = n.TypeName
	nodeCommonMap["is_error_type"] = n.IsErrorType
	return nodeCommonMap
}

func (n *InstructionNode) ToMap() map[string]any {
	nodeCommonMap := graphcommon.NodeCommonAsMap(n.NodeCommon)
	nodeCommonMap["label"] = "Instruction"
	nodeCommonMap["instruction_type"] = n.InstructionType
	return nodeCommonMap
}

func (e *OrderingEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "And_Then"
	return edgeCommonMap
}

func (e *ControlFlowEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Control_Flow"
	edgeCommonMap["condition"] = e.Condition
	return edgeCommonMap
}

func (e *OperandEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Uses_Operand"
	return edgeCommonMap
}

func (e *ResultEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Produces_Result"
	return edgeCommonMap
}

func (e *ResolvedCallEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Resolved_Call"
	edgeCommonMap["edge_cardinality"] = e.EdgeCardinality
	return edgeCommonMap
}

type GraphVisitor struct {
	BaseSSAVisitor
	fileSet           *token.FileSet
	instructionNodes  []InstructionNode
	orderingEdges     []OrderingEdge
	controlFlowEdges  []ControlFlowEdge
	operandEdges      []OperandEdge
	resultEdges       []ResultEdge
	resolvedCallEdges []ResolvedCallEdge
	valueNodes        []ValueNode
	functionEntries   map[string]bool
}

func (v *GraphVisitor) VisitFunction(f *ssa.Function, pkg *ssa.Package) {

	pos := v.fileSet.Position(f.Pos())
	// TODO: restrict to reachable functions from the call graph
	funcId := f.String()
	if _, ok := v.functionEntries[funcId]; !ok {
		addFunctionEntryNode(v, funcId, f, pkg, pos)
	}
	v.functionEntries[funcId] = true
	// Put an end-then node from the function entry to the first instruction

	for blockInd, b := range f.Blocks {
		var precInstrId string
		if blockInd == 0 {
			precInstrId = funcId
		}
		v.controlFlowEdges = addControlFlowEdges(b, v.controlFlowEdges)
		currentBlockId := blockId(b)

		for instrInd, instr := range b.Instrs {
			instrId := ContextualId(b, instrInd)

			instrPosition := v.fileSet.Position(instr.Pos())
			v.instructionNodes = append(v.instructionNodes, InstructionNode{
				NodeCommon: graphcommon.NodeCommon{
					ID:      instrId,
					Name:    instr.String(),
					Package: pkg.Pkg.Path(),
					PositionInfo: graphcommon.PositionInfo{
						File:   instrPosition.Filename,
						Line:   instrPosition.Line,
						Column: instrPosition.Column,
					},
				},
				InstructionType: instrTypeAsString(instr),
			})

			// Add the and-then edge from the previous instruction to the current instruction
			if precInstrId != "" {
				v.orderingEdges = append(v.orderingEdges, OrderingEdge{
					EdgeCommon: graphcommon.EdgeCommon{
						FromID: precInstrId,
						ToID:   instrId,
					},
				})
			}
			precInstrId = instrId

			for _, op := range instr.Operands(make([]*ssa.Value, 0)) {
				if *op == nil {
					continue
				}
				opAsInstr, ok := (*op).(ssa.Instruction)
				producingBlockId := ""
				if ok {
					producingBlockId = blockId(opAsInstr.Block())
				}

				_, opId := ValueId(v.fileSet, *op, producingBlockId)
				v.operandEdges = append(v.operandEdges, OperandEdge{
					EdgeCommon: graphcommon.EdgeCommon{
						FromID: instrId,
						ToID:   opId,
					},
				})
			}

			if asAnnotatedCall, ok := instr.(*AnnotatedCall); ok {
				for _, returnValue := range asAnnotatedCall.ReturnValues {
					_, returnValueId := ValueId(v.fileSet, returnValue, currentBlockId)
					v.valueNodes = processValue(v.valueNodes, returnValueId, returnValue, pkg, instrPosition)
					v.resultEdges = append(v.resultEdges, ResultEdge{
						EdgeCommon: graphcommon.EdgeCommon{
							FromID: instrId,
							ToID:   returnValueId,
						},
					})
				}
				for _, resolvedTarget := range asAnnotatedCall.ResolvedTargets {
					targetId := resolvedTarget.String()
					if _, ok := v.functionEntries[targetId]; !ok {
						addFunctionEntryNode(v, targetId, resolvedTarget, pkg, v.fileSet.Position(resolvedTarget.Pos()))
						v.functionEntries[targetId] = true
					}
					v.resolvedCallEdges = append(v.resolvedCallEdges, ResolvedCallEdge{
						EdgeCommon: graphcommon.EdgeCommon{
							FromID: instrId,
							ToID:   resolvedTarget.String(),
						},
						EdgeCardinality: len(asAnnotatedCall.ResolvedTargets)})
				}
			} else if asValue, ok := instr.(ssa.Value); ok {
				_, vId := ValueId(v.fileSet, asValue, currentBlockId)
				v.valueNodes = processValue(v.valueNodes, vId, asValue, pkg, instrPosition)

				// If instruction produces a value, add a result edge from the instruction to the value
				v.resultEdges = append(v.resultEdges, ResultEdge{
					EdgeCommon: graphcommon.EdgeCommon{
						FromID: instrId,
						ToID:   vId,
					},
				})
			}
		}
	}
}

func addFunctionEntryNode(v *GraphVisitor, funcId string, f *ssa.Function, pkg *ssa.Package, pos token.Position) {
	v.instructionNodes = append(v.instructionNodes, InstructionNode{
		NodeCommon: graphcommon.NodeCommon{
			ID:      funcId,
			Name:    f.Name(),
			Package: pkg.Pkg.Path(),
			PositionInfo: graphcommon.PositionInfo{
				File:   pos.Filename,
				Line:   pos.Line,
				Column: pos.Column,
			},
		},
		InstructionType: "function-entry",
	})
}

func (v *GraphVisitor) VisitTypeMethod(_method *types.Func, ssaFunc *ssa.Function, _namedType *types.Named, _pkg *ssa.Package) {
	v.VisitFunction(ssaFunc, _pkg)
}

func (v *GraphVisitor) VisitValue(valueObj ssa.Value, pkg *ssa.Package) {
	valuePosition, vId := ValueId(v.fileSet, valueObj, "")
	v.valueNodes = processValue(v.valueNodes, vId, valueObj, pkg, valuePosition)
}

func ExtractSSAGraphData(ssaProgram *ssa.Program, packagePrefixes []string) SSAGraphData {
	fileSet := ssaProgram.Fset

	visitor := &GraphVisitor{
		BaseSSAVisitor:  BaseSSAVisitor{},
		fileSet:         fileSet,
		functionEntries: make(map[string]bool),
	}
	traverser := NewSSATraverser(packagePrefixes)
	traverser.Traverse(ssaProgram, visitor)

	return SSAGraphData{
		ValueNodes:        visitor.valueNodes,
		InstructionNodes:  visitor.instructionNodes,
		OrderingEdges:     visitor.orderingEdges,
		OperandEdges:      visitor.operandEdges,
		ControlFlowEdges:  visitor.controlFlowEdges,
		ResultEdges:       visitor.resultEdges,
		ResolvedCallEdges: visitor.resolvedCallEdges,
	}
}

func addControlFlowEdges(b *ssa.BasicBlock, controlFlowEdges []ControlFlowEdge) []ControlFlowEdge {
	lastInstrId := ContextualId(b, len(b.Instrs)-1)
	lastInstr := b.Instrs[len(b.Instrs)-1]
	_, lastInstrIsIf := lastInstr.(*ssa.If)
	if !lastInstrIsIf {
		_, lastInstrIsIf = lastInstr.(*AnnotatedIf)
	}
	for succInd, succBlk := range b.Succs {
		succId := ContextualId(succBlk, 0)
		edgeCondition := ""
		if lastInstrIsIf && succInd == 0 {
			edgeCondition = "true"
		} else if lastInstrIsIf && succInd == 1 {
			edgeCondition = "false"
		}
		controlFlowEdges = append(controlFlowEdges, ControlFlowEdge{
			EdgeCommon: graphcommon.EdgeCommon{
				FromID: lastInstrId,
				ToID:   succId,
			},
			Condition: edgeCondition,
		})
	}
	return controlFlowEdges
}

func processValue(valueNodes []ValueNode, vId string, v ssa.Value, pkg *ssa.Package, valuePosition token.Position) []ValueNode {
	valueNodes = append(valueNodes, ValueNode{
		NodeCommon: graphcommon.NodeCommon{
			ID:      vId,
			Name:    v.Name(),
			Package: pkg.Pkg.Path(),
			PositionInfo: graphcommon.PositionInfo{
				File:   valuePosition.Filename,
				Line:   valuePosition.Line,
				Column: valuePosition.Column,
			},
		},
		ValueType:   valueTypeAsString(v),
		TypeName:    v.Type().String(),
		IsErrorType: isErrorType(v.Type()),
	})
	return valueNodes
}

func isErrorType(t types.Type) bool {
	errorType := types.Universe.Lookup("error").Type()
	return types.AssignableTo(t, errorType)
}

func ValueId(fileSet *token.FileSet, valueObj ssa.Value, producingBlockId string) (token.Position, string) {
	valuePos := valueObj.Pos()
	if valuePos == token.NoPos {
		if asConst, ok := valueObj.(*ssa.Const); ok {
			if asConst.Value == nil {
				return token.Position{}, valueObj.String()
			}
			asExactString := asConst.Value.ExactString()
			asString := asConst.Value.String()
			if asExactString == asString {
				return token.Position{}, asExactString
			}
			return token.Position{}, asExactString
		} else if asFRP, ok := valueObj.(*FuncReturnPlaceholder); ok {
			valuePos = asFRP.FromCall.Pos()
			valuePosition := fileSet.Position(valuePos)
			valueId := fmt.Sprintf("return-placeholder-%d-%s:%d:%d", asFRP.Index, valuePosition.Filename, valuePosition.Line, valuePosition.Column)
			return valuePosition, valueId
		} else if asGlobal, ok := valueObj.(*ssa.Global); ok {
			valueId := fmt.Sprintf("%s:%s", asGlobal.Pkg.Pkg.Path(), asGlobal.Name())
			return token.Position{}, valueId
		} else if asFunction, ok := valueObj.(*ssa.Function); ok {
			if asFunction.Synthetic != "" {
				valueId := fmt.Sprintf("synthetic-%s-%s", asFunction.Pkg.Pkg.Path(), asFunction.String())
				return token.Position{}, valueId
			}
			log.Printf("WARN: Function being treated as a value has no synthetic ID: %v", asFunction)
			valueId := fmt.Sprintf("%s:%s", asFunction.Pkg.Pkg.Path(), asFunction.String())
			return token.Position{}, valueId
		}
		// This line should only be hit when we're in a synthetic position that doesn't exist in the source code
		if producingBlockId != "" {
			valueId := fmt.Sprintf("value-%s:%s", producingBlockId, valueObj.Name())
			return token.Position{}, valueId
		} else {
			log.Fatalf("Value has no position and no containing block ID: %v", valueObj)
		}
		return token.Position{}, valueObj.String()
	}
	valuePosition := fileSet.Position(valueObj.Pos())
	valueId := fmt.Sprintf("value-%s:%d:%d", valuePosition.Filename, valuePosition.Line, valuePosition.Column)
	return valuePosition, valueId
}

func ContextualId(block *ssa.BasicBlock, instrIndex int) string {
	return fmt.Sprintf("%s:%d", blockId(block), instrIndex)
}

func blockId(block *ssa.BasicBlock) string {
	return fmt.Sprintf("%s:%d", block.Parent().String(), block.Index)
}

func instrTypeAsString(instr ssa.Instruction) string {
	switch instr.(type) {
	case *ssa.Alloc:
		return "Alloc"
	case *ssa.BinOp:
		return "BinOp"
	case *ssa.Call:
		return "Call"
	case *AnnotatedCall:
		return "AnnotatedCall"
	case *ssa.ChangeInterface:
		return "ChangeInterface"
	case *ssa.ChangeType:
		return "ChangeType"
	case *ssa.Convert:
		return "Convert"
	case *ssa.DebugRef:
		return "DebugRef"
	case *ssa.Defer:
		return "Defer"
	case *ssa.Extract:
		return "Extract"
	case *ssa.Field:
		return "Field"
	case *ssa.FieldAddr:
		return "FieldAddr"
	case *ssa.Go:
		return "Go"
	case *ssa.If:
		return "If"
	case *AnnotatedIf:
		return "AnnotatedIf"
	case *ssa.Index:
		return "Index"
	case *ssa.IndexAddr:
		return "IndexAddr"
	case *ssa.Jump:
		return "Jump"
	case *ssa.Lookup:
		return "Lookup"
	case *ssa.MakeChan:
		return "MakeChan"
	case *ssa.MakeClosure:
		return "MakeClosure"
	case *ssa.MakeInterface:
		return "MakeInterface"
	case *ssa.MakeMap:
		return "MakeMap"
	case *ssa.MakeSlice:
		return "MakeSlice"
	case *ssa.MapUpdate:
		return "MapUpdate"
	case *ssa.MultiConvert:
		return "MultiConvert"
	case *ssa.Next:
		return "Next"
	case *ssa.Panic:
		return "Panic"
	case *ssa.Phi:
		return "Phi"
	case *ssa.Range:
		return "Range"
	case *ssa.Return:
		return "Return"
	case *ssa.RunDefers:
		return "RunDefers"
	case *ssa.Select:
		return "Select"
	case *ssa.Send:
		return "Send"
	case *ssa.Slice:
		return "Slice"
	case *ssa.SliceToArrayPointer:
		return "SliceToArrayPointer"
	case *ssa.Store:
		return "Store"
	case *ssa.TypeAssert:
		return "TypeAssert"
	case *ssa.UnOp:
		return "UnOp"
	}
	return "unknown"
}

func valueTypeAsString(value ssa.Value) string {
	switch value.(type) {
	case *ssa.Alloc:
		return "Alloc"
	case *ssa.BinOp:
		return "BinOp"
	case *ssa.Builtin:
		return "Builtin"
	case *AnnotatedCall:
		return "AnnotatedCall"
	case *ssa.Call:
		return "Call"
	case *ssa.ChangeInterface:
		return "ChangeInterface"
	case *ssa.ChangeType:
		return "ChangeType"
	case *ssa.Const:
		return "Const"
	case *ssa.Convert:
		return "Convert"
	case *ssa.Extract:
		return "Extract"
	case *ssa.Field:
		return "Field"
	case *ssa.FieldAddr:
		return "FieldAddr"
	case *ssa.FreeVar:
		return "FreeVar"
	case *ssa.Function:
		return "Function"
	case *ssa.Global:
		return "Global"
	case *ssa.Index:
		return "Index"
	case *ssa.IndexAddr:
		return "IndexAddr"
	case *ssa.Lookup:
		return "Lookup"
	case *ssa.MakeChan:
		return "MakeChan"
	case *ssa.MakeClosure:
		return "MakeClosure"
	case *ssa.MakeInterface:
		return "MakeInterface"
	case *ssa.MakeMap:
		return "MakeMap"
	case *ssa.MakeSlice:
		return "MakeSlice"
	case *ssa.MultiConvert:
		return "MultiConvert"
	case *ssa.Next:
		return "Next"
	case *ssa.Parameter:
		return "Parameter"
	case *ssa.Phi:
		return "Phi"
	case *ssa.Range:
		return "Range"
	case *ssa.Select:
		return "Select"
	case *ssa.Slice:
		return "Slice"
	case *ssa.SliceToArrayPointer:
		return "SliceToArrayPointer"
	case *ssa.TypeAssert:
		return "TypeAssert"
	case *ssa.UnOp:
		return "UnOp"
	case *FuncReturnPlaceholder:
		return "FuncReturnPlaceholder"
	}
	return "unknown"
}
