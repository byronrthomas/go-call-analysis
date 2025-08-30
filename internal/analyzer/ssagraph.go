package analyzer

import (
	"fmt"
	"go/token"

	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
	"golang.org/x/tools/go/ssa"
)

type SSAGraphData struct {
	ValueNodes       []ValueNode
	InstructionNodes []InstructionNode
	ReferEdges       []ReferEdge
	OrderingEdges    []OrderingEdge
	OperandEdges     []OperandEdge
	SSAOrderingEdges []SSAOrderingEdge
}

type ValueNode struct {
	graphcommon.NodeCommon
	ValueType string
}

type InstructionNode struct {
	graphcommon.NodeCommon
	InstructionType string
}

type ReferEdge struct {
	graphcommon.EdgeCommon
}

type OrderingEdge struct {
	graphcommon.EdgeCommon
}

type SSAOrderingEdge struct {
	graphcommon.EdgeCommon
}

type OperandEdge struct {
	graphcommon.EdgeCommon
}

func ExtractSSAGraphData(result *CallGraphResult) SSAGraphData {
	var valueNodes []ValueNode
	var instructionNodes []InstructionNode
	var referEdges []ReferEdge
	var orderingEdges []OrderingEdge
	var ssaOrderingEdges []SSAOrderingEdge
	var operandEdges []OperandEdge
	fileSet := result.SSAProgram.Fset

	for _, pkg := range result.SSAProgram.AllPackages() {
		// TODO: hard-coded for now, ideally this should be a list of package prefixes we will include
		if pkg.Pkg.Path() == "github.com/sei-protocol/sei-chain/oracle/price-feeder/oracle/client" {
			for _, mem := range pkg.Members {
				if f, ok := mem.(*ssa.Function); ok {
					pos := fileSet.Position(f.Pos())
					// TODO: restrict to reachable functions from the call graph
					entryId := f.String()
					instructionNodes = append(instructionNodes, InstructionNode{
						NodeCommon: graphcommon.NodeCommon{
							ID:      entryId,
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
					for blockInd, b := range f.Blocks {
						_, firstInstrId := instructionId(fileSet, b.Instrs[0])
						_, lastInstrId := instructionId(fileSet, b.Instrs[len(b.Instrs)-1])
						for _, pred := range b.Preds {
							_, predId := instructionId(fileSet, pred.Instrs[len(pred.Instrs)-1])
							orderingEdges = append(orderingEdges, OrderingEdge{
								EdgeCommon: graphcommon.EdgeCommon{
									FromID: predId,
									ToID:   firstInstrId,
								},
							})
						}
						for _, succ := range b.Succs {
							_, succId := instructionId(fileSet, succ.Instrs[0])
							ssaOrderingEdges = append(ssaOrderingEdges, SSAOrderingEdge{
								EdgeCommon: graphcommon.EdgeCommon{
									FromID: lastInstrId,
									ToID:   succId,
								},
							})
						}

						for instrInd, instr := range b.Instrs {
							instrPosition, instrId := instructionId(fileSet, instr)
							if blockInd == 0 && instrInd == 0 {
								if b.Preds != nil && len(b.Preds) > 0 {
									panic("function-entry block has predecessors")
								}
								ssaOrderingEdges = append(ssaOrderingEdges, SSAOrderingEdge{
									EdgeCommon: graphcommon.EdgeCommon{
										FromID: entryId,
										ToID:   instrId,
									},
								})
							}
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
							for _, op := range instr.Operands(make([]*ssa.Value, 0)) {
								_, opId := valueId(fileSet, *op)
								operandEdges = append(operandEdges, OperandEdge{
									EdgeCommon: graphcommon.EdgeCommon{
										FromID: instrId,
										ToID:   opId,
									},
								})
							}

							if asValue, ok := instr.(ssa.Value); ok {
								valuePosition, vId := valueId(fileSet, asValue)
								valueNodes = append(valueNodes, ValueNode{
									NodeCommon: graphcommon.NodeCommon{
										ID:      vId,
										Name:    asValue.Name(),
										Package: pkg.Pkg.Path(),
										PositionInfo: graphcommon.PositionInfo{
										File:   valuePosition.Filename,
										Line:   valuePosition.Line,
										Column: valuePosition.Column,
									},
									},
									ValueType: valueTypeAsString(asValue),
								})
								for _, refr := range *asValue.Referrers() {
									_, referId := instructionId(fileSet, refr)
									referEdges = append(referEdges, ReferEdge{
										EdgeCommon: graphcommon.EdgeCommon{
											FromID: referId,
											ToID:   vId,
										},
									})
								}
							}
						}
					}
				} else if v, ok := mem.(ssa.Value); ok {
					valuePosition, vId := valueId(fileSet, v)
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
						ValueType: valueTypeAsString(v),
					})
					for _, instr := range *v.Referrers() {
						_, instrId := instructionId(fileSet, instr)
						operandEdges = append(operandEdges, OperandEdge{
							EdgeCommon: graphcommon.EdgeCommon{
								FromID: instrId,
								ToID:   vId,
							},
						})
					}
				}
			}
		}
	}

	return SSAGraphData{
		ValueNodes:       valueNodes,
		InstructionNodes: instructionNodes,
		ReferEdges:       referEdges,
		OrderingEdges:    orderingEdges,
		OperandEdges:     operandEdges,
		SSAOrderingEdges: ssaOrderingEdges,
	}
}

func instructionId(fileSet *token.FileSet, instr ssa.Instruction) (token.Position, string) {
	instrPosition := fileSet.Position(instr.Pos())
	instrId := fmt.Sprintf("%s:%d:%d", instrPosition.Filename, instrPosition.Line, instrPosition.Column)
	return instrPosition, instrId
}

func valueId(fileSet *token.FileSet, instr ssa.Value) (token.Position, string) {
	instrPosition := fileSet.Position(instr.Pos())
	instrId := fmt.Sprintf("value-%s:%d:%d", instrPosition.Filename, instrPosition.Line, instrPosition.Column)
	return instrPosition, instrId
}

func instrTypeAsString(instr ssa.Instruction) string {
	switch instr.(type) {
	case *ssa.Alloc:
		return "Alloc"
	case *ssa.BinOp:
		return "BinOp"
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
	case *ssa.Go:
		return "Go"
	case *ssa.If:
		return "If"
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
	}
	return "unknown"
}
