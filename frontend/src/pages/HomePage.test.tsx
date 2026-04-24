import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { HomePage } from './HomePage'

vi.mock('react-cytoscapejs', () => ({
  default: ({ elements }: { elements: Array<{ data: Record<string, unknown> }> }) => (
    <div
      data-testid="cytoscape-graph"
      data-element-count={String(elements.length)}
    />
  ),
}))

describe('HomePage graph view', () => {
  it('renders a cytoscape graph', () => {
    render(<HomePage />)
    expect(screen.getByTestId('cytoscape-graph')).toBeInTheDocument()
  })

  it('populates the graph with package nodes and dependency edges from sample data', () => {
    render(<HomePage />)
    const graph = screen.getByTestId('cytoscape-graph')
    const count = Number(graph.dataset.elementCount)
    // Sample data contains 1595 paths - expect many unique nodes and edges
    expect(count).toBeGreaterThan(100)
  })
})
