import type Cytoscape from 'cytoscape'

interface PathNode {
  id: number
  properties: { package_path: string }
}

interface PathRelationship {
  start: number
  end: number
}

interface PathLine {
  p: {
    nodes: PathNode[]
    relationships: PathRelationship[]
  }
}

export function buildElements(raw: string): Cytoscape.ElementDefinition[] {
  const nodeIdToPackage = new Map<number, string>()
  const packages = new Set<string>()
  const edgeSet = new Set<string>()
  const edges: Array<{ source: string; target: string; id: string }> = []

  for (const line of raw.trim().split('\n')) {
    if (!line.trim()) continue
    const { p }: PathLine = JSON.parse(line)

    for (const node of p.nodes) {
      nodeIdToPackage.set(node.id, node.properties.package_path)
      packages.add(node.properties.package_path)
    }

    for (const rel of p.relationships) {
      const source = nodeIdToPackage.get(rel.start)
      const target = nodeIdToPackage.get(rel.end)
      if (source && target && source !== target) {
        const key = `${source}::${target}`
        if (!edgeSet.has(key)) {
          edgeSet.add(key)
          edges.push({ source, target, id: `e${edges.length}` })
        }
      }
    }
  }

  const cytoscapeNodes = Array.from(packages).map(pkg => ({
    data: { id: pkg, label: pkg.split('/').pop() ?? pkg },
  }))

  const cytoscapeEdges = edges.map(e => ({
    data: { id: e.id, source: e.source, target: e.target },
  }))

  return [...cytoscapeNodes, ...cytoscapeEdges]
}
