package lib

import "github.com/throwin5tone7/go-call-analysis/internal/analyzer"

const appendBothVaryingQueryPrefix = `
MATCH
(i:Function {id: "^builtin^append"})
<-[:Resolved_Call]-(cs)
-[:Produces_Result]->(v),
(cs)-[:Uses_Operand {index: 0}]->(appArg1),
(cs)-[:Uses_Operand {index: 1}]->(appArg2)
WHERE
appArg1.fixed_width_value_kind IS NULL
AND appArg2.fixed_width_value_kind IS NULL
AND v.fixed_width_value_kind IS NULL
AND v.known_two_component_varying IS NULL
`

const appendBothVaryingQueryCount = appendBothVaryingQueryPrefix + `
RETURN count(DISTINCT v)
`

const appendBothVaryingQueryUpdate = appendBothVaryingQueryPrefix + `
SET v.known_two_component_varying = "append(varying, varying)"
`

var appendBothVaryingQuery = analyzer.PropagationQuery{
	CountQuery:     appendBothVaryingQueryCount,
	UpdateQuery:    appendBothVaryingQueryUpdate,
	CountFieldName: "count(DISTINCT v)",
	QueryName:      "append(varying, varying)",
}

const appendTwoVaryingQueryPrefix = `
MATCH 
(i:Function {id: "^builtin^append"})
<-[:Resolved_Call]-(cs)
-[:Produces_Result]->(v),
(cs)-[:Uses_Operand]->(appArg1)
WHERE
appArg1.known_two_component_varying IS NOT NULL
AND v.known_two_component_varying IS NULL
`

const appendTwoVaryingQueryCount = appendTwoVaryingQueryPrefix + `
RETURN count(DISTINCT v)
`

const appendTwoVaryingQueryUpdate = appendTwoVaryingQueryPrefix + `
SET v.known_two_component_varying = "append(2comp, any)"
`

var appendTwoVaryingQuery = analyzer.PropagationQuery{
	CountQuery:     appendTwoVaryingQueryCount,
	UpdateQuery:    appendTwoVaryingQueryUpdate,
	CountFieldName: "count(DISTINCT v)",
	QueryName:      "append(2comp, any)",
}

const funcSingleReturnTwoVaryingQueryPrefix = `
MATCH
(ftgt:Function {num_return_points: 1})
-[:Has_Return_Point]->(ri:Instruction)
-[:Uses_Operand {index: 0}]->(v)
WHERE
v.known_two_component_varying IS NOT NULL
AND v.type_name = "[]byte"
AND NOT coalesce(ftgt.func_returns_two_comp_varying, false)
`

const funcSingleReturnTwoVaryingQueryCount = funcSingleReturnTwoVaryingQueryPrefix + `
RETURN count(ftgt)
`

const funcSingleReturnTwoVaryingQueryUpdate = funcSingleReturnTwoVaryingQueryPrefix + `
SET ftgt.func_returns_two_comp_varying = true
`

var funcSingleReturnTwoVaryingQuery = analyzer.PropagationQuery{
	CountQuery:     funcSingleReturnTwoVaryingQueryCount,
	UpdateQuery:    funcSingleReturnTwoVaryingQueryUpdate,
	CountFieldName: "count(ftgt)",
	QueryName:      "func has single return two varying",
}

const labelFuncToRetValTwoVaryingQueryPrefix = `
MATCH
(v:Value)<-[:Produces_Result {index: 0}]-
(cs:Instruction)-[:Resolved_Call]->(ftgt {func_returns_two_comp_varying: true})
WHERE v.fixed_width_value_kind IS NULL
AND v.known_two_component_varying IS NULL
`

const labelFuncToRetValTwoVaryingQueryCount = labelFuncToRetValTwoVaryingQueryPrefix + `
RETURN count(distinct v)
`

const labelFuncToRetValTwoVaryingQueryUpdate = labelFuncToRetValTwoVaryingQueryPrefix + `
SET v.known_two_component_varying = "two comp varying width func result"
`

var labelFuncToRetValTwoVaryingQuery = analyzer.PropagationQuery{
	CountQuery:     labelFuncToRetValTwoVaryingQueryCount,
	UpdateQuery:    labelFuncToRetValTwoVaryingQueryUpdate,
	CountFieldName: "count(distinct v)",
	QueryName:      "two comp varying width func result",
}

var twoVaryingPropagationQueries = []analyzer.PropagationQuery{appendBothVaryingQuery, appendTwoVaryingQuery, funcSingleReturnTwoVaryingQuery, labelFuncToRetValTwoVaryingQuery}
