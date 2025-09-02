package analyzer

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
	"golang.org/x/tools/go/ssa"
)

type SSAGraphData struct {
	ValueNodes       []ValueNode
	InstructionNodes []InstructionNode
	OrderingEdges    []OrderingEdge
	OperandEdges     []OperandEdge
	ControlFlowEdges []ControlFlowEdge
	ResultEdges      []ResultEdge
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

func ExtractSSAGraphData(ssaProgram *ssa.Program, packagePrefixes []string) SSAGraphData {
	var valueNodes []ValueNode
	var instructionNodes []InstructionNode
	var orderingEdges []OrderingEdge
	var controlFlowEdges []ControlFlowEdge
	var operandEdges []OperandEdge
	var resultEdges []ResultEdge
	fileSet := ssaProgram.Fset

	// Helper function to check if a package path matches any of the prefixes
	matchesPrefix := PackageMatcher(packagePrefixes)

	for _, pkg := range ssaProgram.AllPackages() {
		// Check if the package path matches any of the provided prefixes
		if matchesPrefix(pkg.Pkg.Path()) {
			for _, mem := range pkg.Members {
				if f, ok := mem.(*ssa.Function); ok {
					pos := fileSet.Position(f.Pos())
					// TODO: restrict to reachable functions from the call graph
					funcId := f.String()
					instructionNodes = append(instructionNodes, InstructionNode{
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
					// Put an end-then node from the function entry to the first instruction

					for blockInd, b := range f.Blocks {
						var precInstrId string
						if blockInd == 0 {
							precInstrId = funcId
						}
						controlFlowEdges = addControlFlowEdges(b, controlFlowEdges)

						for instrInd, instr := range b.Instrs {
							instrId := ContextualId(b, instrInd)

							instrPosition := fileSet.Position(instr.Pos())
							instructionNodes = append(instructionNodes, InstructionNode{
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
								orderingEdges = append(orderingEdges, OrderingEdge{
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
								_, opId := ValueId(fileSet, *op)
								operandEdges = append(operandEdges, OperandEdge{
									EdgeCommon: graphcommon.EdgeCommon{
										FromID: instrId,
										ToID:   opId,
									},
								})
							}

							if asAnnotatedCall, ok := instr.(*AnnotatedCall); ok {
								for _, returnValue := range asAnnotatedCall.ReturnValues {
									_, returnValueId := ValueId(fileSet, returnValue)
									valueNodes = processValue(valueNodes, returnValueId, returnValue, pkg, instrPosition)
									resultEdges = append(resultEdges, ResultEdge{
										EdgeCommon: graphcommon.EdgeCommon{
											FromID: instrId,
											ToID:   returnValueId,
										},
									})
								}
							} else if asValue, ok := instr.(ssa.Value); ok {
								_, vId := ValueId(fileSet, asValue)
								valueNodes = processValue(valueNodes, vId, asValue, pkg, instrPosition)

								// If instruction produces a value, add a result edge from the instruction to the value
								resultEdges = append(resultEdges, ResultEdge{
									EdgeCommon: graphcommon.EdgeCommon{
										FromID: instrId,
										ToID:   vId,
									},
								})
							}
						}
					}
				} else if v, ok := mem.(ssa.Value); ok {
					valuePosition, vId := ValueId(fileSet, v)
					valueNodes = processValue(valueNodes, vId, v, pkg, valuePosition)
				}
			}
		}
	}

	return SSAGraphData{
		ValueNodes:       valueNodes,
		InstructionNodes: instructionNodes,
		OrderingEdges:    orderingEdges,
		OperandEdges:     operandEdges,
		ControlFlowEdges: controlFlowEdges,
		ResultEdges:      resultEdges,
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

func ValueId(fileSet *token.FileSet, instr ssa.Value) (token.Position, string) {
	instrPos := instr.Pos()
	if instrPos == token.NoPos {
		if asConst, ok := instr.(*ssa.Const); ok {
			if asConst.Value == nil {
				return token.Position{}, instr.String()
			}
			asExactString := asConst.Value.ExactString()
			asString := asConst.Value.String()
			if asExactString == asString {
				return token.Position{}, asExactString
			}
			return token.Position{}, asExactString
		} else if asFRP, ok := instr.(*FuncReturnPlaceholder); ok {
			instrPos = asFRP.FromCall.Pos()
			instrPosition := fileSet.Position(instrPos)
			instrId := fmt.Sprintf("return-placeholder-%d-%s:%d:%d", asFRP.Index, instrPosition.Filename, instrPosition.Line, instrPosition.Column)
			return instrPosition, instrId
		}
		return token.Position{}, instr.String()
	}
	instrPosition := fileSet.Position(instr.Pos())
	instrId := fmt.Sprintf("value-%s:%d:%d", instrPosition.Filename, instrPosition.Line, instrPosition.Column)
	return instrPosition, instrId
}

func ContextualId(block *ssa.BasicBlock, instrIndex int) string {
	funcId := block.Parent().String()
	return fmt.Sprintf("%s:%d:%d", funcId, block.Index, instrIndex)
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
	case *AnnotatedCall:
		return "AnnotatedCall"
	case *ssa.Call:
		return "Call"
	case *ssa.ChangeInterface:
		return "ChangeInterface"
	case *ssa.ChangeType:
		return "ChangeType"
	case *ssa.Convert:
		return "Convert"
	case *ssa.Extract:
		return "Extract"
	case *ssa.Field:
		return "Field"
	case *ssa.FieldAddr:
		return "FieldAddr"
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
