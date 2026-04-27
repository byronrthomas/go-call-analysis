import { useCallback, useMemo, useRef } from 'react'
import CytoscapeComponent from 'react-cytoscapejs'
import Cytoscape from 'cytoscape'
import dagre from 'cytoscape-dagre'
import rawData from '../../test_resources/sample-package-deps-paths.jsonl?raw'
import { buildElements } from './graphData'

Cytoscape.use(dagre)

const layout = {
  name: 'dagre',
  rankDir: 'TB',
  nodeSep: 6,
  rankSep: 60,
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
      'background-color': 'data(color)',
      color: '#fff',
      'font-size': 12,
      'min-zoomed-font-size': 7,  // labels vanish below this screen-px size
      padding: '4px',
      shape: 'roundrectangle',
    },
  },
  {
    // Group nodes: a package subtree auto-detected by trie and collapsed to one unit.
    selector: 'node.group-node',
    style: {
      'border-width': 2,
      'border-color': '#fff',
      'border-opacity': 0.6,
      'font-weight': 'bold',
    },
  },
  {
    selector: 'edge',
    style: {
      'curve-style': 'bezier',
      'target-arrow-shape': 'triangle',
      'line-color': '#555',
      'target-arrow-color': '#555',
      width: 1,
      opacity: 0.5,
    },
  },
]

export function HomePage() {
  const elements = useMemo(() => buildElements(rawData), [])
  const cyRef = useRef<Cytoscape.Core | null>(null)

  const handleCy = useCallback((cy: Cytoscape.Core) => {
    if (cyRef.current === cy) return
    cyRef.current = cy
    // Fit the whole graph; labels appear automatically as the user zooms in
    // (min-zoomed-font-size controls the threshold).
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
