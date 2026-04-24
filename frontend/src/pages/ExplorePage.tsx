import { QueryEditor } from '../components/QueryEditor'
import { ResultsTable } from '../components/ResultsTable'
import { useQuery } from '../hooks/useQuery'

export function ExplorePage() {
  const { records, loading, error, execute } = useQuery()

  return (
    <>
      <QueryEditor onExecute={execute} loading={loading} />
      {error && <div className="error">{error}</div>}
      <ResultsTable records={records} />
    </>
  )
}
