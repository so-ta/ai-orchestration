import type { BlockGroup } from '~/types/api'
import { GRID_SIZE, snapToGrid } from '../utils/dagHelpers'
import { useDropZone } from './useDropZone'

// Push direction type for unified cascade direction
export type PushDirection = 'left' | 'right' | 'up' | 'down'

// Moved group info for groups that were pushed by blocks
export interface MovedGroup {
  groupId: string
  position: { x: number; y: number }
  delta: { x: number; y: number }
}

interface UseCascadePushOptions {
  blockGroups: Ref<BlockGroup[] | undefined>
  updateNode: (nodeId: string, updates: { position: { x: number; y: number } }) => void
}

export function useCascadePush(options: UseCascadePushOptions) {
  const { blockGroups, updateNode } = options
  const { findGroupCollision } = useDropZone({ blockGroups })

  /**
   * Determine push direction based on relative positions
   */
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

  /**
   * Calculate push position for an element being pushed
   * When fixedDirection is provided, always push in that direction (for cascade consistency)
   */
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

  /**
   * Process cascade group push when a block is pushed out and collides with other groups
   */
  function processCascadeGroupPush(
    blockX: number,
    blockY: number,
    blockWidth: number,
    blockHeight: number,
    excludeGroupIds: Set<string>
  ): MovedGroup[] {
    const groups = blockGroups.value
    if (!groups) return []

    const movedGroups: MovedGroup[] = []
    const processedGroups = new Set<string>(excludeGroupIds)
    const groupPositions = new Map<string, { x: number; y: number }>()
    const groupSizes = new Map<string, { width: number; height: number }>()

    // Initialize group positions and sizes
    for (const group of groups) {
      groupPositions.set(group.id, { x: group.position_x, y: group.position_y })
      groupSizes.set(group.id, { width: group.width, height: group.height })
    }

    // Track cascade direction - determined by the first push
    let cascadeDirection: PushDirection | null = null

    // Queue of items to process
    interface PushItem {
      type: 'block' | 'group'
      x: number
      y: number
      width: number
      height: number
      groupId?: string
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

  return {
    determinePushDirection,
    calculatePushPosition,
    processCascadeGroupPush,
  }
}
