<script setup lang="ts">
import { VueFlow, useVueFlow, Handle, Position, MarkerType, type Node, type Edge as FlowEdge } from '@vue-flow/core'
import { MiniMap } from '@vue-flow/minimap'
import { NodeResizer, type OnResizeStart, type OnResize, type OnResizeEnd } from '@vue-flow/node-resizer'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/minimap/dist/style.css'
import '@vue-flow/node-resizer/dist/style.css'
import type { Step, Edge, StepType, StepRun, BlockDefinition, InputPort, OutputPort, BlockGroup, BlockGroupType, GroupRole } from '~/types/api'
import NodeIcon from './NodeIcon.vue'
import { getBlockIcon } from '~/composables/useBlockIcons'
import { useCopilotOffset } from '~/composables/useFloatingLayout'
import { useEditorState } from '~/composables/useEditorState'

// Constants for group node ID prefix
const GROUP_NODE_PREFIX = 'group_'

// Helper function to extract plain group UUID from Vue Flow node ID
function getGroupUuidFromNodeId(nodeId: string): string {
  if (nodeId.startsWith(GROUP_NODE_PREFIX)) {
    return nodeId.slice(GROUP_NODE_PREFIX.length)
  }
  return nodeId
}

// Helper function to convert group UUID to Vue Flow node ID
function getNodeIdFromGroupUuid(groupUuid: string): string {
  return `${GROUP_NODE_PREFIX}${groupUuid}`
}

// Grid size constant - must match Vue Flow's snap-grid setting
const GRID_SIZE = 20

// Helper function to snap a value to the grid
function snapToGrid(value: number): number {
  return Math.round(value / GRID_SIZE) * GRID_SIZE
}

// Preview state for Copilot changes
interface PreviewState {
  addedStepIds: Set<string>
  modifiedStepIds: Set<string>
  deletedStepIds: Set<string>
  addedEdgeIds: Set<string>
  deletedEdgeIds: Set<string>
}

const props = defineProps<{
  steps: Step[]
  edges: Edge[]
  blockGroups?: BlockGroup[]  // Block groups for control flow constructs
  readonly?: boolean
  selectedStepId?: string | null
  selectedGroupId?: string | null  // Selected block group
  stepRuns?: StepRun[]  // Optional: for showing step execution status
  blockDefinitions?: BlockDefinition[] // Block definitions for output ports
  showMinimap?: boolean // Show/hide minimap (default: true)
  previewState?: PreviewState | null // Copilot preview highlighting
}>()

// Pushed block info for boundary violations (pushed outside)
interface PushedBlock {
  stepId: string
  position: { x: number; y: number }
}

// Added block info for blocks that should be added to the group (pushed inside)
interface AddedBlock {
  stepId: string
  position: { x: number; y: number }
  role: GroupRole
}

// Moved group info for groups that were pushed by blocks
interface MovedGroup {
  groupId: string
  position: { x: number; y: number }
  delta: { x: number; y: number }
}

const emit = defineEmits<{
  (e: 'step:select', step: Step): void
  (e: 'step:update', stepId: string, position: { x: number; y: number }, movedGroups?: MovedGroup[]): void
  (e: 'step:drop', data: { type: StepType; name: string; position: { x: number; y: number }; groupId?: string; groupRole?: GroupRole }): void
  (e: 'step:assign-group', stepId: string, groupId: string | null, position: { x: number; y: number }, role?: GroupRole, movedGroups?: MovedGroup[]): void
  (e: 'edge:add', source: string, target: string, sourcePort?: string, targetPort?: string): void
  (e: 'edge:delete', edgeId: string): void
  (e: 'pane:click' | 'autoLayout'): void
  (e: 'step:showDetails', stepRun: StepRun): void
  // Block group events
  (e: 'group:select', group: BlockGroup): void
  (e: 'group:update', groupId: string, updates: { position?: { x: number; y: number }; size?: { width: number; height: number } }): void
  (e: 'group:drop', data: { type: BlockGroupType; name: string; position: { x: number; y: number } }): void
  // Group move complete - includes delta for updating internal blocks, pushed/added blocks, and cascaded group movements
  (e: 'group:move-complete', groupId: string, data: {
    position: { x: number; y: number }
    delta: { x: number; y: number }
    pushedBlocks: PushedBlock[]
    addedBlocks: AddedBlock[]
    movedGroups: MovedGroup[]
  }): void
  // Group resize complete - includes size change, pushed/added blocks, and cascaded group movements
  (e: 'group:resize-complete', groupId: string, data: {
    position: { x: number; y: number }
    size: { width: number; height: number }
    pushedBlocks: PushedBlock[]
    addedBlocks: AddedBlock[]
    movedGroups: MovedGroup[]
  }): void
}>()

const { onConnect, onNodeDragStop, onPaneClick, onEdgeClick, project, updateNode, setNodes, getNodes, removeNodes, viewport, zoomIn, zoomOut, zoomTo } = useVueFlow()

// Copilot Sidebar を考慮した右端オフセット
const { value: copilotOffset, isResizing: copilotResizing } = useCopilotOffset(12)
const { copilotSidebarOpen, toggleCopilotSidebar } = useEditorState()

// Right offset for auto-layout button (shift left when properties panel is open)
const autoLayoutRightOffset = computed(() => {
  // 基本: copilotOffset (CopilotSidebarの現在の幅 + baseOffset)
  // パネル開時: さらに 360px (FloatingRightPanel幅) + 12px (gap) を追加
  if (props.selectedStepId || props.selectedGroupId) {
    return copilotOffset.value + 360 + 12
  }
  return copilotOffset.value
})

// Selected edge for deletion
const selectedEdgeId = ref<string | null>(null)
// Store the click position in flow coordinates for delete button placement
const edgeClickFlowPosition = ref<{ x: number; y: number } | null>(null)
// Reference to the dag editor container for coordinate conversion
const dagEditorRef = ref<HTMLElement | null>(null)

// Compute button position from flow coordinates, reactive to viewport changes
const edgeDeleteButtonPosition = computed(() => {
  if (!edgeClickFlowPosition.value) return null

  // Convert flow coordinates to screen position using current viewport
  // This makes the button follow pan/zoom just like blocks do
  const screenX = edgeClickFlowPosition.value.x * viewport.value.zoom + viewport.value.x
  const screenY = edgeClickFlowPosition.value.y * viewport.value.zoom + viewport.value.y

  return {
    x: screenX + 10, // Offset to top-right
    y: screenY - 30,
  }
})

// Drag state
const isDragOver = ref(false)

// Create a map of step ID to step run for quick lookup
const stepRunMap = computed(() => {
  const map = new Map<string, StepRun>()
  if (props.stepRuns) {
    for (const run of props.stepRuns) {
      map.set(run.step_id, run)
    }
  }
  return map
})

// Create a map of block slug to input ports
const inputPortsMap = computed(() => {
  const map = new Map<string, InputPort[]>()
  if (props.blockDefinitions) {
    for (const block of props.blockDefinitions) {
      map.set(block.slug, block.input_ports || [])
    }
  }
  return map
})

// Create a map of block slug to output ports
const outputPortsMap = computed(() => {
  const map = new Map<string, OutputPort[]>()
  if (props.blockDefinitions) {
    for (const block of props.blockDefinitions) {
      map.set(block.slug, block.output_ports || [])
    }
  }
  return map
})

// Get input ports for a step type
function getInputPorts(stepType: string): InputPort[] {
  const ports = inputPortsMap.value.get(stepType)
  if (ports && ports.length > 0) {
    return ports
  }
  // Default single input port
  return [{ name: 'input', label: 'Input', required: true }]
}

// Get output ports for a step type (or step for dynamic ports like switch)
function getOutputPorts(stepType: string, step?: Step): OutputPort[] {
  const config = step?.config as Record<string, unknown> | undefined

  // Special handling for switch blocks - generate dynamic ports from cases
  if (stepType === 'switch' && config?.cases) {
    const casesConfig = config.cases
    const dynamicPorts: OutputPort[] = []

    // Handle both array format and object format for cases
    if (Array.isArray(casesConfig)) {
      // Array format: [{ name: string; expression?: string; is_default?: boolean }, ...]
      for (const caseItem of casesConfig as Array<{ name: string; expression?: string; is_default?: boolean }>) {
        if (caseItem.is_default) {
          dynamicPorts.push({
            name: 'default',
            label: 'Default',
            is_default: true,
          })
        } else {
          dynamicPorts.push({
            name: caseItem.name || `case_${dynamicPorts.length + 1}`,
            label: caseItem.name || `Case ${dynamicPorts.length + 1}`,
            is_default: false,
          })
        }
      }
    } else if (typeof casesConfig === 'object' && casesConfig !== null) {
      // Object format: { "case_name": "value", ... }
      for (const caseName of Object.keys(casesConfig as Record<string, unknown>)) {
        dynamicPorts.push({
          name: caseName,
          label: caseName,
          is_default: false,
        })
      }
    }

    // If no cases defined, return default port
    if (dynamicPorts.length === 0) {
      return [{ name: 'default', label: 'Default', is_default: true }]
    }

    return dynamicPorts
  }

  // Special handling for router blocks - generate dynamic ports from routes
  if (stepType === 'router' && config?.routes) {
    const routes = config.routes as Array<{ name: string; description?: string }>
    const dynamicPorts: OutputPort[] = [
      { name: 'default', label: 'Default', is_default: true }
    ]

    for (const route of routes) {
      dynamicPorts.push({
        name: route.name,
        label: route.name,
        is_default: false,
      })
    }

    return dynamicPorts
  }

  // Build output ports dynamically from block definition and step config
  let basePorts = outputPortsMap.value.get(stepType) || []
  if (basePorts.length === 0) {
    basePorts = [{ name: 'output', label: 'Output', is_default: true }]
  }

  // Check for custom_output_ports in config (for code/function blocks)
  const customOutputPorts = config?.custom_output_ports as string[] | undefined
  if (customOutputPorts && customOutputPorts.length > 0) {
    // Replace default ports with custom ports
    basePorts = customOutputPorts.map((name, index) => ({
      name,
      label: name,
      is_default: index === 0,
    }))
  }

  // Check for enable_error_port in config
  const enableErrorPort = config?.enable_error_port as boolean | undefined
  if (enableErrorPort) {
    // Add error port if not already present
    const hasErrorPort = basePorts.some(p => p.name === 'error')
    if (!hasErrorPort) {
      basePorts = [
        ...basePorts,
        { name: 'error', label: 'Error', is_default: false }
      ]
    }
  }

  return basePorts
}

// Check if a step type has multiple inputs (reserved for future use)
function _hasMultipleInputs(stepType: string): boolean {
  const ports = getInputPorts(stepType)
  return ports.length > 1
}

// Check if a step type has multiple outputs (reserved for future use)
function _hasMultipleOutputs(stepType: string): boolean {
  const ports = getOutputPorts(stepType)
  return ports.length > 1
}

// Get block group color based on type
function getGroupColor(type: BlockGroupType): string {
  const colors: Record<BlockGroupType, string> = {
    parallel: '#8b5cf6',    // Purple
    try_catch: '#ef4444',   // Red
    foreach: '#22c55e',     // Green
    while: '#14b8a6',       // Teal
    agent: '#10b981',       // Emerald
  }
  return colors[type] || '#64748b'
}

// Get block group icon
function getGroupIcon(type: BlockGroupType): string {
  const icons: Record<BlockGroupType, string> = {
    parallel: '⫲',
    try_catch: '⚡',
    foreach: '∀',
    while: '↻',
    agent: '🤖',
  }
  return icons[type] || '▢'
}

// Get group label suffix based on type
function getGroupTypeLabel(type: BlockGroupType): string {
  const labels: Record<BlockGroupType, string> = {
    parallel: 'Parallel',
    try_catch: 'Try-Catch',
    foreach: 'ForEach',
    while: 'While',
    agent: 'Agent',
  }
  return labels[type] || type
}

// Group output port definitions
interface GroupPort {
  name: string
  label: string
  color: string
}

const GROUP_OUTPUT_PORTS: Record<BlockGroupType, GroupPort[]> = {
  parallel: [
    { name: 'out', label: 'Output', color: '#22c55e' },
    { name: 'error', label: 'Error', color: '#ef4444' },
  ],
  try_catch: [
    { name: 'out', label: 'Output', color: '#22c55e' },
    { name: 'error', label: 'Error', color: '#ef4444' },
  ],
  foreach: [
    { name: 'out', label: 'Output', color: '#22c55e' },
    { name: 'error', label: 'Error', color: '#ef4444' },
  ],
  while: [
    { name: 'out', label: 'Output', color: '#22c55e' },
  ],
  agent: [
    { name: 'out', label: 'Response', color: '#22c55e' },
    { name: 'error', label: 'Error', color: '#ef4444' },
  ],
}

// Get group output ports
function getGroupOutputPorts(type: BlockGroupType): GroupPort[] {
  return GROUP_OUTPUT_PORTS[type] || [{ name: 'out', label: 'Output', color: '#22c55e' }]
}

// Multi-section zone configuration
interface GroupZone {
  role: string
  label: string
  // Position as percentage of content area (after header)
  top: number
  bottom: number
  left: number
  right: number
}

const GROUP_ZONES: Record<BlockGroupType, GroupZone[] | null> = {
  parallel: null, // Single body zone
  try_catch: null, // Phase A: simplified to single body zone
  foreach: null,
  while: null,
  agent: null, // Single body zone - child steps become tools
}

// Get zones for a group type
function getGroupZones(type: BlockGroupType): GroupZone[] | null {
  return GROUP_ZONES[type] || null
}

// Check if group has multiple zones
function hasMultipleZones(type: BlockGroupType): boolean {
  return GROUP_ZONES[type] !== null
}

// Constants for group layout
const GROUP_HEADER_HEIGHT = 32
const GROUP_PADDING = 10
const GROUP_BOUNDARY_WIDTH = 20

// Default step node size (used as fallback when actual size is not available)
// Miro-style: 48px icon + label width (~80px total), 48px icon + 2px gap + ~14px label (~70px total)
const DEFAULT_STEP_NODE_WIDTH = 80
const DEFAULT_STEP_NODE_HEIGHT = 70

// Default group size for new groups
const DEFAULT_GROUP_WIDTH = 400
const DEFAULT_GROUP_HEIGHT = 300

// Drop zone result interface
interface DropZoneResult {
  group: BlockGroup | null
  zone: 'inside' | 'boundary' | 'outside'
  role?: GroupRole
}

// Determine role based on position within multi-section group
// Phase A: Simplified to always return 'body' since multi-zone was removed
function determineRoleInGroup(_x: number, _y: number, _group: BlockGroup): GroupRole {
  // All groups now have a single body zone only
  return 'body'
}

// Find which group contains the given position and determine zone
// Considers the full bounding box of the block (not just the position point)
function findDropZone(
  x: number,
  y: number,
  blockWidth: number = DEFAULT_STEP_NODE_WIDTH,
  blockHeight: number = DEFAULT_STEP_NODE_HEIGHT,
  excludeGroupId?: string  // Exclude this group from checking (used when dragging a group)
): DropZoneResult {
  if (!props.blockGroups) return { group: null, zone: 'outside' }

  // Block bounding box
  const blockLeft = x
  const blockRight = x + blockWidth
  const blockTop = y
  const blockBottom = y + blockHeight

  // Check groups in reverse order (later groups are on top)
  for (let i = props.blockGroups.length - 1; i >= 0; i--) {
    const group = props.blockGroups[i]

    // Skip excluded group
    if (excludeGroupId && group.id === excludeGroupId) continue

    // Group outer bounds
    const outerLeft = group.position_x
    const outerRight = group.position_x + group.width
    const outerTop = group.position_y
    const outerBottom = group.position_y + group.height

    // Check if any part of the block overlaps with the group's outer bounds
    const overlapsOuterBounds =
      blockRight > outerLeft && blockLeft < outerRight &&
      blockBottom > outerTop && blockTop < outerBottom

    if (!overlapsOuterBounds) continue

    // Valid inside area (where the entire block must fit)
    const innerLeft = group.position_x + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const innerRight = group.position_x + group.width - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
    const innerTop = group.position_y + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const innerBottom = group.position_y + group.height - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

    // Check if the ENTIRE block fits within the valid inner area
    const entireBlockInside =
      blockLeft >= innerLeft && blockRight <= innerRight &&
      blockTop >= innerTop && blockBottom <= innerBottom

    if (entireBlockInside) {
      // Determine role based on block center position for multi-section groups
      const centerX = x + blockWidth / 2
      const centerY = y + blockHeight / 2
      const role = determineRoleInGroup(centerX, centerY, group)
      return { group, zone: 'inside', role }
    }

    // Block overlaps with group but doesn't fully fit in inner area = boundary zone
    return { group, zone: 'boundary' }
  }

  return { group: null, zone: 'outside' }
}

// Legacy function for backward compatibility (reserved for future use)
function _findGroupAtPosition(x: number, y: number): BlockGroup | null {
  const result = findDropZone(x, y)
  return result.zone === 'inside' ? result.group : null
}

// Snap position to valid location (inside or outside group, not on boundary)
// Returns the closest non-boundary position where the entire block fits
function snapToValidPosition(
  x: number,
  y: number,
  group: BlockGroup,
  blockWidth: number = DEFAULT_STEP_NODE_WIDTH,
  blockHeight: number = DEFAULT_STEP_NODE_HEIGHT
): { x: number; y: number; inside: boolean } {
  // Valid inside area bounds (where the block's top-left corner can be placed
  // so that the entire block fits within the inner area)
  const innerLeft = group.position_x + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
  const innerRight = group.position_x + group.width - GROUP_PADDING - GROUP_BOUNDARY_WIDTH - blockWidth
  const innerTop = group.position_y + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
  const innerBottom = group.position_y + group.height - GROUP_PADDING - GROUP_BOUNDARY_WIDTH - blockHeight

  // Check if inner area is large enough to contain the block
  const canFitInside = innerRight >= innerLeft && innerBottom >= innerTop

  if (canFitInside) {
    // Calculate clamped inside position (snapped to grid)
    const insideX = snapToGrid(Math.max(innerLeft, Math.min(x, innerRight)))
    const insideY = snapToGrid(Math.max(innerTop, Math.min(y, innerBottom)))
    const distToInside = Math.sqrt(Math.pow(x - insideX, 2) + Math.pow(y - insideY, 2))

    // Calculate outside positions (block completely outside the group, snapped to grid)
    const outsideGap = GRID_SIZE // Gap between block and group when outside (aligned to grid)
    const edgeDistances = [
      { edge: 'left', dist: Math.abs(x - (group.position_x - blockWidth - outsideGap)),
        outX: snapToGrid(group.position_x - blockWidth - outsideGap), outY: snapToGrid(y) },
      { edge: 'right', dist: Math.abs(x - (group.position_x + group.width + outsideGap)),
        outX: snapToGrid(group.position_x + group.width + outsideGap), outY: snapToGrid(y) },
      { edge: 'top', dist: Math.abs(y - (group.position_y - blockHeight - outsideGap)),
        outX: snapToGrid(x), outY: snapToGrid(group.position_y - blockHeight - outsideGap) },
      { edge: 'bottom', dist: Math.abs(y - (group.position_y + group.height + outsideGap)),
        outX: snapToGrid(x), outY: snapToGrid(group.position_y + group.height + outsideGap) },
    ]

    // Find closest outside position
    edgeDistances.sort((a, b) => a.dist - b.dist)
    const closestEdge = edgeDistances[0]
    const distToOutside = closestEdge.dist

    // Snap to the closer valid position
    if (distToInside <= distToOutside) {
      return { x: insideX, y: insideY, inside: true }
    } else {
      return { x: closestEdge.outX, y: closestEdge.outY, inside: false }
    }
  } else {
    // Block doesn't fit inside - must snap to outside (snapped to grid)
    const outsideGap = GRID_SIZE
    const edgeDistances = [
      { edge: 'left', dist: Math.abs(x - (group.position_x - blockWidth - outsideGap)),
        outX: snapToGrid(group.position_x - blockWidth - outsideGap), outY: snapToGrid(y) },
      { edge: 'right', dist: Math.abs(x - (group.position_x + group.width + outsideGap)),
        outX: snapToGrid(group.position_x + group.width + outsideGap), outY: snapToGrid(y) },
      { edge: 'top', dist: Math.abs(y - (group.position_y - blockHeight - outsideGap)),
        outX: snapToGrid(x), outY: snapToGrid(group.position_y - blockHeight - outsideGap) },
      { edge: 'bottom', dist: Math.abs(y - (group.position_y + group.height + outsideGap)),
        outX: snapToGrid(x), outY: snapToGrid(group.position_y + group.height + outsideGap) },
    ]

    edgeDistances.sort((a, b) => a.dist - b.dist)
    const closestEdge = edgeDistances[0]
    return { x: closestEdge.outX, y: closestEdge.outY, inside: false }
  }
}

// Calculate new position for a group to avoid collision with a block (reserved for future use)
// The group is pushed away from the block in the direction that requires minimum movement
function _pushGroupAwayFromBlock(
  group: BlockGroup,
  blockX: number,
  blockY: number,
  blockWidth: number,
  blockHeight: number
): { x: number; y: number; deltaX: number; deltaY: number } {
  const outsideGap = GRID_SIZE // Use grid-aligned gap

  // Calculate how much overlap exists in each direction
  const blockLeft = blockX
  const blockRight = blockX + blockWidth
  const blockTop = blockY
  const blockBottom = blockY + blockHeight

  const groupLeft = group.position_x
  const groupRight = group.position_x + group.width
  const groupTop = group.position_y
  const groupBottom = group.position_y + group.height

  // Calculate minimum push distance for each direction (snapped to grid)
  const pushDistances = [
    { dir: 'left', dist: groupRight - blockLeft + outsideGap, newX: snapToGrid(blockLeft - group.width - outsideGap), newY: snapToGrid(group.position_y) },
    { dir: 'right', dist: blockRight - groupLeft + outsideGap, newX: snapToGrid(blockRight + outsideGap), newY: snapToGrid(group.position_y) },
    { dir: 'up', dist: groupBottom - blockTop + outsideGap, newX: snapToGrid(group.position_x), newY: snapToGrid(blockTop - group.height - outsideGap) },
    { dir: 'down', dist: blockBottom - groupTop + outsideGap, newX: snapToGrid(group.position_x), newY: snapToGrid(blockBottom + outsideGap) },
  ]

  // Find the direction with minimum push distance
  pushDistances.sort((a, b) => a.dist - b.dist)
  const minPush = pushDistances[0]

  return {
    x: minPush.newX,
    y: minPush.newY,
    deltaX: minPush.newX - group.position_x,
    deltaY: minPush.newY - group.position_y,
  }
}

// Check if a block collides with any group's boundary (excluding specified groups)
function findGroupBoundaryCollision(
  blockX: number,
  blockY: number,
  blockWidth: number,
  blockHeight: number,
  excludeGroupIds: Set<string>,
  groupPositions: Map<string, { x: number; y: number }>  // Current positions (may be updated during cascade)
): BlockGroup | null {
  if (!props.blockGroups) return null

  const blockLeft = blockX
  const blockRight = blockX + blockWidth
  const blockTop = blockY
  const blockBottom = blockY + blockHeight

  for (const group of props.blockGroups) {
    if (excludeGroupIds.has(group.id)) continue

    // Get current position (may have been moved during cascade)
    const pos = groupPositions.get(group.id) || { x: group.position_x, y: group.position_y }

    const outerLeft = pos.x
    const outerRight = pos.x + group.width
    const outerTop = pos.y
    const outerBottom = pos.y + group.height

    // Check if block overlaps with group's outer bounds
    const overlapsGroup =
      blockRight > outerLeft && blockLeft < outerRight &&
      blockBottom > outerTop && blockTop < outerBottom

    if (!overlapsGroup) continue

    // Check if block is fully inside the valid inner area
    const innerLeft = pos.x + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const innerRight = pos.x + group.width - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
    const innerTop = pos.y + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const innerBottom = pos.y + group.height - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

    const fullyInside =
      blockLeft >= innerLeft && blockRight <= innerRight &&
      blockTop >= innerTop && blockBottom <= innerBottom

    // If on boundary (overlaps but not fully inside), return this group
    if (!fullyInside) {
      return group
    }
  }

  return null
}

// Check if a group collides with another group (excluding specified groups)
function findGroupCollision(
  groupX: number,
  groupY: number,
  groupWidth: number,
  groupHeight: number,
  excludeGroupIds: Set<string>,
  groupPositions: Map<string, { x: number; y: number }>,
  groupSizes?: Map<string, { width: number; height: number }>
): BlockGroup | null {
  if (!props.blockGroups) return null

  const gap = 10  // Minimum gap between groups

  for (const group of props.blockGroups) {
    if (excludeGroupIds.has(group.id)) continue

    const pos = groupPositions.get(group.id) || { x: group.position_x, y: group.position_y }
    // Use tracked size if available, otherwise use group's stored size
    const size = groupSizes?.get(group.id) || { width: group.width, height: group.height }

    // Check if groups overlap (with gap consideration)
    const overlaps =
      groupX < pos.x + size.width + gap &&
      groupX + groupWidth + gap > pos.x &&
      groupY < pos.y + size.height + gap &&
      groupY + groupHeight + gap > pos.y

    if (overlaps) {
      return group
    }
  }

  return null
}

// Push direction type for unified cascade direction
type PushDirection = 'left' | 'right' | 'up' | 'down'

// Determine push direction based on relative positions
function determinePushDirection(
  pushedCenterX: number,
  pushedCenterY: number,
  pusherCenterX: number,
  pusherCenterY: number
): PushDirection {
  const dirX = pushedCenterX - pusherCenterX
  const dirY = pushedCenterY - pusherCenterY

  if (Math.abs(dirX) > Math.abs(dirY)) {
    return dirX > 0 ? 'right' : 'left'
  } else {
    return dirY > 0 ? 'down' : 'up'
  }
}

// Generic function to calculate push position
// When fixedDirection is provided, always push in that direction (for cascade consistency)
// Returns the new position, delta, and the direction used
function calculatePushPosition(
  pushedX: number,
  pushedY: number,
  pushedWidth: number,
  pushedHeight: number,
  pusherX: number,
  pusherY: number,
  pusherWidth: number,
  pusherHeight: number,
  fixedDirection?: PushDirection
): { x: number; y: number; deltaX: number; deltaY: number; direction: PushDirection } {
  const gap = GRID_SIZE  // Gap after pushing (grid-aligned)

  // Calculate center points
  const pushedCenterX = pushedX + pushedWidth / 2
  const pushedCenterY = pushedY + pushedHeight / 2
  const pusherCenterX = pusherX + pusherWidth / 2
  const pusherCenterY = pusherY + pusherHeight / 2

  // Use fixed direction if provided, otherwise determine from relative positions
  const direction = fixedDirection ?? determinePushDirection(
    pushedCenterX, pushedCenterY, pusherCenterX, pusherCenterY
  )

  // Calculate new position based on direction (snapped to grid)
  let newX = pushedX
  let newY = pushedY

  switch (direction) {
    case 'left':
      newX = snapToGrid(pusherX - pushedWidth - gap)
      break
    case 'right':
      newX = snapToGrid(pusherX + pusherWidth + gap)
      break
    case 'up':
      newY = snapToGrid(pusherY - pushedHeight - gap)
      break
    case 'down':
      newY = snapToGrid(pusherY + pusherHeight + gap)
      break
  }

  return {
    x: newX,
    y: newY,
    deltaX: newX - pushedX,
    deltaY: newY - pushedY,
    direction,
  }
}

// Push a group away from another group (reserved for future use)
function _pushGroupAwayFromGroup(
  pushedGroup: BlockGroup,
  pushedGroupPos: { x: number; y: number },
  pusherGroup: { x: number; y: number; width: number; height: number }
): { x: number; y: number; deltaX: number; deltaY: number } {
  return calculatePushPosition(
    pushedGroupPos.x, pushedGroupPos.y, pushedGroup.width, pushedGroup.height,
    pusherGroup.x, pusherGroup.y, pusherGroup.width, pusherGroup.height
  )
}

// Process cascade group push when a block is pushed out and collides with other groups
// Returns an array of moved groups with their new positions
function processCascadeGroupPush(
  blockX: number,
  blockY: number,
  blockWidth: number,
  blockHeight: number,
  excludeGroupIds: Set<string>
): MovedGroup[] {
  if (!props.blockGroups) return []

  const movedGroups: MovedGroup[] = []
  const processedGroups = new Set<string>(excludeGroupIds)
  const groupPositions = new Map<string, { x: number; y: number }>()
  const groupSizes = new Map<string, { width: number; height: number }>()

  // Initialize group positions and sizes
  for (const group of props.blockGroups) {
    groupPositions.set(group.id, { x: group.position_x, y: group.position_y })
    groupSizes.set(group.id, { width: group.width, height: group.height })
  }

  // Track cascade direction - determined by the first push
  let cascadeDirection: PushDirection | null = null

  // Queue of items to process: can be block or group
  interface PushItem {
    type: 'block' | 'group'
    x: number
    y: number
    width: number
    height: number
    groupId?: string  // Only for group type
  }

  const pushQueue: PushItem[] = [{
    type: 'block',
    x: blockX,
    y: blockY,
    width: blockWidth,
    height: blockHeight,
  }]

  const MAX_CASCADE_DEPTH = 10
  let cascadeDepth = 0

  while (pushQueue.length > 0 && cascadeDepth < MAX_CASCADE_DEPTH) {
    cascadeDepth++
    const item = pushQueue.shift()!

    // Find groups that collide with this item
    const collidingGroup = findGroupCollision(
      item.x, item.y, item.width, item.height,
      processedGroups, groupPositions, groupSizes
    )

    if (!collidingGroup) continue

    // Mark as processed
    processedGroups.add(collidingGroup.id)

    // Get current position and size
    const currentPos = groupPositions.get(collidingGroup.id) || { x: collidingGroup.position_x, y: collidingGroup.position_y }
    const currentSize = groupSizes.get(collidingGroup.id) || { width: collidingGroup.width, height: collidingGroup.height }

    // Calculate push position using cascade direction
    const pushResult = calculatePushPosition(
      currentPos.x, currentPos.y, currentSize.width, currentSize.height,
      item.x, item.y, item.width, item.height,
      cascadeDirection ?? undefined
    )

    // Set cascade direction if this is the first push
    if (!cascadeDirection) {
      cascadeDirection = pushResult.direction
    }

    // Update position tracking
    groupPositions.set(collidingGroup.id, { x: pushResult.x, y: pushResult.y })

    // Add to moved groups
    movedGroups.push({
      groupId: collidingGroup.id,
      position: { x: pushResult.x, y: pushResult.y },
      delta: { x: pushResult.deltaX, y: pushResult.deltaY },
    })

    // Update Vue Flow's internal state (use prefixed ID to match groupNodes)
    updateNode(`group_${collidingGroup.id}`, { position: { x: pushResult.x, y: pushResult.y } })

    // Add the pushed group to the queue for further cascade checks
    pushQueue.push({
      type: 'group',
      x: pushResult.x,
      y: pushResult.y,
      width: currentSize.width,
      height: currentSize.height,
      groupId: collidingGroup.id,
    })
  }

  return movedGroups
}

// Convert block groups to Vue Flow group nodes
// NOTE: Group node IDs use 'group_' prefix to distinguish from step node IDs
// This matches the format used in flowEdges for group connections
const groupNodes = computed<Node[]>(() => {
  if (!props.blockGroups) return []

  return props.blockGroups.map(group => ({
    id: `group_${group.id}`,
    type: 'group',
    position: { x: group.position_x, y: group.position_y },
    style: {
      width: `${group.width}px`,
      height: `${group.height}px`,
      backgroundColor: `${getGroupColor(group.type)}10`,
      borderColor: getGroupColor(group.type),
      borderWidth: '2px',
      borderStyle: 'dashed',
      borderRadius: '12px',
    },
    data: {
      label: group.name,
      type: group.type,
      group,
      isSelected: group.id === props.selectedGroupId,
      color: getGroupColor(group.type),
      icon: getGroupIcon(group.type),
      typeLabel: getGroupTypeLabel(group.type),
      outputPorts: getGroupOutputPorts(group.type),
      height: group.height,
      width: group.width,
      zones: getGroupZones(group.type),
      hasMultipleZones: hasMultipleZones(group.type),
    },
    // Group nodes should be rendered behind step nodes
    zIndex: -1,
  }))
})

// Get preview class for a step based on Copilot preview state
function getPreviewClass(stepId: string): string | undefined {
  if (!props.previewState) return undefined
  if (props.previewState.addedStepIds?.has(stepId)) return 'preview-added'
  if (props.previewState.modifiedStepIds?.has(stepId)) return 'preview-modified'
  if (props.previewState.deletedStepIds?.has(stepId)) return 'preview-deleted'
  return undefined
}

// Get preview class for an edge based on Copilot preview state
function getEdgePreviewClass(edgeId: string, sourceId: string, targetId: string): string | undefined {
  if (!props.previewState) return undefined

  // Check if edge is marked for deletion by ID
  if (props.previewState.deletedEdgeIds?.has(edgeId)) return 'preview-edge-deleted'

  // Check if edge is marked as added by source->target composite key
  const compositeKey = `${sourceId}->${targetId}`
  if (props.previewState.addedEdgeIds?.has(compositeKey)) return 'preview-edge-added'

  return undefined
}

// Convert steps to Vue Flow nodes
const stepNodes = computed<Node[]>(() => {
  return props.steps.map(step => {
    const stepRun = stepRunMap.value.get(step.id)
    const inputPorts = getInputPorts(step.type)
    // Pass step for dynamic port generation (switch, router)
    const outputPorts = getOutputPorts(step.type, step)

    // Calculate position relative to parent group if step belongs to a group
    const parentGroupId = step.block_group_id || undefined
    let position = { x: step.position_x, y: step.position_y }

    // If step has a parent group, position is relative to the group
    if (parentGroupId && props.blockGroups) {
      const parentGroup = props.blockGroups.find(g => g.id === parentGroupId)
      if (parentGroup) {
        // Position relative to group's top-left corner
        // Add padding for the group header
        position = {
          x: step.position_x - parentGroup.position_x,
          y: step.position_y - parentGroup.position_y,
        }
      }
    }

    const previewClass = getPreviewClass(step.id)

    return {
      id: step.id,
      type: 'custom',
      position,
      parentNode: parentGroupId ? `group_${parentGroupId}` : undefined,
      class: previewClass || undefined,
      // Note: expandParent is intentionally NOT set - group should not auto-expand
      data: {
        label: step.name,
        type: step.type,
        step,
        isSelected: step.id === props.selectedStepId,
        stepRun,  // Include step run data if available
        inputPorts,   // Include input ports for multiple handles
        outputPorts,  // Include output ports for multiple handles
        icon: getStepIcon(step.type), // Icon for Miro-style display
        previewClass,  // Preview highlighting for Copilot changes
      },
    }
  })
})

// Combine group nodes and step nodes
const nodes = computed<Node[]>(() => {
  return [...groupNodes.value, ...stepNodes.value]
})

// Sync Vue Flow's internal state when steps/groups change (needed for Undo/Redo)
// Vue Flow maintains its own internal state that doesn't automatically sync with props
// We need to explicitly remove deleted nodes and update existing ones
function syncVueFlowNodes() {
  // Use nextTick to ensure Vue's reactivity has fully propagated
  nextTick(() => {
    const currentVueFlowNodes = getNodes.value
    const newNodes = nodes.value
    const newNodeIds = new Set(newNodes.map(n => n.id))

    console.log('[DagEditor] syncVueFlowNodes called:', {
      vueFlowNodeCount: currentVueFlowNodes.length,
      newNodeCount: newNodes.length,
      vueFlowIds: currentVueFlowNodes.map(n => n.id),
      newIds: newNodes.map(n => n.id),
    })

    // Find and remove nodes that no longer exist
    const nodesToRemove = currentVueFlowNodes.filter(n => !newNodeIds.has(n.id))
    if (nodesToRemove.length > 0) {
      console.log('[DagEditor] Removing nodes:', nodesToRemove.map(n => n.id))
      removeNodes(nodesToRemove)
    }

    // Update all nodes (this will add new ones and update existing ones)
    setNodes(newNodes)
  })
}

// Watch step IDs (more reliable than length for detecting add/remove)
watch(
  () => props.steps.map(s => s.id).join(','),
  (newIds, oldIds) => {
    console.log('[DagEditor] step IDs changed:', oldIds, '->', newIds)
    syncVueFlowNodes()
  }
)

// Watch group IDs
watch(
  () => (props.blockGroups || []).map(g => g.id).join(','),
  (newIds, oldIds) => {
    console.log('[DagEditor] group IDs changed:', oldIds, '->', newIds)
    syncVueFlowNodes()
  }
)

// Get edge color based on source port
function getEdgeColor(sourcePort?: string): string {
  if (!sourcePort) return '#94a3b8'

  const portColors: Record<string, string> = {
    // Condition ports
    'true': '#22c55e',      // green for Yes/True
    'false': '#ef4444',     // red for No/False
    // Human in loop
    'approved': '#22c55e',  // green
    'rejected': '#ef4444',  // red
    'timeout': '#f59e0b',   // amber
    // Filter
    'matched': '#22c55e',   // green
    'unmatched': '#94a3b8', // gray
    // Loop/Map
    'loop': '#3b82f6',      // blue
    'complete': '#22c55e',  // green
    'item': '#3b82f6',      // blue
    // Block Group output
    'out': '#22c55e',       // green
    // Default
    'default': '#94a3b8',   // gray
    'output': '#94a3b8',    // gray
  }

  return portColors[sourcePort] || '#6366f1' // indigo for custom cases
}

// Get edge label text
function getEdgeLabel(sourcePort?: string, condition?: string): string | undefined {
  if (condition) return condition
  if (!sourcePort || sourcePort === 'output') return undefined

  // Pretty labels for known ports
  const portLabels: Record<string, string> = {
    'true': 'Yes',
    'false': 'No',
    'approved': 'Approved',
    'rejected': 'Rejected',
    'timeout': 'Timeout',
    'matched': 'Matched',
    'unmatched': 'Unmatched',
    'loop': 'Loop',
    'complete': 'Complete',
    'item': 'Item',
    'out': 'Output',
    'default': 'Default',
  }

  return portLabels[sourcePort] || sourcePort
}

// Get edge data flow status based on step runs
function getEdgeFlowStatus(sourceStepId: string, targetStepId: string): { animated: boolean; status: 'idle' | 'flowing' | 'completed' | 'error' } {
  if (!props.stepRuns || props.stepRuns.length === 0) {
    return { animated: false, status: 'idle' }
  }

  const sourceRun = stepRunMap.value.get(sourceStepId)
  const targetRun = stepRunMap.value.get(targetStepId)

  // If source completed and target is running, data is flowing
  if (sourceRun?.status === 'completed' && targetRun?.status === 'running') {
    return { animated: true, status: 'flowing' }
  }

  // If both completed, data has flowed
  if (sourceRun?.status === 'completed' && targetRun?.status === 'completed') {
    return { animated: false, status: 'completed' }
  }

  // If target failed, mark as error
  if (targetRun?.status === 'failed') {
    return { animated: false, status: 'error' }
  }

  return { animated: false, status: 'idle' }
}

// Convert edges to Vue Flow edges
const flowEdges = computed<FlowEdge[]>(() => {
  const result: FlowEdge[] = []

  for (const edge of props.edges) {
    // Determine source: step ID or group ID (with 'group_' prefix for Vue Flow)
    const source = edge.source_step_id || (edge.source_block_group_id ? `group_${edge.source_block_group_id}` : '')
    const target = edge.target_step_id || (edge.target_block_group_id ? `group_${edge.target_block_group_id}` : '')

    // Skip edges with missing source/target
    if (!source || !target) {
      continue
    }

    const baseColor = getEdgeColor(edge.source_port)
    const flowStatus = edge.source_step_id && edge.target_step_id
      ? getEdgeFlowStatus(edge.source_step_id, edge.target_step_id)
      : { animated: false, status: 'idle' }

    // Override color based on flow status
    let color = baseColor
    let strokeWidth = 2
    if (flowStatus.status === 'flowing') {
      color = '#3b82f6' // blue for active flow
      strokeWidth = 3
    } else if (flowStatus.status === 'completed') {
      color = '#22c55e' // green for completed flow
    } else if (flowStatus.status === 'error') {
      color = '#ef4444' // red for error
    }

    // Use special styling for group edges
    const isGroupEdge = edge.source_block_group_id || edge.target_block_group_id
    if (isGroupEdge) {
      color = '#8b5cf6' // purple for group edges
    }

    // Check if this edge is selected (use same blue color as selected blocks)
    const isSelected = selectedEdgeId.value === edge.id
    if (isSelected) {
      color = '#3b82f6' // blue for selected edge (matches selected blocks)
      strokeWidth = 3
    }

    // Skip edge labels for edges involving group blocks (both from and to groups)
    const edgeLabel = isGroupEdge ? undefined : getEdgeLabel(edge.source_port, edge.condition)

    // Get preview class for Copilot changes
    const edgePreviewClass = getEdgePreviewClass(edge.id, source, target)

    result.push({
      id: edge.id,
      source,
      target,
      sourceHandle: edge.source_port || undefined, // Connect from specific output port
      targetHandle: edge.target_port || undefined, // Connect to specific input port
      type: 'smoothstep',
      animated: flowStatus.animated || isSelected,
      label: edgeLabel,
      labelBgStyle: { fill: 'white', fillOpacity: 0.9 },
      labelStyle: { fill: color, fontWeight: 500, fontSize: 11 },
      labelShowBg: true,
      style: { stroke: color, strokeWidth },
      markerEnd: { type: MarkerType.ArrowClosed, color },
      interactionWidth: 20, // Make edge easier to click
      class: edgePreviewClass || undefined,
      data: { isSelected, edgeId: edge.id },
    })
  }

  return result
})

// Check if a branching block (condition/switch) is inside a Block Group (reserved for future use)
function _isBranchingBlockInGroup(stepId: string): boolean {
  const step = props.steps.find(s => s.id === stepId)
  if (!step) return false

  // Only check condition and switch blocks
  if (step.type !== 'condition' && step.type !== 'switch') {
    return true // Non-branching blocks are always allowed
  }

  // Branching blocks must be in a Block Group if they have multiple outputs
  return !!step.block_group_id
}

// Get existing outgoing edges from a step
function getOutgoingEdgesFromStep(stepId: string): Edge[] {
  return props.edges.filter(e => e.source_step_id === stepId)
}

// Get existing outgoing edges from a group
function getOutgoingEdgesFromGroup(groupId: string): Edge[] {
  return props.edges.filter(e => e.source_block_group_id === groupId)
}

// Get default source port for a node (step or group)
function getDefaultSourcePort(nodeId: string, step: Step | undefined): string | undefined {
  if (step) {
    const outputPorts = getOutputPorts(step.type, step)
    return outputPorts.length > 0 ? outputPorts[0].name : undefined
  }
  if (nodeId?.startsWith(GROUP_NODE_PREFIX)) {
    const groupUuid = getGroupUuidFromNodeId(nodeId)
    const group = props.blockGroups?.find(g => g.id === groupUuid)
    if (group) {
      const groupPorts = getGroupOutputPorts(group.type)
      return groupPorts.length > 0 ? groupPorts[0].name : undefined
    }
  }
  return undefined
}

// Get default target port for a node (step or group)
// Groups use 'in' as their single input port
function getDefaultTargetPort(nodeId: string): string | undefined {
  const step = props.steps.find(s => s.id === nodeId)
  if (step) {
    const inputPorts = getInputPorts(step.type)
    return inputPorts.length > 0 ? inputPorts[0].name : undefined
  }
  if (nodeId?.startsWith(GROUP_NODE_PREFIX)) {
    // Groups have a single input port named 'in'
    return 'in'
  }
  return undefined
}

// Handle new connection
onConnect((params) => {
  if (!props.readonly) {
    const sourceNodeId = params.source
    const sourceStep = props.steps.find(s => s.id === sourceNodeId)

    // Check if source is a group
    const isSourceGroup = sourceNodeId?.startsWith(GROUP_NODE_PREFIX)

    // Delete existing outgoing edges before creating new one (auto-replace)
    // Exception: branching blocks inside Block Groups can have multiple outputs
    if (isSourceGroup) {
      // Source is a Block Group - delete existing outgoing edges
      const groupId = getGroupUuidFromNodeId(sourceNodeId)
      const existingEdges = getOutgoingEdgesFromGroup(groupId)
      for (const edge of existingEdges) {
        emit('edge:delete', edge.id)
      }
    } else if (sourceStep) {
      // Source is a step
      const isBranchingBlock = sourceStep.type === 'condition' || sourceStep.type === 'switch'
      const existingEdges = getOutgoingEdgesFromStep(sourceNodeId)

      // Branching blocks inside a Block Group are allowed multiple outputs
      const allowMultiple = isBranchingBlock && sourceStep.block_group_id

      if (!allowMultiple && existingEdges.length > 0) {
        // Delete existing edges before adding new one
        for (const edge of existingEdges) {
          emit('edge:delete', edge.id)
        }
      }
    }

    // sourceHandle/targetHandle contain the port names when connecting from/to specific ports
    // If not set, use the default (first) port of the source/target (supports both steps and groups)
    const sourcePort = params.sourceHandle || getDefaultSourcePort(sourceNodeId, sourceStep)
    const targetPort = params.targetHandle || getDefaultTargetPort(params.target)
    emit('edge:add', params.source, params.target, sourcePort, targetPort)
  }
})

// Handle edge click - select edge for deletion
onEdgeClick(({ edge, event }) => {
  if (!props.readonly) {
    // Find the actual edge ID from our edges array using source/target
    const actualEdge = props.edges.find(e => {
      // Match by VueFlow edge ID format
      const flowEdgeId = edge.id
      // VueFlow edge ID format: "source-target" or with handles
      return e.id === flowEdgeId || flowEdgeId.includes(e.id)
    })

    if (actualEdge && dagEditorRef.value) {
      selectedEdgeId.value = actualEdge.id
      // Store click position in flow coordinates
      // This allows the delete button to follow pan/zoom
      const mouseEvent = event as MouseEvent
      const rect = dagEditorRef.value.getBoundingClientRect()
      const flowPos = project({
        x: mouseEvent.clientX - rect.left,
        y: mouseEvent.clientY - rect.top,
      })
      edgeClickFlowPosition.value = {
        x: flowPos.x,
        y: flowPos.y,
      }
    }
  }
})

// Handle keyboard events for edge deletion
function handleKeyDown(event: KeyboardEvent) {
  if (props.readonly) return

  if ((event.key === 'Delete' || event.key === 'Backspace') && selectedEdgeId.value) {
    handleDeleteSelectedEdge()
    event.preventDefault()
  }
}

// Handle delete button click for selected edge
function handleDeleteSelectedEdge() {
  if (selectedEdgeId.value) {
    emit('edge:delete', selectedEdgeId.value)
    selectedEdgeId.value = null
    edgeClickFlowPosition.value = null
  }
}

// Clear edge selection when clicking on pane
onPaneClick(() => {
  selectedEdgeId.value = null
  edgeClickFlowPosition.value = null
  emit('pane:click')
})

// Handle node drag
onNodeDragStop((event) => {
  if (!props.readonly) {
    const node = event.node

    // Check if this is a group node
    if (node.type === 'group') {
      const groupData = node.data.group as BlockGroup
      let newX = snapToGrid(node.position.x)
      let newY = snapToGrid(node.position.y)

      // Calculate delta from original position
      const originalX = groupData.position_x
      const originalY = groupData.position_y

      // Get actual group dimensions (prefer Vue Flow dimensions, fallback to stored data)
      const groupWidth = node.dimensions?.width || groupData.width || DEFAULT_GROUP_WIDTH
      const groupHeight = node.dimensions?.height || groupData.height || DEFAULT_GROUP_HEIGHT

      // Save the drop position (before any snapping)
      const dropX = newX
      const dropY = newY

      // Result arrays
      const pushedBlocks: PushedBlock[] = []
      const addedBlocks: AddedBlock[] = []
      const movedGroups: MovedGroup[] = []

      // Track cascade direction - all pushes will use the same direction for consistency
      let cascadeDirection: PushDirection | null = null

      // =================================================================
      // STEP 1: At the DROP position, find blocks that are FULLY INSIDE
      //         and add them to the group BEFORE any snapping
      // =================================================================
      const dropPositionGroup: BlockGroup = {
        ...groupData,
        position_x: dropX,
        position_y: dropY,
        width: groupWidth,
        height: groupHeight,
      }

      // Valid inside area at drop position
      const dropInnerLeft = dropX + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
      const dropInnerRight = dropX + groupWidth - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
      const dropInnerTop = dropY + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
      const dropInnerBottom = dropY + groupHeight - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

      // Set to track which steps are added to the group (won't be pushed out later)
      const addedStepIds = new Set<string>()

      for (const step of props.steps) {
        // Skip steps already in this group
        // Note: step.block_group_id is plain UUID, node.id is "group_${uuid}" format
        if (step.block_group_id === groupData.id) continue
        // Skip start steps - they cannot be added to groups
        if (step.type === 'start') continue

        const stepWidth = DEFAULT_STEP_NODE_WIDTH
        const stepHeight = DEFAULT_STEP_NODE_HEIGHT

        const stepLeft = step.position_x
        const stepRight = step.position_x + stepWidth
        const stepTop = step.position_y
        const stepBottom = step.position_y + stepHeight

        // Check if step is FULLY inside the valid inner area at drop position
        const fullyInsideAtDrop =
          stepLeft >= dropInnerLeft && stepRight <= dropInnerRight &&
          stepTop >= dropInnerTop && stepBottom <= dropInnerBottom

        if (fullyInsideAtDrop) {
          // Add this block to the group
          const role = determineRoleInGroup(
            step.position_x + stepWidth / 2,
            step.position_y + stepHeight / 2,
            dropPositionGroup
          )

          // Store relative position (relative to drop position)
          const relativeX = step.position_x - dropX
          const relativeY = step.position_y - dropY

          addedBlocks.push({
            stepId: step.id,
            position: { x: step.position_x, y: step.position_y },  // Will be updated after snapping
            role,
          })
          addedStepIds.add(step.id)

          // Update Vue Flow state with relative position and parent
          updateNode(step.id, {
            position: { x: relativeX, y: relativeY },
            parentNode: node.id,
          })
        }
      }

      // =================================================================
      // STEP 2: Check for collision with other groups and snap if needed
      // =================================================================
      // Note: node.id is in format "group_${uuid}", but blockGroups use plain UUID
      const groupUuid = getGroupUuidFromNodeId(node.id)
      const dropZone = findDropZone(newX, newY, groupWidth, groupHeight, groupUuid)

      let wasGroupPushed = false
      let pushedAwayFromGroupId: string | null = null

      if (dropZone.zone === 'boundary' && dropZone.group) {
        // Snap group to outside the other group (groups can't be nested)
        const snapped = snapToValidPosition(newX, newY, dropZone.group, groupWidth, groupHeight)
        newX = snapped.x
        newY = snapped.y
        wasGroupPushed = true
        pushedAwayFromGroupId = dropZone.group.id

        // Set cascade direction based on the direction the group was pushed
        const pushDeltaX = newX - dropX
        const pushDeltaY = newY - dropY
        if (Math.abs(pushDeltaX) > Math.abs(pushDeltaY)) {
          cascadeDirection = pushDeltaX > 0 ? 'right' : 'left'
        } else {
          cascadeDirection = pushDeltaY > 0 ? 'down' : 'up'
        }

        // Update Vue Flow's internal position
        updateNode(node.id, { position: { x: newX, y: newY } })
      } else if (dropZone.zone === 'inside' && dropZone.group) {
        // Groups can't be nested - snap to outside (grid-aligned)
        const outsideGap = GRID_SIZE
        const edgeDistances = [
          { outX: snapToGrid(dropZone.group.position_x - groupWidth - outsideGap), outY: snapToGrid(newY) },
          { outX: snapToGrid(dropZone.group.position_x + dropZone.group.width + outsideGap), outY: snapToGrid(newY) },
          { outX: snapToGrid(newX), outY: snapToGrid(dropZone.group.position_y - groupHeight - outsideGap) },
          { outX: snapToGrid(newX), outY: snapToGrid(dropZone.group.position_y + dropZone.group.height + outsideGap) },
        ]
        let minDist = Infinity
        for (const pos of edgeDistances) {
          const dist = Math.sqrt(Math.pow(newX - pos.outX, 2) + Math.pow(newY - pos.outY, 2))
          if (dist < minDist) {
            minDist = dist
            newX = pos.outX
            newY = pos.outY
          }
        }
        wasGroupPushed = true
        pushedAwayFromGroupId = dropZone.group.id

        // Set cascade direction based on the direction the group was pushed
        const pushDeltaX = newX - dropX
        const pushDeltaY = newY - dropY
        if (Math.abs(pushDeltaX) > Math.abs(pushDeltaY)) {
          cascadeDirection = pushDeltaX > 0 ? 'right' : 'left'
        } else {
          cascadeDirection = pushDeltaY > 0 ? 'down' : 'up'
        }

        // Update Vue Flow's internal position
        updateNode(node.id, { position: { x: newX, y: newY } })
      }

      // =================================================================
      // STEP 3: If group was snapped, update the added blocks' absolute positions
      //         (they move WITH the group - their relative position stays the same)
      // =================================================================
      if (wasGroupPushed && addedBlocks.length > 0) {
        const snapDeltaX = newX - dropX
        const snapDeltaY = newY - dropY

        for (const addedBlock of addedBlocks) {
          // Update the absolute position to reflect the group's new position
          addedBlock.position.x += snapDeltaX
          addedBlock.position.y += snapDeltaY
        }
        // Note: Vue Flow positions don't need updating because they're relative to the group
      }

      // Track current group positions AND sizes (updated during cascade)
      const groupPositions = new Map<string, { x: number; y: number }>()
      const groupSizes = new Map<string, { width: number; height: number }>()
      // Initialize with the moving group's new position and size
      // Note: Use plain UUID (groupData.id) for consistency with other code that uses group.id
      groupPositions.set(groupData.id, { x: newX, y: newY })
      groupSizes.set(groupData.id, { width: groupWidth, height: groupHeight })

      // Create a temporary group object with the new position for boundary checking
      const movedGroup: BlockGroup = {
        ...groupData,
        position_x: newX,
        position_y: newY,
        width: groupWidth,
        height: groupHeight,
      }

      // Set of groups that have been processed (to avoid infinite loops)
      // Note: Use plain UUID (groupData.id) for consistency with other code that uses group.id
      const processedGroups = new Set<string>([groupData.id])
      // Also exclude the group we were pushed away from (if any)
      if (pushedAwayFromGroupId) {
        processedGroups.add(pushedAwayFromGroupId)
      }

      // Queue of groups that need to be pushed (for cascading)
      interface GroupToPush {
        group: BlockGroup
        pusherX: number
        pusherY: number
        pusherWidth: number
        pusherHeight: number
      }
      const groupsToPush: GroupToPush[] = []

      // =================================================================
      // STEP 4: Check remaining blocks for boundary collision and push them out
      //         (blocks already added in STEP 1 are skipped)
      // =================================================================
      for (const step of props.steps) {
        // Skip steps that are already inside this group
        // Note: step.block_group_id is plain UUID, groupData.id is also plain UUID
        if (step.block_group_id === groupData.id) continue

        // Skip steps that were added in STEP 1 (at drop position)
        if (addedStepIds.has(step.id)) continue

        // Skip start steps - they cannot be added to groups
        if (step.type === 'start') continue

        // Get step dimensions (use defaults)
        const stepWidth = DEFAULT_STEP_NODE_WIDTH
        const stepHeight = DEFAULT_STEP_NODE_HEIGHT

        // Check if this step overlaps with the moved group's boundary (at FINAL position)
        const stepLeft = step.position_x
        const stepRight = step.position_x + stepWidth
        const stepTop = step.position_y
        const stepBottom = step.position_y + stepHeight

        // Group bounds (at final position after snapping)
        const groupLeft = movedGroup.position_x
        const groupRight = movedGroup.position_x + movedGroup.width
        const groupTop = movedGroup.position_y
        const groupBottom = movedGroup.position_y + movedGroup.height

        // Check if step overlaps with group's outer bounds
        const overlapsGroup =
          stepRight > groupLeft && stepLeft < groupRight &&
          stepBottom > groupTop && stepTop < groupBottom

        if (!overlapsGroup) continue

        // Check if step is fully inside the valid inner area at final position
        const innerLeft = movedGroup.position_x + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
        const innerRight = movedGroup.position_x + movedGroup.width - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
        const innerTop = movedGroup.position_y + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
        const innerBottom = movedGroup.position_y + movedGroup.height - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

        const fullyInsideAtFinal =
          stepLeft >= innerLeft && stepRight <= innerRight &&
          stepTop >= innerTop && stepBottom <= innerBottom

        // If block is on the boundary at final position, push it out
        // If block is fully inside at final position but wasn't at drop position, also push it out
        if (!fullyInsideAtFinal) {
          // Push to outside - use cascade direction for consistency
          const pushResult = calculatePushPosition(
            step.position_x, step.position_y, stepWidth, stepHeight,
            movedGroup.position_x, movedGroup.position_y, movedGroup.width, movedGroup.height,
            cascadeDirection ?? undefined  // Use cascade direction if set, otherwise auto-detect
          )

          // Set cascade direction if this is the first push
          if (!cascadeDirection) {
            cascadeDirection = pushResult.direction
          }

          pushedBlocks.push({
            stepId: step.id,
            position: { x: pushResult.x, y: pushResult.y },
          })

          // Update Vue Flow's internal state for the pushed block
          let relativeX = pushResult.x
          let relativeY = pushResult.y
          if (step.block_group_id && props.blockGroups) {
            const stepGroup = props.blockGroups.find(g => g.id === step.block_group_id)
            if (stepGroup) {
              const stepGroupPos = groupPositions.get(stepGroup.id) || { x: stepGroup.position_x, y: stepGroup.position_y }
              relativeX = pushResult.x - stepGroupPos.x
              relativeY = pushResult.y - stepGroupPos.y
            }
          }
          updateNode(step.id, { position: { x: relativeX, y: relativeY } })

          // Check if pushed block now collides with another group's boundary
          const collidingGroup = findGroupBoundaryCollision(
            pushResult.x, pushResult.y, stepWidth, stepHeight,
            processedGroups, groupPositions
          )
          if (collidingGroup) {
            groupsToPush.push({
              group: collidingGroup,
              pusherX: pushResult.x,
              pusherY: pushResult.y,
              pusherWidth: stepWidth,
              pusherHeight: stepHeight,
            })
          }
        }
      }

      // Check if the moved group collides with any other group (group-to-group collision)
      const collidingGroupInitial = findGroupCollision(
        newX, newY, groupWidth, groupHeight,
        processedGroups, groupPositions, groupSizes
      )
      if (collidingGroupInitial) {
        groupsToPush.push({
          group: collidingGroupInitial,
          pusherX: newX,
          pusherY: newY,
          pusherWidth: groupWidth,
          pusherHeight: groupHeight,
        })
      }

      // Process cascading group pushes
      const MAX_CASCADE_DEPTH = 10  // Safety limit
      let cascadeDepth = 0

      while (groupsToPush.length > 0 && cascadeDepth < MAX_CASCADE_DEPTH) {
        cascadeDepth++
        const { group, pusherX, pusherY, pusherWidth, pusherHeight } = groupsToPush.shift()!

        // Skip if already processed
        if (processedGroups.has(group.id)) continue
        processedGroups.add(group.id)

        // Get current position and size of the group
        const currentPos = groupPositions.get(group.id) || { x: group.position_x, y: group.position_y }
        const currentSize = groupSizes.get(group.id) || { width: group.width, height: group.height }

        // Create temporary group with current position and size
        const _tempGroup: BlockGroup = {
          ...group,
          position_x: currentPos.x,
          position_y: currentPos.y,
        }

        // Calculate new position for the group to avoid collision
        // Use cascade direction for consistency (if already set)
        const pushed = calculatePushPosition(
          currentPos.x, currentPos.y, currentSize.width, currentSize.height,
          pusherX, pusherY, pusherWidth, pusherHeight,
          cascadeDirection ?? undefined
        )

        // Set cascade direction if this is the first push
        if (!cascadeDirection) {
          cascadeDirection = pushed.direction
        }

        // Update group position and size tracking
        groupPositions.set(group.id, { x: pushed.x, y: pushed.y })
        groupSizes.set(group.id, currentSize)  // Keep the same size

        // Add to movedGroups
        movedGroups.push({
          groupId: group.id,
          position: { x: pushed.x, y: pushed.y },
          delta: { x: pushed.deltaX, y: pushed.deltaY },
        })

        // Update Vue Flow's internal state (use prefixed ID to match groupNodes)
        updateNode(`group_${group.id}`, { position: { x: pushed.x, y: pushed.y } })

        // Check if the pushed group now causes any block collisions
        // (blocks inside the pushed group will move with it, but we need to check external blocks)
        for (const step of props.steps) {
          // Skip steps inside this group (they move with the group)
          if (step.block_group_id === group.id) continue
          // Skip start steps
          if (step.type === 'start') continue

          const stepWidth = DEFAULT_STEP_NODE_WIDTH
          const stepHeight = DEFAULT_STEP_NODE_HEIGHT

          // Get step's current position (may have been pushed already)
          const existingPush = pushedBlocks.find(p => p.stepId === step.id)
          const stepX = existingPush ? existingPush.position.x : step.position_x
          const stepY = existingPush ? existingPush.position.y : step.position_y

          // Check if step collides with the pushed group's boundary
          // Use currentSize for accurate dimensions
          const _pushedGroupTemp: BlockGroup = {
            ...group,
            position_x: pushed.x,
            position_y: pushed.y,
            width: currentSize.width,
            height: currentSize.height,
          }

          // Check overlap with pushed group's outer bounds
          const outerLeft = pushed.x
          const outerRight = pushed.x + currentSize.width
          const outerTop = pushed.y
          const outerBottom = pushed.y + currentSize.height

          const overlapsGroup =
            (stepX + stepWidth) > outerLeft && stepX < outerRight &&
            (stepY + stepHeight) > outerTop && stepY < outerBottom

          if (!overlapsGroup) continue

          // Check if fully inside
          const innerLeft = pushed.x + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
          const innerRight = pushed.x + currentSize.width - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
          const innerTop = pushed.y + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
          const innerBottom = pushed.y + currentSize.height - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

          const fullyInside =
            stepX >= innerLeft && (stepX + stepWidth) <= innerRight &&
            stepY >= innerTop && (stepY + stepHeight) <= innerBottom

          if (!fullyInside) {
            // This step is on the pushed group's boundary - push it using cascade direction
            const blockPushResult = calculatePushPosition(
              stepX, stepY, stepWidth, stepHeight,
              pushed.x, pushed.y, currentSize.width, currentSize.height,
              cascadeDirection ?? undefined
            )

            // Set cascade direction if not already set
            if (!cascadeDirection) {
              cascadeDirection = blockPushResult.direction
            }

            // Update or add to pushedBlocks
            const existingIndex = pushedBlocks.findIndex(p => p.stepId === step.id)
            if (existingIndex >= 0) {
              pushedBlocks[existingIndex].position = { x: blockPushResult.x, y: blockPushResult.y }
            } else {
              pushedBlocks.push({
                stepId: step.id,
                position: { x: blockPushResult.x, y: blockPushResult.y },
              })
            }

            // Update Vue Flow
            let relativeX = blockPushResult.x
            let relativeY = blockPushResult.y
            if (step.block_group_id && props.blockGroups) {
              const stepGroup = props.blockGroups.find(g => g.id === step.block_group_id)
              if (stepGroup) {
                const stepGroupPos = groupPositions.get(stepGroup.id) || { x: stepGroup.position_x, y: stepGroup.position_y }
                relativeX = blockPushResult.x - stepGroupPos.x
                relativeY = blockPushResult.y - stepGroupPos.y
              }
            }
            updateNode(step.id, { position: { x: relativeX, y: relativeY } })

            // Check for further group collisions
            const nextCollision = findGroupBoundaryCollision(
              blockPushResult.x, blockPushResult.y, stepWidth, stepHeight,
              processedGroups, groupPositions
            )
            if (nextCollision) {
              groupsToPush.push({
                group: nextCollision,
                pusherX: blockPushResult.x,
                pusherY: blockPushResult.y,
                pusherWidth: stepWidth,
                pusherHeight: stepHeight,
              })
            }
          }
        }

        // Check if the pushed group collides with another group (group-to-group cascade)
        const nextGroupCollision = findGroupCollision(
          pushed.x, pushed.y, currentSize.width, currentSize.height,
          processedGroups, groupPositions, groupSizes
        )
        if (nextGroupCollision) {
          groupsToPush.push({
            group: nextGroupCollision,
            pusherX: pushed.x,
            pusherY: pushed.y,
            pusherWidth: currentSize.width,
            pusherHeight: currentSize.height,
          })
        }
      }

      // Recalculate delta after any snapping
      const finalDeltaX = newX - originalX
      const finalDeltaY = newY - originalY

      // Emit the move complete event with all the data (use plain UUID for parent)
      emit('group:move-complete', getGroupUuidFromNodeId(node.id), {
        position: { x: newX, y: newY },
        delta: { x: finalDeltaX, y: finalDeltaY },
        pushedBlocks,
        addedBlocks,
        movedGroups,
      })
    } else {
      // For step nodes, calculate absolute position if they have a parent
      let absoluteX = snapToGrid(node.position.x)
      let absoluteY = snapToGrid(node.position.y)

      // Get actual node dimensions (use defaults if not available)
      const nodeWidth = node.dimensions?.width ?? DEFAULT_STEP_NODE_WIDTH
      const nodeHeight = node.dimensions?.height ?? DEFAULT_STEP_NODE_HEIGHT

      if (node.parentNode && props.blockGroups) {
        const parentGroupUuid = getGroupUuidFromNodeId(node.parentNode)
        const parentGroup = props.blockGroups.find(g => g.id === parentGroupUuid)
        if (parentGroup) {
          absoluteX += parentGroup.position_x
          absoluteY += parentGroup.position_y
        }
      }

      // Check drop zone (inside, boundary, or outside) using actual node dimensions
      const dropZone = findDropZone(absoluteX, absoluteY, nodeWidth, nodeHeight)
      const step = node.data.step as Step
      const currentGroupId = step.block_group_id || null

      // Handle boundary zone - snap to valid position
      if (dropZone.zone === 'boundary' && dropZone.group) {
        const snapped = snapToValidPosition(absoluteX, absoluteY, dropZone.group, nodeWidth, nodeHeight)
        absoluteX = snapped.x
        absoluteY = snapped.y

        // Calculate relative position for Vue Flow (relative to parent if inside group)
        let relativeX = absoluteX
        let relativeY = absoluteY
        if (snapped.inside) {
          relativeX = absoluteX - dropZone.group.position_x
          relativeY = absoluteY - dropZone.group.position_y
        }

        // Update Vue Flow's internal node position to the snapped position
        updateNode(node.id, {
          position: { x: relativeX, y: relativeY },
          parentNode: snapped.inside ? getNodeIdFromGroupUuid(dropZone.group.id) : undefined,
        })

        const targetGroupId = snapped.inside ? dropZone.group.id : null
        const role = snapped.inside ? determineRoleInGroup(absoluteX, absoluteY, dropZone.group) : undefined

        // If block was pushed outside, check for cascade group collisions
        let movedGroups: MovedGroup[] = []
        if (!snapped.inside) {
          movedGroups = processCascadeGroupPush(
            absoluteX, absoluteY, nodeWidth, nodeHeight,
            new Set([dropZone.group.id])  // Exclude the group we were pushed from
          )
        }

        if (currentGroupId !== targetGroupId) {
          emit('step:assign-group', node.id, targetGroupId, { x: absoluteX, y: absoluteY }, role, movedGroups.length > 0 ? movedGroups : undefined)
        } else {
          emit('step:update', node.id, { x: absoluteX, y: absoluteY }, movedGroups.length > 0 ? movedGroups : undefined)
        }
      } else {
        // Normal handling - inside or outside
        const targetGroupId = dropZone.zone === 'inside' ? dropZone.group?.id || null : null
        const role = dropZone.zone === 'inside' ? dropZone.role : undefined

        if (currentGroupId !== targetGroupId) {
          // Parent is changing - update Vue Flow's internal state
          let relativeX = absoluteX
          let relativeY = absoluteY
          if (targetGroupId && dropZone.group) {
            relativeX = absoluteX - dropZone.group.position_x
            relativeY = absoluteY - dropZone.group.position_y
          }
          updateNode(node.id, {
            position: { x: relativeX, y: relativeY },
            parentNode: targetGroupId ? getNodeIdFromGroupUuid(targetGroupId) : undefined,
          })
          emit('step:assign-group', node.id, targetGroupId, { x: absoluteX, y: absoluteY }, role)
        } else {
          emit('step:update', node.id, { x: absoluteX, y: absoluteY })
        }
      }
    }
  }
})

// Note: pane click handler is defined above (with edge selection clearing)

// Handle node click
function onNodeClick(event: { node: Node }) {
  // Clear edge selection when clicking on a node
  selectedEdgeId.value = null
  edgeClickFlowPosition.value = null

  // Check if this is a group node
  if (event.node.type === 'group') {
    emit('group:select', event.node.data.group)
  } else {
    emit('step:select', event.node.data.step)
    // If step run data exists, also emit step details event
    if (event.node.data.stepRun) {
      emit('step:showDetails', event.node.data.stepRun)
    }
  }
}

// Get step run status color
function getStepRunStatusColor(status?: string): string {
  if (!status) return ''
  const colors: Record<string, string> = {
    pending: '#f59e0b',  // amber
    running: '#3b82f6',  // blue
    completed: '#22c55e', // green
    failed: '#ef4444',   // red
    skipped: '#94a3b8',  // gray
  }
  return colors[status] || '#94a3b8'
}

// Get step run status icon (unused in Miro design but kept for potential future use)
function _getStepRunStatusIcon(status?: string): string {
  if (!status) return ''
  const icons: Record<string, string> = {
    pending: '○',
    running: '●',
    completed: '✓',
    failed: '✕',
    skipped: '−',
  }
  return icons[status] || ''
}

// Drag and drop handlers
const dragEnterCounter = ref(0)

function handleDragEnter(event: DragEvent) {
  if (props.readonly) return
  event.preventDefault()
  dragEnterCounter.value++
  isDragOver.value = true
}

function handleDragOver(event: DragEvent) {
  if (props.readonly) return

  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'copy'
  }
}

function handleDragLeave(event: DragEvent) {
  if (props.readonly) return
  event.preventDefault()
  dragEnterCounter.value--
  // Only set isDragOver to false when we've left the root element
  if (dragEnterCounter.value <= 0) {
    dragEnterCounter.value = 0
    isDragOver.value = false
  }
}

function handleDrop(event: DragEvent) {
  if (props.readonly) return

  event.preventDefault()
  dragEnterCounter.value = 0
  isDragOver.value = false

  if (!event.dataTransfer) return

  // Get the drop position relative to the viewport
  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()

  // Convert screen coordinates to flow coordinates using Vue Flow's project function
  const flowPosition = project({
    x: event.clientX - rect.left,
    y: event.clientY - rect.top,
  })

  // Check if this is a block group drop
  const groupType = event.dataTransfer.getData('group-type') as BlockGroupType
  const groupName = event.dataTransfer.getData('group-name')

  if (groupType) {
    // Position for group (larger than step nodes), snapped to grid
    const position = {
      x: snapToGrid(flowPosition.x - 200),
      y: snapToGrid(flowPosition.y - 150),
    }
    emit('group:drop', { type: groupType, name: groupName || 'New Group', position })
    return
  }

  // Handle step drop
  const stepType = event.dataTransfer.getData('step-type') as StepType
  const stepName = event.dataTransfer.getData('step-name') || 'New Step'

  if (!stepType) return

  // Center the node at the drop position (node width ~150px, height ~60px), snapped to grid
  let positionX = snapToGrid(flowPosition.x - 75)
  let positionY = snapToGrid(flowPosition.y - 30)

  // Check drop zone with boundary detection
  const dropZone = findDropZone(positionX, positionY)
  let targetGroupId: string | undefined
  let targetRole: GroupRole | undefined

  if (dropZone.zone === 'boundary' && dropZone.group) {
    // Snap to valid position
    const snapped = snapToValidPosition(positionX, positionY, dropZone.group)
    positionX = snapped.x
    positionY = snapped.y
    targetGroupId = snapped.inside ? dropZone.group.id : undefined
    if (snapped.inside) {
      targetRole = determineRoleInGroup(positionX, positionY, dropZone.group)
    }
  } else if (dropZone.zone === 'inside' && dropZone.group) {
    targetGroupId = dropZone.group.id
    targetRole = dropZone.role
  }

  emit('step:drop', {
    type: stepType,
    name: stepName,
    position: { x: positionX, y: positionY },
    groupId: targetGroupId,
    groupRole: targetRole,
  })
}

// Get port handle color based on port name
function getPortColor(portName: string): string {
  const portColors: Record<string, string> = {
    // Condition ports
    'true': '#22c55e',      // green for Yes/True
    'false': '#ef4444',     // red for No/False
    // Human in loop
    'approved': '#22c55e',  // green
    'rejected': '#ef4444',  // red
    'timeout': '#f59e0b',   // amber
    // Filter
    'matched': '#22c55e',   // green
    'unmatched': '#94a3b8', // gray
    // Loop/Map
    'loop': '#3b82f6',      // blue
    'complete': '#22c55e',  // green
    'item': '#3b82f6',      // blue
    // Block Group output
    'out': '#22c55e',       // green
    // Default
    'default': '#94a3b8',   // gray
    'output': '#94a3b8',    // gray
    'input': '#94a3b8',     // gray
  }

  return portColors[portName] || '#6366f1' // indigo for custom cases
}

// Get step type color
function getStepColor(type: string) {
  const colors: Record<string, string> = {
    start: '#10b981', // Emerald - distinct entry point color
    llm: '#3b82f6',
    tool: '#22c55e',
    condition: '#f59e0b',
    switch: '#eab308',
    map: '#8b5cf6',
    join: '#6366f1',
    subflow: '#ec4899',
    loop: '#14b8a6',
    wait: '#64748b',
    function: '#f97316',
    router: '#a855f7',
    human_in_loop: '#ef4444',
    filter: '#06b6d4',
    split: '#0ea5e9',
    aggregate: '#0284c7',
    error: '#dc2626',
    note: '#9ca3af',
  }
  return colors[type] || '#64748b'
}

// Get step icon based on type
function getStepIcon(type: string): string {
  // Use blockDefinitions if available to get custom icon
  if (props.blockDefinitions) {
    const blockDef = props.blockDefinitions.find(b => b.slug === type)
    if (blockDef?.icon) {
      return blockDef.icon
    }
  }
  // Fall back to default icon mapping
  return getBlockIcon(type)
}

// Check if step type is a trigger block (start, schedule_trigger, webhook_trigger等)
const triggerBlockTypes = ['start', 'schedule_trigger', 'webhook_trigger']
function isStartNode(type: string): boolean {
  return triggerBlockTypes.includes(type)
}

// Minimum size for group nodes (Miro-style: fits one 80x70 node with padding)
const MIN_GROUP_WIDTH = 160
const MIN_GROUP_HEIGHT = 150

// Track resize state for position compensation
interface ResizeState {
  groupId: string
  initialGroupX: number
  initialGroupY: number
  // Map of stepId -> initial relative position
  initialChildPositions: Map<string, { relX: number; relY: number }>
}
const resizeState = ref<ResizeState | null>(null)

// Handle group resize start - record initial positions
function onGroupResizeStart(nodeId: string, _event: OnResizeStart) {
  if (props.readonly) return

  // Convert Vue Flow node ID to plain group UUID
  const groupUuid = getGroupUuidFromNodeId(nodeId)
  const group = props.blockGroups?.find(g => g.id === groupUuid)
  if (!group) return

  // Record initial child positions (relative to group)
  const childPositions = new Map<string, { relX: number; relY: number }>()
  for (const step of props.steps) {
    if (step.block_group_id !== groupUuid) continue
    // Calculate relative position from absolute position
    const relX = step.position_x - group.position_x
    const relY = step.position_y - group.position_y
    childPositions.set(step.id, { relX, relY })
  }

  resizeState.value = {
    groupId: nodeId,
    initialGroupX: group.position_x,
    initialGroupY: group.position_y,
    initialChildPositions: childPositions,
  }
}

// Handle group resize (during drag) - compensate child positions in real-time
function onGroupResize(nodeId: string, event: OnResize) {
  if (props.readonly) return
  if (!resizeState.value || resizeState.value.groupId !== nodeId) return

  const currentX = event.params.x
  const currentY = event.params.y

  // Calculate delta from initial position
  const deltaX = currentX - resizeState.value.initialGroupX
  const deltaY = currentY - resizeState.value.initialGroupY

  // If no position change, no need to compensate
  if (deltaX === 0 && deltaY === 0) return

  // Compensate each child's position to maintain absolute position
  for (const [stepId, initialPos] of resizeState.value.initialChildPositions) {
    // New relative position = initial relative - delta
    const newRelX = initialPos.relX - deltaX
    const newRelY = initialPos.relY - deltaY

    updateNode(stepId, {
      position: { x: newRelX, y: newRelY },
    })
  }
}

// Handle group resize end with node context
function onGroupResizeEnd(nodeId: string, event: OnResizeEnd) {
  // Clear resize state
  resizeState.value = null
  if (props.readonly) return

  // Convert Vue Flow node ID to plain group UUID for API calls
  const groupUuid = getGroupUuidFromNodeId(nodeId)

  const newX = Math.round(event.params.x)
  const newY = Math.round(event.params.y)
  const newWidth = Math.round(event.params.width)
  const newHeight = Math.round(event.params.height)

  // Find the original group data
  const group = props.blockGroups?.find(g => g.id === groupUuid)
  if (!group) {
    // Fallback: just emit size update
    emit('group:update', groupUuid, {
      size: { width: newWidth, height: newHeight }
    })
    return
  }

  // Result arrays
  const pushedBlocks: PushedBlock[] = []
  const addedBlocks: AddedBlock[] = []
  const movedGroups: MovedGroup[] = []

  // Create a virtual group with new dimensions for calculations
  const resizedGroup: BlockGroup = {
    ...group,
    position_x: newX,
    position_y: newY,
    width: newWidth,
    height: newHeight,
  }

  // Valid inside area bounds for the resized group
  const innerLeft = newX + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
  const innerRight = newX + newWidth - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
  const innerTop = newY + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
  const innerBottom = newY + newHeight - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

  // Set to track which steps are processed
  const processedStepIds = new Set<string>()

  // =================================================================
  // STEP 1: Compensate internal blocks for group position change
  //         and check if they need to be pushed out
  // =================================================================
  for (const step of props.steps) {
    if (step.block_group_id !== groupUuid) continue
    if (step.type === 'start') continue

    const stepWidth = DEFAULT_STEP_NODE_WIDTH
    const stepHeight = DEFAULT_STEP_NODE_HEIGHT

    // Step's absolute position (stored in props)
    const stepAbsX = step.position_x
    const stepAbsY = step.position_y

    const stepLeft = stepAbsX
    const stepRight = stepAbsX + stepWidth
    const stepTop = stepAbsY
    const stepBottom = stepAbsY + stepHeight

    // Check if block is fully inside the valid inner area
    const fullyInside =
      stepLeft >= innerLeft && stepRight <= innerRight &&
      stepTop >= innerTop && stepBottom <= innerBottom

    // Check if block is fully outside the group (no overlap at all)
    const fullyOutside =
      stepRight <= newX || stepLeft >= newX + newWidth ||
      stepBottom <= newY || stepTop >= newY + newHeight

    // Check if block overlaps with group boundary (partially inside/outside)
    const onBoundary = !fullyInside && !fullyOutside

    if (onBoundary) {
      // Block overlaps with boundary - push out to nearest outside position (grid-aligned)
      const outsideGap = GRID_SIZE
      const edgeDistances = [
        { outX: snapToGrid(newX - stepWidth - outsideGap), outY: snapToGrid(stepAbsY) },
        { outX: snapToGrid(newX + newWidth + outsideGap), outY: snapToGrid(stepAbsY) },
        { outX: snapToGrid(stepAbsX), outY: snapToGrid(newY - stepHeight - outsideGap) },
        { outX: snapToGrid(stepAbsX), outY: snapToGrid(newY + newHeight + outsideGap) },
      ]

      // Find closest outside position based on current position
      let bestPos = edgeDistances[0]
      let bestDist = Infinity
      for (const pos of edgeDistances) {
        const dist = Math.sqrt(Math.pow(stepAbsX - pos.outX, 2) + Math.pow(stepAbsY - pos.outY, 2))
        if (dist < bestDist) {
          bestDist = dist
          bestPos = pos
        }
      }

      pushedBlocks.push({
        stepId: step.id,
        position: { x: bestPos.outX, y: bestPos.outY },
      })

      // Update Vue Flow state - remove from parent (use absolute position)
      updateNode(step.id, {
        position: { x: bestPos.outX, y: bestPos.outY },
        parentNode: undefined,
      })

      processedStepIds.add(step.id)
    } else if (fullyOutside) {
      // Block is fully outside - just remove from group, keep its absolute position
      pushedBlocks.push({
        stepId: step.id,
        position: { x: stepAbsX, y: stepAbsY },
      })

      // Update Vue Flow state - remove from parent, keep absolute position
      updateNode(step.id, {
        position: { x: stepAbsX, y: stepAbsY },
        parentNode: undefined,
      })

      processedStepIds.add(step.id)
    } else {
      // Block is fully inside - position already compensated by onGroupResize
      // No need to update position again
      processedStepIds.add(step.id)
    }
  }

  // =================================================================
  // STEP 2: Check blocks NOT in this group - add if now fully inside
  // =================================================================
  for (const step of props.steps) {
    // Skip if already in this group or already processed
    if (step.block_group_id === groupUuid) continue
    if (processedStepIds.has(step.id)) continue
    if (step.type === 'start') continue

    const stepWidth = DEFAULT_STEP_NODE_WIDTH
    const stepHeight = DEFAULT_STEP_NODE_HEIGHT

    const stepLeft = step.position_x
    const stepRight = step.position_x + stepWidth
    const stepTop = step.position_y
    const stepBottom = step.position_y + stepHeight

    // Check if block is now fully inside the valid inner area
    const fullyInside =
      stepLeft >= innerLeft && stepRight <= innerRight &&
      stepTop >= innerTop && stepBottom <= innerBottom

    if (fullyInside) {
      // Determine role based on position
      const role = determineRoleInGroup(
        step.position_x + stepWidth / 2,
        step.position_y + stepHeight / 2,
        resizedGroup
      )

      addedBlocks.push({
        stepId: step.id,
        position: { x: step.position_x, y: step.position_y },
        role,
      })

      // Update Vue Flow state - set parent
      const relativeX = step.position_x - newX
      const relativeY = step.position_y - newY
      updateNode(step.id, {
        position: { x: relativeX, y: relativeY },
        parentNode: nodeId,
      })

      processedStepIds.add(step.id)
    }
  }

  // =================================================================
  // STEP 3: Check for collision with other groups and push them away
  // =================================================================
  const processedGroups = new Set<string>([groupUuid])
  const groupPositions = new Map<string, { x: number; y: number }>()
  const groupSizes = new Map<string, { width: number; height: number }>()

  // Initialize group positions and sizes
  if (props.blockGroups) {
    for (const g of props.blockGroups) {
      if (g.id === groupUuid) {
        groupPositions.set(g.id, { x: newX, y: newY })
        groupSizes.set(g.id, { width: newWidth, height: newHeight })
      } else {
        groupPositions.set(g.id, { x: g.position_x, y: g.position_y })
        groupSizes.set(g.id, { width: g.width, height: g.height })
      }
    }
  }

  // Check for collisions with the resized group
  const collidingGroup = findGroupCollision(
    newX, newY, newWidth, newHeight,
    processedGroups, groupPositions, groupSizes
  )

  if (collidingGroup) {
    // Push the colliding group away
    const collidingPos = groupPositions.get(collidingGroup.id) || { x: collidingGroup.position_x, y: collidingGroup.position_y }
    const collidingSize = groupSizes.get(collidingGroup.id) || { width: collidingGroup.width, height: collidingGroup.height }

    const pushResult = calculatePushPosition(
      collidingPos.x, collidingPos.y, collidingSize.width, collidingSize.height,
      newX, newY, newWidth, newHeight
    )

    movedGroups.push({
      groupId: collidingGroup.id,
      position: { x: pushResult.x, y: pushResult.y },
      delta: { x: pushResult.deltaX, y: pushResult.deltaY },
    })

    // Update Vue Flow state (use prefixed ID to match groupNodes)
    updateNode(`group_${collidingGroup.id}`, { position: { x: pushResult.x, y: pushResult.y } })

    // Continue cascade check
    processedGroups.add(collidingGroup.id)
    groupPositions.set(collidingGroup.id, { x: pushResult.x, y: pushResult.y })

    // Process cascade (similar to processCascadeGroupPush but simplified)
    const cascadeDirection = pushResult.direction
    let currentPos = { x: pushResult.x, y: pushResult.y }
    let currentSize = collidingSize
    let cascadeDepth = 0
    const MAX_CASCADE = 10

    while (cascadeDepth < MAX_CASCADE) {
      cascadeDepth++
      const nextCollision = findGroupCollision(
        currentPos.x, currentPos.y, currentSize.width, currentSize.height,
        processedGroups, groupPositions, groupSizes
      )

      if (!nextCollision) break

      const nextPos = groupPositions.get(nextCollision.id) || { x: nextCollision.position_x, y: nextCollision.position_y }
      const nextSize = groupSizes.get(nextCollision.id) || { width: nextCollision.width, height: nextCollision.height }

      const nextPush = calculatePushPosition(
        nextPos.x, nextPos.y, nextSize.width, nextSize.height,
        currentPos.x, currentPos.y, currentSize.width, currentSize.height,
        cascadeDirection
      )

      movedGroups.push({
        groupId: nextCollision.id,
        position: { x: nextPush.x, y: nextPush.y },
        delta: { x: nextPush.deltaX, y: nextPush.deltaY },
      })

      // Use prefixed ID to match groupNodes
      updateNode(`group_${nextCollision.id}`, { position: { x: nextPush.x, y: nextPush.y } })
      processedGroups.add(nextCollision.id)
      groupPositions.set(nextCollision.id, { x: nextPush.x, y: nextPush.y })

      currentPos = { x: nextPush.x, y: nextPush.y }
      currentSize = nextSize
    }
  }

  // First, emit group:update to persist the new size and position (use plain UUID)
  emit('group:update', groupUuid, {
    position: { x: newX, y: newY },
    size: { width: newWidth, height: newHeight },
  })

  // Then emit the resize complete event with all the collision/push changes (use plain UUID)
  emit('group:resize-complete', groupUuid, {
    position: { x: newX, y: newY },
    size: { width: newWidth, height: newHeight },
    pushedBlocks,
    addedBlocks,
    movedGroups,
  })
}

// Expose zoom functions and viewport for parent components
defineExpose({
  viewport,
  zoomIn,
  zoomOut,
  zoomTo,
})
</script>

<template>
  <div
    ref="dagEditorRef"
    :class="['dag-editor', { 'drag-over': isDragOver }]"
    tabindex="0"
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
    @keydown="handleKeyDown"
  >
    <VueFlow
      :nodes="nodes"
      :edges="flowEdges"
      :default-viewport="{ zoom: 1, x: 50, y: 50 }"
      :default-edge-options="{
        type: 'smoothstep',
        animated: true,
        style: { strokeWidth: 2, stroke: '#94a3b8' },
        markerEnd: { type: MarkerType.ArrowClosed, color: '#94a3b8' }
      }"
      :min-zoom="0.25"
      :max-zoom="2"
      fit-view-on-init
      :snap-to-grid="true"
      :snap-grid="[20, 20]"
      @node-click="onNodeClick"
    >
      <!-- Block Group Node Template -->
      <template #node-group="{ data, id }">
        <!-- Resizer for group nodes (only in edit mode) -->
        <NodeResizer
          v-if="!readonly"
          :min-width="MIN_GROUP_WIDTH"
          :min-height="MIN_GROUP_HEIGHT"
          :color="data.color"
          :handle-class-name="'dag-group-resize-handle'"
          :line-class-name="'dag-group-resize-line'"
          @resize-start="(event: OnResizeStart) => onGroupResizeStart(id, event)"
          @resize="(event: OnResize) => onGroupResize(id, event)"
          @resize-end="(event: OnResizeEnd) => onGroupResizeEnd(id, event)"
        />
        <div
          :class="[
            'dag-group',
            { 'dag-group-selected': data.isSelected }
          ]"
          :style="{
            borderColor: data.color,
            '--group-color': data.color,
            '--group-height': `${data.height}px`,
          }"
        >
          <!-- Group Input Handle (left side) -->
          <Handle
            id="group-input"
            type="target"
            :position="Position.Left"
            class="dag-group-handle dag-group-handle-input"
            :style="{ top: '50%' }"
          />

          <!-- Group Header - Minimal Linear -->
          <div class="dag-group-header">
            <span class="dag-group-icon">{{ data.icon }}</span>
            <span class="dag-group-name">{{ data.label }}</span>
          </div>

          <!-- Entry Point Indicator (for single-zone groups) -->
          <div v-if="!data.hasMultipleZones" class="dag-group-entry" :style="{ color: data.color }">
            <span class="dag-group-entry-arrow">→</span>
            <span class="dag-group-entry-label">Start</span>
          </div>

          <!-- Multi-Section Zone Dividers and Labels -->
          <template v-if="data.hasMultipleZones && data.zones">
            <!-- Section Labels and Dividers -->
            <template v-for="(zone, index) in data.zones" :key="zone.role">
              <!-- Section Label -->
              <div
                class="dag-group-section-label"
                :style="{
                  top: `${32 + (data.height - 32) * zone.top + 8}px`,
                  left: zone.left === 0 ? '12px' : `${data.width * zone.left + 12}px`,
                  color: data.color,
                }"
              >
                <span class="dag-group-section-arrow">→</span>
                {{ zone.label }}
              </div>

              <!-- Horizontal Divider (for try_catch style - vertical stacking) -->
              <div
                v-if="index > 0 && zone.left === 0"
                class="dag-group-divider-h"
                :style="{
                  top: `${32 + (data.height - 32) * zone.top}px`,
                  backgroundColor: data.color,
                }"
              />

              <!-- Vertical Divider (for if_else style - horizontal stacking) -->
              <div
                v-if="index > 0 && zone.left > 0"
                class="dag-group-divider-v"
                :style="{
                  left: `${data.width * zone.left - 2}px`,
                  backgroundColor: data.color,
                }"
              />
            </template>
          </template>

          <!-- Group Output Handles (right side) -->
          <div class="dag-group-outputs">
            <Handle
              v-for="(port, index) in data.outputPorts"
              :id="port.name"
              :key="port.name"
              type="source"
              :position="Position.Right"
              class="dag-group-handle dag-group-handle-output"
              :style="{
                top: `${50 + (index - (data.outputPorts.length - 1) / 2) * 40}%`,
                '--handle-color': port.color,
              }"
            />
            <!-- Output Port Labels (hidden for cleaner look) -->
          </div>

          <!-- Group content area is handled by Vue Flow's built-in group functionality -->
        </div>
      </template>

      <!-- Custom Node Template - Miro-style Icon Design -->
      <template #node-custom="{ data }">
        <div
          :class="[
            'dag-node-miro',
            { 'dag-node-selected': data.isSelected },
            { 'dag-node-start': isStartNode(data.type) },
            { 'dag-node-has-run': data.stepRun },
            { 'dag-node-running': data.stepRun?.status === 'running' },
            { 'dag-node-completed': data.stepRun?.status === 'completed' },
            { 'dag-node-failed': data.stepRun?.status === 'failed' }
          ]"
          :style="{
            '--node-color': getStepColor(data.type),
          }"
        >
          <!-- Step Run Status Indicator (top-right corner of icon box) -->
          <div
            v-if="data.stepRun"
            class="dag-node-status-miro"
            :style="{ backgroundColor: getStepRunStatusColor(data.stepRun.status) }"
            :title="`${data.stepRun.status} - Click for details`"
          />

          <!-- Input Handles (left side of icon box) -->
          <template v-if="!isStartNode(data.type) && data.inputPorts && data.inputPorts.length > 1">
            <Handle
              v-for="(port, index) in data.inputPorts"
              :id="port.name"
              :key="port.name"
              type="target"
              :position="Position.Left"
              :class="['dag-handle-miro', 'dag-handle-target']"
              :style="{
                '--handle-top': `${24 + (index - (data.inputPorts.length - 1) / 2) * 12}px`,
              }"
              :title="port.label"
            />
          </template>
          <!-- Single input handle for standard blocks (hidden for Start nodes) -->
          <Handle
            v-else-if="!isStartNode(data.type)"
            id="input"
            type="target"
            :position="Position.Left"
            class="dag-handle-miro dag-handle-target dag-handle-center"
          />

          <!-- Icon Box (main visual element) -->
          <div class="dag-node-icon-box">
            <NodeIcon :icon="data.icon" :color="getStepColor(data.type)" />
          </div>

          <!-- Label (below icon box) -->
          <div class="dag-node-label-miro">{{ data.label }}</div>

          <!-- Duration indicator for completed step runs -->
          <div v-if="data.stepRun?.duration_ms" class="dag-node-duration-miro">
            {{ data.stepRun.duration_ms < 1000 ? `${data.stepRun.duration_ms}ms` : `${(data.stepRun.duration_ms / 1000).toFixed(1)}s` }}
          </div>

          <!-- Output Handles (right side of icon box) - Multiple for branching blocks -->
          <template v-if="data.outputPorts && data.outputPorts.length > 1">
            <Handle
              v-for="(port, index) in data.outputPorts"
              :id="port.name"
              :key="port.name"
              type="source"
              :position="Position.Right"
              :class="['dag-handle-miro', 'dag-handle-source', 'dag-handle-colored']"
              :style="{
                '--handle-top': `${24 + (index - (data.outputPorts.length - 1) / 2) * 12}px`,
                '--handle-bg': getPortColor(port.name),
                '--handle-border': getPortColor(port.name),
              }"
              :title="port.label"
            />
            <!-- Port labels (below icon, next to output) -->
            <div class="dag-port-labels-miro">
              <div
                v-for="(port, index) in data.outputPorts"
                :key="`label-${port.name}`"
                class="dag-port-label-miro"
                :style="{
                  '--label-top': `${24 + (index - (data.outputPorts.length - 1) / 2) * 12}px`,
                  color: getPortColor(port.name),
                }"
              >
                {{ port.label }}
              </div>
            </div>
          </template>
          <!-- Single output handle for non-branching blocks -->
          <Handle
            v-else
            id="output"
            type="source"
            :position="Position.Right"
            class="dag-handle-miro dag-handle-source dag-handle-center"
          />
        </div>
      </template>

      <!-- Mini Map -->
      <MiniMap
        v-if="props.showMinimap !== false"
        :pannable="true"
        :zoomable="true"
        :node-color="(node: Node) => node.type === 'group' ? node.data.color : getStepColor(node.data.type)"
        class="dag-minimap"
      />

    </VueFlow>

    <!-- Edge Delete Button (positioned relative to dag-editor container) -->
    <div
      v-if="edgeDeleteButtonPosition && selectedEdgeId && !readonly"
      class="edge-delete-button-container"
      :style="{
        left: `${edgeDeleteButtonPosition.x}px`,
        top: `${edgeDeleteButtonPosition.y}px`,
      }"
    >
      <button
        class="edge-delete-button-floating"
        title="エッジを削除 (Delete)"
        @click.stop="handleDeleteSelectedEdge"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="3 6 5 6 21 6"/>
          <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
        </svg>
      </button>
    </div>

    <!-- Drop indicator overlay -->
    <div v-if="isDragOver" class="drop-indicator">
      <div class="drop-indicator-content">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        <span>Drop to add step</span>
      </div>
    </div>

    <div v-if="!readonly && !isDragOver" class="dag-editor-hint">
      Drag blocks here to add steps
    </div>

    <!-- Bottom Right Button Group -->
    <div
      class="bottom-right-buttons"
      :class="{ 'no-transition': copilotResizing }"
      :style="{ right: autoLayoutRightOffset + 'px' }"
    >
      <!-- Auto Layout Button -->
      <button
        v-if="!readonly"
        class="action-button"
        data-tooltip="整形する"
        @click="emit('autoLayout')"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="7" height="7" rx="1"/>
          <rect x="14" y="3" width="7" height="7" rx="1"/>
          <rect x="14" y="14" width="7" height="7" rx="1"/>
          <rect x="3" y="14" width="7" height="7" rx="1"/>
        </svg>
      </button>

      <!-- Copilot Toggle Button -->
      <button
        class="action-button copilot-toggle"
        :class="{ active: copilotSidebarOpen }"
        data-tooltip="AI Copilot"
        @click="toggleCopilotSidebar"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73A2 2 0 0 1 10 4a2 2 0 0 1 2-2z"/>
          <circle cx="8" cy="14" r="2"/>
          <circle cx="16" cy="14" r="2"/>
        </svg>
      </button>
    </div>

  </div>
</template>

<style scoped>
.dag-editor {
  width: 100%;
  height: 100%;
  background-color: #fafafa;
  background-image:
    radial-gradient(circle, #e5e5e5 1px, transparent 1px);
  background-size: 24px 24px;
  position: relative;
  overflow: hidden;
  transition: background-color 0.2s;
}

.dag-editor.drag-over {
  background-color: rgba(59, 130, 246, 0.05);
}

/* ========================================
   Minimal Linear Design System
   ======================================== */

.dag-node {
  background: #ffffff;
  border: 1px solid #e5e5e5;
  border-radius: 8px;
  min-width: 160px;
  box-shadow: none;
  transition: border-color 0.15s, background-color 0.15s;
  position: relative;
}

.dag-node:hover {
  border-color: #d4d4d4;
  background-color: #fafafa;
}

.dag-node-selected {
  border-color: #3b82f6;
  box-shadow: 0 0 0 1px #3b82f6;
}

/* Running Node - pulsing animation */
.dag-node-running {
  animation: node-pulse 1.5s ease-in-out infinite;
  border-color: #3b82f6 !important;
}

.dag-node-running .dag-node-status {
  animation: status-pulse 1s ease-in-out infinite;
}

@keyframes node-pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4);
  }
  50% {
    box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.2);
  }
}

@keyframes status-pulse {
  0%, 100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.3);
    opacity: 0.8;
  }
}

/* Completed Node - green accent */
.dag-node-completed {
  border-color: #22c55e !important;
  box-shadow: 0 0 0 1px #22c55e;
}

/* Failed Node - red accent */
.dag-node-failed {
  border-color: #ef4444 !important;
  box-shadow: 0 0 0 1px #ef4444;
}

/* Start Node - subtle enhancement */
.dag-node-start {
  min-width: 160px;
}

.dag-node-start .dag-node-indicator {
  width: 10px;
  height: 10px;
}

/* Node Header - Minimal Linear style */
.dag-node-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px 4px;
}

/* Type Indicator (small dot) */
.dag-node-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* Type Label - Subtle gray text */
.dag-node-type {
  font-size: 11px;
  font-weight: 500;
  color: #737373;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Node Label */
.dag-node-label {
  padding: 4px 12px 12px;
  font-size: 14px;
  font-weight: 500;
  color: #171717;
  line-height: 1.4;
}

/* Step Run Status Indicator - Minimal Linear (left border accent) */
.dag-node-status {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  z-index: 10;
}

.dag-node-has-run {
  cursor: pointer;
}

/* Running state - animated border */
.dag-node-running {
  border-color: #3b82f6;
  animation: pulse-border 2s ease-in-out infinite;
}

/* Completed state - green left border */
.dag-node-completed {
  border-left: 3px solid #22c55e;
}

/* Failed state - red left border */
.dag-node-failed {
  border-left: 3px solid #ef4444;
}

@keyframes pulse-border {
  0%, 100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.3); }
  50% { box-shadow: 0 0 0 3px rgba(59, 130, 246, 0); }
}

/* Duration indicator */
.dag-node-duration {
  position: absolute;
  bottom: -6px;
  left: 50%;
  transform: translateX(-50%);
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 0.65rem;
  color: #64748b;
  font-family: 'SF Mono', Monaco, monospace;
  white-space: nowrap;
}

/* Trigger Badge for Start blocks */
.dag-node-trigger-badge {
  position: absolute;
  top: -8px;
  right: -8px;
  z-index: 10;
  pointer-events: auto;
}

/* Handle Styles - Minimal Linear (subtle, visible on hover) */
.dag-handle {
  width: 8px !important;
  height: 8px !important;
  background: #ffffff !important;
  border: 1.5px solid #d4d4d4 !important;
  border-radius: 50% !important;
  opacity: 0;
  transition: opacity 0.15s, border-color 0.15s, background-color 0.15s, transform 0.15s;
}

.dag-node:hover .dag-handle,
.vue-flow__node:hover .dag-handle {
  opacity: 1;
}

.dag-handle:hover {
  background: #3b82f6 !important;
  border-color: #3b82f6 !important;
  transform: scale(1.25) translateY(-50%);
}

.dag-handle-target {
  left: -4px !important;
  top: 50% !important;
  transform: translateY(-50%);
}

.dag-handle-source {
  right: -4px !important;
  top: 50% !important;
  transform: translateY(-50%);
}

/* Multi-handle styling for branching/merging blocks */
.dag-handle-multi {
  width: 8px !important;
  height: 8px !important;
}

.dag-handle-multi.dag-handle-source {
  right: -4px !important;
  top: var(--handle-top, 50%) !important;
  transform: translateY(-50%);
}

.dag-handle-multi.dag-handle-target {
  left: -4px !important;
  top: var(--handle-top, 50%) !important;
  transform: translateY(-50%);
}

/* Colored handles for branching blocks - use CSS custom properties */
.dag-handle-colored {
  background: var(--handle-bg, white) !important;
  border-color: var(--handle-border, #94a3b8) !important;
}

.dag-handle-colored:hover {
  filter: brightness(1.1);
  transform: scale(1.3) translateY(-50%);
}

/* Port labels container - right side (outputs) */
.dag-port-labels {
  position: absolute;
  right: 8px;
  top: 0;
  height: 100%;
  pointer-events: none;
}

/* Port labels container - left side (inputs) */
.dag-port-labels-left {
  right: auto;
  left: 8px;
}

.dag-port-label {
  position: absolute;
  right: 0;
  transform: translateY(-50%);
  font-size: 0.6rem;
  color: #64748b;
  text-transform: uppercase;
  white-space: nowrap;
  letter-spacing: 0.02em;
}

/* Left side port labels */
.dag-port-label-left {
  right: auto;
  left: 0;
}

/* Editor Hint */
.dag-editor-hint {
  position: absolute;
  bottom: 12px;
  left: 12px;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  background: white;
  padding: 6px 12px;
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 6px;
}

.dag-editor-hint::before {
  content: '💡';
  font-size: 0.875rem;
}

/* Drop Indicator */
.drop-indicator {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(59, 130, 246, 0.08);
  border: 2px dashed var(--color-primary);
  pointer-events: none;
  z-index: 100;
}

.drop-indicator-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  color: var(--color-primary);
  font-weight: 500;
}

/* Mini Map Customization */
.dag-minimap {
  background: white !important;
  border: 1px solid #e2e8f0 !important;
  border-radius: 8px !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1) !important;
}

/* Controls Customization */
:deep(.vue-flow__controls) {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

:deep(.vue-flow__controls-button) {
  width: 32px;
  height: 32px;
  border: none;
  background: white;
  color: #64748b;
  transition: all 0.15s;
}

:deep(.vue-flow__controls-button:hover) {
  background: #f1f5f9;
  color: #3b82f6;
}

:deep(.vue-flow__controls-button svg) {
  width: 16px;
  height: 16px;
}

/* Edge Styles - Minimal Linear */
:deep(.vue-flow__edge-path) {
  stroke: #d4d4d4;
  stroke-width: 1.5;
}

:deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke: #3b82f6;
  stroke-width: 2;
}

:deep(.vue-flow__edge:hover .vue-flow__edge-path) {
  stroke: #a3a3a3;
}

/* Connection line when dragging */
:deep(.vue-flow__connection-line) {
  stroke: #3b82f6;
  stroke-width: 2;
  stroke-dasharray: 5;
}

/* Block Group Styles - Minimal Linear */
.dag-group {
  width: 100%;
  height: 100%;
  border: none;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.01);
  position: relative;
  transition: border-color 0.15s;
}

.dag-group:hover {
  /* No border change on hover */
}

.dag-group-selected {
  box-shadow: 0 0 0 2px var(--group-color);
}

.dag-group-header {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 32px;
  border-radius: 12px 12px 0 0;
  border-bottom: none;
  background: transparent;
  display: flex;
  align-items: center;
  padding: 0 14px;
  gap: 8px;
  font-size: 0.75rem;
  font-weight: 500;
}

/* Group Icon */
.dag-group-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.dag-group-name {
  font-size: 13px;
  font-weight: 600;
  color: #171717;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Group Input/Output Handles - Minimal Linear */
.dag-group-handle {
  width: 8px !important;
  height: 8px !important;
  border-radius: 50% !important;
  border: 1.5px solid !important;
  transition: all 0.15s ease;
  z-index: 10;
  opacity: 0;
  /* Override Vue Flow default positioning */
  position: absolute !important;
}

.dag-group:hover .dag-group-handle {
  opacity: 1;
}

.dag-group-handle-input {
  left: -4px !important;
  top: 50% !important;
  right: auto !important;
  bottom: auto !important;
  transform: translateY(-50%);
  background: white !important;
  border-color: #d4d4d4 !important;
}

.dag-group-handle-input:hover {
  background: #3b82f6 !important;
  border-color: #3b82f6 !important;
  transform: scale(1.25) translateY(-50%);
}

.dag-group-handle-output {
  right: -4px !important;
  left: auto !important;
  bottom: auto !important;
  transform: translateY(-50%);
  background: var(--handle-color, #22c55e) !important;
  border-color: var(--handle-color, #22c55e) !important;
}

.dag-group-handle-output:hover {
  filter: brightness(1.1);
  transform: scale(1.25) translateY(-50%);
}

/* Group Entry Point Indicator - Minimal Linear */
.dag-group-entry {
  position: absolute;
  top: 44px;
  left: 14px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-weight: 600;
  opacity: 0.5;
}

.dag-group-entry-arrow {
  font-size: 0.85rem;
  font-weight: bold;
}

.dag-group-entry-label {
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

/* Group Output Port Labels */
.dag-group-outputs {
  position: absolute;
  right: 8px;
  top: 0;
  height: var(--group-height, 100%);
  pointer-events: none;
}

.dag-group-port-label {
  position: absolute;
  right: 0;
  transform: translateY(-50%);
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  white-space: nowrap;
  letter-spacing: 0.03em;
}

/* Multi-Section Zone Styles - Minimal Linear */
.dag-group-section-label {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  opacity: 0.7;
  z-index: 5;
}

.dag-group-section-arrow {
  font-size: 0.85rem;
  font-weight: bold;
}

/* Horizontal Divider (for try_catch) - Minimal */
.dag-group-divider-h {
  position: absolute;
  left: 12px;
  right: 12px;
  height: 1px;
  background: #e5e5e5 !important;
  opacity: 1;
  z-index: 5;
}

/* Vertical Divider (for if_else) - Minimal */
.dag-group-divider-v {
  position: absolute;
  top: 42px;
  bottom: 12px;
  width: 1px;
  background: #e5e5e5 !important;
  opacity: 1;
  z-index: 5;
}

/* Nested group indicator */
:deep(.vue-flow__node-group) {
  cursor: grab;
}

:deep(.vue-flow__node-group:active) {
  cursor: grabbing;
}

/* Make group nodes resizable appearance */
:deep(.vue-flow__node-group .vue-flow__resize-control) {
  width: 12px;
  height: 12px;
  background: white;
  border: 2px solid #94a3b8;
  border-radius: 4px;
}

:deep(.vue-flow__node-group .vue-flow__resize-control:hover) {
  background: #3b82f6;
  border-color: #3b82f6;
}

/* Custom resize handle styles for node-resizer - Minimal Linear */
:deep(.dag-group-resize-handle) {
  width: 8px !important;
  height: 8px !important;
  background: white !important;
  border: 1.5px solid #d4d4d4 !important;
  border-radius: 2px !important;
  opacity: 0;
  transition: all 0.15s ease;
}

:deep(.vue-flow__node-group:hover .dag-group-resize-handle) {
  opacity: 1;
}

:deep(.dag-group-resize-handle:hover) {
  background: var(--group-color, #3b82f6) !important;
  transform: scale(1.2);
}

/* Resize line styles */
:deep(.dag-group-resize-line) {
  border-color: var(--group-color, #94a3b8) !important;
  border-width: 1px !important;
  opacity: 0;
  transition: opacity 0.15s ease;
}

:deep(.vue-flow__node-group:hover .dag-group-resize-line) {
  opacity: 0.5;
}

/* Show resize handles on hover */
:deep(.vue-flow__node-group .vue-flow__resize-control) {
  opacity: 0;
  transition: opacity 0.15s ease;
}

:deep(.vue-flow__node-group:hover .vue-flow__resize-control),
:deep(.vue-flow__node-group.selected .vue-flow__resize-control) {
  opacity: 1;
}

/* Edge Delete Button (positioned relative to dag-editor container) */
.edge-delete-button-container {
  position: absolute;
  z-index: 100;
  pointer-events: auto;
}

.edge-delete-button-floating {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: white;
  color: #64748b;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.edge-delete-button-floating:hover {
  background: #fef2f2;
  color: #dc2626;
  border-color: #fecaca;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

.edge-delete-button-floating:active {
  background: #fee2e2;
}

/* Bottom Right Button Group */
.bottom-right-buttons {
  position: absolute;
  bottom: 70px;
  /* right is set dynamically via inline style */
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 10;
  transition: right 0.3s ease;
}

.bottom-right-buttons.no-transition {
  transition: none;
}

/* Action Button (shared style for auto-layout and copilot toggle) */
.action-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  background: white;
  color: #64748b;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  position: relative;
}

.action-button:hover {
  background: #f8fafc;
  color: #3b82f6;
  border-color: #3b82f6;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.15);
}

.action-button:active {
  transform: scale(0.95);
}

/* Copilot Toggle Active State */
.action-button.copilot-toggle.active {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.1) 0%, rgba(139, 92, 246, 0.1) 100%);
  color: #6366f1;
  border-color: #6366f1;
}

/* Tooltip for action buttons */
.action-button[data-tooltip]::before {
  content: attr(data-tooltip);
  position: absolute;
  right: calc(100% + 8px);
  top: 50%;
  transform: translateY(-50%);
  padding: 6px 10px;
  background: #1f2937;
  color: white;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  border-radius: 6px;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.15s, visibility 0.15s;
  pointer-events: none;
  z-index: 100;
}

.action-button[data-tooltip]::after {
  content: '';
  position: absolute;
  right: calc(100% + 4px);
  top: 50%;
  transform: translateY(-50%);
  border: 4px solid transparent;
  border-left-color: #1f2937;
  opacity: 0;
  visibility: hidden;
  transition: opacity 0.15s, visibility 0.15s;
  pointer-events: none;
  z-index: 100;
}

.action-button[data-tooltip]:hover::before,
.action-button[data-tooltip]:hover::after {
  opacity: 1;
  visibility: visible;
}

/* ========================================
   Miro-style Icon Node Design
   ======================================== */

/* Hide Vue Flow's default node selection outline for Miro-style nodes */
:deep(.vue-flow__node-custom.selected),
:deep(.vue-flow__node-custom:focus),
:deep(.vue-flow__node-custom:focus-visible) {
  outline: none !important;
  box-shadow: none !important;
}

.dag-node-miro {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  width: 48px; /* Fixed width to match icon box */
  min-width: 48px;
  overflow: visible; /* Allow label to extend beyond node bounds */
}

/* Icon Box - Main visual element */
.dag-node-icon-box {
  position: relative;
  z-index: 1;
  flex-shrink: 0;
}

/* Selected state for Miro nodes - remove outer box-shadow, only style icon wrapper */
.dag-node-miro.dag-node-selected {
  box-shadow: none !important;
  border-color: transparent !important;
}

.dag-node-miro.dag-node-selected .dag-node-icon-box :deep(.node-icon-wrapper) {
  border-color: #3b82f6 !important;
  box-shadow: none !important;
}

/* Running state - pulse animation */
.dag-node-running .dag-node-icon-box :deep(.node-icon-wrapper) {
  animation: miro-node-pulse 1.5s ease-in-out infinite;
  border-color: #3b82f6 !important;
}

@keyframes miro-node-pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4);
  }
  50% {
    box-shadow: 0 0 0 6px rgba(59, 130, 246, 0.1);
  }
}

/* Completed state - green border */
.dag-node-completed .dag-node-icon-box :deep(.node-icon-wrapper) {
  border-color: #22c55e !important;
  box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.2);
}

/* Failed state - red border */
.dag-node-failed .dag-node-icon-box :deep(.node-icon-wrapper) {
  border-color: #ef4444 !important;
  box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.2);
}

/* Label below icon */
.dag-node-label-miro {
  margin-top: 2px;
  font-size: 11px;
  font-weight: 500;
  color: #374151;
  text-align: center;
  max-width: 72px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 1.2;
}

/* Status indicator - top-right corner */
.dag-node-status-miro {
  position: absolute;
  top: 0;
  right: 0;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid white;
  z-index: 10;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.dag-node-running .dag-node-status-miro {
  animation: status-pulse 1s ease-in-out infinite;
}

/* Duration indicator */
.dag-node-duration-miro {
  margin-top: 2px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  padding: 1px 6px;
  font-size: 0.6rem;
  color: #64748b;
  font-family: 'SF Mono', Monaco, monospace;
  white-space: nowrap;
}

/* Trigger badge position for Miro design */
.dag-node-trigger-badge-miro {
  position: absolute;
  top: -4px;
  left: -4px;
  z-index: 10;
  pointer-events: auto;
}

/* Miro-style Handle Styles */
.dag-handle-miro {
  width: 10px !important;
  height: 10px !important;
  background: #ffffff !important;
  border: 2px solid #d1d5db !important;
  border-radius: 50% !important;
  opacity: 0;
  transition: opacity 0.15s, border-color 0.15s, background-color 0.15s, transform 0.15s;
}

.dag-node-miro:hover .dag-handle-miro {
  opacity: 1;
}

.dag-handle-miro:hover {
  background: #3b82f6 !important;
  border-color: #3b82f6 !important;
  transform: scale(1.3);
}

/* Handle positioning - attached to 48px icon box edges */
.dag-handle-miro.dag-handle-target {
  left: -5px !important; /* Center handle on icon edge (handle is 10px wide) */
  top: var(--handle-top, 24px) !important;
  transform: translateY(-50%);
}

.dag-handle-miro.dag-handle-source {
  right: -5px !important; /* Center handle on icon edge (handle is 10px wide) */
  top: var(--handle-top, 24px) !important;
  transform: translateY(-50%);
}

/* Center position for single handles - vertically centered on icon box */
.dag-handle-miro.dag-handle-center {
  top: 24px !important; /* Center of 48px icon box */
}

/* Colored handles for branching blocks */
.dag-handle-miro.dag-handle-colored {
  background: var(--handle-bg, white) !important;
  border-color: var(--handle-border, #94a3b8) !important;
}

.dag-handle-miro.dag-handle-colored:hover {
  filter: brightness(1.1);
  transform: scale(1.3);
}

/* Port labels for Miro design */
.dag-port-labels-miro {
  position: absolute;
  left: 100%;
  top: 0;
  margin-left: 8px;
  pointer-events: none;
}

.dag-port-label-miro {
  position: absolute;
  top: var(--label-top, 24px);
  transform: translateY(-50%);
  font-size: 0.5rem;
  font-weight: 600;
  text-transform: uppercase;
  white-space: nowrap;
  letter-spacing: 0.02em;
}

/* Copilot Preview Highlighting */
:deep(.preview-added .dag-node-miro) {
  outline: 2px dashed #22c55e;
  outline-offset: 2px;
  animation: preview-pulse-green 1.5s ease-in-out infinite;
}

:deep(.preview-modified .dag-node-miro) {
  outline: 2px dashed #3b82f6;
  outline-offset: 2px;
  animation: preview-pulse-blue 1.5s ease-in-out infinite;
}

:deep(.preview-deleted .dag-node-miro) {
  outline: 2px dashed #ef4444;
  outline-offset: 2px;
  opacity: 0.5;
  animation: preview-pulse-red 1.5s ease-in-out infinite;
}

/* Edge Preview Highlighting */
:deep(.preview-edge-added .vue-flow__edge-path) {
  stroke: #22c55e !important;
  stroke-width: 3 !important;
  stroke-dasharray: 8 4;
  animation: edge-pulse-green 1.5s ease-in-out infinite;
}

:deep(.preview-edge-deleted .vue-flow__edge-path) {
  stroke: #ef4444 !important;
  stroke-width: 3 !important;
  stroke-dasharray: 8 4;
  opacity: 0.5;
  animation: edge-pulse-red 1.5s ease-in-out infinite;
}

@keyframes edge-pulse-green {
  0%, 100% { stroke: #22c55e; }
  50% { stroke: #16a34a; }
}

@keyframes edge-pulse-red {
  0%, 100% { stroke: #ef4444; }
  50% { stroke: #dc2626; }
}

@keyframes preview-pulse-green {
  0%, 100% {
    outline-color: #22c55e;
    box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.3);
  }
  50% {
    outline-color: #16a34a;
    box-shadow: 0 0 8px 2px rgba(34, 197, 94, 0.4);
  }
}

@keyframes preview-pulse-blue {
  0%, 100% {
    outline-color: #3b82f6;
    box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.3);
  }
  50% {
    outline-color: #2563eb;
    box-shadow: 0 0 8px 2px rgba(59, 130, 246, 0.4);
  }
}

@keyframes preview-pulse-red {
  0%, 100% {
    outline-color: #ef4444;
    box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.3);
  }
  50% {
    outline-color: #dc2626;
    box-shadow: 0 0 8px 2px rgba(239, 68, 68, 0.4);
  }
}
</style>
