import { useMemo } from 'react'
import CytoscapeComponent from 'react-cytoscapejs'
import Cytoscape from 'cytoscape'
import dagre from 'cytoscape-dagre'
import rawData from '../../test_resources/sample-package-deps-paths.jsonl?raw'
import { buildElements } from './graphData'

Cytoscape.use(dagre)

const layout = {
  name: 'dagre',
  rankDir: 'LR',
  nodeSep: 40,
  rankSep: 120,
  padding: 20,
  animate: false,
} as Cytoscape.LayoutOptions

const stylesheet: Array<{ selector: string; style: Record<string, unknown> }> = [
  {
    selector: 'node',
    style: {
      label: 'data(label)',
      'text-valign': 'center',
      'text-halign': 'center',
      'background-color': '#4a90d9',
      color: '#fff',
      'font-size': 10,
      padding: '6px',
      shape: 'roundrectangle',
    },
  },
  {
    selector: 'edge',
    style: {
      'curve-style': 'bezier',
      'target-arrow-shape': 'triangle',
      'line-color': '#aaa',
      'target-arrow-color': '#aaa',
      width: 1,
    },
  },
]

export function HomePage() {
  const elements = useMemo(() => buildElements(rawData), [])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <h2>Package Dependency Graph</h2>
      <div data-testid="package-graph-container" style={{ width: '100%', height: '80vh' }}>
        <CytoscapeComponent
          elements={elements}
          layout={layout}
          stylesheet={stylesheet}
          style={{ width: '100%', height: '100%' }}
        />
      </div>
    </div>
  )
}
