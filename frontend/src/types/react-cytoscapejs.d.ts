declare module 'react-cytoscapejs' {
  import { CSSProperties, Component } from 'react'
  import Cytoscape from 'cytoscape'

  interface CytoscapeComponentProps {
    elements: Cytoscape.ElementDefinition[]
    layout?: Cytoscape.LayoutOptions
    stylesheet?: Array<{ selector: string; style: Record<string, unknown> }>
    style?: CSSProperties
    cy?: (cy: Cytoscape.Core) => void
  }

  export default class CytoscapeComponent extends Component<CytoscapeComponentProps> {}
}
