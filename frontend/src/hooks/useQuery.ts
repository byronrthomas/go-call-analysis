import { useState, useCallback, useRef } from 'react'
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
  // Incremented each time execute() is called; lets us discard stale results.
  const requestId = useRef(0)

  const execute = useCallback(async (cypher: string) => {
    const id = ++requestId.current
    setState({ records: [], loading: true, error: null })
    try {
      const result = await runQuery(cypher)
      if (id !== requestId.current) return
      setState({ records: result.records, loading: false, error: null })
    } catch (err) {
      if (id !== requestId.current) return
      setState({
        records: [],
        loading: false,
        error: err instanceof Error ? err.message : String(err),
      })
    }
  }, [])

  return { ...state, execute }
}
