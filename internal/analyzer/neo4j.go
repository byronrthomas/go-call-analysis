package analyzer

import (
	"context"
	"fmt"
	"log"
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
func GenerateNodeQuery(nodeMap map[string]interface{}) (string, error) {
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

	// Construct the query
	query := fmt.Sprintf("UNWIND $nodes AS node CREATE (n:%s {%s})",
		labelStr,
		joinProperties(properties))

	return query, nil
}

// joinProperties joins property assignments with commas
func joinProperties(properties []string) string {
	if len(properties) == 0 {
		return ""
	}

	result := ""
	for i, prop := range properties {
		if i > 0 {
			result += ", "
		}
		result += prop
	}
	return result
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
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
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

			params := map[string]interface{}{
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
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
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

			params := map[string]interface{}{
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

func runSSAInNeoSession(ctx context.Context, session neo4j.SessionWithContext, graphData SSAGraphData) error {
	// Start timing
	startTime := time.Now()

	// Import file version nodes
	log.Printf("Starting node import of %d file version nodes...", len(graphData.FileVersionNodes))
	fileRevisionStartTime := time.Now()
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.FileVersionNodes); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize

			if end > len(graphData.FileVersionNodes) {
				end = len(graphData.FileVersionNodes)
			}

			batch := graphData.FileVersionNodes[i:end]
			query := "UNWIND $nodes AS node CREATE (n:node.label {id: node.id, name: node.name, last_git_revision: node.last_git_revision})"

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, node := range batch {
				mappableBatch[i] = &node
			}

			params := map[string]interface{}{
				"nodes": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create nodes batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed nodes %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.FileVersionNodes), float64(end)/float64(len(graphData.FileVersionNodes))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import file version nodes: %v", err)
	}
	log.Printf("file version node import completed in %v", time.Since(fileRevisionStartTime))

	// Create index
	log.Println("Creating index on FileVersion node IDs...")
	_, err = session.Run(ctx, "CREATE INDEX ON :FileVersion(id)", nil)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	// Import instruction nodes
	log.Printf("Starting node import of %d instruction nodes...", len(graphData.InstructionNodes))
	nodeStartTime := time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.InstructionNodes); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(graphData.InstructionNodes) {
				end = len(graphData.InstructionNodes)
			}

			batch := graphData.InstructionNodes[i:end]
			query := "UNWIND $nodes AS node CREATE (n:node.label {id: node.id, name: node.name, package: node.package, line: node.line, column: node.column, instruction_type: node.instruction_type, annotation: node.annotation})"

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, node := range batch {
				mappableBatch[i] = &node
			}

			params := map[string]interface{}{
				"nodes": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create nodes batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed nodes %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.InstructionNodes), float64(end)/float64(len(graphData.InstructionNodes))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import instruction nodes: %v", err)
	}
	log.Printf("Instruction node import completed in %v", time.Since(nodeStartTime))

	// Create index
	log.Println("Creating index on Instruction node IDs...")
	_, err = session.Run(ctx, "CREATE INDEX ON :Instruction(id)", nil)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	// Import value nodes
	log.Printf("Starting node import of %d value nodes...", len(graphData.ValueNodes))
	valueStartTime := time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.ValueNodes); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(graphData.ValueNodes) {
				end = len(graphData.ValueNodes)
			}

			batch := graphData.ValueNodes[i:end]
			query := "UNWIND $nodes AS node CREATE (n:node.label {id: node.id, name: node.name, package: node.package, line: node.line, column: node.column, value_type: node.value_type, type_name: node.type_name, is_error_type: node.is_error_type})"

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, node := range batch {
				mappableBatch[i] = &node
			}

			params := map[string]interface{}{
				"nodes": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create nodes batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed nodes %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.ValueNodes), float64(end)/float64(len(graphData.ValueNodes))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import value nodes: %v", err)
	}
	log.Printf("Value node import completed in %v", time.Since(valueStartTime))

	// Create index
	log.Println("Creating index on Value node IDs...")
	_, err = session.Run(ctx, "CREATE INDEX ON :Value(id)", nil)
	if err != nil {
		return fmt.Errorf("failed to create index: %v", err)
	}

	// Import ordering edges
	log.Printf("Starting edge import of %d ordering edges...", len(graphData.OrderingEdges))
	edgeStartTime := time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.OrderingEdges); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(graphData.OrderingEdges) {
				end = len(graphData.OrderingEdges)
			}

			batch := graphData.OrderingEdges[i:end]
			query := `
				UNWIND $edges AS edge
				MATCH (from:Instruction {id: edge.from_id}), (to:Instruction {id: edge.to_id})
				CREATE (from)-[:edge.type]->(to)
			`

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, edge := range batch {
				mappableBatch[i] = &edge
			}

			params := map[string]interface{}{
				"edges": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create edges batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed edges %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.OrderingEdges), float64(end)/float64(len(graphData.OrderingEdges))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import ordering edges: %v", err)
	}
	log.Printf("Ordering edge import completed in %v", time.Since(edgeStartTime))

	// Import operand edges
	log.Printf("Starting edge import of %d operand edges...", len(graphData.OperandEdges))
	edgeStartTime = time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.OperandEdges); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(graphData.OperandEdges) {
				end = len(graphData.OperandEdges)
			}

			batch := graphData.OperandEdges[i:end]
			query := `
				UNWIND $edges AS edge
				MATCH (from:Instruction {id: edge.from_id}), (to:Value {id: edge.to_id})
				CREATE (from)-[:edge.type]->(to)
			`

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, edge := range batch {
				mappableBatch[i] = &edge
			}

			params := map[string]interface{}{
				"edges": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create edges batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed edges %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.OperandEdges), float64(end)/float64(len(graphData.OperandEdges))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import operand edges: %v", err)
	}
	log.Printf("Operand edge import completed in %v", time.Since(edgeStartTime))

	// Import result edges
	log.Printf("Starting edge import of %d result edges...", len(graphData.ResultEdges))
	edgeStartTime = time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.ResultEdges); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(graphData.ResultEdges) {
				end = len(graphData.ResultEdges)
			}

			batch := graphData.ResultEdges[i:end]
			query := `
				UNWIND $edges AS edge
				MATCH (from:Instruction {id: edge.from_id}), (to:Value {id: edge.to_id})
				CREATE (from)-[:edge.type]->(to)
			`

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, edge := range batch {
				mappableBatch[i] = &edge
			}

			params := map[string]interface{}{
				"edges": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create edges batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed edges %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.ResultEdges), float64(end)/float64(len(graphData.ResultEdges))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import result edges: %v", err)
	}
	log.Printf("Result edge import completed in %v", time.Since(edgeStartTime))

	// Import control flow edges
	log.Printf("Starting edge import of %d control flow edges...", len(graphData.ControlFlowEdges))
	edgeStartTime = time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.ControlFlowEdges); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(graphData.ControlFlowEdges) {
				end = len(graphData.ControlFlowEdges)
			}

			batch := graphData.ControlFlowEdges[i:end]
			query := `
				UNWIND $edges AS edge
				MATCH (from:Instruction {id: edge.from_id}), (to:Instruction {id: edge.to_id})
				CREATE (from)-[:edge.type {condition: edge.condition}]->(to)
			`

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, edge := range batch {
				mappableBatch[i] = &edge
			}

			params := map[string]interface{}{
				"edges": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create edges batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed edges %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.ControlFlowEdges), float64(end)/float64(len(graphData.ControlFlowEdges))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import control flow edges: %v", err)
	}
	log.Printf("Control flow edge import completed in %v", time.Since(edgeStartTime))

	// Import resolved call edges
	log.Printf("Starting edge import of %d resolved call edges...", len(graphData.ResolvedCallEdges))
	edgeStartTime = time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.ResolvedCallEdges); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize
			if end > len(graphData.ResolvedCallEdges) {
				end = len(graphData.ResolvedCallEdges)
			}

			batch := graphData.ResolvedCallEdges[i:end]
			query := `
				UNWIND $edges AS edge
				MATCH (from:Instruction {id: edge.from_id}), (to:Instruction {id: edge.to_id})
				CREATE (from)-[:edge.type {edge_cardinality: edge.edge_cardinality}]->(to)
			`

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, edge := range batch {
				mappableBatch[i] = &edge
			}

			params := map[string]interface{}{
				"edges": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create edges batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed edges %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.ResolvedCallEdges), float64(end)/float64(len(graphData.ResolvedCallEdges))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import resolved call edges: %v", err)
	}
	log.Printf("Resolved call edge import completed in %v", time.Since(edgeStartTime))

	// Import belongs to edges
	log.Printf("Starting edge import of %d belongs to edges...", len(graphData.BelongsToEdges))
	edgeStartTime = time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for i := 0; i < len(graphData.BelongsToEdges); i += defaultBatchSize {
			batchStartTime := time.Now()
			end := i + defaultBatchSize

			if end > len(graphData.BelongsToEdges) {
				end = len(graphData.BelongsToEdges)
			}

			batch := graphData.BelongsToEdges[i:end]
			query := `
				UNWIND $edges AS edge
				MATCH (from:Instruction {id: edge.from_id}), (to:FileVersion {id: edge.to_id})
				CREATE (from)-[:edge.type]->(to)
			`

			mappableBatch := make([]graphcommon.Mappable, len(batch))
			for i, edge := range batch {
				mappableBatch[i] = &edge
			}

			params := map[string]interface{}{
				"edges": mapify(mappableBatch),
			}

			_, err := tx.Run(ctx, query, params)
			if err != nil {
				return nil, fmt.Errorf("failed to create edges batch %d-%d: %v", i, end, err)
			}

			batchDuration := time.Since(batchStartTime)
			log.Printf("Processed edges %d-%d/%d (%.2f%%) in %v", i, end, len(graphData.BelongsToEdges), float64(end)/float64(len(graphData.BelongsToEdges))*100, batchDuration)
		}
		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to import belongs to edges: %v", err)
	}
	log.Printf("Belongs to edge import completed in %v", time.Since(edgeStartTime))

	log.Printf("Total import completed in %v", time.Since(startTime))
	return nil
}
