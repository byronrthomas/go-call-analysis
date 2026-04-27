import { describe, expect, it } from 'vitest'
import { buildElements } from './graphData'
import rawData from '../../test_resources/sample-package-deps-paths.jsonl?raw'

const elements = buildElements(rawData)
const nodes = elements.filter(e => !e.data.source)
const edges = elements.filter(e => e.data.source)

describe('buildElements', () => {
  it('produces at least one node and one edge', () => {
    expect(nodes.length).toBeGreaterThan(0)
    expect(edges.length).toBeGreaterThan(0)
  })

  it('assigns a unique id to every node', () => {
    const ids = nodes.map(n => n.data.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('assigns a unique id to every edge', () => {
    const ids = edges.map(e => e.data.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('excludes self-loop edges', () => {
    expect(edges.every(e => e.data.source !== e.data.target)).toBe(true)
  })

  it('produces no duplicate edges', () => {
    const keys = edges.map(e => `${e.data.source}::${e.data.target}`)
    expect(new Set(keys).size).toBe(keys.length)
  })

  it('all edge endpoints reference existing nodes', () => {
    const nodeIds = new Set(nodes.map(n => n.data.id))
    for (const edge of edges) {
      expect(nodeIds.has(edge.data.source)).toBe(true)
      expect(nodeIds.has(edge.data.target)).toBe(true)
    }
  })

  it('sets node label to the last path segment', () => {
    for (const node of nodes) {
      const id = node.data.id as string
      expect(node.data.label).toBe(id.split('/').pop())
    }
  })
})
