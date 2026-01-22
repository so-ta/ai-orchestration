import type { BlockGroupType, GroupRole } from '~/types/api'

// Constants for group node ID prefix
export const GROUP_NODE_PREFIX = 'group_'

// Grid size constant - must match Vue Flow's snap-grid setting
export const GRID_SIZE = 20

// Constants for group layout
export const GROUP_HEADER_HEIGHT = 32
export const GROUP_PADDING = 10
export const GROUP_BOUNDARY_WIDTH = 20

// Default step node size (used as fallback when actual size is not available)
export const DEFAULT_STEP_NODE_WIDTH = 80
export const DEFAULT_STEP_NODE_HEIGHT = 70

// Default group size for new groups
export const DEFAULT_GROUP_WIDTH = 400
export const DEFAULT_GROUP_HEIGHT = 300

// Helper function to extract plain group UUID from Vue Flow node ID
export function getGroupUuidFromNodeId(nodeId: string): string {
  if (nodeId.startsWith(GROUP_NODE_PREFIX)) {
    return nodeId.slice(GROUP_NODE_PREFIX.length)
  }
  return nodeId
}

// Helper function to convert group UUID to Vue Flow node ID
export function getNodeIdFromGroupUuid(groupUuid: string): string {
  return `${GROUP_NODE_PREFIX}${groupUuid}`
}

// Helper function to snap a value to the grid
export function snapToGrid(value: number): number {
  return Math.round(value / GRID_SIZE) * GRID_SIZE
}

// Get block group color based on type
export function getGroupColor(type: BlockGroupType): string {
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
export function getGroupIcon(type: BlockGroupType): string {
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
export function getGroupTypeLabel(type: BlockGroupType): string {
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
export interface GroupPort {
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
export function getGroupOutputPorts(type: BlockGroupType): GroupPort[] {
  return GROUP_OUTPUT_PORTS[type] || [{ name: 'out', label: 'Output', color: '#22c55e' }]
}

// Multi-section zone configuration
export interface GroupZone {
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
export function getGroupZones(type: BlockGroupType): GroupZone[] | null {
  return GROUP_ZONES[type] || null
}

// Check if group has multiple zones
export function hasMultipleZones(type: BlockGroupType): boolean {
  return GROUP_ZONES[type] !== null
}

// Determine role based on position within multi-section group
// Phase A: Simplified to always return 'body' since multi-zone was removed
export function determineRoleInGroup(_x: number, _y: number, _group: unknown): GroupRole {
  // All groups now have a single body zone only
  return 'body'
}

// Get step color based on type
export function getStepColor(type: string): string {
  const colors: Record<string, string> = {
    start: '#22c55e',
    llm: '#8b5cf6',
    tool: '#3b82f6',
    condition: '#f59e0b',
    switch: '#f97316',
    map: '#06b6d4',
    join: '#84cc16',
    subflow: '#ec4899',
    wait: '#64748b',
    function: '#6366f1',
    router: '#f43f5e',
    human_in_loop: '#a855f7',
    filter: '#14b8a6',
    split: '#f472b6',
    aggregate: '#10b981',
    error: '#ef4444',
    note: '#94a3b8',
    log: '#71717a',
  }
  return colors[type] || '#64748b'
}

// Get step icon based on type
export function getStepIcon(type: string): string {
  const icons: Record<string, string> = {
    start: 'play',
    llm: 'brain',
    tool: 'wrench',
    condition: 'git-branch',
    switch: 'list-checks',
    map: 'map',
    join: 'git-merge',
    subflow: 'folder-tree',
    wait: 'clock',
    function: 'code',
    router: 'route',
    human_in_loop: 'user-check',
    filter: 'filter',
    split: 'split',
    aggregate: 'layers',
    error: 'alert-triangle',
    note: 'sticky-note',
    log: 'terminal',
  }
  return icons[type] || 'box'
}

// Check if type is a start node
export function isStartNode(type: string): boolean {
  return type === 'start' || type.startsWith('start_')
}

// Port color definitions
const PORT_COLORS: Record<string, string> = {
  true: '#22c55e',    // Green for true
  false: '#ef4444',   // Red for false
  error: '#ef4444',   // Red for error
  success: '#22c55e', // Green for success
  default: '#64748b', // Gray for default
  out: '#22c55e',     // Green for output
}

// Get port color based on port name
export function getPortColor(portName: string): string {
  return PORT_COLORS[portName] || '#3b82f6' // Default blue
}
