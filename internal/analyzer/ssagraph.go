package analyzer

import (
	"fmt"
	"go/token"
	"go/types"
	"log"
	"maps"
	"slices"
	"strings"

	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
	"golang.org/x/tools/go/ssa"
)

type SSAGraphData struct {
	FileVersionNodes   []graphcommon.FileVersionNode
	BelongsToEdges     []BelongsToEdge
	ValueNodes         []ValueNode
	InstructionNodes   []InstructionNode
	FunctionNodes      []SSAGraphFunctionNode
	OrderingEdges      []OrderingEdge
	OperandEdges       []OperandEdge
	ControlFlowEdges   []ControlFlowEdge
	ResultEdges        []ResultEdge
	ResolvedCallEdges  []ResolvedCallEdge
	FunctionEntryEdges []FunctionEntryEdge
	HasParameterEdges  []HasParameterEdge
	ReturnPointEdges   []ReturnPointEdge
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
	// Used to store the condition of an annotated if
	Annotation string
}

type SSAGraphFunctionNode struct {
	graphcommon.NodeCommon
	// Used to store the condition of an annotated if
	Annotation string
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
	Index int
}

type ResolvedCallEdge struct {
	graphcommon.EdgeCommon
	EdgeCardinality int
}

type ResultEdge struct {
	graphcommon.EdgeCommon
	Index int
}

type BelongsToEdge struct {
	graphcommon.EdgeCommon
}

type FunctionEntryEdge struct {
	graphcommon.EdgeCommon
}

type HasParameterEdge struct {
	graphcommon.EdgeCommon
	Index int
}

type ReturnPointEdge struct {
	graphcommon.EdgeCommon
}

func (n ValueNode) ToMap() map[string]any {
	nodeCommonMap := graphcommon.NodeCommonAsMap(n.NodeCommon)
	nodeCommonMap["label"] = "Value"
	nodeCommonMap["value_type"] = n.ValueType
	nodeCommonMap["type_name"] = n.TypeName
	nodeCommonMap["is_error_type"] = n.IsErrorType
	return nodeCommonMap
}

func (n InstructionNode) ToMap() map[string]any {
	nodeCommonMap := graphcommon.NodeCommonAsMap(n.NodeCommon)
	nodeCommonMap["label"] = "Instruction"
	nodeCommonMap["instruction_type"] = n.InstructionType
	nodeCommonMap["annotation"] = n.Annotation
	return nodeCommonMap
}

func (n SSAGraphFunctionNode) ToMap() map[string]any {
	nodeCommonMap := graphcommon.NodeCommonAsMap(n.NodeCommon)
	nodeCommonMap["label"] = "Function"
	nodeCommonMap["annotation"] = n.Annotation
	return nodeCommonMap
}

func (e OrderingEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "And_Then"
	return edgeCommonMap
}

func (e OrderingEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Instruction",
		ToLabel:   "Instruction",
	}
}

func (e ControlFlowEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Control_Flow"
	edgeCommonMap["condition"] = e.Condition
	return edgeCommonMap
}

func (e ControlFlowEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Instruction",
		ToLabel:   "Instruction",
	}
}

func (e OperandEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Uses_Operand"
	edgeCommonMap["index"] = e.Index
	return edgeCommonMap
}

func (e OperandEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Instruction",
		ToLabel:   "Value",
	}
}

func (e ResultEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Produces_Result"
	edgeCommonMap["index"] = e.Index
	return edgeCommonMap
}

func (e ResultEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Instruction",
		ToLabel:   "Value",
	}
}

func (e ResolvedCallEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Resolved_Call"
	edgeCommonMap["edge_cardinality"] = e.EdgeCardinality
	return edgeCommonMap
}

func (e ResolvedCallEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Instruction",
		ToLabel:   "Function",
	}
}

func (e BelongsToEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Belongs_To"
	return edgeCommonMap
}

func (e BelongsToEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Function",
		ToLabel:   "FileVersion",
	}
}

func (e FunctionEntryEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Function_Entry"
	return edgeCommonMap
}

func (e FunctionEntryEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Function",
		ToLabel:   "Instruction",
	}
}

func (e HasParameterEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Has_Parameter"
	edgeCommonMap["index"] = e.Index
	return edgeCommonMap
}

func (e HasParameterEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Function",
		ToLabel:   "Value",
	}
}

func (e ReturnPointEdge) ToMap() map[string]any {
	edgeCommonMap := graphcommon.EdgeCommonAsMap(e.EdgeCommon)
	edgeCommonMap["type"] = "Has_Return_Point"
	return edgeCommonMap
}

func (e ReturnPointEdge) NodeTypes() graphcommon.NodeTypes {
	return graphcommon.NodeTypes{
		FromLabel: "Function",
		ToLabel:   "Instruction",
	}
}

type GraphVisitor struct {
	BaseSSAVisitor
	SSASimplificationResult *SSASimplificationResult
	fileSet                 *token.FileSet
	gitRevisionCache        *GitRevisionCache
	fileVersionNodes        map[string]graphcommon.FileVersionNode
	instructionNodes        []InstructionNode
	functionNodes           []SSAGraphFunctionNode
	orderingEdges           []OrderingEdge
	controlFlowEdges        []ControlFlowEdge
	operandEdges            []OperandEdge
	resultEdges             []ResultEdge
	resolvedCallEdges       []ResolvedCallEdge
	valueNodes              []ValueNode
	belongsToEdges          []BelongsToEdge
	functionEntryEdges      []FunctionEntryEdge
	hasParameterEdges       []HasParameterEdge
	functionEntries         map[string]bool
	processedValues         map[string]bool
	returnPointEdges        []ReturnPointEdge
}

func (v *GraphVisitor) VisitFunction(f *ssa.Function, pkg *ssa.Package) {

	pos := v.fileSet.Position(f.Pos())
	// TODO: restrict to reachable functions from the call graph
	funcId := f.String()
	if !v.SSASimplificationResult.ShouldVisitFunction(f) {
		log.Printf("INFO: Skipping function %s because it is marked as unreachable", funcId)
		return
	}
	if _, ok := v.fileVersionNodes[pos.Filename]; !ok {
		packageName := "unknown-package"
		if pkg != nil {
			packageName = pkg.Pkg.Path()
		}
		v.fileVersionNodes[pos.Filename] = graphcommon.FileVersionNode{
			Id:              pos.Filename,
			LastGitRevision: v.gitRevisionCache.GetFileRevision(pos.Filename),
			Package:         packageName,
		}
	}
	if _, ok := v.functionEntries[funcId]; !ok {
		addFunctionEntryNode(v, funcId, f, pkg, pos)
	}
	v.belongsToEdges = append(v.belongsToEdges, BelongsToEdge{
		EdgeCommon: graphcommon.EdgeCommon{
			FromID: funcId,
			ToID:   pos.Filename,
		},
	})
	v.functionEntries[funcId] = true

	for paramIndex, param := range f.Params {
		_, paramId := ValueId(v.fileSet, param, "")
		v.valueNodes = processValue(v.valueNodes, paramId, param, pkg, pos, v.gitRevisionCache, v.processedValues)
		v.hasParameterEdges = append(v.hasParameterEdges, HasParameterEdge{
			EdgeCommon: graphcommon.EdgeCommon{
				FromID: funcId,
				ToID:   paramId,
			},
			Index: paramIndex,
		})
	}

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
			annotation := ""
			if asAnnotatedIf, ok := instr.(*AnnotatedIf); ok {
				annotation = asAnnotatedIf.ConditionDescription
			}

			instrPosition := v.fileSet.Position(instr.Pos())
			v.instructionNodes = append(v.instructionNodes, InstructionNode{
				NodeCommon: graphcommon.NodeCommon{
					ID:   instrId,
					Name: instr.String(),
					PositionInfo: graphcommon.PositionInfo{
						Line:   instrPosition.Line,
						Column: instrPosition.Column,
					},
				},
				InstructionType: instrTypeAsString(instr),
				Annotation:      annotation,
			})

			// Add the and-then edge from the previous instruction to the current instruction
			if precInstrId != "" {
				if precInstrId == funcId {
					// This is the first instruction in the first block, use FunctionEntryEdge
					v.functionEntryEdges = append(v.functionEntryEdges, FunctionEntryEdge{
						EdgeCommon: graphcommon.EdgeCommon{
							FromID: precInstrId,
							ToID:   instrId,
						},
					})
				} else {
					// Regular ordering edge between instructions
					v.orderingEdges = append(v.orderingEdges, OrderingEdge{
						EdgeCommon: graphcommon.EdgeCommon{
							FromID: precInstrId,
							ToID:   instrId,
						},
					})
				}
			}
			precInstrId = instrId

			operands := instr.Operands(make([]*ssa.Value, 0))
			if asAnnotatedCall, ok := instr.(*AnnotatedCall); ok {
				operands = make([]*ssa.Value, len(asAnnotatedCall.Args))
				copy(operands, asAnnotatedCall.Args)
			}

			for opIndex, op := range operands {
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
					Index: opIndex,
				})
			}

			if asAnnotatedCall, ok := instr.(*AnnotatedCall); ok {
				for returnValueIndex, returnValue := range asAnnotatedCall.ReturnValues {
					_, returnValueId := ValueId(v.fileSet, returnValue, currentBlockId)
					v.valueNodes = processValue(v.valueNodes, returnValueId, returnValue, pkg, instrPosition, v.gitRevisionCache, v.processedValues)
					v.resultEdges = append(v.resultEdges, ResultEdge{
						EdgeCommon: graphcommon.EdgeCommon{
							FromID: instrId,
							ToID:   returnValueId,
						},
						Index: returnValueIndex,
					})
				}
				if len(asAnnotatedCall.ResolvedTargets) > 0 {
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
				} else if asAnnotatedCall.DynamicCallee != nil {
					dynamicCallee := asAnnotatedCall.DynamicCallee
					if asBuiltin, ok := dynamicCallee.(*ssa.Builtin); ok {
						builtinId := "^builtin^" + asBuiltin.Name()
						if _, ok := v.functionEntries[builtinId]; !ok {
							v.functionNodes = append(v.functionNodes, SSAGraphFunctionNode{
								NodeCommon: graphcommon.NodeCommon{
									ID:           builtinId,
									Name:         asBuiltin.String(),
									PositionInfo: graphcommon.PositionInfo{},
								},
								Annotation: "builtin",
							})
							v.functionEntries[builtinId] = true
						}
						v.resolvedCallEdges = append(v.resolvedCallEdges, ResolvedCallEdge{
							EdgeCommon: graphcommon.EdgeCommon{
								FromID: instrId,
								ToID:   builtinId,
							},
							EdgeCardinality: len(asAnnotatedCall.ResolvedTargets)})
					}

				}
			} else if asValue, ok := instr.(ssa.Value); ok {

				_, vId := ValueId(v.fileSet, asValue, currentBlockId)
				v.valueNodes = processValue(v.valueNodes, vId, asValue, pkg, instrPosition, v.gitRevisionCache, v.processedValues)

				// If instruction produces a value, add a result edge from the instruction to the value
				v.resultEdges = append(v.resultEdges, ResultEdge{
					EdgeCommon: graphcommon.EdgeCommon{
						FromID: instrId,
						ToID:   vId,
					},
					Index: 0,
				})
			}
		}

		lastInstr := b.Instrs[len(b.Instrs)-1]
		if _, ok := lastInstr.(*ssa.Return); ok {
			v.returnPointEdges = append(v.returnPointEdges, ReturnPointEdge{
				EdgeCommon: graphcommon.EdgeCommon{
					FromID: funcId,
					ToID:   precInstrId,
				},
			})
		}
	}
}

func addFunctionEntryNode(v *GraphVisitor, funcId string, f *ssa.Function, pkg *ssa.Package, pos token.Position) {
	v.functionNodes = append(v.functionNodes, SSAGraphFunctionNode{
		NodeCommon: graphcommon.NodeCommon{
			ID:   funcId,
			Name: f.Name(),
			PositionInfo: graphcommon.PositionInfo{
				Line:   pos.Line,
				Column: pos.Column,
			},
		},
		Annotation: "",
	})
}

func (v *GraphVisitor) VisitTypeMethod(_method *types.Func, ssaFunc *ssa.Function, _namedType *types.Named, _pkg *ssa.Package) {
	if !v.SSASimplificationResult.ShouldVisitFunction(ssaFunc) {
		log.Printf("INFO: Skipping type method %s because it is marked as unreachable", ssaFunc.String())
		return
	}
	v.VisitFunction(ssaFunc, _pkg)
}

func (v *GraphVisitor) VisitValue(valueObj ssa.Value, pkg *ssa.Package) {
	valuePosition, vId := ValueId(v.fileSet, valueObj, "")
	v.valueNodes = processValue(v.valueNodes, vId, valueObj, pkg, valuePosition, v.gitRevisionCache, v.processedValues)
}

func ExtractSSAGraphData(simplificationResult *SSASimplificationResult, packagePrefixes []string, projectPath string) SSAGraphData {
	fileSet := simplificationResult.SSAProgram.Fset

	visitor := &GraphVisitor{
		BaseSSAVisitor:          BaseSSAVisitor{},
		SSASimplificationResult: simplificationResult,
		fileSet:                 fileSet,
		gitRevisionCache:        NewGitRevisionCache(projectPath),
		functionEntries:         make(map[string]bool),
		fileVersionNodes:        make(map[string]graphcommon.FileVersionNode),
		processedValues:         make(map[string]bool),
	}
	traverser := NewSSATraverser(packagePrefixes)
	traverser.Traverse(simplificationResult.SSAProgram, visitor)

	return SSAGraphData{
		FileVersionNodes:   slices.Collect(maps.Values(visitor.fileVersionNodes)),
		ValueNodes:         visitor.valueNodes,
		InstructionNodes:   visitor.instructionNodes,
		FunctionNodes:      visitor.functionNodes,
		OrderingEdges:      visitor.orderingEdges,
		OperandEdges:       visitor.operandEdges,
		ControlFlowEdges:   visitor.controlFlowEdges,
		ResultEdges:        visitor.resultEdges,
		ResolvedCallEdges:  visitor.resolvedCallEdges,
		BelongsToEdges:     visitor.belongsToEdges,
		FunctionEntryEdges: visitor.functionEntryEdges,
		HasParameterEdges:  visitor.hasParameterEdges,
		ReturnPointEdges:   visitor.returnPointEdges,
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

func processValue(valueNodes []ValueNode, vId string, v ssa.Value, pkg *ssa.Package, valuePosition token.Position, gitCache *GitRevisionCache, processedValues map[string]bool) []ValueNode {
	if _, ok := processedValues[vId]; ok {
		return valueNodes
	}
	processedValues[vId] = true
	valueNodes = append(valueNodes, ValueNode{
		NodeCommon: graphcommon.NodeCommon{
			ID:   vId,
			Name: v.Name(),
			PositionInfo: graphcommon.PositionInfo{
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

func checkComparableType(t types.Type) bool {
	switch t.(type) {
	case *types.Basic:
		return true

	case *types.Array:
		return true

	case *types.Slice:
		return true

	case *types.Struct:
		return true

	case *types.Pointer:
		return true

	case *types.Tuple:
		return true

	case *types.Signature:
		return true

	case *types.Union:
		return true

	case *types.Interface:
		return true

	case *types.Map:
		return true

	case *types.Chan:
		return true

	case *types.Named:
		return true

	case *types.TypeParam:
		return true
	case nil:
		return true
	}

	return false
}

func isErrorType(t types.Type) bool {
	if checkComparableType(t) {
		errorType := types.Universe.Lookup("error").Type()
		return types.AssignableTo(t, errorType)
	}
	return false
}

func ValueId(fileSet *token.FileSet, valueObj ssa.Value, producingBlockId string) (token.Position, string) {
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
		valuePos := asFRP.FromCall.Pos()
		valuePosition := fileSet.Position(valuePos)
		valueId := fmt.Sprintf("return-placeholder-%d-%s:%d:%d", asFRP.Index, valuePosition.Filename, valuePosition.Line, valuePosition.Column)
		return valuePosition, valueId
	} else if asGlobal, ok := valueObj.(*ssa.Global); ok {
		valueId := fmt.Sprintf("%s:%s", asGlobal.Pkg.Pkg.Path(), asGlobal.Name())
		return token.Position{}, valueId
	} else if asFunction, ok := valueObj.(*ssa.Function); ok {
		if asFunction.Synthetic != "" {
			if asFunction.Pkg == nil && asFunction.Pos() != token.NoPos {
				valuePosition := fileSet.Position(asFunction.Pos())
				return valuePosition, strings.ReplaceAll(asFunction.Synthetic, " ", "_")
			}
			valueId := fmt.Sprintf("synthetic-%s-%s", asFunction.Pkg.Pkg.Path(), asFunction.String())
			return token.Position{}, valueId
		}
		//log.Printf("WARN: Function being treated as a value has no synthetic ID: %v", asFunction)
		valueId := fmt.Sprintf("%s:%s", asFunction.Pkg.Pkg.Path(), asFunction.String())
		return fileSet.Position(asFunction.Pos()), valueId
	} else if asParameter, ok := valueObj.(*ssa.Parameter); ok {
		valueId := fmt.Sprintf("parameter-%s.%s", asParameter.Parent().String(), asParameter.Name())
		return token.Position{}, valueId
	} else if asBuiltin, ok := valueObj.(*ssa.Builtin); ok {
		valueId := fmt.Sprintf("builtin-%s", asBuiltin.Name())
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

func ContextualId(block *ssa.BasicBlock, instrIndex int) string {
	return fmt.Sprintf("%s:%d", blockId(block), instrIndex)
}

func blockId(block *ssa.BasicBlock) string {
	return fmt.Sprintf("%s:%d", block.Parent().String(), block.Index)
}

func instrTypeAsString(instr ssa.Instruction) string {
	switch instr := instr.(type) {
	case *ssa.Alloc:
		return "Alloc"
	case *ssa.BinOp:
		return fmt.Sprintf("BinOp(%s)", instr.Op.String())
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
		return fmt.Sprintf("UnOp(%s)", instr.Op.String())
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
