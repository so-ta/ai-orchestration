import type { GraphNode } from '@vue-flow/core'
import type { BlockGroup, Step, GroupRole } from '~/types/api'
import {
  GRID_SIZE,
  GROUP_PADDING,
  GROUP_BOUNDARY_WIDTH,
  GROUP_HEADER_HEIGHT,
  DEFAULT_STEP_NODE_WIDTH,
  DEFAULT_STEP_NODE_HEIGHT,
  DEFAULT_GROUP_WIDTH,
  DEFAULT_GROUP_HEIGHT,
  getGroupUuidFromNodeId,
  getNodeIdFromGroupUuid,
  snapToGrid,
  determineRoleInGroup,
} from '../utils/dagHelpers'
import { useDropZone } from './useDropZone'
import { useCascadePush, type MovedGroup, type PushDirection } from './useCascadePush'

// Pushed block info
export interface PushedBlock {
  stepId: string
  position: { x: number; y: number }
}

// Added block info
export interface AddedBlock {
  stepId: string
  position: { x: number; y: number }
  role: GroupRole
}

interface UseNodeDragHandlerOptions {
  steps: Ref<Step[]>
  blockGroups: Ref<BlockGroup[] | undefined>
  readonly: Ref<boolean | undefined>
  updateNode: (nodeId: string, updates: { position?: { x: number; y: number }; parentNode?: string }) => void
  emit: {
    stepUpdate: (stepId: string, position: { x: number; y: number }, movedGroups?: MovedGroup[]) => void
    stepAssignGroup: (stepId: string, groupId: string | null, position: { x: number; y: number }, role?: GroupRole, movedGroups?: MovedGroup[]) => void
    groupMoveComplete: (groupId: string, data: {
      position: { x: number; y: number }
      delta: { x: number; y: number }
      pushedBlocks: PushedBlock[]
      addedBlocks: AddedBlock[]
      movedGroups: MovedGroup[]
    }) => void
  }
}

export function useNodeDragHandler(options: UseNodeDragHandlerOptions) {
  const { steps, blockGroups, readonly, updateNode, emit } = options

  const { findDropZone, snapToValidPosition, findGroupBoundaryCollision, findGroupCollision } = useDropZone({ blockGroups })
  const { calculatePushPosition, processCascadeGroupPush } = useCascadePush({ blockGroups, updateNode })

  /**
   * Handle step node drag stop
   */
  function handleStepDragStop(node: GraphNode) {
    let absoluteX = snapToGrid(node.position.x)
    let absoluteY = snapToGrid(node.position.y)

    const nodeWidth = node.dimensions?.width ?? DEFAULT_STEP_NODE_WIDTH
    const nodeHeight = node.dimensions?.height ?? DEFAULT_STEP_NODE_HEIGHT

    if (node.parentNode && blockGroups.value) {
      const parentGroupUuid = getGroupUuidFromNodeId(node.parentNode)
      const parentGroup = blockGroups.value.find(g => g.id === parentGroupUuid)
      if (parentGroup) {
        absoluteX += parentGroup.position_x
        absoluteY += parentGroup.position_y
      }
    }

    const dropZone = findDropZone(absoluteX, absoluteY, nodeWidth, nodeHeight)
    const step = node.data.step as Step
    const currentGroupId = step.block_group_id || null

    if (dropZone.zone === 'boundary' && dropZone.group) {
      const snapped = snapToValidPosition(absoluteX, absoluteY, dropZone.group, nodeWidth, nodeHeight)
      absoluteX = snapped.x
      absoluteY = snapped.y

      let relativeX = absoluteX
      let relativeY = absoluteY
      if (snapped.inside) {
        relativeX = absoluteX - dropZone.group.position_x
        relativeY = absoluteY - dropZone.group.position_y
      }

      updateNode(node.id, {
        position: { x: relativeX, y: relativeY },
        parentNode: snapped.inside ? getNodeIdFromGroupUuid(dropZone.group.id) : undefined,
      })

      const targetGroupId = snapped.inside ? dropZone.group.id : null
      const role = snapped.inside ? determineRoleInGroup(absoluteX, absoluteY, dropZone.group) : undefined

      let movedGroups: MovedGroup[] = []
      if (!snapped.inside) {
        movedGroups = processCascadeGroupPush(absoluteX, absoluteY, nodeWidth, nodeHeight, new Set([dropZone.group.id]))
      }

      if (currentGroupId !== targetGroupId) {
        emit.stepAssignGroup(node.id, targetGroupId, { x: absoluteX, y: absoluteY }, role, movedGroups.length > 0 ? movedGroups : undefined)
      } else {
        emit.stepUpdate(node.id, { x: absoluteX, y: absoluteY }, movedGroups.length > 0 ? movedGroups : undefined)
      }
    } else {
      const targetGroupId = dropZone.zone === 'inside' ? dropZone.group?.id || null : null
      const role = dropZone.zone === 'inside' ? dropZone.role : undefined

      if (currentGroupId !== targetGroupId) {
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
        emit.stepAssignGroup(node.id, targetGroupId, { x: absoluteX, y: absoluteY }, role)
      } else {
        emit.stepUpdate(node.id, { x: absoluteX, y: absoluteY })
      }
    }
  }

  /**
   * Handle group node drag stop
   */
  function handleGroupDragStop(node: GraphNode) {
    const groupData = node.data.group as BlockGroup
    let newX = snapToGrid(node.position.x)
    let newY = snapToGrid(node.position.y)

    const originalX = groupData.position_x
    const originalY = groupData.position_y
    const groupWidth = node.dimensions?.width || groupData.width || DEFAULT_GROUP_WIDTH
    const groupHeight = node.dimensions?.height || groupData.height || DEFAULT_GROUP_HEIGHT

    const dropX = newX
    const dropY = newY

    const pushedBlocks: PushedBlock[] = []
    const addedBlocks: AddedBlock[] = []
    const movedGroups: MovedGroup[] = []

    let cascadeDirection: PushDirection | null = null

    // STEP 1: Find blocks fully inside at drop position
    const dropPositionGroup: BlockGroup = {
      ...groupData,
      position_x: dropX,
      position_y: dropY,
      width: groupWidth,
      height: groupHeight,
    }

    const dropInnerLeft = dropX + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const dropInnerRight = dropX + groupWidth - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
    const dropInnerTop = dropY + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
    const dropInnerBottom = dropY + groupHeight - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

    const addedStepIds = new Set<string>()

    for (const step of steps.value) {
      if (step.block_group_id === groupData.id) continue
      if (step.type === 'start') continue

      const stepWidth = DEFAULT_STEP_NODE_WIDTH
      const stepHeight = DEFAULT_STEP_NODE_HEIGHT
      const stepLeft = step.position_x
      const stepRight = step.position_x + stepWidth
      const stepTop = step.position_y
      const stepBottom = step.position_y + stepHeight

      const fullyInsideAtDrop =
        stepLeft >= dropInnerLeft && stepRight <= dropInnerRight &&
        stepTop >= dropInnerTop && stepBottom <= dropInnerBottom

      if (fullyInsideAtDrop) {
        const role = determineRoleInGroup(
          step.position_x + stepWidth / 2,
          step.position_y + stepHeight / 2,
          dropPositionGroup
        )

        const relativeX = step.position_x - dropX
        const relativeY = step.position_y - dropY

        addedBlocks.push({
          stepId: step.id,
          position: { x: step.position_x, y: step.position_y },
          role,
        })
        addedStepIds.add(step.id)

        updateNode(step.id, {
          position: { x: relativeX, y: relativeY },
          parentNode: node.id,
        })
      }
    }

    // STEP 2: Check for collision with other groups
    const groupUuid = getGroupUuidFromNodeId(node.id)
    const dropZone = findDropZone(newX, newY, groupWidth, groupHeight, groupUuid)

    let wasGroupPushed = false
    let pushedAwayFromGroupId: string | null = null

    if (dropZone.zone === 'boundary' && dropZone.group) {
      const snapped = snapToValidPosition(newX, newY, dropZone.group, groupWidth, groupHeight)
      newX = snapped.x
      newY = snapped.y
      wasGroupPushed = true
      pushedAwayFromGroupId = dropZone.group.id

      const pushDeltaX = newX - dropX
      const pushDeltaY = newY - dropY
      cascadeDirection = Math.abs(pushDeltaX) > Math.abs(pushDeltaY)
        ? (pushDeltaX > 0 ? 'right' : 'left')
        : (pushDeltaY > 0 ? 'down' : 'up')

      updateNode(node.id, { position: { x: newX, y: newY } })
    } else if (dropZone.zone === 'inside' && dropZone.group) {
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

      const pushDeltaX = newX - dropX
      const pushDeltaY = newY - dropY
      cascadeDirection = Math.abs(pushDeltaX) > Math.abs(pushDeltaY)
        ? (pushDeltaX > 0 ? 'right' : 'left')
        : (pushDeltaY > 0 ? 'down' : 'up')

      updateNode(node.id, { position: { x: newX, y: newY } })
    }

    // STEP 3: Update added blocks' positions if group was snapped
    if (wasGroupPushed && addedBlocks.length > 0) {
      const snapDeltaX = newX - dropX
      const snapDeltaY = newY - dropY

      for (const addedBlock of addedBlocks) {
        addedBlock.position.x += snapDeltaX
        addedBlock.position.y += snapDeltaY
      }
    }

    // Track group positions and sizes
    const groupPositions = new Map<string, { x: number; y: number }>()
    const groupSizes = new Map<string, { width: number; height: number }>()
    groupPositions.set(groupData.id, { x: newX, y: newY })
    groupSizes.set(groupData.id, { width: groupWidth, height: groupHeight })

    const movedGroup: BlockGroup = {
      ...groupData,
      position_x: newX,
      position_y: newY,
      width: groupWidth,
      height: groupHeight,
    }

    const processedGroups = new Set<string>([groupData.id])
    if (pushedAwayFromGroupId) {
      processedGroups.add(pushedAwayFromGroupId)
    }

    interface GroupToPush {
      group: BlockGroup
      pusherX: number
      pusherY: number
      pusherWidth: number
      pusherHeight: number
    }
    const groupsToPush: GroupToPush[] = []

    // STEP 4: Check remaining blocks for boundary collision
    for (const step of steps.value) {
      if (step.block_group_id === groupData.id) continue
      if (addedStepIds.has(step.id)) continue
      if (step.type === 'start') continue

      const stepWidth = DEFAULT_STEP_NODE_WIDTH
      const stepHeight = DEFAULT_STEP_NODE_HEIGHT

      const stepLeft = step.position_x
      const stepRight = step.position_x + stepWidth
      const stepTop = step.position_y
      const stepBottom = step.position_y + stepHeight

      const groupLeft = movedGroup.position_x
      const groupRight = movedGroup.position_x + movedGroup.width
      const groupTop = movedGroup.position_y
      const groupBottom = movedGroup.position_y + movedGroup.height

      const overlapsGroup =
        stepRight > groupLeft && stepLeft < groupRight &&
        stepBottom > groupTop && stepTop < groupBottom

      if (!overlapsGroup) continue

      const innerLeft = movedGroup.position_x + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
      const innerRight = movedGroup.position_x + movedGroup.width - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
      const innerTop = movedGroup.position_y + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
      const innerBottom = movedGroup.position_y + movedGroup.height - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

      const fullyInsideAtFinal =
        stepLeft >= innerLeft && stepRight <= innerRight &&
        stepTop >= innerTop && stepBottom <= innerBottom

      if (!fullyInsideAtFinal) {
        const pushResult = calculatePushPosition(
          step.position_x, step.position_y, stepWidth, stepHeight,
          movedGroup.position_x, movedGroup.position_y, movedGroup.width, movedGroup.height,
          cascadeDirection ?? undefined
        )

        if (!cascadeDirection) {
          cascadeDirection = pushResult.direction
        }

        pushedBlocks.push({
          stepId: step.id,
          position: { x: pushResult.x, y: pushResult.y },
        })

        let relativeX = pushResult.x
        let relativeY = pushResult.y
        if (step.block_group_id && blockGroups.value) {
          const stepGroup = blockGroups.value.find(g => g.id === step.block_group_id)
          if (stepGroup) {
            const stepGroupPos = groupPositions.get(stepGroup.id) || { x: stepGroup.position_x, y: stepGroup.position_y }
            relativeX = pushResult.x - stepGroupPos.x
            relativeY = pushResult.y - stepGroupPos.y
          }
        }
        updateNode(step.id, { position: { x: relativeX, y: relativeY } })

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

    // Check group-to-group collision
    const collidingGroupInitial = findGroupCollision(newX, newY, groupWidth, groupHeight, processedGroups, groupPositions, groupSizes)
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
    const MAX_CASCADE_DEPTH = 10
    let cascadeDepth = 0

    while (groupsToPush.length > 0 && cascadeDepth < MAX_CASCADE_DEPTH) {
      cascadeDepth++
      const { group, pusherX, pusherY, pusherWidth, pusherHeight } = groupsToPush.shift()!

      if (processedGroups.has(group.id)) continue
      processedGroups.add(group.id)

      const currentPos = groupPositions.get(group.id) || { x: group.position_x, y: group.position_y }
      const currentSize = groupSizes.get(group.id) || { width: group.width, height: group.height }

      const pushed = calculatePushPosition(
        currentPos.x, currentPos.y, currentSize.width, currentSize.height,
        pusherX, pusherY, pusherWidth, pusherHeight,
        cascadeDirection ?? undefined
      )

      if (!cascadeDirection) {
        cascadeDirection = pushed.direction
      }

      groupPositions.set(group.id, { x: pushed.x, y: pushed.y })
      groupSizes.set(group.id, currentSize)

      movedGroups.push({
        groupId: group.id,
        position: { x: pushed.x, y: pushed.y },
        delta: { x: pushed.deltaX, y: pushed.deltaY },
      })

      updateNode(`group_${group.id}`, { position: { x: pushed.x, y: pushed.y } })

      // Check if pushed group causes block collisions
      for (const step of steps.value) {
        if (step.block_group_id === group.id) continue
        if (step.type === 'start') continue

        const stepWidth = DEFAULT_STEP_NODE_WIDTH
        const stepHeight = DEFAULT_STEP_NODE_HEIGHT

        const existingPush = pushedBlocks.find(p => p.stepId === step.id)
        const stepX = existingPush ? existingPush.position.x : step.position_x
        const stepY = existingPush ? existingPush.position.y : step.position_y

        const outerLeft = pushed.x
        const outerRight = pushed.x + currentSize.width
        const outerTop = pushed.y
        const outerBottom = pushed.y + currentSize.height

        const overlapsGroup =
          (stepX + stepWidth) > outerLeft && stepX < outerRight &&
          (stepY + stepHeight) > outerTop && stepY < outerBottom

        if (!overlapsGroup) continue

        const innerLeft = pushed.x + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
        const innerRight = pushed.x + currentSize.width - GROUP_PADDING - GROUP_BOUNDARY_WIDTH
        const innerTop = pushed.y + GROUP_HEADER_HEIGHT + GROUP_PADDING + GROUP_BOUNDARY_WIDTH
        const innerBottom = pushed.y + currentSize.height - GROUP_PADDING - GROUP_BOUNDARY_WIDTH

        const fullyInside =
          stepX >= innerLeft && (stepX + stepWidth) <= innerRight &&
          stepY >= innerTop && (stepY + stepHeight) <= innerBottom

        if (!fullyInside) {
          const blockPushResult = calculatePushPosition(
            stepX, stepY, stepWidth, stepHeight,
            pushed.x, pushed.y, currentSize.width, currentSize.height,
            cascadeDirection ?? undefined
          )

          if (!cascadeDirection) {
            cascadeDirection = blockPushResult.direction
          }

          const existingIndex = pushedBlocks.findIndex(p => p.stepId === step.id)
          if (existingIndex >= 0) {
            pushedBlocks[existingIndex].position = { x: blockPushResult.x, y: blockPushResult.y }
          } else {
            pushedBlocks.push({ stepId: step.id, position: { x: blockPushResult.x, y: blockPushResult.y } })
          }

          let relativeX = blockPushResult.x
          let relativeY = blockPushResult.y
          if (step.block_group_id && blockGroups.value) {
            const stepGroup = blockGroups.value.find(g => g.id === step.block_group_id)
            if (stepGroup) {
              const stepGroupPos = groupPositions.get(stepGroup.id) || { x: stepGroup.position_x, y: stepGroup.position_y }
              relativeX = blockPushResult.x - stepGroupPos.x
              relativeY = blockPushResult.y - stepGroupPos.y
            }
          }
          updateNode(step.id, { position: { x: relativeX, y: relativeY } })

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

      // Check group-to-group cascade
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

    const finalDeltaX = newX - originalX
    const finalDeltaY = newY - originalY

    emit.groupMoveComplete(getGroupUuidFromNodeId(node.id), {
      position: { x: newX, y: newY },
      delta: { x: finalDeltaX, y: finalDeltaY },
      pushedBlocks,
      addedBlocks,
      movedGroups,
    })
  }

  /**
   * Main handler for node drag stop
   */
  function onNodeDragStop(node: GraphNode) {
    if (readonly.value) return

    if (node.type === 'group') {
      handleGroupDragStop(node)
    } else {
      handleStepDragStop(node)
    }
  }

  return {
    onNodeDragStop,
  }
}
