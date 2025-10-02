package lib

import "github.com/throwin5tone7/go-call-analysis/internal/analyzer"

const derefPropagationQueryPrefix = `
MATCH (vIn:Value)<-[:Uses_Operand {index: 0}]-(deref:Instruction {instruction_type: "UnOp(*)"})
-[:Produces_Result {index: 0}]->(vOut:Value)
WHERE vIn.fixed_width_value_kind IS NOT NULL
AND vOut.fixed_width_value_kind IS NULL
`

const derefPropagationQueryCount = derefPropagationQueryPrefix + `
RETURN count(vOut)
`

const derefPropagationQueryUpdate = `
` + derefPropagationQueryPrefix + `
SET vOut.fixed_width_value_kind = "deref(" + vIn.fixed_width_value_kind + ")"
`

var derefPropagationQuery = analyzer.PropagationQuery{
	CountQuery:     derefPropagationQueryCount,
	UpdateQuery:    derefPropagationQueryUpdate,
	CountFieldName: "count(vOut)",
	QueryName:      "Deref",
}

const appendFixedQueryPrefix = `
MATCH 
(i:Function {id: "^builtin^append"})
<-[:Resolved_Call]-(cs)
-[:Produces_Result]->(v),
(cs)-[:Uses_Operand {index: 0}]->(appArg1),
(cs)-[:Uses_Operand {index: 1}]->(appArg2)
WHERE
appArg1.fixed_width_value_kind IS NOT NULL
AND appArg2.fixed_width_value_kind IS NOT NULL
AND v.fixed_width_value_kind IS NULL`

const appendFixedQueryCount = appendFixedQueryPrefix + `
RETURN count(DISTINCT v)
`

const appendFixedQueryUpdate = `
` + appendFixedQueryPrefix + `
SET v.fixed_width_value_kind = "append(fixed, fixed)"
`

var appendFixedQuery = analyzer.PropagationQuery{
	CountQuery:     appendFixedQueryCount,
	UpdateQuery:    appendFixedQueryUpdate,
	CountFieldName: "count(DISTINCT v)",
	QueryName:      "append(fixed, fixed)",
}

const funcSingleReturnFixedQueryPrefix = `
MATCH
(ftgt:Function {num_return_points: 1})
-[:Has_Return_Point]->(ri:Instruction)
-[:Uses_Operand {index: 0}]->(v)
WHERE
v.fixed_width_value_kind IS NOT NULL
AND v.type_name = "[]byte"
AND NOT coalesce(ftgt.func_returns_fixed_width, false)
`

const funcSingleReturnFixedQueryCount = funcSingleReturnFixedQueryPrefix + `
RETURN count(ftgt)
`

const funcSingleReturnFixedQueryUpdate = funcSingleReturnFixedQueryPrefix + `
SET ftgt.func_returns_fixed_width = true
`

var funcSingleReturnFixedQuery = analyzer.PropagationQuery{
	CountQuery:     funcSingleReturnFixedQueryCount,
	UpdateQuery:    funcSingleReturnFixedQueryUpdate,
	CountFieldName: "count(ftgt)",
	QueryName:      "func has single return fixed",
}

const labelFuncToRetValPrefix = `
MATCH
(v:Value)<-[:Produces_Result {index: 0}]-
(cs:Instruction)-[:Resolved_Call]->(ftgt {func_returns_fixed_width: true})
WHERE v.fixed_width_value_kind IS NULL
`

const labelFuncToRetValQueryCount = labelFuncToRetValPrefix + `
RETURN count(distinct v)
`

const labelFuncToRetValQueryUpdate = labelFuncToRetValPrefix + `
SET v.fixed_width_value_kind = "fixed width func result"
`

var labelFuncToRetValQuery = analyzer.PropagationQuery{
	CountQuery:     labelFuncToRetValQueryCount,
	UpdateQuery:    labelFuncToRetValQueryUpdate,
	CountFieldName: "count(distinct v)",
	QueryName:      "label func to ret val",
}

var fixedWidthPropagationQueries = []analyzer.PropagationQuery{derefPropagationQuery, appendFixedQuery, funcSingleReturnFixedQuery, labelFuncToRetValQuery}
