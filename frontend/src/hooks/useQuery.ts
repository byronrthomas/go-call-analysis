import { useState, useCallback } from 'react'
import { Record as Neo4jRecord } from 'neo4j-driver'
import { runQuery } from '../lib/neo4j'

interface QueryState {
  records: Neo4jRecord[]
  loading: boolean
  error: string | null
}

export function useQuery() {
  const [state, setState] = useState<QueryState>({
    records: [],
    loading: false,
    error: null,
  })

  const execute = useCallback(async (cypher: string) => {
    setState({ records: [], loading: true, error: null })
    try {
      const result = await runQuery(cypher)
      setState({ records: result.records, loading: false, error: null })
    } catch (err) {
      setState({
        records: [],
        loading: false,
        error: err instanceof Error ? err.message : String(err),
      })
    }
  }, [])

  return { ...state, execute }
}
