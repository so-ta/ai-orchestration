import dagre from 'dagre'
import type { Step, Edge } from '~/types/api'

export interface LayoutOptions {
  direction?: 'TB' | 'BT' | 'LR' | 'RL' // Top-Bottom, Bottom-Top, Left-Right, Right-Left
  nodeWidth?: number
  nodeHeight?: number
  nodeSeparation?: number
  rankSeparation?: number
}

export interface LayoutResult {
  stepId: string
  x: number
  y: number
}

const DEFAULT_OPTIONS: Required<LayoutOptions> = {
  direction: 'LR', // Left-to-Right layout
  nodeWidth: 180,
  nodeHeight: 60,
  nodeSeparation: 60,
  rankSeparation: 120,
}

/**
 * Calculate optimized positions for all steps in a DAG using dagre layout algorithm
 */
export function calculateLayout(
  steps: Step[],
  edges: Edge[],
  options: LayoutOptions = {}
): LayoutResult[] {
  const opts = { ...DEFAULT_OPTIONS, ...options }

  // Create a new directed graph
  const g = new dagre.graphlib.Graph()

  // Set graph options
  g.setGraph({
    rankdir: opts.direction,
    nodesep: opts.nodeSeparation,
    ranksep: opts.rankSeparation,
    marginx: 50,
    marginy: 50,
  })

  // Default edge label (required by dagre)
  g.setDefaultEdgeLabel(() => ({}))

  // Add nodes
  for (const step of steps) {
    g.setNode(step.id, {
      width: opts.nodeWidth,
      height: opts.nodeHeight,
    })
  }

  // Add edges
  for (const edge of edges) {
    g.setEdge(edge.source_step_id, edge.target_step_id)
  }

  // Calculate layout
  dagre.layout(g)

  // Extract positions
  const results: LayoutResult[] = []
  for (const step of steps) {
    const node = g.node(step.id)
    if (node) {
      results.push({
        stepId: step.id,
        // dagre returns center position, adjust to top-left
        x: Math.round(node.x - opts.nodeWidth / 2),
        y: Math.round(node.y - opts.nodeHeight / 2),
      })
    }
  }

  return results
}

/**
 * Find the Start node in a list of steps
 */
export function findStartNode(steps: Step[]): Step | undefined {
  return steps.find(step => step.type === 'start')
}

/**
 * Check if a step is a Start node
 */
export function isStartNode(step: Step): boolean {
  return step.type === 'start'
}
