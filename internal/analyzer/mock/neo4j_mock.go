package mock

import (
	"context"
	"fmt"
	"log"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type QueryExecutionCapture struct {
	Query          string
	ParamMapValues []map[string]any
}

type MockSession struct {
	neo4j.SessionWithContext
	capturedQueries []QueryExecutionCapture
}

func (ms *MockSession) BeginTransaction(ctx context.Context, configurers ...func(*neo4j.TransactionConfig)) (neo4j.ExplicitTransaction, error) {
	return nil, nil
}
func (ms *MockSession) ReadTransaction(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (any, error) {
	return nil, nil
}

func (ms *MockSession) WriteTransaction(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (any, error) {
	return nil, nil
}

func (ms *MockSession) ExecuteRead(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (any, error) {
	return nil, nil
}

func (ms *MockSession) ExecuteWrite(ctx context.Context, work neo4j.ManagedTransactionWork, configurers ...func(*neo4j.TransactionConfig)) (any, error) {
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

		excludedFromSort := []string{"id", "from_id", "to_id"}

		// Format parameters if any exist
		if len(capture.ParamMapValues) > 0 {
			// Get sorted keys from the first parameter map (assuming all maps have same keys)
			var keys []string
			for key := range capture.ParamMapValues[0] {
				if slices.Contains(excludedFromSort, key) {
					continue
				}
				keys = append(keys, key)
			}
			sort.Strings(keys)

			// Write header line with map keys
			sb.WriteString("Parameters:\n")
			if capture.ParamMapValues[0]["id"] != nil {
				if capture.ParamMapValues[0]["from_id"] != nil || capture.ParamMapValues[0]["to_id"] != nil {
					log.Fatalf("Do not support mixed id, from_id, and to_id in param values for output - keys: %s", strings.Join(slices.Collect(maps.Keys(capture.ParamMapValues[0])), ", "))
				}
				keys = slices.Insert(keys, 0, "id")
			} else if capture.ParamMapValues[0]["from_id"] != nil && capture.ParamMapValues[0]["to_id"] != nil {
				keys = slices.Insert(keys, 0, "from_id")
				keys = slices.Insert(keys, 1, "to_id")
			} else {
				log.Fatalf("Do not support mixed id, from_id, and to_id in param values for output - keys: %s", strings.Join(slices.Collect(maps.Keys(capture.ParamMapValues[0])), ", "))
			}

			for j, key := range keys {
				if j > 0 {
					sb.WriteString("\t")
				}
				sb.WriteString(key)
			}
			sb.WriteString("\n")

			mapValues := make([]string, len(capture.ParamMapValues))
			// Write each parameter map as a row
			for pIndex, paramMap := range capture.ParamMapValues {
				innerSb := strings.Builder{}
				for j, key := range keys {
					if j > 0 {
						innerSb.WriteString("\t")
					}
					innerSb.WriteString(fmt.Sprintf("%v", paramMap[key]))
				}
				mapValues[pIndex] = innerSb.String()
			}
			sort.Strings(mapValues)
			for _, mapValue := range mapValues {
				sb.WriteString(mapValue)
				sb.WriteString("\n")
			}
		}
	}
}

func extractParamMapValues(params map[string]any) []map[string]any {
	if len(params) > 1 {
		log.Fatalf("Cannot handle more than one parameter in mock session")
	}
	capturedParams := make([]map[string]any, 0)
	for _, v := range params {
		// The parameter value is an array of maps (e.g., "nodes": []map[string]any)
		if mapArray, ok := v.([]map[string]any); ok {
			capturedParams = append(capturedParams, mapArray...)
		} else {
			// Fallback for single map parameters
			capturedParams = append(capturedParams, v.(map[string]any))
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
