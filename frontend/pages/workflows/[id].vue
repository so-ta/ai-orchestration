<script setup lang="ts">
import type { Workflow, Step, StepType, BlockDefinition, BlockGroup, BlockGroupType, Run, GroupRole, OutputPort } from '~/types/api'
import type { GenerateWorkflowResponse } from '~/composables/useCopilot'
import { calculateLayout, calculateLayoutWithGroups, parseNodeId } from '~/utils/graph-layout'

const { t } = useI18n()
const route = useRoute()
const workflowId = route.params.id as string

const workflows = useWorkflows()
const runs = useRuns()
const blocksApi = useBlocks()
const blockGroupsApi = useBlockGroups()
const toast = useToast()
const { confirm } = useConfirm()

const workflow = ref<Workflow | null>(null)
const blockDefinitions = ref<BlockDefinition[]>([])
const blockGroups = ref<BlockGroup[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const saving = ref(false)

// Run dialog state
const showRunDialog = ref(false)

// Execution state
const latestRun = ref<Run | null>(null)

// Tab state
const activeTab = ref<'editor' | 'history'>('editor')

// Editor state composable
const {
  selectedStepId,
  selectedStep,
  leftPanelWidth,
  rightPanelWidth,
  selectStep,
  clearSelection,
  setLeftPanelWidth,
  setRightPanelWidth,
} = useEditorState(workflow)

// Selected block group
const selectedGroupId = ref<string | null>(null)

// Workflows are always editable - versioning handles history
const isReadonly = computed(() => false)

// Step form data for editing
const stepForm = ref({
  name: '',
  type: 'tool' as string,
  config: {} as Record<string, any>,
})

// Sync step form when selection changes
watch(selectedStep, (step) => {
  if (step) {
    stepForm.value = {
      name: step.name,
      type: step.type,
      config: { ...step.config },
    }
  }
}, { immediate: true })

// Load workflow
async function loadWorkflow() {
  try {
    loading.value = true
    error.value = null
    const [workflowResponse, groupsResponse] = await Promise.all([
      workflows.get(workflowId),
      blockGroupsApi.list(workflowId).catch(() => ({ data: [] })),
    ])
    workflow.value = workflowResponse.data
    blockGroups.value = groupsResponse.data || []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load workflow'
  } finally {
    loading.value = false
  }
}

// Add step from palette drop
async function handleStepDrop(data: { type: StepType; name: string; position: { x: number; y: number }; groupId?: string; groupRole?: GroupRole }) {
  if (!workflow.value || isReadonly.value) return

  const defaultConfigs: Record<string, Record<string, any>> = {
    llm: { provider: 'mock', model: 'mock', prompt: '' },
    tool: { adapter_id: 'mock' },
    condition: { expression: '' },
    map: { input_path: '$.items', parallel: 10 },
    join: {},
    subflow: { workflow_id: '' },
    loop: { loop_type: 'for', count: 10, max_iterations: 100 },
    wait: { duration_ms: 5000 },
    function: { code: '', timeout_ms: 5000 },
    router: { provider: 'mock', model: 'mock', prompt: '', routes: [] },
    human_in_loop: { instructions: '', timeout_hours: 24 },
  }

  try {
    const response = await workflows.createStep(workflowId, {
      name: data.name,
      type: data.type,
      config: defaultConfigs[data.type] || {},
      position: data.position,
    })

    // If dropped inside a group, add the step to the group
    if (data.groupId) {
      await blockGroupsApi.addStep(workflowId, data.groupId, {
        step_id: response.data.id,
        group_role: data.groupRole || 'body',
      })
    }

    // Add step to local state instead of reloading entire workflow
    if (workflow.value) {
      workflow.value.steps = [...(workflow.value.steps || []), response.data]
    }
    selectStep(response.data.id)
  } catch (e) {
    toast.error('Failed to add step', e instanceof Error ? e.message : undefined)
  }
}

// Select step
function handleSelectStep(step: Step) {
  selectStep(step.id)
}

// Handle pane click (deselect)
function handlePaneClick() {
  clearSelection()
  selectedGroupId.value = null
}

// Block Group Handlers
function handleSelectGroup(group: BlockGroup) {
  selectedGroupId.value = group.id
  clearSelection() // Deselect step when selecting group
}

// Delete block group
async function handleDeleteGroup() {
  if (!selectedGroupId.value || isReadonly.value) return

  const groupId = selectedGroupId.value

  try {
    saving.value = true
    selectedGroupId.value = null

    // Remove group from local state immediately (optimistic update)
    blockGroups.value = blockGroups.value.filter(g => g.id !== groupId)

    // Clear block_group_id from steps that were in this group
    if (workflow.value?.steps) {
      for (const step of workflow.value.steps) {
        if (step.block_group_id === groupId) {
          step.block_group_id = undefined
          step.group_role = undefined
        }
      }
    }

    // Delete from API
    await blockGroupsApi.remove(workflowId, groupId)
    toast.success(t('editor.groupDeleted'))
  } catch (e) {
    toast.error(t('editor.groupDeleteFailed'), e instanceof Error ? e.message : undefined)
    // On error, reload to get correct state
    await loadWorkflow()
  } finally {
    saving.value = false
  }
}

async function handleUpdateGroupPosition(groupId: string, updates: { position?: { x: number; y: number }; size?: { width: number; height: number } }) {
  if (isReadonly.value) return
  try {
    // Optimistic update - update local state immediately
    const group = blockGroups.value.find(g => g.id === groupId)
    if (group) {
      if (updates.position) {
        group.position_x = updates.position.x
        group.position_y = updates.position.y
      }
      if (updates.size) {
        group.width = updates.size.width
        group.height = updates.size.height
      }
    }
    // Save to API (no reload)
    await blockGroupsApi.update(workflowId, groupId, updates)
  } catch (e) {
    toast.error('Failed to update group', e instanceof Error ? e.message : undefined)
    // On error, reload to get correct state
    await loadWorkflow()
  }
}

async function handleGroupDrop(data: { type: BlockGroupType; name: string; position: { x: number; y: number } }) {
  if (!workflow.value || isReadonly.value) return
  try {
    const response = await blockGroupsApi.create(workflowId, {
      name: data.name,
      type: data.type,
      position: data.position,
      size: { width: 400, height: 300 },
    })
    // Add group to local state instead of reloading
    if (response?.data) {
      blockGroups.value = [...blockGroups.value, response.data]
    }
  } catch (e) {
    toast.error('Failed to create group', e instanceof Error ? e.message : undefined)
  }
}

// Handle group move complete - update internal blocks, pushed external blocks, added blocks, and cascading group movements
async function handleGroupMoveComplete(
  groupId: string,
  data: {
    position: { x: number; y: number }
    delta: { x: number; y: number }
    pushedBlocks: Array<{ stepId: string; position: { x: number; y: number } }>
    addedBlocks: Array<{ stepId: string; position: { x: number; y: number }; role: GroupRole }>
    movedGroups: Array<{ groupId: string; position: { x: number; y: number }; delta: { x: number; y: number } }>
  }
) {
  if (!workflow.value || isReadonly.value) return

  try {
    // 1. Update group position (optimistic update)
    const group = blockGroups.value.find(g => g.id === groupId)
    if (group) {
      group.position_x = data.position.x
      group.position_y = data.position.y
    }

    // 2. Update all blocks inside this group by adding delta to their absolute positions
    const stepsInGroup = workflow.value.steps?.filter(s => s.block_group_id === groupId) || []
    for (const step of stepsInGroup) {
      step.position_x += data.delta.x
      step.position_y += data.delta.y
    }

    // 3. Update pushed external blocks
    for (const pushed of data.pushedBlocks) {
      const step = workflow.value.steps?.find(s => s.id === pushed.stepId)
      if (step) {
        step.position_x = pushed.position.x
        step.position_y = pushed.position.y
      }
    }

    // 4. Handle blocks being added to the group (crossing into group)
    // Delete edges for steps being added to group (crossing boundary)
    const edgesToDelete: string[] = []
    for (const added of data.addedBlocks) {
      const step = workflow.value.steps?.find(s => s.id === added.stepId)
      if (step) {
        // Collect edges to delete
        // This includes:
        // - step-to-step edges (source_step_id or target_step_id matches)
        // - group-to-step edges (target_step_id matches, for edges from groups to this step)
        // - step-to-group edges (source_step_id matches, for edges from this step to groups)
        const connectedEdges = workflow.value.edges?.filter(e =>
          e.source_step_id === added.stepId || e.target_step_id === added.stepId
        ) || []
        for (const edge of connectedEdges) {
          edgesToDelete.push(edge.id)
        }

        // Update local state
        step.block_group_id = groupId
        step.group_role = added.role
        step.position_x = added.position.x
        step.position_y = added.position.y
      }
    }

    // Remove edges from local state
    if (edgesToDelete.length > 0 && workflow.value.edges) {
      workflow.value.edges = workflow.value.edges.filter(e => !edgesToDelete.includes(e.id))
    }

    // 5. Handle cascading group movements (groups pushed by pushed blocks)
    const stepsInMovedGroups: Array<{ step: Step; delta: { x: number; y: number } }> = []
    for (const movedGroup of data.movedGroups) {
      // Update moved group position in local state
      const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
      if (targetGroup) {
        targetGroup.position_x = movedGroup.position.x
        targetGroup.position_y = movedGroup.position.y
      }

      // Find and track all steps inside the moved group
      const stepsInThisGroup = workflow.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
      for (const step of stepsInThisGroup) {
        step.position_x += movedGroup.delta.x
        step.position_y += movedGroup.delta.y
        stepsInMovedGroups.push({ step, delta: movedGroup.delta })
      }
    }

    // 6. Save to API (parallel updates for performance)
    const updatePromises: Promise<unknown>[] = []

    // Update group position
    updatePromises.push(blockGroupsApi.update(workflowId, groupId, { position: data.position }))

    // Update blocks inside group (position only)
    for (const step of stepsInGroup) {
      updatePromises.push(workflows.updateStep(workflowId, step.id, {
        position: { x: step.position_x, y: step.position_y },
      }))
    }

    // Update pushed blocks
    for (const pushed of data.pushedBlocks) {
      updatePromises.push(workflows.updateStep(workflowId, pushed.stepId, {
        position: pushed.position,
      }))
    }

    // Delete edges for added blocks
    for (const edgeId of edgesToDelete) {
      updatePromises.push(workflows.deleteEdge(workflowId, edgeId).catch(() => {
        // Ignore edge deletion errors
      }))
    }

    // Add blocks to group
    for (const added of data.addedBlocks) {
      updatePromises.push(
        blockGroupsApi.addStep(workflowId, groupId, {
          step_id: added.stepId,
          group_role: added.role,
        }).then(() => {
          // Also update position
          return workflows.updateStep(workflowId, added.stepId, {
            position: added.position,
          })
        })
      )
    }

    // Update cascading moved groups
    for (const movedGroup of data.movedGroups) {
      updatePromises.push(blockGroupsApi.update(workflowId, movedGroup.groupId, {
        position: movedGroup.position,
      }))
    }

    // Update steps inside moved groups
    for (const { step } of stepsInMovedGroups) {
      updatePromises.push(workflows.updateStep(workflowId, step.id, {
        position: { x: step.position_x, y: step.position_y },
      }))
    }

    await Promise.all(updatePromises)
  } catch (e) {
    toast.error('Failed to update group', e instanceof Error ? e.message : undefined)
    // On error, reload to get correct state
    await loadWorkflow()
  }
}

// Handle group resize complete - update blocks that were pushed out or added, and groups that were moved
async function handleGroupResizeComplete(
  groupId: string,
  data: {
    position: { x: number; y: number }
    size: { width: number; height: number }
    pushedBlocks: Array<{ stepId: string; position: { x: number; y: number } }>
    addedBlocks: Array<{ stepId: string; position: { x: number; y: number }; role: GroupRole }>
    movedGroups: Array<{ groupId: string; position: { x: number; y: number }; delta: { x: number; y: number } }>
  }
) {
  if (!workflow.value || isReadonly.value) return

  try {
    // 1. Update pushed blocks (removed from group)
    for (const pushed of data.pushedBlocks) {
      const step = workflow.value.steps?.find(s => s.id === pushed.stepId)
      if (step) {
        step.block_group_id = undefined
        step.group_role = undefined
        step.position_x = pushed.position.x
        step.position_y = pushed.position.y
      }
    }

    // 2. Handle blocks being added to the group
    const edgesToDelete: string[] = []
    for (const added of data.addedBlocks) {
      const step = workflow.value.steps?.find(s => s.id === added.stepId)
      if (step) {
        // Collect edges to delete (crossing boundary)
        const connectedEdges = workflow.value.edges?.filter(e =>
          e.source_step_id === added.stepId || e.target_step_id === added.stepId
        ) || []
        for (const edge of connectedEdges) {
          edgesToDelete.push(edge.id)
        }

        // Update local state
        step.block_group_id = groupId
        step.group_role = added.role
        step.position_x = added.position.x
        step.position_y = added.position.y
      }
    }

    // Remove edges from local state
    if (edgesToDelete.length > 0 && workflow.value.edges) {
      workflow.value.edges = workflow.value.edges.filter(e => !edgesToDelete.includes(e.id))
    }

    // 3. Handle cascading group movements
    const stepsInMovedGroups: Array<{ step: Step; delta: { x: number; y: number } }> = []
    for (const movedGroup of data.movedGroups) {
      // Update moved group position in local state
      const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
      if (targetGroup) {
        targetGroup.position_x = movedGroup.position.x
        targetGroup.position_y = movedGroup.position.y
      }

      // Find and track all steps inside the moved group
      const stepsInThisGroup = workflow.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
      for (const step of stepsInThisGroup) {
        step.position_x += movedGroup.delta.x
        step.position_y += movedGroup.delta.y
        stepsInMovedGroups.push({ step, delta: movedGroup.delta })
      }
    }

    // 4. Save to API (parallel updates for performance)
    const updatePromises: Promise<unknown>[] = []

    // Update pushed blocks (remove from group and update position)
    for (const pushed of data.pushedBlocks) {
      updatePromises.push(
        blockGroupsApi.removeStep(workflowId, groupId, pushed.stepId).catch(() => {
          // May fail if already removed
        }).then(() => {
          return workflows.updateStep(workflowId, pushed.stepId, {
            position: pushed.position,
          })
        })
      )
    }

    // Delete edges for added blocks
    for (const edgeId of edgesToDelete) {
      updatePromises.push(workflows.deleteEdge(workflowId, edgeId).catch(() => {
        // Ignore edge deletion errors
      }))
    }

    // Add blocks to group
    for (const added of data.addedBlocks) {
      updatePromises.push(
        blockGroupsApi.addStep(workflowId, groupId, {
          step_id: added.stepId,
          group_role: added.role,
        }).then(() => {
          return workflows.updateStep(workflowId, added.stepId, {
            position: added.position,
          })
        })
      )
    }

    // Update cascading moved groups
    for (const movedGroup of data.movedGroups) {
      updatePromises.push(blockGroupsApi.update(workflowId, movedGroup.groupId, {
        position: movedGroup.position,
      }))
    }

    // Update steps inside moved groups
    for (const { step } of stepsInMovedGroups) {
      updatePromises.push(workflows.updateStep(workflowId, step.id, {
        position: { x: step.position_x, y: step.position_y },
      }))
    }

    await Promise.all(updatePromises)
  } catch (e) {
    toast.error('Failed to update after resize', e instanceof Error ? e.message : undefined)
    // On error, reload to get correct state
    await loadWorkflow()
  }
}

// Assign step to a group (or remove from group)
async function handleStepAssignGroup(
  stepId: string,
  groupId: string | null,
  position: { x: number; y: number },
  role?: GroupRole,
  movedGroups?: Array<{ groupId: string; position: { x: number; y: number }; delta: { x: number; y: number } }>
) {
  if (!workflow.value || isReadonly.value) return

  try {
    // Find the step to get its current group
    const step = workflow.value.steps?.find(s => s.id === stepId)
    if (!step) return

    const currentGroupId = step.block_group_id

    // If group is changing (crossing boundary), delete all connected edges
    if (currentGroupId !== groupId) {
      const edges = workflow.value.edges || []
      const connectedEdges = edges.filter(e =>
        e.source_step_id === stepId || e.target_step_id === stepId
      )

      // Optimistic update - remove edges from local state
      if (workflow.value.edges) {
        workflow.value.edges = workflow.value.edges.filter(e =>
          e.source_step_id !== stepId && e.target_step_id !== stepId
        )
      }

      // Delete edges from API
      for (const edge of connectedEdges) {
        try {
          await workflows.deleteEdge(workflowId, edge.id)
        } catch {
          console.warn(`Failed to delete edge ${edge.id}`)
        }
      }
    }

    // Update local state optimistically
    step.block_group_id = groupId || undefined
    step.group_role = role || undefined
    step.position_x = position.x
    step.position_y = position.y

    const updatePromises: Promise<unknown>[] = []

    // Remove from current group if any
    if (currentGroupId) {
      await blockGroupsApi.removeStep(workflowId, currentGroupId, stepId)
    }

    // Add to new group if specified
    if (groupId) {
      await blockGroupsApi.addStep(workflowId, groupId, {
        step_id: stepId,
        group_role: role || 'body',
      })
    }

    // Update step position (position is already in absolute coordinates from DagEditor)
    updatePromises.push(workflows.updateStep(workflowId, stepId, { position }))

    // Handle cascading group movements (if any)
    if (movedGroups && movedGroups.length > 0) {
      for (const movedGroup of movedGroups) {
        // Update moved group position in local state
        const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
        if (targetGroup) {
          targetGroup.position_x = movedGroup.position.x
          targetGroup.position_y = movedGroup.position.y
        }

        // Update all steps inside the moved group
        const stepsInThisGroup = workflow.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
        for (const groupStep of stepsInThisGroup) {
          groupStep.position_x += movedGroup.delta.x
          groupStep.position_y += movedGroup.delta.y
          // Save step position
          updatePromises.push(workflows.updateStep(workflowId, groupStep.id, {
            position: { x: groupStep.position_x, y: groupStep.position_y },
          }))
        }

        // Save group position
        updatePromises.push(blockGroupsApi.update(workflowId, movedGroup.groupId, {
          position: movedGroup.position,
        }))
      }
    }

    await Promise.all(updatePromises)

    // No reload - frontend is the source of truth for positions
  } catch (e) {
    toast.error('Failed to update step group', e instanceof Error ? e.message : undefined)
    // On error, reload to get correct state
    await loadWorkflow()
  }
}

// Update step position
async function handleUpdateStepPosition(
  stepId: string,
  position: { x: number; y: number },
  movedGroups?: Array<{ groupId: string; position: { x: number; y: number }; delta: { x: number; y: number } }>
) {
  if (!workflow.value || isReadonly.value) return

  try {
    // Optimistic update - update local state immediately
    const step = workflow.value.steps?.find(s => s.id === stepId)
    if (step) {
      step.position_x = position.x
      step.position_y = position.y
    }

    const updatePromises: Promise<unknown>[] = []

    // Save step position to API
    updatePromises.push(workflows.updateStep(workflowId, stepId, { position }))

    // Handle cascading group movements (if any)
    if (movedGroups && movedGroups.length > 0) {
      for (const movedGroup of movedGroups) {
        // Update moved group position in local state
        const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
        if (targetGroup) {
          targetGroup.position_x = movedGroup.position.x
          targetGroup.position_y = movedGroup.position.y
        }

        // Update all steps inside the moved group
        const stepsInThisGroup = workflow.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
        for (const groupStep of stepsInThisGroup) {
          groupStep.position_x += movedGroup.delta.x
          groupStep.position_y += movedGroup.delta.y
          // Save step position
          updatePromises.push(workflows.updateStep(workflowId, groupStep.id, {
            position: { x: groupStep.position_x, y: groupStep.position_y },
          }))
        }

        // Save group position
        updatePromises.push(blockGroupsApi.update(workflowId, movedGroup.groupId, {
          position: movedGroup.position,
        }))
      }
    }

    await Promise.all(updatePromises)
  } catch (e) {
    console.error('Failed to update step position:', e)
  }
}

// Add edge (with optional source/target ports for branching/merging blocks)
// Handles both step-to-step and group-to-step connections
async function handleAddEdge(source: string, target: string, sourcePort?: string, targetPort?: string) {
  if (!workflow.value || isReadonly.value) return

  // Parse source and target to determine if they are groups or steps
  const sourceInfo = parseNodeId(source)
  const targetInfo = parseNodeId(target)

  try {
    // Build edge request, only including defined fields
    const edgeRequest: Parameters<typeof workflows.createEdge>[1] = {
      source_port: sourcePort,
      target_port: targetPort,
    }

    // Set source based on node type
    if (sourceInfo.isGroup) {
      edgeRequest.source_block_group_id = sourceInfo.id
    } else {
      edgeRequest.source_step_id = sourceInfo.id
    }

    // Set target based on node type
    if (targetInfo.isGroup) {
      edgeRequest.target_block_group_id = targetInfo.id
    } else {
      edgeRequest.target_step_id = targetInfo.id
    }

    const response = await workflows.createEdge(workflowId, edgeRequest)
    // Add edge to local state instead of reloading entire workflow
    if (workflow.value && response?.data) {
      workflow.value.edges = [...(workflow.value.edges || []), response.data]
    }
  } catch (e) {
    toast.error('Failed to add edge', e instanceof Error ? e.message : undefined)
  }
}

// Prepare workflow data for save
function prepareWorkflowData() {
  if (!workflow.value) return null

  return {
    name: workflow.value.name,
    description: workflow.value.description,
    input_schema: workflow.value.input_schema,
    steps: (workflow.value.steps || []).map(s => ({
      id: s.id,
      name: s.name,
      type: s.type,
      config: s.config,
      position_x: s.position_x,
      position_y: s.position_y,
    })),
    edges: (workflow.value.edges || []).map(e => ({
      id: e.id,
      source_step_id: e.source_step_id,
      target_step_id: e.target_step_id,
      source_block_group_id: e.source_block_group_id,
      target_block_group_id: e.target_block_group_id,
      source_port: e.source_port,
      target_port: e.target_port,
      condition: e.condition,
    })),
  }
}

// Save workflow (creates a new version)
async function handleSave() {
  const data = prepareWorkflowData()
  if (!data) return

  const newVersion = workflow.value!.version + 1
  const confirmed = await confirm({
    title: t('workflows.saveVersionTitle'),
    message: t('workflows.confirmSave', { version: newVersion }),
    confirmText: t('common.save'),
    cancelText: t('common.cancel'),
  })
  if (!confirmed) return

  try {
    saving.value = true
    const response = await workflows.save(workflowId, data)
    // Update local state with response data instead of reloading
    if (response?.data && workflow.value) {
      workflow.value.version = response.data.version
      workflow.value.status = response.data.status
      workflow.value.has_draft = response.data.has_draft
      workflow.value.updated_at = response.data.updated_at
    }
    toast.success(t('workflows.saved'))
  } catch (e) {
    toast.error('Failed to save workflow', e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Save workflow as draft
async function handleSaveDraft() {
  const data = prepareWorkflowData()
  if (!data) return

  try {
    saving.value = true
    const response = await workflows.saveDraft(workflowId, data)
    // Update local state with response data instead of reloading
    if (response?.data && workflow.value) {
      workflow.value.has_draft = true
      workflow.value.updated_at = response.data.updated_at
    }
    toast.success(t('workflows.draftSaved'))
  } catch (e) {
    toast.error('Failed to save draft', e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Discard draft
async function handleDiscardDraft() {
  if (!workflow.value?.has_draft) return

  const confirmed = await confirm({
    title: t('workflows.discardDraftTitle'),
    message: t('workflows.confirmDiscardDraft'),
    confirmText: t('workflows.discardDraft'),
    cancelText: t('common.cancel'),
    variant: 'danger',
  })
  if (!confirmed) return

  try {
    saving.value = true
    const response = await workflows.discardDraft(workflowId)
    // Update local state with response data instead of reloading
    // discardDraft returns the published version, so we need to update steps and edges
    if (response?.data && workflow.value) {
      workflow.value.has_draft = false
      workflow.value.steps = response.data.steps || []
      workflow.value.edges = response.data.edges || []
      workflow.value.name = response.data.name
      workflow.value.description = response.data.description
      workflow.value.input_schema = response.data.input_schema
      workflow.value.updated_at = response.data.updated_at
    }
    toast.success(t('workflows.draftDiscarded'))
  } catch (e) {
    toast.error('Failed to discard draft', e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Run workflow - show dialog
function handleRun() {
  if (!workflow.value) return
  showRunDialog.value = true
}

// Execute workflow from dialog
async function handleRunFromDialog(input: Record<string, unknown>) {
  if (!workflow.value) return

  try {
    const response = await runs.create(workflowId, { triggered_by: 'manual', input })
    showRunDialog.value = false
    toast.success(t('workflows.runStarted'))
    // Open run page in a new tab
    window.open(`/runs/${response.data.id}`, '_blank')
  } catch (e) {
    toast.error(t('workflows.runFailed'), e instanceof Error ? e.message : undefined)
  }
}

// Update step name reactively (without saving to API)
function handleUpdateStepName(name: string) {
  if (!selectedStep.value || !workflow.value) return

  const stepIndex = workflow.value.steps?.findIndex(s => s.id === selectedStep.value!.id)
  if (stepIndex !== undefined && stepIndex >= 0 && workflow.value.steps) {
    workflow.value.steps[stepIndex] = {
      ...workflow.value.steps[stepIndex],
      name,
    }
  }
}

// Save step from properties panel
async function handleSaveStep(formData: { name: string; type: string; config: Record<string, any> }) {
  if (!selectedStep.value || !workflow.value) return

  try {
    saving.value = true
    const stepId = selectedStep.value.id

    // Optimistic update - update local state immediately
    const stepIndex = workflow.value.steps?.findIndex(s => s.id === stepId)
    if (stepIndex !== undefined && stepIndex >= 0 && workflow.value.steps) {
      workflow.value.steps[stepIndex] = {
        ...workflow.value.steps[stepIndex],
        name: formData.name,
        config: formData.config as object,
      }
    }

    // Save to API (no reload needed)
    await workflows.updateStep(workflowId, stepId, formData)
  } catch (e) {
    toast.error('Failed to save step', e instanceof Error ? e.message : undefined)
    // On error, reload to get correct state
    await loadWorkflow()
  } finally {
    saving.value = false
  }
}

// Delete step from properties panel (confirmation handled in PropertiesPanel)
async function handleDeleteStep() {
  if (!selectedStep.value || !workflow.value) return

  try {
    saving.value = true
    const stepId = selectedStep.value.id
    clearSelection()
    await workflows.deleteStep(workflowId, stepId)

    // Remove step and related edges from local state instead of reloading
    // Note: Also filter edges where this step is the source/target via block group
    workflow.value.steps = (workflow.value.steps || []).filter(s => s.id !== stepId)
    workflow.value.edges = (workflow.value.edges || []).filter(
      e => e.source_step_id !== stepId && e.target_step_id !== stepId
    )
    // Note: Group edges (source_block_group_id/target_block_group_id) are preserved
    // since they connect to groups, not individual steps
  } catch (e) {
    toast.error('Failed to delete step', e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Paste step handler for keyboard shortcuts
async function handlePasteStep(data: { type: StepType; name: string; config: Record<string, any> }) {
  if (!workflow.value || isReadonly.value) return

  try {
    const response = await workflows.createStep(workflowId, {
      name: data.name,
      type: data.type,
      config: data.config,
      position: { x: 200, y: 200 }, // Default paste position
    })
    // Add step to local state instead of reloading entire workflow
    workflow.value.steps = [...(workflow.value.steps || []), response.data]
    selectStep(response.data.id)
  } catch (e) {
    toast.error('Failed to paste step', e instanceof Error ? e.message : undefined)
  }
}

// Get output ports for a step type (used for auto-layout port ordering)
function getOutputPortsForLayout(stepType: StepType, step?: Step): OutputPort[] {
  const config = step?.config as Record<string, unknown> | undefined

  // Special handling for switch blocks - generate dynamic ports from cases
  if (stepType === 'switch' && config?.cases) {
    const cases = config.cases as Array<{ name: string; expression?: string; is_default?: boolean }>
    const dynamicPorts: OutputPort[] = []

    for (const caseItem of cases) {
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

    // Add error port for switch
    dynamicPorts.push({ name: 'error', label: 'Error', is_default: false })

    return dynamicPorts
  }

  // Look up from block definitions
  const blockDef = blockDefinitions.value.find(b => b.slug === stepType)
  let basePorts: OutputPort[] = blockDef?.output_ports || [{ name: 'out', label: 'Output', is_default: false }]

  // If the step has error port enabled, add error port if not already present
  if (config?.enable_error_port) {
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

// Get output ports for a block group type (for auto-layout)
// This mirrors GROUP_OUTPUT_PORTS in DagEditor.vue
const GROUP_OUTPUT_PORTS_FOR_LAYOUT: Record<BlockGroupType, OutputPort[]> = {
  parallel: [
    { name: 'out', label: 'Output', is_default: true },
    { name: 'error', label: 'Error', is_default: false },
  ],
  try_catch: [
    { name: 'out', label: 'Output', is_default: true },
    { name: 'error', label: 'Error', is_default: false },
  ],
  foreach: [
    { name: 'out', label: 'Output', is_default: true },
    { name: 'error', label: 'Error', is_default: false },
  ],
  while: [
    { name: 'out', label: 'Output', is_default: true },
  ],
}

function getGroupOutputPortsForLayout(groupType: BlockGroupType): OutputPort[] {
  return GROUP_OUTPUT_PORTS_FOR_LAYOUT[groupType] || [{ name: 'out', label: 'Output', is_default: true }]
}

// Auto-layout using dagre
async function handleAutoLayout() {
  if (!workflow.value || isReadonly.value) return
  const steps = workflow.value.steps || []
  const edges = workflow.value.edges || []

  if (steps.length === 0) return

  try {
    saving.value = true

    // Check if there are block groups
    if (blockGroups.value.length > 0) {
      // Use layout with groups support
      const layoutResults = calculateLayoutWithGroups(steps, edges, blockGroups.value, {
        getOutputPorts: getOutputPortsForLayout,
        getGroupOutputPorts: getGroupOutputPortsForLayout,
      })

      // Update local state immediately (optimistic update)
      for (const result of layoutResults.steps) {
        const step = workflow.value.steps?.find(s => s.id === result.stepId)
        if (step) {
          step.position_x = result.x
          step.position_y = result.y
        }
      }
      for (const result of layoutResults.groups) {
        const group = blockGroups.value.find(g => g.id === result.groupId)
        if (group) {
          group.position_x = result.x
          group.position_y = result.y
          group.width = result.width
          group.height = result.height
        }
      }

      // Save to API
      const stepUpdatePromises = layoutResults.steps.map(result =>
        workflows.updateStep(workflowId, result.stepId, {
          position: { x: result.x, y: result.y },
        })
      )
      const groupUpdatePromises = layoutResults.groups.map(result =>
        blockGroupsApi.update(workflowId, result.groupId, {
          position: { x: result.x, y: result.y },
          size: { width: result.width, height: result.height },
        })
      )

      await Promise.all([...stepUpdatePromises, ...groupUpdatePromises])
    } else {
      // Use simple layout without groups
      const layoutResults = calculateLayout(steps, edges, {
        getOutputPorts: getOutputPortsForLayout,
      })

      // Update local state immediately (optimistic update)
      for (const result of layoutResults) {
        const step = workflow.value.steps?.find(s => s.id === result.stepId)
        if (step) {
          step.position_x = result.x
          step.position_y = result.y
        }
      }

      // Save to API
      const updatePromises = layoutResults.map(result =>
        workflows.updateStep(workflowId, result.stepId, {
          position: { x: result.x, y: result.y },
        })
      )

      await Promise.all(updatePromises)
    }
  } catch (e) {
    toast.error('Failed to auto-layout', e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Apply generated workflow from Copilot
async function handleApplyWorkflow(generatedWorkflow: GenerateWorkflowResponse) {
  if (!workflow.value || isReadonly.value) return

  // Debug: Log the generated workflow
  console.log('Generated workflow:', JSON.stringify(generatedWorkflow, null, 2))

  try {
    saving.value = true

    // Map temp_id to actual step id
    const idMapping: Record<string, string> = {}
    const newSteps: Step[] = []
    const newEdges: typeof workflow.value.edges = []

    // Create all steps first
    for (const genStep of generatedWorkflow.steps) {
      // Skip if this is a start step and we already have one
      if (genStep.type === 'start') {
        const existingStart = workflow.value.steps?.find(s => s.type === 'start')
        if (existingStart) {
          idMapping[genStep.temp_id] = existingStart.id
          continue
        }
      }

      const response = await workflows.createStep(workflowId, {
        name: genStep.name,
        type: genStep.type as StepType,
        config: genStep.config || {},
        position: { x: genStep.position_x, y: genStep.position_y },
      })
      idMapping[genStep.temp_id] = response.data.id
      newSteps.push(response.data)
    }

    // Debug: Log ID mapping
    console.log('ID Mapping:', idMapping)
    console.log('Edges to create:', generatedWorkflow.edges)

    // Create edges using the id mapping
    for (const genEdge of generatedWorkflow.edges) {
      const sourceId = idMapping[genEdge.source_temp_id]
      const targetId = idMapping[genEdge.target_temp_id]

      console.log(`Edge: ${genEdge.source_temp_id} -> ${genEdge.target_temp_id}`)
      console.log(`  Resolved: ${sourceId} -> ${targetId}`)

      if (sourceId && targetId) {
        const edgeResponse = await workflows.createEdge(workflowId, {
          source_step_id: sourceId,
          target_step_id: targetId,
          source_port: genEdge.source_port,
        })
        if (edgeResponse?.data) {
          newEdges.push(edgeResponse.data)
        }
        console.log('  Edge created successfully')
      } else {
        console.warn(`  Edge skipped: sourceId=${sourceId}, targetId=${targetId}`)
      }
    }

    // Update local state instead of reloading
    if (workflow.value) {
      workflow.value.steps = [...(workflow.value.steps || []), ...newSteps]
      workflow.value.edges = [...(workflow.value.edges || []), ...newEdges]
    }
    toast.success(t('copilot.workflowGenerated'))
  } catch (e) {
    toast.error(t('copilot.errors.generateFailed'), e instanceof Error ? e.message : undefined)
    console.error('Failed to apply generated workflow:', e)
  } finally {
    saving.value = false
  }
}

// Keyboard shortcuts (Delete, Cmd/Ctrl+C, Cmd/Ctrl+V, Escape)
useKeyboardShortcuts({
  selectedStep,
  selectedGroupId,
  isReadonly,
  onDelete: handleDeleteStep,
  onDeleteGroup: handleDeleteGroup,
  onCopy: () => {},
  onPaste: handlePasteStep,
  onClearSelection: () => {
    clearSelection()
    selectedGroupId.value = null
  },
})



// Load block definitions for output port information
async function loadBlockDefinitions() {
  try {
    const response = await blocksApi.list()
    blockDefinitions.value = response.blocks
  } catch (e) {
    console.error('Failed to load block definitions:', e)
  }
}

// Load latest run for step re-execution
async function loadLatestRun() {
  try {
    const response = await runs.list(workflowId, { limit: 1 })
    if (response.data && response.data.length > 0) {
      latestRun.value = response.data[0]
    }
  } catch (e) {
    console.error('Failed to load latest run:', e)
  }
}

// Handle execute workflow from execution tab
async function handleExecuteWorkflowFromTab(triggered_by: 'test' | 'manual', input: object) {
  if (!workflow.value) return

  try {
    const response = await runs.create(workflowId, { triggered_by, input })
    latestRun.value = response.data
    // Don't open in new tab, just update latest run reference
    toast.success(t('workflows.runStarted'))
  } catch (e) {
    toast.error(t('workflows.runFailed'), e instanceof Error ? e.message : undefined)
  }
}

onMounted(() => {
  loadWorkflow()
  loadBlockDefinitions()
  loadLatestRun()
})
</script>

<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="text-secondary mt-2">{{ t('editor.loading') }}</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-banner">
      <div class="error-icon">!</div>
      <div>
        <div class="error-title">{{ t('editor.loadFailed') }}</div>
        <div class="error-message">{{ error }}</div>
      </div>
      <button class="btn btn-outline btn-sm" @click="loadWorkflow">{{ t('common.retry') }}</button>
    </div>

    <!-- Workflow -->
    <div v-else-if="workflow">
      <!-- Header -->
      <div class="page-header">
        <div class="page-header-info">
          <div class="breadcrumb">
            <NuxtLink to="/workflows" class="breadcrumb-link">{{ t('workflows.title') }}</NuxtLink>
            <span class="breadcrumb-separator">/</span>
            <span class="breadcrumb-current">{{ workflow.name }}</span>
          </div>
          <h1 class="page-title">{{ workflow.name }}</h1>
          <p class="page-subtitle">{{ workflow.description || t('editor.noDescription') }}</p>
        </div>
        <div class="page-header-actions">
          <div class="status-badges">
            <span :class="['badge', workflow.status === 'published' ? 'badge-success' : 'badge-warning']">
              {{ t(`workflows.status.${workflow.status}`) }}
            </span>
            <span class="badge badge-info">v{{ workflow.version }}</span>
            <span v-if="workflow.has_draft" class="badge badge-draft">
              {{ t('workflows.hasDraft') }}
            </span>
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="tabs-container">
        <div class="tabs">
          <button
            :class="['tab', { active: activeTab === 'editor' }]"
            @click="activeTab = 'editor'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
              <line x1="9" y1="3" x2="9" y2="21"></line>
            </svg>
            {{ t('workflows.tabs.editor') }}
          </button>
          <button
            :class="['tab', { active: activeTab === 'history' }]"
            @click="activeTab = 'history'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
            {{ t('workflows.tabs.history') }}
          </button>
        </div>
      </div>

      <!-- Editor Tab Content -->
      <div v-show="activeTab === 'editor'" class="editor-tab-content">
        <!-- Actions Bar -->
        <div class="actions-bar">
          <div class="actions-left">
            <button
              class="btn btn-primary"
              :disabled="saving"
              @click="handleSave"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path>
                <polyline points="17 21 17 13 7 13 7 21"></polyline>
                <polyline points="7 3 7 8 15 8"></polyline>
              </svg>
              {{ t('common.save') }}
            </button>
            <button
              class="btn btn-outline"
              :disabled="saving"
              @click="handleSaveDraft"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
                <line x1="16" y1="13" x2="8" y2="13"></line>
                <line x1="16" y1="17" x2="8" y2="17"></line>
              </svg>
              {{ t('workflows.saveDraft') }}
            </button>
            <button
              v-if="workflow.has_draft"
              class="btn btn-outline btn-warning"
              :disabled="saving"
              @click="handleDiscardDraft"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
              </svg>
              {{ t('workflows.discardDraft') }}
            </button>
            <div class="separator"></div>
            <button
              class="btn btn-outline"
              @click="handleRun()"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="5 3 19 12 5 21 5 3"></polygon>
              </svg>
              {{ t('workflows.run') }}
            </button>
          </div>
          <div class="actions-right">
            <button
              class="btn btn-outline btn-sm"
              :disabled="saving || !workflow.steps?.length"
              @click="handleAutoLayout"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="7" height="7"></rect>
                <rect x="14" y="3" width="7" height="7"></rect>
                <rect x="14" y="14" width="7" height="7"></rect>
                <rect x="3" y="14" width="7" height="7"></rect>
                <path d="M10 6h4M6 10v4M18 10v4M10 18h4"></path>
              </svg>
              {{ t('editor.autoLayout') }}
            </button>
            <span class="step-count">{{ t('editor.stepsCount', { count: workflow.steps?.length || 0 }) }}</span>
          </div>
        </div>

        <!-- 3-Column Editor Layout -->
      <WorkflowEditorLayout
        :left-width="leftPanelWidth"
        :right-width="rightPanelWidth"
        @update:left-width="setLeftPanelWidth"
        @update:right-width="setRightPanelWidth"
      >
        <!-- Left Sidebar: Step Palette -->
        <template #palette>
          <StepPalette :readonly="isReadonly" />
        </template>

        <!-- Center: DAG Canvas -->
        <template #canvas>
          <DagEditor
            :steps="workflow.steps || []"
            :edges="workflow.edges || []"
            :block-groups="blockGroups"
            :block-definitions="blockDefinitions"
            :readonly="isReadonly"
            :selected-step-id="selectedStepId"
            :selected-group-id="selectedGroupId"
            @step:select="handleSelectStep"
            @step:update="handleUpdateStepPosition"
            @step:drop="handleStepDrop"
            @step:assign-group="handleStepAssignGroup"
            @edge:add="handleAddEdge"
            @pane:click="handlePaneClick"
            @group:select="handleSelectGroup"
            @group:update="handleUpdateGroupPosition"
            @group:drop="handleGroupDrop"
            @group:move-complete="handleGroupMoveComplete"
            @group:resize-complete="handleGroupResizeComplete"
          />
        </template>

        <!-- Right Sidebar: Properties Panel -->
        <template #properties>
          <PropertiesPanel
            :step="selectedStep"
            :workflow-id="workflowId"
            :readonly-mode="isReadonly"
            :saving="saving"
            :latest-run="latestRun"
            :steps="workflow?.steps || []"
            :edges="workflow?.edges || []"
            :block-definitions="blockDefinitions"
            @save="handleSaveStep"
            @delete="handleDeleteStep"
            @apply-workflow="handleApplyWorkflow"
            @execute-workflow="handleExecuteWorkflowFromTab"
            @update:name="handleUpdateStepName"
          />
        </template>
      </WorkflowEditorLayout>
      </div>

      <!-- History Tab Content -->
      <div v-show="activeTab === 'history'">
        <WorkflowRunHistory :workflow-id="workflowId" />
      </div>
    </div>

    <!-- Run Dialog -->
    <RunDialog
      v-if="workflow"
      :show="showRunDialog"
      :workflow-id="workflowId"
      :workflow-name="workflow.name"
      :steps="workflow.steps || []"
      :edges="workflow.edges || []"
      :blocks="blockDefinitions"
      @close="showRunDialog = false"
      @run="handleRunFromDialog"
    />

  </div>
</template>

<style scoped>
/* Loading & Error */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-banner {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.5rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: var(--radius);
}

.error-icon {
  width: 32px;
  height: 32px;
  background: var(--color-error);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  flex-shrink: 0;
}

.error-title {
  font-weight: 600;
  color: var(--color-error);
}

.error-message {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
}

.breadcrumb-separator {
  color: var(--color-text-secondary);
}

.breadcrumb-current {
  color: var(--color-text-secondary);
}

.page-title {
  font-size: 1.5rem;
  font-weight: 700;
  margin: 0;
}

.page-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.status-badges {
  display: flex;
  gap: 0.5rem;
}

/* Actions Bar */
.actions-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
  padding: 0.75rem 1rem;
  background: var(--color-surface);
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
}

.actions-left {
  display: flex;
  gap: 0.75rem;
}

.actions-left .btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.actions-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.actions-right .btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.step-count {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.separator {
  width: 1px;
  height: 24px;
  background: var(--color-border);
  margin: 0 0.25rem;
}

.badge-draft {
  background: #fef3c7;
  color: #92400e;
}

/* Button Variants */
.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

.btn-warning {
  color: #92400e;
  border-color: #fbbf24;
}

.btn-warning:hover {
  background: #fef3c7;
  border-color: #f59e0b;
}

/* Tabs */
.tabs-container {
  margin-bottom: 1rem;
}

.tabs {
  display: flex;
  gap: 0.25rem;
  background: var(--color-surface);
  padding: 0.25rem;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  width: fit-content;
}

.tab {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-radius: calc(var(--radius) - 2px);
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab:hover {
  color: var(--color-text);
  background: rgba(0, 0, 0, 0.05);
}

.tab.active {
  color: var(--color-primary);
  background: white;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.tab svg {
  flex-shrink: 0;
}

/* Editor Tab Content */
.editor-tab-content {
  position: relative;
}
</style>
