import { Record as Neo4jRecord } from 'neo4j-driver'

interface Props {
  records: Neo4jRecord[]
}

function cellValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object' && 'properties' in (value as object)) {
    return JSON.stringify((value as { properties: unknown }).properties)
  }
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
