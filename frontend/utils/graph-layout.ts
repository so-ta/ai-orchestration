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
  rankSeparation: 40, // Reduced from 120 to 40 (1/3)
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

  // Grid size for snapping (must match Vue Flow's snap-grid)
  const GRID_SIZE = 20

  // Set graph options (margins must be divisible by grid size)
  g.setGraph({
    rankdir: opts.direction,
    nodesep: opts.nodeSeparation,
    ranksep: opts.rankSeparation,
    marginx: 40, // Changed from 50 to 40 (divisible by 20)
    marginy: 40, // Changed from 50 to 40 (divisible by 20)
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

  // Helper function to snap value to grid
  const snapToGrid = (value: number): number => {
    return Math.round(value / GRID_SIZE) * GRID_SIZE
  }

  // Extract positions and snap to grid
  const results: LayoutResult[] = []
  for (const step of steps) {
    const node = g.node(step.id)
    if (node) {
      // dagre returns center position, adjust to top-left and snap to grid
      const rawX = node.x - opts.nodeWidth / 2
      const rawY = node.y - opts.nodeHeight / 2
      results.push({
        stepId: step.id,
        x: snapToGrid(rawX),
        y: snapToGrid(rawY),
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
