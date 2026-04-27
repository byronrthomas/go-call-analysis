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

// Returns the longest slash-terminated prefix shared by all paths.
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

// Bottom-up, no-nesting compound detection.
// A path becomes a compound when it directly parents ≥2 path segments and none
// of its descendants is already a picked compound.
function buildCompoundSet(paths: string[], modulePrefix: string): Set<string> {
  const trie = new Map<string, Set<string>>()

  for (const path of paths) {
    if (!path.startsWith(modulePrefix)) continue
    const segs = path.slice(modulePrefix.length).split('/')
    for (let i = 1; i < segs.length; i++) {
      const parentPath = modulePrefix + segs.slice(0, i).join('/')
      let set = trie.get(parentPath)
      if (!set) { set = new Set(); trie.set(parentPath, set) }
      set.add(segs[i])
    }
  }

  const candidates = Array.from(trie.entries())
    .filter(([, children]) => children.size >= 2)
    .map(([path]) => path)
    .sort((a, b) => {
      const d = b.split('/').length - a.split('/').length
      return d !== 0 ? d : a.localeCompare(b)
    })

  const compounds = new Set<string>()
  for (const candidate of candidates) {
    const prefix = candidate + '/'
    if (![...compounds].some(c => c.startsWith(prefix))) compounds.add(candidate)
  }
  return compounds
}

// Returns the deepest compound ancestor of pkg, or undefined if standalone.
function findParent(pkg: string, modulePrefix: string, compounds: Set<string>): string | undefined {
  if (!pkg.startsWith(modulePrefix)) return undefined
  const segs = pkg.slice(modulePrefix.length).split('/')
  for (let i = segs.length - 1; i >= 1; i--) {
    const candidate = modulePrefix + segs.slice(0, i).join('/')
    if (compounds.has(candidate)) return candidate
  }
  return undefined
}

export function buildElements(raw: string): Cytoscape.ElementDefinition[] {
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
  const compounds = buildCompoundSet(allPaths, modulePrefix)

  // Each package maps to its "group node": its compound ancestor if inside one,
  // otherwise itself. This collapses the 260-node graph to ~60 group/standalone nodes.
  function groupId(pkg: string): string {
    return findParent(pkg, modulePrefix, compounds) ?? pkg
  }

  // Collect unique group node IDs.
  const groupIds = new Set<string>()
  for (const pkg of allPaths) groupIds.add(groupId(pkg))

  // Assign a distinct hue to each auto-detected compound group.
  const compoundList = Array.from(compounds).sort()
  const groupColor = new Map<string, string>()
  compoundList.forEach((c, i) => {
    const hue = Math.round((i * 360) / compoundList.length)
    groupColor.set(c, `hsl(${hue},60%,48%)`)
  })

  // Build nodes: one per group ID.
  const cytoscapeNodes: Cytoscape.ElementDefinition[] = []
  for (const gid of groupIds) {
    const isCompound = compounds.has(gid)
    const label = gid.split('/').pop() ?? gid
    const color = isCompound ? (groupColor.get(gid) ?? '#4a90d9') : '#4a90d9'
    // Tooltip shows the full grouped path and member count.
    const memberCount = allPaths.filter(p => groupId(p) === gid).length
    const fullPath = gid.replace(modulePrefix, '')
    const tooltip = isCompound ? `${fullPath} (${memberCount} packages)` : fullPath
    cytoscapeNodes.push({
      data: { id: gid, label, color, tooltip, isGroup: isCompound },
      classes: isCompound ? 'group-node' : '',
    })
  }

  // Build edges: aggregate to group level, deduplicate, drop self-loops.
  const groupEdgeSet = new Set<string>()
  const cytoscapeEdges: Cytoscape.ElementDefinition[] = []
  for (const { source, target } of edges) {
    const src = groupId(source)
    const tgt = groupId(target)
    if (src === tgt) continue
    const key = `${src}::${tgt}`
    if (!groupEdgeSet.has(key)) {
      groupEdgeSet.add(key)
      cytoscapeEdges.push({
        data: { id: `e${cytoscapeEdges.length}`, source: src, target: tgt },
      })
    }
  }

  return [...cytoscapeNodes, ...cytoscapeEdges]
}
