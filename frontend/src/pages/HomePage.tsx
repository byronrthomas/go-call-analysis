import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import CytoscapeComponent from 'react-cytoscapejs'
import Cytoscape from 'cytoscape'
import dagre from 'cytoscape-dagre'
import rawData from '../../test_resources/sample-package-deps-paths.jsonl?raw'
import { buildElements } from './graphData'

Cytoscape.use(dagre)

const dagreOptions = {
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
      'min-zoomed-font-size': 7,
      padding: '4px',
      shape: 'roundrectangle',
    },
  },
  {
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
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(() => new Set())
  const elements = useMemo(() => buildElements(rawData, expandedGroups), [expandedGroups])
  const cyRef = useRef<Cytoscape.Core | null>(null)
  const isInitialMount = useRef(true)

  // Re-run layout whenever elements change (skip the first render — layout prop handles it).
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false
      return
    }
    const cy = cyRef.current
    if (!cy) return
    cy.layout(dagreOptions).run()
    cy.fit(undefined, 30)
  }, [elements])

  const handleNodeClick = useCallback((event: Cytoscape.EventObject) => {
    const node = event.target as Cytoscape.NodeSingular
    if (!node.data('isGroup')) return
    const id = node.data('id') as string
    setExpandedGroups(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        // Collapse this group and all expanded descendants.
        for (const g of [...next]) {
          if (g === id || g.startsWith(id + '/')) next.delete(g)
        }
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  const handleCy = useCallback((cy: Cytoscape.Core) => {
    if (cyRef.current === cy) return
    cyRef.current = cy
    cy.fit(undefined, 30)
    cy.on('tap', 'node', handleNodeClick)
  }, [handleNodeClick])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <h2>Package Dependency Graph</h2>
      <p style={{ margin: '0 0 8px', fontSize: 13, color: '#888' }}>
        Click a bold group node to expand · click again to collapse
      </p>
      <div data-testid="package-graph-container" style={{ width: '100%', height: '80vh' }}>
        <CytoscapeComponent
          elements={elements}
          layout={dagreOptions}
          stylesheet={stylesheet}
          style={{ width: '100%', height: '100%' }}
          cy={handleCy}
        />
      </div>
    </div>
  )
}
