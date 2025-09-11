package mock

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type QueryExecutionCapture struct {
	Query          string
	ParamMapValues []map[string]interface{}
}

type MockSession struct {
	neo4j.SessionWithContext
	capturedQueries []QueryExecutionCapture
}

func (ms *MockSession) BeginTransaction(ctx context.Context, configurers ...func(*neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error) {
	return nil, nil
}
func (ms *MockSession) ReadTransaction(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (interface{}, error) {
	return nil, nil
}

func (ms *MockSession) WriteTransaction(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (interface{}, error) {
	return nil, nil
}

func (ms *MockSession) ExecuteRead(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (interface{}, error) {
	return nil, nil
}

func (ms *MockSession) ExecuteWrite(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (interface{}, error) {
	tx := &MockTransaction{MySession: ms}
	return work(tx)
}

func (ms *MockSession) Close(ctx context.Context) error {
	return nil
}

func (ms *MockSession) LastBookmarks() []string {
	return nil
}

func (ms *MockSession) Run(ctx context.Context, query string, params map[string]any, configurers ...func(*neo4j.TransactionConfig)) (neo4j.ResultWithContext, error) {
	ms.capturedQueries = append(ms.capturedQueries, QueryExecutionCapture{Query: query, ParamMapValues: extractParamMapValues(params)})
	return nil, nil
}

// FormatCapturedQueries appends formatted details of each captured query to the provided string builder
func (ms *MockSession) FormatCapturedQueries(sb *strings.Builder) {
	for i, capture := range ms.capturedQueries {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		// Write the query
		sb.WriteString("Query: ")
		sb.WriteString(capture.Query)
		sb.WriteString("\n")

		// Format parameters if any exist
		if len(capture.ParamMapValues) > 0 {
			// Get sorted keys from the first parameter map (assuming all maps have same keys)
			var keys []string
			for key := range capture.ParamMapValues[0] {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			// Write header line with map keys
			sb.WriteString("Parameters:\n")
			for j, key := range keys {
				if j > 0 {
					sb.WriteString("\t")
				}
				sb.WriteString(key)
			}
			sb.WriteString("\n")

			// Write each parameter map as a row
			for _, paramMap := range capture.ParamMapValues {
				for j, key := range keys {
					if j > 0 {
						sb.WriteString("\t")
					}
					sb.WriteString(fmt.Sprintf("%v", paramMap[key]))
				}
				sb.WriteString("\n")
			}
		}
	}
}

func extractParamMapValues(params map[string]any) []map[string]interface{} {
	if len(params) > 1 {
		log.Fatalf("Cannot handle more than one parameter in mock session")
	}
	capturedParams := make([]map[string]interface{}, 0)
	for _, v := range params {
		// The parameter value is an array of maps (e.g., "nodes": []map[string]any)
		if mapArray, ok := v.([]map[string]any); ok {
			for _, m := range mapArray {
				capturedParams = append(capturedParams, m)
			}
		} else {
			// Fallback for single map parameters
			capturedParams = append(capturedParams, v.(map[string]interface{}))
		}
	}
	return capturedParams
}

type MockTransaction struct {
	neo4j.ManagedTransaction
	MySession *MockSession
}

func (mt *MockTransaction) Run(ctx context.Context, query string, params map[string]any) (neo4j.ResultWithContext, error) {
	mt.MySession.capturedQueries = append(mt.MySession.capturedQueries, QueryExecutionCapture{Query: query, ParamMapValues: extractParamMapValues(params)})
	return nil, nil
}
