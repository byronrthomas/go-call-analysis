package analyzer

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/throwin5tone7/go-call-analysis/internal/analyzer/mock"
	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
)

const (
	defaultBatchSize = 10000
)

// GenerateNodeQuery dynamically generates a Neo4j CREATE query for a node
// based on the properties in the provided map.
// The map must contain "label" and "id" keys, otherwise the function will return an error.
func GenerateNodeQuery(nodeMap map[string]any) (string, error) {
	// Validate required fields
	label, hasLabel := nodeMap["label"]
	if !hasLabel {
		return "", fmt.Errorf("node map must contain 'label' key")
	}

	_, hasID := nodeMap["id"]
	if !hasID {
		return "", fmt.Errorf("node map must contain 'id' key")
	}

	// Convert label to string
	labelStr, ok := label.(string)
	if !ok {
		return "", fmt.Errorf("label must be a string, got %T", label)
	}

	if labelStr == "" {
		return "", fmt.Errorf("label cannot be empty")
	}

	// Build the property list dynamically
	var properties []string
	for key := range nodeMap {
		if key != "label" { // Skip label as it's used for the node type
			properties = append(properties, fmt.Sprintf("%s: node.%s", key, key))
		}
	}

	sort.Strings(properties)
	// Construct the query
	query := fmt.Sprintf("UNWIND $nodes AS node CREATE (n:node.label {%s})",
		strings.Join(properties, ", "))

	return query, nil
}

// GenerateEdgeQuery dynamically generates a Neo4j CREATE query for an edge
// based on the properties in the provided map.
// The map must contain "type", "from_id", "to_id", "from_label", and "to_label" keys.
func GenerateEdgeQuery(edgeMap map[string]any, fromLabel string, toLabel string) (string, error) {
	// Validate required fields
	requiredFields := []string{"type", "from_id", "to_id"}
	for _, field := range requiredFields {
		if _, exists := edgeMap[field]; !exists {
			return "", fmt.Errorf("edge map must contain '%s' key", field)
		}
	}

	// Extract and validate required string fields
	edgeType, ok := edgeMap["type"].(string)
	if !ok {
		return "", fmt.Errorf("type must be a string, got %T", edgeMap["type"])
	}
	if edgeType == "" {
		return "", fmt.Errorf("type cannot be empty")
	}

	if fromLabel == "" {
		return "", fmt.Errorf("from_label cannot be empty")
	}

	if toLabel == "" {
		return "", fmt.Errorf("to_label cannot be empty")
	}

	// Build edge properties (excluding the structural fields)
	excludedFields := map[string]bool{
		"type":    true,
		"from_id": true,
		"to_id":   true,
	}

	var edgeProperties []string
	for key := range edgeMap {
		if !excludedFields[key] {
			edgeProperties = append(edgeProperties, fmt.Sprintf("%s: edge.%s", key, key))
		}
	}

	sort.Strings(edgeProperties)

	// Build the relationship part
	var relationshipPart string
	if len(edgeProperties) > 0 {
		relationshipPart = fmt.Sprintf("[:edge.type {%s}]", strings.Join(edgeProperties, ", "))
	} else {
		relationshipPart = "[:edge.type]"
	}

	// Construct the complete query
	query := fmt.Sprintf(`
				UNWIND $edges AS edge
				MATCH (from:%s {id: edge.from_id}), (to:%s {id: edge.to_id})
				CREATE (from)-%s->(to)
			`, fromLabel, toLabel, relationshipPart)

	return query, nil
}

// Neo4jConfig holds the connection configuration for Neo4j
type Neo4jConfig struct {
	URI      string // Full URI including protocol, host, port
	Username string
	Password string
	Database string
}

func mapify(batch []graphcommon.Mappable) []map[string]any {
	result := make([]map[string]any, len(batch))
	for i, node := range batch {
		result[i] = node.ToMap()
	}
	return result
}

// ExportCallGraphToNeo4j exports the call graph data to a Neo4j database
func ExportCallGraphToNeo4j(nodes []FunctionNode, edges []CallEdge, config Neo4jConfig) error {
	return runInNeoSession(config, func(ctx context.Context, session neo4j.SessionWithContext) error {
		return runCallGraphInNeoSession(ctx, session, nodes, edges)
	})

}

func runCallGraphInNeoSession(ctx context.Context, session neo4j.SessionWithContext, nodes []FunctionNode, edges []CallEdge) error {
	// Start timing
	startTime := time.Now()

	// Import nodes
	log.Printf("Starting node import of %d nodes...", len(nodes))
	nodeStartTime := time.Now()
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for i := 0; i < len(nodes); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(nodes) {
				end = len(nodes)
			}

			batch := nodes[i:end]
			query := "UNWIND $nodes AS node CREATE (n:node.label {id: node.id, name: node.name, package: node.package, file: node.file, line: node.line, column: node.column})"

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, node := range batch {
				mappableBatch[i] = &node
			}

			params := map[string]any{
				"nodes": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create nodes batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed nodes %d-%d/%d (%.2f%%) in %v", i, end, len(nodes), float64(end)/float64(len(nodes))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import nodes: %v", err)
	}
	log.Printf("Node import completed in %v", time.Since(nodeStartTime))

	// Create index
	log.Println("Creating index on Function node IDs...")
	_, err = session.Run(ctx, "CREATE INDEX ON :Function(id)", nil)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	// Import edges
	log.Printf("Starting edge import of %d edges...", len(edges))
	edgeStartTime := time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for i := 0; i < len(edges); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(edges) {
				end = len(edges)
			}

			batch := edges[i:end]
			query := `
				UNWIND $edges AS edge
				MATCH (from:Function {id: edge.from_id}), (to:Function {id: edge.to_id})
				CREATE (from)-[:edge.type {call_site_file: edge.call_site_file, call_site_line: edge.call_site_line, call_site_column: edge.call_site_column, call_site_text: edge.call_site_text}]->(to)
			`

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, edge := range batch {
				mappableBatch[i] = &edge
			}

			params := map[string]any{
				"edges": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create edges batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed edges %d-%d/%d (%.2f%%) in %v", i, end, len(edges), float64(end)/float64(len(edges))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import edges: %v", err)
	}
	log.Printf("Edge import completed in %v", time.Since(edgeStartTime))

	log.Printf("Total import completed in %v", time.Since(startTime))
	return nil
}

func runInNeoSession(config Neo4jConfig, runnerFunc func(ctx context.Context, session neo4j.SessionWithContext) error) error {
	driver, err := neo4j.NewDriverWithContext(config.URI, neo4j.BasicAuth(config.Username, config.Password, ""))
	if err != nil {
		return fmt.Errorf("failed to create Neo4j driver: %v", err)
	}
	ctx := context.Background()
	defer driver.Close(ctx)

	// Create session
	session := createSession(ctx, driver, config)
	defer session.Close(ctx)

	err = runnerFunc(ctx, session)
	return err
}

var InMockMode = false
var MockSession neo4j.SessionWithContext

func createSession(ctx context.Context, driver neo4j.DriverWithContext, config Neo4jConfig) neo4j.SessionWithContext {
	if InMockMode {
		MockSession = &mock.MockSession{
			SessionWithContext: driver.NewSession(ctx, neo4j.SessionConfig{
				DatabaseName: config.Database,
			}),
		}
		return MockSession
	}
	return driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: config.Database,
	})
}

// ExportSSAGraphToNeo4j exports the SSA graph data to a Neo4j database
func ExportSSAGraphToNeo4j(graphData SSAGraphData, config Neo4jConfig) error {

	return runInNeoSession(config, func(ctx context.Context, session neo4j.SessionWithContext) error {
		return runSSAInNeoSession(ctx, session, graphData)
	})
}

// importNodesInBatches is a generic helper function to import nodes of any type
func importNodesInBatches[T graphcommon.Mappable](ctx context.Context, session neo4j.SessionWithContext, nodes *[]T, nodeTypeName string) error {
	if len(*nodes) == 0 {
		return nil
	}

	log.Printf("Starting node import of %d %s nodes...", len(*nodes), nodeTypeName)
	startTime := time.Now()

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for i := 0; i < len(*nodes); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(*nodes) {
				end = len(*nodes)
			}

			mappableBatch := make([]graphcommon.Mappable, end-i)
			for j := i; j < end; j++ {
				mappableBatch[j-i] = (*nodes)[j]
			}

			query, err := GenerateNodeQuery(mappableBatch[0].ToMap())
			if err != nil {
				return nil, fmt.Errorf("failed to generate node query: %v", err)
			}

			params := map[string]any{
				"nodes": mapify(mappableBatch),
			}

			_, err = tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create nodes batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed nodes %d-%d/%d (%.2f%%) in %v", i, end, len(*nodes), float64(end)/float64(len(*nodes))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import %s nodes: %v", nodeTypeName, err)
	}

	log.Printf("%s node import completed in %v", nodeTypeName, time.Since(startTime))
	return nil
}

// importEdgesInBatches is a generic helper function to import edges of any type
func importEdgesInBatches[T graphcommon.EdgeMappable](ctx context.Context, session neo4j.SessionWithContext, edges *[]T, edgeTypeName string) error {
	if len(*edges) == 0 {
		return nil
	}

	log.Printf("Starting edge import of %d %s edges...", len(*edges), edgeTypeName)
	startTime := time.Now()

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		for i := 0; i < len(*edges); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(*edges) {
				end = len(*edges)
			}

			types := (*edges)[i].NodeTypes()
			mappableBatch := make([]graphcommon.Mappable, end-i)
			for j := i; j < end; j++ {
				mappableBatch[j-i] = (*edges)[j]
			}

			query, err := GenerateEdgeQuery(mappableBatch[0].ToMap(), types.FromLabel, types.ToLabel)
			if err != nil {
				return nil, fmt.Errorf("failed to generate edge query: %v", err)
			}

			params := map[string]any{
				"edges": mapify(mappableBatch),
			}

			_, err = tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create edges batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed edges %d-%d/%d (%.2f%%) in %v", i, end, len(*edges), float64(end)/float64(len(*edges))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import %s edges: %v", edgeTypeName, err)
	}

	log.Printf("%s edge import completed in %v", edgeTypeName, time.Since(startTime))
	return nil
}

// createNodeIdIndex creates an index for the specified node type
func createNodeIdIndex(ctx context.Context, session neo4j.SessionWithContext, nodeType string) error {
	return createNodeIndex(ctx, fmt.Sprintf(":%s(id)", nodeType), session)
}

func createNodeIndex(ctx context.Context, indexExpr string, session neo4j.SessionWithContext) error {
	log.Printf("Creating node index on %s...", indexExpr)
	_, err := session.Run(ctx, fmt.Sprintf("CREATE INDEX ON %s", indexExpr), nil)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}
	return nil
}

var additionalNodeIndexes []string = []string{
	":Instruction(instruction_type)",
}

func runSSAInNeoSession(ctx context.Context, session neo4j.SessionWithContext, graphData SSAGraphData) error {
	// Start timing
	startTime := time.Now()

	// Import file version nodes
	if err := importNodesInBatches(ctx, session, &graphData.FileVersionNodes, "file version"); err != nil {
		return err
	}
	if err := createNodeIdIndex(ctx, session, "FileVersion"); err != nil {
		return err
	}

	// Import function nodes
	if err := importNodesInBatches(ctx, session, &graphData.FunctionNodes, "function"); err != nil {
		return err
	}
	if err := createNodeIdIndex(ctx, session, "Function"); err != nil {
		return err
	}

	// Import instruction nodes
	if err := importNodesInBatches(ctx, session, &graphData.InstructionNodes, "instruction"); err != nil {
		return err
	}
	if err := createNodeIdIndex(ctx, session, "Instruction"); err != nil {
		return err
	}

	// Import value nodes
	if err := importNodesInBatches(ctx, session, &graphData.ValueNodes, "value"); err != nil {
		return err
	}
	if err := createNodeIdIndex(ctx, session, "Value"); err != nil {
		return err
	}

	// Import function entry edges
	if err := importEdgesInBatches(ctx, session, &graphData.FunctionEntryEdges, "function entry"); err != nil {
		return err
	}

	// Import has parameter edges
	if err := importEdgesInBatches(ctx, session, &graphData.HasParameterEdges, "has parameter"); err != nil {
		return err
	}

	// Import ordering edges
	if err := importEdgesInBatches(ctx, session, &graphData.OrderingEdges, "ordering"); err != nil {
		return err
	}

	// Import operand edges
	if err := importEdgesInBatches(ctx, session, &graphData.OperandEdges, "operand"); err != nil {
		return err
	}

	// Import result edges
	if err := importEdgesInBatches(ctx, session, &graphData.ResultEdges, "result"); err != nil {
		return err
	}

	// Import control flow edges
	if err := importEdgesInBatches(ctx, session, &graphData.ControlFlowEdges, "control flow"); err != nil {
		return err
	}

	// Import resolved call edges
	if err := importEdgesInBatches(ctx, session, &graphData.ResolvedCallEdges, "resolved call"); err != nil {
		return err
	}

	// Import belongs to edges
	if err := importEdgesInBatches(ctx, session, &graphData.BelongsToEdges, "belongs to"); err != nil {
		return err
	}

	// Import return point edges
	if err := importEdgesInBatches(ctx, session, &graphData.ReturnPointEdges, "has return point"); err != nil {
		return err
	}

	log.Printf("Creating Edge type index")
	_, err := session.Run(ctx, "CREATE EDGE INDEX ON :EDGE_TYPE;", nil)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	for _, index := range additionalNodeIndexes {
		if err := createNodeIndex(ctx, index, session); err != nil {
			return err
		}
	}

	log.Printf("Total import completed in %v", time.Since(startTime))
	return nil
}

const derefPropagationQueryCount = `
MATCH 
(vIn:Value)<-[:Uses_Operand {index: 0}]-(deref:Instruction {instruction_type: "UnOp(*)"})
-[:Produces_Result {index: 0}]->(vOut:Value)
WHERE vIn.fixed_width_value_kind IS NOT NULL
AND vOut.fixed_width_value_kind IS NULL
RETURN count(vOut)
`

const derefPropagationQueryUpdate = `
MATCH 
(vIn:Value)<-[:Uses_Operand {index: 0}]-(deref:Instruction {instruction_type: "UnOp(*)"})
-[:Produces_Result {index: 0}]->(vOut:Value)
WHERE vIn.fixed_width_value_kind IS NOT NULL
AND vOut.fixed_width_value_kind IS NULL
SET vOut.fixed_width_value_kind = "deref(" + vIn.fixed_width_value_kind + ")"
`

type PropagationQuery struct {
	CountQuery     string
	UpdateQuery    string
	CountFieldName string
	QueryName      string
}

var derefPropagationQuery = PropagationQuery{
	CountQuery:     derefPropagationQueryCount,
	UpdateQuery:    derefPropagationQueryUpdate,
	CountFieldName: "count(vOut)",
	QueryName:      "Deref",
}

const ITERATION_LIMIT = 100

func runPropagationQueryInNeoSession(ctx context.Context, session neo4j.SessionWithContext, query PropagationQuery) error {

	count, err := runCountQuery(ctx, session, query)
	if err != nil {
		return err
	}
	log.Printf("%s propagation count: %d", query.QueryName, count)
	iteration := 0
	for count > 0 && iteration < ITERATION_LIMIT {
		_, err := session.Run(ctx, query.UpdateQuery, nil)
		if err != nil {
			return fmt.Errorf("failed to run propagation update query: %v", err)
		}
		log.Printf("Propagation update completed")

		count, err = runCountQuery(ctx, session, query)
		if err != nil {
			return err
		}
		log.Printf("%s propagation count: %d", query.QueryName, count)
		iteration++
	}
	return nil
}

func runCountQuery(ctx context.Context, session neo4j.SessionWithContext, query PropagationQuery) (int64, error) {
	r1, err := session.Run(ctx, query.CountQuery, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to run propagation counting query: %v", err)
	}
	r1S, err := r1.Single(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get single result of propagation counting query: %v", err)
	}
	count, ok := r1S.AsMap()[query.CountFieldName].(int64)
	if !ok {
		return 0, fmt.Errorf("failed to get count of %s propagation: %v", query.QueryName, err)
	}
	return count, nil
}

func runPropagationQueries(ctx context.Context, session neo4j.SessionWithContext, queries []PropagationQuery) error {
	for _, query := range queries {
		if err := runPropagationQueryInNeoSession(ctx, session, query); err != nil {
			return err
		}
	}
	return nil
}

func RunPropagationQueries(config Neo4jConfig) error {
	return runInNeoSession(config, func(ctx context.Context, session neo4j.SessionWithContext) error {
		return runPropagationQueries(ctx, session, []PropagationQuery{derefPropagationQuery})
	})
}
