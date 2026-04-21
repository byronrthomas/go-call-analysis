import { useState } from 'react'

interface Props {
  onExecute: (cypher: string) => void
  loading: boolean
}

const DEFAULT_QUERY = 'MATCH (n) RETURN n LIMIT 25'

export function QueryEditor({ onExecute, loading }: Props) {
  const [value, setValue] = useState(DEFAULT_QUERY)

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault()
      if (!loading) onExecute(value)
    }
  }

  return (
    <div className="query-editor">
      <textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        rows={4}
        spellCheck={false}
        placeholder="Enter a Cypher query…"
      />
      <button onClick={() => onExecute(value)} disabled={loading}>
        {loading ? 'Running…' : 'Run (Ctrl+Enter)'}
      </button>
    </div>
  )
}
