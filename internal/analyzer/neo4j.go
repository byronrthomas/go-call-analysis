package analyzer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/throwin5tone7/go-call-analysis/internal/graphcommon"
)

const (
	defaultBatchSize = 10000
)

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
	// Create driver
	driver, err := neo4j.NewDriverWithContext(config.URI, neo4j.BasicAuth(config.Username, config.Password, ""))
	if err != nil {
		return fmt.Errorf("failed to create Neo4j driver: %v", err)
	}
	ctx := context.Background()
	defer driver.Close(ctx)

	// Create session
	session := driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: config.Database,
	})
	defer session.Close(ctx)

	// Start timing
	startTime := time.Now()

	// Import nodes
	log.Printf("Starting node import of %d nodes...", len(nodes))
	nodeStartTime := time.Now()
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
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
