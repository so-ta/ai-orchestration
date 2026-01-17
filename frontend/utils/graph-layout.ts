import dagre from 'dagre'
import type { Step, Edge, BlockGroup, OutputPort, StepType, BlockGroupType } from '~/types/api'

/**
 * Prefix used for group node IDs in Vue Flow to distinguish from step node IDs.
 * Group node IDs are formatted as `group_${uuid}` while step node IDs are just `${uuid}`.
 */
export const GROUP_NODE_PREFIX = 'group_'

/**
 * Parse a Vue Flow node ID to determine if it's a group or step.
 * @param nodeId The node ID from Vue Flow (e.g., "group_abc123" or "abc123")
 * @returns Object with isGroup flag and the actual ID (without prefix)
 */
export function parseNodeId(nodeId: string): { isGroup: boolean; id: string } {
  if (nodeId.startsWith(GROUP_NODE_PREFIX)) {
    return { isGroup: true, id: nodeId.slice(GROUP_NODE_PREFIX.length) }
  }
  return { isGroup: false, id: nodeId }
}

export interface LayoutOptions {
  direction?: 'TB' | 'BT' | 'LR' | 'RL' // Top-Bottom, Bottom-Top, Left-Right, Right-Left
  nodeWidth?: number
  nodeHeight?: number
  nodeSeparation?: number
  rankSeparation?: number
}

/**
 * Extended layout options with port ordering information
 */
export interface LayoutOptionsWithPorts extends LayoutOptions {
  // Function to get output ports for a step type
  // Returns ports in order (top to bottom)
  getOutputPorts?: (stepType: StepType, step?: Step) => OutputPort[]
  // Function to get output ports for a block group type
  // Returns ports in order (top to bottom)
  getGroupOutputPorts?: (groupType: BlockGroupType) => OutputPort[]
}

export interface LayoutResult {
  stepId: string
  x: number
  y: number
}

export interface GroupLayoutResult {
  groupId: string
  x: number
  y: number
  width: number
  height: number
}

const DEFAULT_OPTIONS: Required<LayoutOptions> = {
  direction: 'LR', // Left-to-Right layout
  nodeWidth: 80,   // Miro-style: 48px icon + label (~72px) + margin
  nodeHeight: 70,  // Miro-style: 48px icon + 2px gap + ~14px label + margin
  nodeSeparation: 40,
  rankSeparation: 40,
}

/**
 * Calculate optimized positions for all steps in a DAG using dagre layout algorithm
 */
export function calculateLayout(
  steps: Step[],
  edges: Edge[],
  options: LayoutOptionsWithPorts = {}
): LayoutResult[] {
  const opts = { ...DEFAULT_OPTIONS, ...options }
  const getOutputPorts = options.getOutputPorts

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

  // Add edges (only step-to-step edges)
  for (const edge of edges) {
    if (edge.source_step_id && edge.target_step_id) {
      g.setEdge(edge.source_step_id, edge.target_step_id)
    }
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

  // Apply port-based Y ordering if getOutputPorts is provided
  if (getOutputPorts) {
    adjustYBySourcePort(results, steps, edges, getOutputPorts, opts.nodeSeparation, GRID_SIZE)
  }

  return results
}

/**
 * Adjust Y positions of target nodes based on source port order.
 * When a source node has multiple output ports (e.g., out, out2, error),
 * target nodes are positioned vertically in the order of the ports.
 */
function adjustYBySourcePort(
  results: LayoutResult[],
  steps: Step[],
  edges: Edge[],
  getOutputPorts: (stepType: StepType, step?: Step) => OutputPort[],
  nodeSeparation: number,
  gridSize: number
): void {
  const snapToGrid = (value: number): number => {
    return Math.round(value / gridSize) * gridSize
  }

  // Build step map for quick lookup
  const stepMap = new Map<string, Step>()
  for (const step of steps) {
    stepMap.set(step.id, step)
  }

  // Build result map for quick lookup and modification
  const resultMap = new Map<string, LayoutResult>()
  for (const result of results) {
    resultMap.set(result.stepId, result)
  }

  // Group edges by source step
  const edgesBySource = new Map<string, Edge[]>()
  for (const edge of edges) {
    if (edge.source_step_id && edge.target_step_id) {
      const existing = edgesBySource.get(edge.source_step_id) || []
      existing.push(edge)
      edgesBySource.set(edge.source_step_id, existing)
    }
  }

  // For each source step with multiple outgoing edges to different targets
  for (const [sourceId, sourceEdges] of edgesBySource) {
    if (sourceEdges.length <= 1) continue

    const sourceStep = stepMap.get(sourceId)
    if (!sourceStep) continue

    // Get output ports for the source step
    const outputPorts = getOutputPorts(sourceStep.type, sourceStep)
    if (outputPorts.length <= 1) continue

    // Create port order map (port name -> order index)
    const portOrder = new Map<string, number>()
    outputPorts.forEach((port, index) => {
      portOrder.set(port.name, index)
    })

    // Get unique target step IDs with their port info
    // Group by target step to handle multiple edges to same target
    const targetsByPort = new Map<string, { targetId: string; portIndex: number }[]>()

    for (const edge of sourceEdges) {
      if (!edge.target_step_id || !edge.source_port) continue

      // source_port is required - skip edges without it
      const portName = edge.source_port
      const portIndex = portOrder.get(portName) ?? 999 // Unknown ports go last

      const existing = targetsByPort.get(portName) || []
      // Avoid duplicates
      if (!existing.some(t => t.targetId === edge.target_step_id)) {
        existing.push({ targetId: edge.target_step_id, portIndex })
      }
      targetsByPort.set(portName, existing)
    }

    // Collect all targets with their port indices
    const allTargets: Array<{ targetId: string; portIndex: number }> = []
    for (const targets of targetsByPort.values()) {
      allTargets.push(...targets)
    }

    // Sort targets by port index
    allTargets.sort((a, b) => a.portIndex - b.portIndex)

    // Get unique target IDs in sorted order
    const uniqueTargetIds: string[] = []
    const seen = new Set<string>()
    for (const target of allTargets) {
      if (!seen.has(target.targetId)) {
        uniqueTargetIds.push(target.targetId)
        seen.add(target.targetId)
      }
    }

    if (uniqueTargetIds.length <= 1) continue

    // Calculate the center Y of all target nodes
    const targetResults = uniqueTargetIds.map(id => resultMap.get(id)).filter((r): r is LayoutResult => r !== undefined)
    if (targetResults.length <= 1) continue

    // Calculate center of current positions
    const sumY = targetResults.reduce((sum, r) => sum + r.y, 0)
    const centerY = sumY / targetResults.length

    // Calculate new positions centered around the current center
    const nodeHeight = DEFAULT_OPTIONS.nodeHeight
    const totalHeight = (targetResults.length - 1) * (nodeHeight + nodeSeparation)
    const startY = centerY - totalHeight / 2

    // Assign new Y positions in port order
    uniqueTargetIds.forEach((targetId, index) => {
      const result = resultMap.get(targetId)
      if (result) {
        result.y = snapToGrid(startY + index * (nodeHeight + nodeSeparation))
      }
    })
  }
}

/**
 * Adjust Y positions of target nodes based on block group source port order.
 * Similar to adjustYBySourcePort but handles edges originating from block groups.
 */
/**
 * Adjust Y positions of all targets (both steps and groups) from block groups based on source port order.
 * This unified function handles both step and group targets together to ensure consistent ordering
 * when a group outputs to a mix of steps and groups.
 * Also updates the positions of steps inside moved groups.
 */
function adjustYByGroupSourcePortUnified(
  stepResults: LayoutResult[],
  groupResults: GroupLayoutResult[],
  edges: Edge[],
  groupMap: Map<string, BlockGroup>,
  steps: Step[],
  getGroupOutputPorts: (groupType: BlockGroupType) => OutputPort[],
  nodeSeparation: number,
  nodeHeight: number,
  gridSize: number
): void {
  const snapToGrid = (value: number): number => {
    return Math.round(value / gridSize) * gridSize
  }

  // Build result maps for quick lookup and modification
  const stepResultMap = new Map<string, LayoutResult>()
  for (const result of stepResults) {
    stepResultMap.set(result.stepId, result)
  }

  const groupResultMap = new Map<string, GroupLayoutResult>()
  for (const result of groupResults) {
    groupResultMap.set(result.groupId, result)
  }

  // Build map of steps by their group ID
  const stepsByGroupId = new Map<string, Step[]>()
  for (const step of steps) {
    if (step.block_group_id) {
      const existing = stepsByGroupId.get(step.block_group_id) || []
      existing.push(step)
      stepsByGroupId.set(step.block_group_id, existing)
    }
  }

  // Group edges by source block group (both step and group targets)
  const edgesBySourceGroup = new Map<string, Edge[]>()
  for (const edge of edges) {
    if (edge.source_block_group_id && (edge.target_step_id || edge.target_block_group_id)) {
      const existing = edgesBySourceGroup.get(edge.source_block_group_id) || []
      existing.push(edge)
      edgesBySourceGroup.set(edge.source_block_group_id, existing)
    }
  }

  // For each source group with multiple outgoing edges
  for (const [sourceGroupId, sourceEdges] of edgesBySourceGroup) {
    if (sourceEdges.length <= 1) continue

    const sourceGroup = groupMap.get(sourceGroupId)
    if (!sourceGroup) continue

    // Get output ports for the source group
    const outputPorts = getGroupOutputPorts(sourceGroup.type)
    if (outputPorts.length <= 1) continue

    // Create port order map (port name -> order index)
    const portOrder = new Map<string, number>()
    outputPorts.forEach((port, index) => {
      portOrder.set(port.name, index)
    })

    // Collect all targets with their port info and type
    interface TargetInfo {
      id: string
      isGroup: boolean
      portIndex: number
    }
    const targetsByPort = new Map<string, TargetInfo[]>()

    for (const edge of sourceEdges) {
      if (!edge.source_port) continue

      const portName = edge.source_port
      const portIndex = portOrder.get(portName) ?? 999

      const existing = targetsByPort.get(portName) || []

      if (edge.target_step_id) {
        // Target is a step
        if (!existing.some(t => t.id === edge.target_step_id && !t.isGroup)) {
          existing.push({ id: edge.target_step_id, isGroup: false, portIndex })
        }
      } else if (edge.target_block_group_id) {
        // Target is a group
        if (!existing.some(t => t.id === edge.target_block_group_id && t.isGroup)) {
          existing.push({ id: edge.target_block_group_id, isGroup: true, portIndex })
        }
      }

      targetsByPort.set(portName, existing)
    }

    // Collect all targets with their port indices
    const allTargets: TargetInfo[] = []
    for (const targets of targetsByPort.values()) {
      allTargets.push(...targets)
    }

    // Sort targets by port index
    allTargets.sort((a, b) => a.portIndex - b.portIndex)

    // Get unique targets in sorted order
    const uniqueTargets: TargetInfo[] = []
    const seen = new Set<string>()
    for (const target of allTargets) {
      const key = `${target.isGroup ? 'g' : 's'}_${target.id}`
      if (!seen.has(key)) {
        uniqueTargets.push(target)
        seen.add(key)
      }
    }

    if (uniqueTargets.length <= 1) continue

    // Get current Y positions and heights for all targets
    interface TargetPosition {
      target: TargetInfo
      y: number
      height: number
    }
    const targetPositions: TargetPosition[] = []

    for (const target of uniqueTargets) {
      if (target.isGroup) {
        const result = groupResultMap.get(target.id)
        if (result) {
          targetPositions.push({ target, y: result.y, height: result.height })
        }
      } else {
        const result = stepResultMap.get(target.id)
        if (result) {
          targetPositions.push({ target, y: result.y, height: nodeHeight })
        }
      }
    }

    if (targetPositions.length <= 1) continue

    // Get source group's result to calculate port positions
    const sourceResult = groupResultMap.get(sourceGroupId)
    if (!sourceResult) continue

    // Calculate port Y positions on the source group
    // Ports are distributed vertically on the right edge of the group
    const portCount = outputPorts.length
    const portSpacing = sourceResult.height / (portCount + 1)
    const portYPositions = new Map<number, number>()
    for (let i = 0; i < portCount; i++) {
      // Port Y position relative to the group's top
      portYPositions.set(i, sourceResult.y + portSpacing * (i + 1))
    }

    // Group targets by port index
    const targetsByPortIndex = new Map<number, TargetPosition[]>()
    for (const tp of targetPositions) {
      const existing = targetsByPortIndex.get(tp.target.portIndex) || []
      existing.push(tp)
      targetsByPortIndex.set(tp.target.portIndex, existing)
    }

    // Collect all positioned targets with their final Y positions and heights
    interface PositionedTarget {
      target: TargetInfo
      y: number
      height: number
      portIndex: number
    }
    const positionedTargets: PositionedTarget[] = []

    // Position targets at their corresponding port's Y level
    for (const [portIndex, portTargets] of targetsByPortIndex) {
      const portY = portYPositions.get(portIndex)
      if (portY === undefined) continue

      if (portTargets.length === 1) {
        // Single target for this port - align its center with the port
        const tp = portTargets[0]
        const newY = snapToGrid(portY - tp.height / 2)
        positionedTargets.push({ target: tp.target, y: newY, height: tp.height, portIndex })
      } else {
        // Multiple targets for the same port - stack them vertically around the port Y
        const totalHeight = portTargets.reduce((sum, tp) => sum + tp.height, 0) +
                           (portTargets.length - 1) * nodeSeparation
        let currentY = portY - totalHeight / 2

        for (const tp of portTargets) {
          const newY = snapToGrid(currentY)
          positionedTargets.push({ target: tp.target, y: newY, height: tp.height, portIndex })
          currentY += tp.height + nodeSeparation
        }
      }
    }

    // Sort by port index to process in order
    positionedTargets.sort((a, b) => a.portIndex - b.portIndex)

    // Check for overlaps between targets on different ports and adjust
    for (let i = 1; i < positionedTargets.length; i++) {
      const prev = positionedTargets[i - 1]
      const curr = positionedTargets[i]

      // Skip if same port (already handled by stacking logic)
      if (prev.portIndex === curr.portIndex) continue

      // Check if previous target's bottom overlaps with current target's top
      const prevBottom = prev.y + prev.height
      const currTop = curr.y
      const overlap = prevBottom - currTop + nodeSeparation

      if (overlap > 0) {
        // Move current target down to avoid overlap
        curr.y = snapToGrid(prevBottom + nodeSeparation)
      }
    }

    // Apply the final positions
    for (const pt of positionedTargets) {
      if (pt.target.isGroup) {
        const result = groupResultMap.get(pt.target.id)
        if (result) {
          const deltaY = pt.y - result.y
          result.y = pt.y

          const groupSteps = stepsByGroupId.get(pt.target.id) || []
          for (const step of groupSteps) {
            const stepResult = stepResultMap.get(step.id)
            if (stepResult) {
              stepResult.y = snapToGrid(stepResult.y + deltaY)
            }
          }
        }
      } else {
        const result = stepResultMap.get(pt.target.id)
        if (result) {
          result.y = pt.y
        }
      }
    }
  }
}

/**
 * Align consecutive groups (1:1 connections) to make edges horizontal.
 * This ensures that when group A connects to group B with a single edge,
 * their connection points (output port of A, input port of B) are at the same Y position.
 */
function alignConsecutiveGroups(
  stepResults: LayoutResult[],
  groupResults: GroupLayoutResult[],
  edges: Edge[],
  groupMap: Map<string, BlockGroup>,
  steps: Step[],
  getGroupOutputPorts: (groupType: BlockGroupType) => OutputPort[],
  gridSize: number
): void {
  const snapToGrid = (value: number): number => {
    return Math.round(value / gridSize) * gridSize
  }

  // Build result maps
  const groupResultMap = new Map<string, GroupLayoutResult>()
  for (const result of groupResults) {
    groupResultMap.set(result.groupId, result)
  }

  const stepResultMap = new Map<string, LayoutResult>()
  for (const result of stepResults) {
    stepResultMap.set(result.stepId, result)
  }

  // Build map of steps by their group ID
  const stepsByGroupId = new Map<string, Step[]>()
  for (const step of steps) {
    if (step.block_group_id) {
      const existing = stepsByGroupId.get(step.block_group_id) || []
      existing.push(step)
      stepsByGroupId.set(step.block_group_id, existing)
    }
  }

  // Find group-to-group edges
  const groupToGroupEdges = edges.filter(
    e => e.source_block_group_id && e.target_block_group_id
  )

  // Count incoming group-to-group edges for each target group
  const incomingCountByGroup = new Map<string, number>()
  for (const edge of groupToGroupEdges) {
    if (edge.target_block_group_id) {
      const count = incomingCountByGroup.get(edge.target_block_group_id) || 0
      incomingCountByGroup.set(edge.target_block_group_id, count + 1)
    }
  }

  // Count outgoing group-to-group edges for each source group
  const outgoingCountByGroup = new Map<string, number>()
  for (const edge of groupToGroupEdges) {
    if (edge.source_block_group_id) {
      const count = outgoingCountByGroup.get(edge.source_block_group_id) || 0
      outgoingCountByGroup.set(edge.source_block_group_id, count + 1)
    }
  }

  // Count ALL targets (steps + groups) per port for each source group
  // This is used to skip alignment when a group has multiple targets on the same port
  const targetsPerPortByGroup = new Map<string, Map<string, number>>()
  for (const edge of edges) {
    if (edge.source_block_group_id && edge.source_port && (edge.target_step_id || edge.target_block_group_id)) {
      let portMap = targetsPerPortByGroup.get(edge.source_block_group_id)
      if (!portMap) {
        portMap = new Map<string, number>()
        targetsPerPortByGroup.set(edge.source_block_group_id, portMap)
      }
      const count = portMap.get(edge.source_port) || 0
      portMap.set(edge.source_port, count + 1)
    }
  }

  // Build adjacency list for topological sorting
  const outgoingEdges = new Map<string, Edge[]>()
  const incomingDegree = new Map<string, number>()

  // Initialize all groups with 0 incoming degree
  for (const result of groupResults) {
    incomingDegree.set(result.groupId, 0)
    outgoingEdges.set(result.groupId, [])
  }

  // Build the graph structure (only for edges we want to process)
  for (const edge of groupToGroupEdges) {
    if (!edge.source_block_group_id || !edge.target_block_group_id || !edge.source_port) continue

    const sourceOutgoingCount = outgoingCountByGroup.get(edge.source_block_group_id) || 0
    const targetIncomingCount = incomingCountByGroup.get(edge.target_block_group_id) || 0

    // Skip if source has multiple group-to-group outgoing edges
    if (sourceOutgoingCount !== 1) continue

    // Skip if target has multiple group-to-group incoming edges
    if (targetIncomingCount !== 1) continue

    // Skip if source has multiple targets (steps + groups) on the same port
    // This prevents overlap when a port connects to both a step and a group
    const portMap = targetsPerPortByGroup.get(edge.source_block_group_id)
    const targetsOnPort = portMap?.get(edge.source_port) || 0
    if (targetsOnPort > 1) continue

    // Skip if source group outputs to BOTH groups AND steps (on any port)
    // This prevents overlap when aligning a group would overlap with step targets on other ports
    const allEdgesFromSource = edges.filter(e => e.source_block_group_id === edge.source_block_group_id)
    const hasGroupTargets = allEdgesFromSource.some(e => e.target_block_group_id)
    const hasStepTargets = allEdgesFromSource.some(e => e.target_step_id)
    if (hasGroupTargets && hasStepTargets) continue

    const existing = outgoingEdges.get(edge.source_block_group_id) || []
    existing.push(edge)
    outgoingEdges.set(edge.source_block_group_id, existing)

    const degree = incomingDegree.get(edge.target_block_group_id) || 0
    incomingDegree.set(edge.target_block_group_id, degree + 1)
  }

  // Topological sort using Kahn's algorithm
  const queue: string[] = []
  for (const [groupId, degree] of incomingDegree) {
    if (degree === 0) {
      queue.push(groupId)
    }
  }

  // Process edges in topological order
  while (queue.length > 0) {
    const sourceGroupId = queue.shift()!
    const edges = outgoingEdges.get(sourceGroupId) || []

    for (const edge of edges) {
      if (!edge.target_block_group_id) continue

      const sourceResult = groupResultMap.get(edge.source_block_group_id!)
      const targetResult = groupResultMap.get(edge.target_block_group_id)

      if (sourceResult && targetResult) {
        // Calculate source output port Y position based on actual port location
        const sourceGroup = groupMap.get(edge.source_block_group_id!)
        let sourcePortY = sourceResult.y + sourceResult.height / 2 // default to center

        if (sourceGroup && edge.source_port) {
          const outputPorts = getGroupOutputPorts(sourceGroup.type)
          const portIndex = outputPorts.findIndex(p => p.name === edge.source_port)
          if (portIndex >= 0 && outputPorts.length > 1) {
            // Calculate actual port Y position (ports are distributed vertically)
            const portSpacing = sourceResult.height / (outputPorts.length + 1)
            sourcePortY = sourceResult.y + portSpacing * (portIndex + 1)
          }
        }

        // Calculate target input port Y position (center of group - input is always at center)
        const targetPortY = targetResult.y + targetResult.height / 2

        // Calculate delta to align target's input port with source's output port
        const deltaY = sourcePortY - targetPortY

        if (Math.abs(deltaY) >= gridSize) {
          // Move target group
          const newY = snapToGrid(targetResult.y + deltaY)
          const actualDelta = newY - targetResult.y
          targetResult.y = newY

          // Also move all steps inside the target group
          const targetGroupSteps = stepsByGroupId.get(edge.target_block_group_id) || []
          for (const step of targetGroupSteps) {
            const stepResult = stepResultMap.get(step.id)
            if (stepResult) {
              stepResult.y = snapToGrid(stepResult.y + actualDelta)
            }
          }
        }
      }

      // Decrease incoming degree and add to queue if it becomes 0
      const degree = incomingDegree.get(edge.target_block_group_id) || 0
      incomingDegree.set(edge.target_block_group_id, degree - 1)
      if (degree - 1 === 0) {
        queue.push(edge.target_block_group_id)
      }
    }
  }
}

/**
 * Calculate optimized positions for all steps and block groups using dagre layout algorithm
 * This function handles block groups by:
 * 1. Laying out steps inside each group separately (with multiple entry points stacked vertically)
 * 2. Treating each group as a single node in the main graph (edges connect to groups, not internal steps)
 * 3. Calculating final positions considering group containment
 *
 * Key behaviors:
 * - Group internal steps with multiple entry points (no incoming internal edges) are arranged vertically
 * - Edges from outside the group connect to the group boundary, not directly to internal steps
 * - This creates a clean visual where edges don't cross group boundaries
 * - When getOutputPorts is provided, target nodes are ordered by source port order (top to bottom)
 */
export function calculateLayoutWithGroups(
  steps: Step[],
  edges: Edge[],
  blockGroups: BlockGroup[],
  options: LayoutOptionsWithPorts = {}
): { steps: LayoutResult[]; groups: GroupLayoutResult[] } {
  const opts = { ...DEFAULT_OPTIONS, ...options }
  const GRID_SIZE = 20
  const GROUP_PADDING = 40 // Padding inside groups
  const GROUP_HEADER_HEIGHT = 40 // Height for group header/title

  const snapToGrid = (value: number): number => {
    return Math.round(value / GRID_SIZE) * GRID_SIZE
  }

  // If no block groups, use simple layout
  if (blockGroups.length === 0) {
    return {
      steps: calculateLayout(steps, edges, options),
      groups: [],
    }
  }

  // Separate steps by group membership
  const stepsByGroup = new Map<string, Step[]>()
  const ungroupedSteps: Step[] = []

  for (const step of steps) {
    if (step.block_group_id) {
      const groupSteps = stepsByGroup.get(step.block_group_id) || []
      groupSteps.push(step)
      stepsByGroup.set(step.block_group_id, groupSteps)
    } else {
      ungroupedSteps.push(step)
    }
  }

  // Build step-to-group mapping
  const stepToGroup = new Map<string, string>()
  for (const step of steps) {
    if (step.block_group_id) {
      stepToGroup.set(step.id, step.block_group_id)
    }
  }

  // Calculate internal layout for each group and determine group sizes
  const groupInternalLayouts = new Map<string, LayoutResult[]>()
  const groupSizes = new Map<string, { width: number; height: number }>()

  for (const group of blockGroups) {
    const groupSteps = stepsByGroup.get(group.id) || []
    if (groupSteps.length === 0) {
      // Empty group - use minimum size
      groupSizes.set(group.id, { width: 160, height: 150 })
      groupInternalLayouts.set(group.id, [])
      continue
    }

    const groupStepIds = new Set(groupSteps.map(s => s.id))

    // Get edges that are internal to this group (step-to-step only)
    const internalEdges = edges.filter(
      e => e.source_step_id && e.target_step_id &&
           groupStepIds.has(e.source_step_id) && groupStepIds.has(e.target_step_id)
    )

    // Find entry points: steps that have no incoming internal edges
    // These are steps that receive input from outside the group or are roots
    const hasInternalIncoming = new Set<string>()
    for (const edge of internalEdges) {
      if (edge.target_step_id) {
        hasInternalIncoming.add(edge.target_step_id)
      }
    }

    const entryPoints = groupSteps.filter(s => !hasInternalIncoming.has(s.id))
    const hasMultipleEntryPoints = entryPoints.length > 1

    let internalLayout: LayoutResult[]

    if (hasMultipleEntryPoints) {
      // Multiple entry points: arrange them vertically, then layout their subgraphs
      internalLayout = layoutWithVerticalEntryPoints(
        groupSteps,
        internalEdges,
        entryPoints,
        opts
      )
    } else {
      // Single entry point or no clear entry: use standard TB layout
      internalLayout = calculateLayout(groupSteps, internalEdges, {
        ...options,
        direction: 'TB',
      })
    }

    // Calculate bounding box
    let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity
    for (const result of internalLayout) {
      minX = Math.min(minX, result.x)
      minY = Math.min(minY, result.y)
      maxX = Math.max(maxX, result.x + opts.nodeWidth)
      maxY = Math.max(maxY, result.y + opts.nodeHeight)
    }

    // Handle case where layout is empty or invalid
    if (minX === Infinity) {
      minX = 0
      minY = 0
      maxX = opts.nodeWidth
      maxY = opts.nodeHeight
    }

    // Normalize positions to start from (0, 0) with padding
    const normalizedLayout = internalLayout.map(result => ({
      stepId: result.stepId,
      x: snapToGrid(result.x - minX + GROUP_PADDING),
      y: snapToGrid(result.y - minY + GROUP_PADDING + GROUP_HEADER_HEIGHT),
    }))

    groupInternalLayouts.set(group.id, normalizedLayout)

    const groupWidth = snapToGrid(maxX - minX + GROUP_PADDING * 2)
    const groupHeight = snapToGrid(maxY - minY + GROUP_PADDING * 2 + GROUP_HEADER_HEIGHT)
    groupSizes.set(group.id, {
      width: Math.max(160, groupWidth),
      height: Math.max(150, groupHeight),
    })
  }

  // Create main graph with ungrouped steps and groups as nodes
  // Groups act as single nodes - edges connect to groups, not internal steps
  const mainGraph = new dagre.graphlib.Graph()
  mainGraph.setGraph({
    rankdir: opts.direction,
    nodesep: opts.nodeSeparation,
    ranksep: opts.rankSeparation,
    marginx: 40,
    marginy: 40,
  })
  mainGraph.setDefaultEdgeLabel(() => ({}))

  // Add ungrouped steps as nodes
  for (const step of ungroupedSteps) {
    mainGraph.setNode(step.id, {
      width: opts.nodeWidth,
      height: opts.nodeHeight,
    })
  }

  // Add groups as nodes (using their calculated sizes)
  for (const group of blockGroups) {
    const size = groupSizes.get(group.id) || { width: 200, height: 150 }
    mainGraph.setNode(`group_${group.id}`, {
      width: size.width,
      height: size.height,
    })
  }

  // Add edges - map group-internal endpoints to the group node
  // This ensures edges connect to group boundaries, not crossing into them
  const addedEdges = new Set<string>()
  for (const edge of edges) {
    // Handle edges with explicit group endpoints
    let sourceNode: string | undefined
    let targetNode: string | undefined

    if (edge.source_block_group_id) {
      // Edge originates from a group
      sourceNode = `group_${edge.source_block_group_id}`
    } else if (edge.source_step_id) {
      // Edge originates from a step (possibly inside a group)
      const sourceGroup = stepToGroup.get(edge.source_step_id)
      sourceNode = sourceGroup ? `group_${sourceGroup}` : edge.source_step_id
    }

    if (edge.target_block_group_id) {
      // Edge targets a group
      targetNode = `group_${edge.target_block_group_id}`
    } else if (edge.target_step_id) {
      // Edge targets a step (possibly inside a group)
      const targetGroup = stepToGroup.get(edge.target_step_id)
      targetNode = targetGroup ? `group_${targetGroup}` : edge.target_step_id
    }

    // Skip if source or target couldn't be resolved
    if (!sourceNode || !targetNode) continue

    // Skip internal edges (both endpoints are the same group node)
    if (sourceNode === targetNode && sourceNode.startsWith('group_')) {
      continue
    }

    // Avoid duplicate edges
    const edgeKey = `${sourceNode}->${targetNode}`
    if (addedEdges.has(edgeKey)) continue
    addedEdges.add(edgeKey)

    mainGraph.setEdge(sourceNode, targetNode)
  }

  // Calculate main layout
  dagre.layout(mainGraph)

  // Extract final positions
  const stepResults: LayoutResult[] = []
  const groupResults: GroupLayoutResult[] = []

  // Position ungrouped steps
  for (const step of ungroupedSteps) {
    const node = mainGraph.node(step.id)
    if (node) {
      stepResults.push({
        stepId: step.id,
        x: snapToGrid(node.x - opts.nodeWidth / 2),
        y: snapToGrid(node.y - opts.nodeHeight / 2),
      })
    }
  }

  // Position groups and their internal steps
  for (const group of blockGroups) {
    const groupNode = mainGraph.node(`group_${group.id}`)
    const size = groupSizes.get(group.id) || { width: 200, height: 150 }

    if (groupNode) {
      const groupX = snapToGrid(groupNode.x - size.width / 2)
      const groupY = snapToGrid(groupNode.y - size.height / 2)

      groupResults.push({
        groupId: group.id,
        x: groupX,
        y: groupY,
        width: size.width,
        height: size.height,
      })

      // Position steps inside the group (relative to group position)
      const internalLayout = groupInternalLayouts.get(group.id) || []
      for (const internalResult of internalLayout) {
        stepResults.push({
          stepId: internalResult.stepId,
          x: groupX + internalResult.x,
          y: groupY + internalResult.y,
        })
      }
    }
  }

  // Apply port-based Y ordering for ungrouped steps if getOutputPorts is provided
  const getOutputPorts = options.getOutputPorts
  const ungroupedStepIds = new Set(ungroupedSteps.map(s => s.id))

  if (getOutputPorts) {
    // Filter edges to only include edges between ungrouped steps
    const ungroupedEdges = edges.filter(
      e => e.source_step_id && e.target_step_id &&
           ungroupedStepIds.has(e.source_step_id) && ungroupedStepIds.has(e.target_step_id)
    )

    // Only adjust ungrouped steps (grouped steps have their own internal layout)
    const ungroupedResults = stepResults.filter(r => ungroupedStepIds.has(r.stepId))
    adjustYBySourcePort(ungroupedResults, ungroupedSteps, ungroupedEdges, getOutputPorts, opts.nodeSeparation, GRID_SIZE)
  }

  // Apply port-based Y ordering for edges from block groups (unified handling for both step and group targets)
  const getGroupOutputPorts = options.getGroupOutputPorts
  if (getGroupOutputPorts) {
    // Build group map for quick lookup
    const groupMap = new Map<string, BlockGroup>()
    for (const group of blockGroups) {
      groupMap.set(group.id, group)
    }

    // Filter edges that originate from block groups (targeting either ungrouped steps or groups)
    const groupOutputEdges = edges.filter(
      e => e.source_block_group_id && (
        (e.target_step_id && ungroupedStepIds.has(e.target_step_id)) ||
        e.target_block_group_id
      )
    )

    // Use unified function to handle both step and group targets together
    // Pass all stepResults (not just ungrouped) so that steps inside groups can be moved with their group
    adjustYByGroupSourcePortUnified(
      stepResults,
      groupResults,
      groupOutputEdges,
      groupMap,
      steps,
      getGroupOutputPorts,
      opts.nodeSeparation,
      opts.nodeHeight,
      GRID_SIZE
    )

    // Align consecutive groups (1:1 connections) to make edges horizontal
    alignConsecutiveGroups(
      stepResults,
      groupResults,
      edges,
      groupMap,
      steps,
      getGroupOutputPorts,
      GRID_SIZE
    )
  }

  return { steps: stepResults, groups: groupResults }
}

/**
 * Layout steps with multiple entry points arranged vertically
 * Entry points are placed in a vertical column, with their subgraphs flowing horizontally
 */
function layoutWithVerticalEntryPoints(
  steps: Step[],
  internalEdges: Edge[],
  entryPoints: Step[],
  opts: Required<LayoutOptions>
): LayoutResult[] {
  const results: LayoutResult[] = []
  const VERTICAL_SPACING = opts.nodeHeight + opts.nodeSeparation
  const HORIZONTAL_SPACING = opts.nodeWidth + opts.rankSeparation

  // Build adjacency list for traversal (step-to-step edges only)
  const children = new Map<string, string[]>()
  for (const edge of internalEdges) {
    if (edge.source_step_id && edge.target_step_id) {
      const existing = children.get(edge.source_step_id) || []
      existing.push(edge.target_step_id)
      children.set(edge.source_step_id, existing)
    }
  }

  // Track positioned steps
  const positioned = new Set<string>()
  let currentY = 0

  // Position each entry point and its descendants
  for (const entryPoint of entryPoints) {
    // BFS to find all descendants and their depths
    const depths = new Map<string, number>()
    const queue: Array<{ step: Step; depth: number }> = [{ step: entryPoint, depth: 0 }]
    depths.set(entryPoint.id, 0)

    while (queue.length > 0) {
      const { step, depth } = queue.shift()!
      const stepChildren = children.get(step.id) || []

      for (const childId of stepChildren) {
        if (!depths.has(childId)) {
          const childStep = steps.find(s => s.id === childId)
          if (childStep) {
            depths.set(childId, depth + 1)
            queue.push({ step: childStep, depth: depth + 1 })
          }
        }
      }
    }

    // Group steps by depth level
    const stepsByDepth = new Map<number, Step[]>()
    for (const [stepId, depth] of depths) {
      const step = steps.find(s => s.id === stepId)
      if (step) {
        const existing = stepsByDepth.get(depth) || []
        existing.push(step)
        stepsByDepth.set(depth, existing)
      }
    }

    // Position steps at each depth level
    const maxDepth = Math.max(...depths.values(), 0)
    for (let depth = 0; depth <= maxDepth; depth++) {
      const depthSteps = stepsByDepth.get(depth) || []
      let depthY = currentY

      for (const step of depthSteps) {
        if (!positioned.has(step.id)) {
          results.push({
            stepId: step.id,
            x: depth * HORIZONTAL_SPACING,
            y: depthY,
          })
          positioned.add(step.id)
          depthY += VERTICAL_SPACING
        }
      }
    }

    // Move to next row for next entry point's subgraph
    currentY += VERTICAL_SPACING
  }

  // Position any remaining steps that weren't reachable from entry points
  for (const step of steps) {
    if (!positioned.has(step.id)) {
      results.push({
        stepId: step.id,
        x: 0,
        y: currentY,
      })
      currentY += VERTICAL_SPACING
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
