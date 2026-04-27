import { useCallback, useMemo, useRef } from 'react'
import CytoscapeComponent from 'react-cytoscapejs'
import Cytoscape from 'cytoscape'
import dagre from 'cytoscape-dagre'
import rawData from '../../test_resources/sample-package-deps-paths.jsonl?raw'
import { buildElements } from './graphData'

Cytoscape.use(dagre)

const layout = {
  name: 'dagre',
  rankDir: 'LR',
  nodeSep: 15,
  rankSep: 100,
  padding: 20,
  animate: false,
  fit: false,
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
  const cyRef = useRef<Cytoscape.Core | null>(null)

  // cy prop fires after layout has run (animate:false means layout is synchronous).
  // Fit to show all nodes, but enforce a minimum zoom so labels remain readable.
  const handleCy = useCallback((cy: Cytoscape.Core) => {
    if (cyRef.current === cy) return
    cyRef.current = cy
    cy.fit(undefined, 30)
  }, [])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <h2>Package Dependency Graph</h2>
      <div data-testid="package-graph-container" style={{ width: '100%', height: '80vh' }}>
        <CytoscapeComponent
          elements={elements}
          layout={layout}
          stylesheet={stylesheet}
          style={{ width: '100%', height: '100%' }}
          cy={handleCy}
        />
      </div>
    </div>
  )
}
