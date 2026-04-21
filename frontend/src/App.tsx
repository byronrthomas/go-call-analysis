import { QueryEditor } from './components/QueryEditor'
import { ResultsTable } from './components/ResultsTable'
import { useQuery } from './hooks/useQuery'
import './App.css'

export default function App() {
  const { records, loading, error, execute } = useQuery()

  return (
    <div className="app">
      <header>
        <h1>Go Call Analysis</h1>
      </header>
      <main>
        <QueryEditor onExecute={execute} loading={loading} />
        {error && <div className="error">{error}</div>}
        <ResultsTable records={records} />
      </main>
    </div>
  )
}
