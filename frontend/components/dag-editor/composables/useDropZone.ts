import type { BlockGroup, GroupRole } from '~/types/api'
import {
  GRID_SIZE,
  GROUP_HEADER_HEIGHT,
  GROUP_PADDING,
  GROUP_BOUNDARY_WIDTH,
  DEFAULT_STEP_NODE_WIDTH,
  DEFAULT_STEP_NODE_HEIGHT,
  snapToGrid,
  determineRoleInGroup,
} from '../utils/dagHelpers'

// Drop zone result interface
export interface DropZoneResult {
  group: BlockGroup | null
  zone: 'inside' | 'boundary' | 'outside'
  role?: GroupRole
}

interface UseDropZoneOptions {
  blockGroups: Ref<BlockGroup[] | undefined>
}

export function useDropZone(options: UseDropZoneOptions) {
  const { blockGroups } = options

  /**
   * Find which group contains the given position and determine zone
   * Considers the full bounding box of the block (not just the position point)
   */
  function findDropZone(
    x: number,
    y: number,
    blockWidth: number = DEFAULT_STEP_NODE_WIDTH,
    blockHeight: number = DEFAULT_STEP_NODE_HEIGHT,
    excludeGroupId?: string
  ): DropZoneResult {
    const groups = blockGroups.value
    if (!groups) return { group: null, zone: 'outside' }

    // Block bounding box
    const blockLeft = x
    const blockRight = x + blockWidth
    const blockTop = y
    const blockBottom = y + blockHeight

    // Check groups in reverse order (later groups are on top)
    for (let i = groups.length - 1; i >= 0; i--) {
      const group = groups[i]

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

  /**
   * Snap position to valid location (inside or outside group, not on boundary)
   * Returns the closest non-boundary position where the entire block fits
   */
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

  /**
   * Check if a block collides with any group's boundary (excluding specified groups)
   */
  function findGroupBoundaryCollision(
    blockX: number,
    blockY: number,
    blockWidth: number,
    blockHeight: number,
    excludeGroupIds: Set<string>,
    groupPositions: Map<string, { x: number; y: number }>
  ): BlockGroup | null {
    const groups = blockGroups.value
    if (!groups) return null

    const blockLeft = blockX
    const blockRight = blockX + blockWidth
    const blockTop = blockY
    const blockBottom = blockY + blockHeight

    for (const group of groups) {
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

  /**
   * Check if a group collides with another group (excluding specified groups)
   */
  function findGroupCollision(
    groupX: number,
    groupY: number,
    groupWidth: number,
    groupHeight: number,
    excludeGroupIds: Set<string>,
    groupPositions: Map<string, { x: number; y: number }>,
    groupSizes?: Map<string, { width: number; height: number }>
  ): BlockGroup | null {
    const groups = blockGroups.value
    if (!groups) return null

    const gap = 10  // Minimum gap between groups

    for (const group of groups) {
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

  /**
   * Check if a position is inside a group's valid area
   */
  function isInsideGroupValidArea(
    x: number,
    y: number,
    blockWidth: number,
    blockHeight: number,
    group: BlockGroup
  ): boolean {
    const innerLeft = group.position_x + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const innerRight = group.position_x + group.width - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
    const innerTop = group.position_y + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const innerBottom = group.position_y + group.height - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

    return (
      x >= innerLeft && (x + blockWidth) <= innerRight &&
      y >= innerTop && (y + blockHeight) <= innerBottom
    )
  }

  return {
    findDropZone,
    snapToValidPosition,
    findGroupBoundaryCollision,
    findGroupCollision,
    isInsideGroupValidArea,
  }
}
