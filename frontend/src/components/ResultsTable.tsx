import neo4j, { Record as Neo4jRecord, Node, Relationship, Path, Integer } from 'neo4j-driver'

interface Props {
  records: Neo4jRecord[]
}

function cellValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (neo4j.isInt(value as Integer)) return (value as Integer).toString()
  if (value instanceof Node || value instanceof Relationship) {
    return JSON.stringify((value as Node | Relationship).properties)
  }
  if (value instanceof Path) return `Path(${(value as Path).length} segments)`
  if (Array.isArray(value)) return value.map((v) => cellValue(v)).join(', ')
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

export function ResultsTable({ records }: Props) {
  if (records.length === 0) return null

  const keys = records[0].keys as string[]

  return (
    <div className="results-table-wrapper">
      <table className="results-table">
        <thead>
          <tr>
            {keys.map((k) => (
              <th key={k}>{k}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {records.map((row, i) => (
            <tr key={i}>
              {keys.map((k) => (
                <td key={k}>{cellValue(row.get(k))}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
