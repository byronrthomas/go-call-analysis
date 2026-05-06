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

function findModulePrefix(paths: string[]): string {
  if (paths.length === 0) return ''
  const sorted = [...paths].sort()
  const first = sorted[0]
  const last = sorted[sorted.length - 1]
  let i = 0
  while (i < first.length && i < last.length && first[i] === last[i]) i++
  const common = first.slice(0, i)
  const lastSlash = common.lastIndexOf('/')
  return lastSlash >= 0 ? common.slice(0, lastSlash + 1) : ''
}

// All intermediate trie paths with ≥2 direct children become groups (nesting allowed).
function buildGroupSet(paths: string[], modulePrefix: string): Set<string> {
  const childCount = new Map<string, Set<string>>()

  for (const path of paths) {
    if (!path.startsWith(modulePrefix)) continue
    const segs = path.slice(modulePrefix.length).split('/')
    for (let i = 1; i < segs.length; i++) {
      const parentPath = modulePrefix + segs.slice(0, i).join('/')
      let set = childCount.get(parentPath)
      if (!set) { set = new Set(); childCount.set(parentPath, set) }
      set.add(segs[i])
    }
  }

  const groups = new Set<string>()
  for (const [path, children] of childCount) {
    if (children.size >= 2) groups.add(path)
  }
  return groups
}

// Returns the outermost collapsed group ancestor for `path`, or `path` itself
// if every ancestor group is expanded.
function visibleRep(path: string, modulePrefix: string, groups: Set<string>, expanded: Set<string>): string {
  if (!path.startsWith(modulePrefix)) return path
  const segs = path.slice(modulePrefix.length).split('/')
  for (let i = 1; i < segs.length; i++) {
    const ancestor = modulePrefix + segs.slice(0, i).join('/')
    if (groups.has(ancestor) && !expanded.has(ancestor)) return ancestor
  }
  return path
}

export function buildElements(raw: string, expanded: Set<string> = new Set()): Cytoscape.ElementDefinition[] {
  const nodeIdToPackage = new Map<number, string>()
  const packages = new Set<string>()
  const edgeSet = new Set<string>()
  const edges: Array<{ source: string; target: string }> = []

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
          edges.push({ source, target })
        }
      }
    }
  }

  const allPaths = Array.from(packages)
  const modulePrefix = findModulePrefix(allPaths)
  const groups = buildGroupSet(allPaths, modulePrefix)

  const groupList = Array.from(groups).sort()
  const groupColor = new Map<string, string>()
  groupList.forEach((g, i) => {
    const hue = Math.round((i * 360) / groupList.length)
    groupColor.set(g, `hsl(${hue},60%,48%)`)
  })

  const rep = (pkg: string) => visibleRep(pkg, modulePrefix, groups, expanded)

  // Collect the set of visible node IDs.
  const visibleIds = new Set<string>()
  for (const pkg of allPaths) visibleIds.add(rep(pkg))

  const cytoscapeNodes: Cytoscape.ElementDefinition[] = []
  for (const vid of visibleIds) {
    const isGroup = groups.has(vid)
    const label = vid.split('/').pop() ?? vid
    const color = isGroup ? (groupColor.get(vid) ?? '#4a90d9') : '#4a90d9'
    const memberCount = allPaths.filter(p => rep(p) === vid).length
    const fullPath = vid.replace(modulePrefix, '')
    const tooltip = isGroup ? `${fullPath} (${memberCount} packages)` : fullPath
    cytoscapeNodes.push({
      data: { id: vid, label, color, tooltip, isGroup },
      classes: isGroup ? 'group-node' : '',
    })
  }

  const visEdgeSet = new Set<string>()
  const cytoscapeEdges: Cytoscape.ElementDefinition[] = []
  for (const { source, target } of edges) {
    const src = rep(source)
    const tgt = rep(target)
    if (src === tgt) continue
    const key = `${src}::${tgt}`
    if (!visEdgeSet.has(key)) {
      visEdgeSet.add(key)
      cytoscapeEdges.push({
        data: { id: `e${cytoscapeEdges.length}`, source: src, target: tgt },
      })
    }
  }

  return [...cytoscapeNodes, ...cytoscapeEdges]
}
