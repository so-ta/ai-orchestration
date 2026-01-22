import type { OnResizeStart, OnResize, OnResizeEnd } from '@vue-flow/node-resizer'
import type { BlockGroup, Step, GroupRole } from '~/types/api'
import {
  GRID_SIZE,
  GROUP_PADDING,
  GROUP_BOUNDARY_WIDTH,
  GROUP_HEADER_HEIGHT,
  DEFAULT_STEP_NODE_WIDTH,
  DEFAULT_STEP_NODE_HEIGHT,
  getGroupUuidFromNodeId,
  snapToGrid,
  determineRoleInGroup,
} from '../utils/dagHelpers'
import { useDropZone } from './useDropZone'
import { useCascadePush, type MovedGroup } from './useCascadePush'

// Pushed block info for boundary violations
export interface PushedBlock {
  stepId: string
  position: { x: number; y: number }
}

// Added block info for blocks that should be added to the group
export interface AddedBlock {
  stepId: string
  position: { x: number; y: number }
  role: GroupRole
}

// Track resize state for position compensation
interface ResizeState {
  groupId: string
  initialGroupX: number
  initialGroupY: number
  initialChildPositions: Map<string, { relX: number; relY: number }>
}

interface UseGroupResizeOptions {
  steps: Ref<Step[]>
  blockGroups: Ref<BlockGroup[] | undefined>
  readonly: Ref<boolean | undefined>
  updateNode: (nodeId: string, updates: { position?: { x: number; y: number }; parentNode?: string }) => void
  emit: {
    groupUpdate: (groupId: string, updates: { position?: { x: number; y: number }; size?: { width: number; height: number } }) => void
    groupResizeComplete: (groupId: string, data: {
      position: { x: number; y: number }
      size: { width: number; height: number }
      pushedBlocks: PushedBlock[]
      addedBlocks: AddedBlock[]
      movedGroups: MovedGroup[]
    }) => void
  }
}

export function useGroupResize(options: UseGroupResizeOptions) {
  const { steps, blockGroups, readonly, updateNode, emit } = options

  const resizeState = ref<ResizeState | null>(null)

  const { findGroupCollision } = useDropZone({ blockGroups })
  const { calculatePushPosition } = useCascadePush({ blockGroups, updateNode })

  /**
   * Handle group resize start - record initial positions
   */
  function onGroupResizeStart(nodeId: string, _event: OnResizeStart) {
    if (readonly.value) return

    const groupUuid = getGroupUuidFromNodeId(nodeId)
    const group = blockGroups.value?.find(g => g.id === groupUuid)
    if (!group) return

    const childPositions = new Map<string, { relX: number; relY: number }>()
    for (const step of steps.value) {
      if (step.block_group_id !== groupUuid) continue
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

  /**
   * Handle group resize (during drag) - compensate child positions in real-time
   */
  function onGroupResize(nodeId: string, event: OnResize) {
    if (readonly.value) return
    if (!resizeState.value || resizeState.value.groupId !== nodeId) return

    const currentX = event.params.x
    const currentY = event.params.y

    const deltaX = currentX - resizeState.value.initialGroupX
    const deltaY = currentY - resizeState.value.initialGroupY

    if (deltaX === 0 && deltaY === 0) return

    for (const [stepId, initialPos] of resizeState.value.initialChildPositions) {
      const newRelX = initialPos.relX - deltaX
      const newRelY = initialPos.relY - deltaY
      updateNode(stepId, { position: { x: newRelX, y: newRelY } })
    }
  }

  /**
   * Handle group resize end with all collision handling
   */
  function onGroupResizeEnd(nodeId: string, event: OnResizeEnd) {
    resizeState.value = null
    if (readonly.value) return

    const groupUuid = getGroupUuidFromNodeId(nodeId)
    const newX = Math.round(event.params.x)
    const newY = Math.round(event.params.y)
    const newWidth = Math.round(event.params.width)
    const newHeight = Math.round(event.params.height)

    const group = blockGroups.value?.find(g => g.id === groupUuid)
    if (!group) {
      emit.groupUpdate(groupUuid, { size: { width: newWidth, height: newHeight } })
      return
    }

    const pushedBlocks: PushedBlock[] = []
    const addedBlocks: AddedBlock[] = []
    const movedGroups: MovedGroup[] = []

    const resizedGroup: BlockGroup = {
      ...group,
      position_x: newX,
      position_y: newY,
      width: newWidth,
      height: newHeight,
    }

    const innerLeft = newX + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const innerRight = newX + newWidth - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
    const innerTop = newY + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const innerBottom = newY + newHeight - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

    const processedStepIds = new Set<string>()

    // STEP 1: Handle internal blocks
    for (const step of steps.value) {
      if (step.block_group_id !== groupUuid) continue
      if (step.type === 'start') continue

      const stepWidth = DEFAULT_STEP_NODE_WIDTH
      const stepHeight = DEFAULT_STEP_NODE_HEIGHT
      const stepAbsX = step.position_x
      const stepAbsY = step.position_y

      const stepLeft = stepAbsX
      const stepRight = stepAbsX + stepWidth
      const stepTop = stepAbsY
      const stepBottom = stepAbsY + stepHeight

      const fullyInside =
        stepLeft >= innerLeft && stepRight <= innerRight &&
        stepTop >= innerTop && stepBottom <= innerBottom

      const fullyOutside =
        stepRight <= newX || stepLeft >= newX + newWidth ||
        stepBottom <= newY || stepTop >= newY + newHeight

      const onBoundary = !fullyInside && !fullyOutside

      if (onBoundary) {
        const outsideGap = GRID_SIZE
        const edgeDistances = [
          { outX: snapToGrid(newX - stepWidth - outsideGap), outY: snapToGrid(stepAbsY) },
          { outX: snapToGrid(newX + newWidth + outsideGap), outY: snapToGrid(stepAbsY) },
          { outX: snapToGrid(stepAbsX), outY: snapToGrid(newY - stepHeight - outsideGap) },
          { outX: snapToGrid(stepAbsX), outY: snapToGrid(newY + newHeight + outsideGap) },
        ]

        let bestPos = edgeDistances[0]
        let bestDist = Infinity
        for (const pos of edgeDistances) {
          const dist = Math.sqrt(Math.pow(stepAbsX - pos.outX, 2) + Math.pow(stepAbsY - pos.outY, 2))
          if (dist < bestDist) {
            bestDist = dist
            bestPos = pos
          }
        }

        pushedBlocks.push({ stepId: step.id, position: { x: bestPos.outX, y: bestPos.outY } })
        updateNode(step.id, { position: { x: bestPos.outX, y: bestPos.outY }, parentNode: undefined })
        processedStepIds.add(step.id)
      } else if (fullyOutside) {
        pushedBlocks.push({ stepId: step.id, position: { x: stepAbsX, y: stepAbsY } })
        updateNode(step.id, { position: { x: stepAbsX, y: stepAbsY }, parentNode: undefined })
        processedStepIds.add(step.id)
      } else {
        processedStepIds.add(step.id)
      }
    }

    // STEP 2: Check external blocks for inclusion
    for (const step of steps.value) {
      if (step.block_group_id === groupUuid) continue
      if (processedStepIds.has(step.id)) continue
      if (step.type === 'start') continue

      const stepWidth = DEFAULT_STEP_NODE_WIDTH
      const stepHeight = DEFAULT_STEP_NODE_HEIGHT
      const stepLeft = step.position_x
      const stepRight = step.position_x + stepWidth
      const stepTop = step.position_y
      const stepBottom = step.position_y + stepHeight

      const fullyInside =
        stepLeft >= innerLeft && stepRight <= innerRight &&
        stepTop >= innerTop && stepBottom <= innerBottom

      if (fullyInside) {
        const role = determineRoleInGroup(
          step.position_x + stepWidth / 2,
          step.position_y + stepHeight / 2,
          resizedGroup
        )

        addedBlocks.push({ stepId: step.id, position: { x: step.position_x, y: step.position_y }, role })

        const relativeX = step.position_x - newX
        const relativeY = step.position_y - newY
        updateNode(step.id, { position: { x: relativeX, y: relativeY }, parentNode: nodeId })
        processedStepIds.add(step.id)
      }
    }

    // STEP 3: Handle group collisions
    const processedGroups = new Set<string>([groupUuid])
    const groupPositions = new Map<string, { x: number; y: number }>()
    const groupSizes = new Map<string, { width: number; height: number }>()

    if (blockGroups.value) {
      for (const g of blockGroups.value) {
        if (g.id === groupUuid) {
          groupPositions.set(g.id, { x: newX, y: newY })
          groupSizes.set(g.id, { width: newWidth, height: newHeight })
        } else {
          groupPositions.set(g.id, { x: g.position_x, y: g.position_y })
          groupSizes.set(g.id, { width: g.width, height: g.height })
        }
      }
    }

    const collidingGroup = findGroupCollision(newX, newY, newWidth, newHeight, processedGroups, groupPositions, groupSizes)

    if (collidingGroup) {
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

      updateNode(`group_${collidingGroup.id}`, { position: { x: pushResult.x, y: pushResult.y } })

      processedGroups.add(collidingGroup.id)
      groupPositions.set(collidingGroup.id, { x: pushResult.x, y: pushResult.y })

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

        updateNode(`group_${nextCollision.id}`, { position: { x: nextPush.x, y: nextPush.y } })
        processedGroups.add(nextCollision.id)
        groupPositions.set(nextCollision.id, { x: nextPush.x, y: nextPush.y })

        currentPos = { x: nextPush.x, y: nextPush.y }
        currentSize = nextSize
      }
    }

    emit.groupUpdate(groupUuid, {
      position: { x: newX, y: newY },
      size: { width: newWidth, height: newHeight },
    })

    emit.groupResizeComplete(groupUuid, {
      position: { x: newX, y: newY },
      size: { width: newWidth, height: newHeight },
      pushedBlocks,
      addedBlocks,
      movedGroups,
    })
  }

  return {
    resizeState,
    onGroupResizeStart,
    onGroupResize,
    onGroupResizeEnd,
  }
}
