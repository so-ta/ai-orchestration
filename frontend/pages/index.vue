<script setup lang="ts">
/**
 * Miro-Style Fullscreen Project Editor
 *
 * UI特徴:
 * - 100%キャンバス（DAGエディタが画面全体）
 * - フローティングヘッダー（中央上部）
 * - フローティングツールバー（左側縦配置）
 * - コンテキストプロパティパネル（選択ノード近くに表示）
 * - フローティングズームコントロール（右下）
 * - クイック検索モーダル（⌘K）
 */

import type { Project, Step, StepType, BlockDefinition, BlockGroup, BlockGroupType, Run, GroupRole, OutputPort, StepRun } from '~/types/api'
import type DagEditor from '~/components/dag-editor/DagEditor.vue'
import type { SlideOutPanel } from '~/composables/useEditorState'
import { calculateLayout, calculateLayoutWithGroups, parseNodeId } from '~/utils/graph-layout'

// Use editor layout (fullscreen, no padding)
definePageMeta({
  layout: 'editor',
})

const { t } = useI18n()

const projects = useProjects()
const runs = useRuns()
const blocksApi = useBlocks()
const blockGroupsApi = useBlockGroups()
const toast = useToast()
const { confirm: _confirm } = useConfirm()

// URL sync
const { projectIdFromUrl, setProjectInUrl } = useProjectUrlSync()

// Editor state
const {
  selectedStepId,
  activeSlideOut,
  selectStep,
  clearSelection,
  toggleSlideOut,
  closeSlideOut,
  setCurrentProjectId,
  getLastProjectId,
  // Bottom panel state
  bottomPanelHeight,
  bottomPanelResizing,
  selectedRun,
  selectedStepRun,
  setBottomPanelHeight,
  setSelectedRun,
  setSelectedStepRun,
} = useEditorState()

// Project state
const project = ref<Project | null>(null)
const blockDefinitions = ref<BlockDefinition[]>([])
const blockGroups = ref<BlockGroup[]>([])
const loading = ref(true)
const initializing = ref(true)
const error = ref<string | null>(null)
const saving = ref(false)
const running = ref(false)

// Run dialog state
const showRunDialog = ref(false)

// Right panel mode: 'block' for properties, 'run' for run details
type RightPanelMode = 'block' | 'run'

// Quick search modal
const showQuickSearch = ref(false)

// Release modal state
const showReleaseModal = ref(false)

// Auto-save status
const saveStatus = ref<'saved' | 'saving' | 'unsaved' | 'error'>('saved')

// DagEditor ref for zoom control
const dagEditorRef = ref<InstanceType<typeof DagEditor> | null>(null)

// Zoom level (read from DagEditor's viewport)
const zoomLevel = computed(() => dagEditorRef.value?.viewport?.zoom ?? 1)

// Execution state
const latestRun = ref<Run | null>(null)

// Selected block group
const selectedGroupId = ref<string | null>(null)

// Projects are always editable
const isReadonly = computed(() => false)

// Pass project ref to useEditorState for selectedStep computed
const editorState = useEditorState(project)


// Step form data for editing
const stepForm = ref({
  name: '',
  type: 'tool' as string,
  config: {} as Record<string, unknown>,
})

// Sync step form when selection changes
watch(() => editorState.selectedStep.value, (step) => {
  if (step) {
    stepForm.value = {
      name: step.name,
      type: step.type,
      config: { ...step.config } as Record<string, unknown>,
    }
  }
}, { immediate: true })

// Initialize project
async function initializeProject() {
  initializing.value = true
  loading.value = true
  error.value = null

  try {
    // 1. Check URL for project ID
    let projectId = projectIdFromUrl.value

    // 2. If no URL param, check localStorage for last project
    if (!projectId) {
      projectId = getLastProjectId()
    }

    // 3. If we have a project ID, try to load it
    if (projectId) {
      try {
        await loadProject(projectId)
        setProjectInUrl(projectId)
        return
      } catch {
        // Project doesn't exist, fall through to create new
        console.warn('Project not found, creating new one')
      }
    }

    // 4. No project found, create a new one
    await createNewProject()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to initialize'
  } finally {
    initializing.value = false
    loading.value = false
  }
}

// Load project by ID
async function loadProject(projectId: string) {
  loading.value = true
  error.value = null

  try {
    const [projectResponse, groupsResponse] = await Promise.all([
      projects.get(projectId),
      blockGroupsApi.list(projectId).catch(() => ({ data: [] })),
    ])
    project.value = projectResponse.data
    blockGroups.value = groupsResponse.data || []
    setCurrentProjectId(projectId)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load project'
    throw e
  } finally {
    loading.value = false
  }
}

// Create new project
async function createNewProject() {
  try {
    const response = await projects.create({
      name: t('editor.untitledProject'),
      description: '',
    })
    project.value = response.data
    blockGroups.value = []
    setCurrentProjectId(response.data.id)
    setProjectInUrl(response.data.id)
    toast.success(t('editor.projectCreated'))
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to create project'
    throw e
  }
}

// Switch project
async function handleSelectProject(projectId: string) {
  if (projectId === project.value?.id) return

  try {
    loading.value = true
    await loadProject(projectId)
    setProjectInUrl(projectId)
    clearSelection()
    selectedGroupId.value = null
    setSelectedRun(null)
    closeSlideOut()
  } catch {
    toast.error(t('projects.loadFailed'))
  }
}

// Create project from selector
async function handleCreateProject() {
  try {
    loading.value = true
    await createNewProject()
    clearSelection()
    selectedGroupId.value = null
    setSelectedRun(null)
    closeSlideOut()
  } catch {
    toast.error(t('projects.createFailed'))
  }
}

// Add step from palette drop
async function handleStepDrop(data: { type: StepType; name: string; position: { x: number; y: number }; groupId?: string; groupRole?: GroupRole }) {
  if (!project.value || isReadonly.value) return

  const defaultConfigs: Record<string, Record<string, unknown>> = {
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
    const response = await projects.createStep(project.value.id, {
      name: data.name,
      type: data.type,
      config: defaultConfigs[data.type] || {},
      position: data.position,
    })

    // If dropped inside a group, add the step to the group
    if (data.groupId) {
      await blockGroupsApi.addStep(project.value.id, data.groupId, {
        step_id: response.data.id,
        group_role: data.groupRole || 'body',
      })
    }

    // Add step to local state instead of reloading entire project
    if (project.value) {
      project.value.steps = [...(project.value.steps || []), response.data]
    }
    selectStep(response.data.id)
  } catch (e) {
    toast.error('Failed to add step', e instanceof Error ? e.message : undefined)
  }
}

// Add block from quick search
function handleSelectBlock(block: BlockDefinition) {
  if (!project.value || isReadonly.value) return

  // Create step at center of viewport
  const position = { x: window.innerWidth / 2 - 100, y: window.innerHeight / 2 - 50 }
  handleStepDrop({
    type: block.slug as StepType,
    name: block.name,
    position,
  })
}

// Add group from quick search
function handleSelectGroupType(type: BlockGroupType) {
  if (!project.value || isReadonly.value) return

  const position = { x: window.innerWidth / 2 - 200, y: window.innerHeight / 2 - 150 }
  handleGroupDrop({
    type,
    name: type.charAt(0).toUpperCase() + type.slice(1),
    position,
  })
}

// Select step
function handleSelectStep(step: Step) {
  selectStep(step.id)
  selectedGroupId.value = null
  // Clear run selection when editing a block
  setSelectedRun(null)
}

// Bottom panel event handlers
function handleRunSelect(run: Run) {
  setSelectedRun(run)
  // Clear block selection when viewing run details
  clearSelection()
  selectedGroupId.value = null
}

function handleBottomPanelHeightChange(height: number) {
  setBottomPanelHeight(height)
}

// Step runs for DAG visualization
const stepRunsForDag = computed(() => {
  return selectedRun.value?.step_runs || []
})

// Right panel visibility and mode
const showRightPanel = computed(() => {
  return editorState.selectedStep.value !== null || selectedRun.value !== null
})

const rightPanelMode = computed<RightPanelMode>(() => {
  // Block editing takes priority
  if (editorState.selectedStep.value !== null) return 'block'
  // Otherwise show run details
  return 'run'
})

const rightPanelTitle = computed(() => {
  if (rightPanelMode.value === 'block') {
    return t('editor.blockDetails')
  }
  return undefined // RunDetailPanel has its own header
})

// Nested panel for step details
const showNestedPanel = computed(() => selectedStepRun.value !== null)

// Primary panel shift when nested panel is open (360px width + 12px gap)
const primaryPanelShift = computed(() => showNestedPanel.value ? 372 : 0)

// Step run selection handler
function handleStepRunSelect(stepRun: StepRun) {
  setSelectedStepRun(stepRun)
}

// Close nested panel
function handleCloseNestedPanel() {
  setSelectedStepRun(null)
}

// Handle run created/updated from execution panel - auto-transition to run details
function handleRunCreated(run: Run) {
  // Check if this is an update to the current run (polling update)
  if (selectedRun.value?.id === run.id) {
    // Update run data without resetting step selection
    const currentStepRunId = selectedStepRun.value?.id
    selectedRun.value = run

    // Update selected step run with fresh data
    if (currentStepRunId && run.step_runs) {
      const updatedStepRun = run.step_runs.find(sr => sr.id === currentStepRunId)
      if (updatedStepRun) {
        setSelectedStepRun(updatedStepRun)
      }
    }
  } else {
    // New run - clear block selection and show run details
    clearSelection()
    selectedGroupId.value = null
    setSelectedRun(run)
  }
}

// Handle run refresh request from RunDetailPanel
async function handleRunRefresh() {
  if (!selectedRun.value) return
  try {
    const currentStepRunId = selectedStepRun.value?.id
    const response = await runs.get(selectedRun.value.id)

    // Update selected run without triggering step reset
    selectedRun.value = response.data

    // If a step was selected, update it with fresh data from the new run
    if (currentStepRunId && response.data.step_runs) {
      const updatedStepRun = response.data.step_runs.find(sr => sr.id === currentStepRunId)
      if (updatedStepRun) {
        setSelectedStepRun(updatedStepRun)
      }
    }
  } catch (e) {
    console.error('Failed to refresh run:', e)
  }
}

// Reset step selection only when run ID changes (not on refresh)
watch(() => selectedRun.value?.id, (newId, oldId) => {
  if (newId !== oldId) {
    setSelectedStepRun(null)
  }
})

function handleCloseRightPanel() {
  clearSelection()
  setSelectedRun(null)
  setSelectedStepRun(null)
}

// Handle pane click (deselect)
function handlePaneClick() {
  clearSelection()
  selectedGroupId.value = null
  setSelectedRun(null)
}

// Block Group Handlers
function handleSelectGroup(group: BlockGroup) {
  selectedGroupId.value = group.id
  clearSelection()
  setSelectedRun(null)
}

// Delete block group
async function handleDeleteGroup() {
  if (!selectedGroupId.value || isReadonly.value || !project.value) return

  const groupId = selectedGroupId.value

  try {
    saving.value = true
    selectedGroupId.value = null

    // Remove group from local state immediately (optimistic update)
    blockGroups.value = blockGroups.value.filter(g => g.id !== groupId)

    // Clear block_group_id from steps that were in this group
    if (project.value?.steps) {
      for (const step of project.value.steps) {
        if (step.block_group_id === groupId) {
          step.block_group_id = undefined
          step.group_role = undefined
        }
      }
    }

    // Delete from API
    await blockGroupsApi.remove(project.value.id, groupId)
    toast.success(t('editor.groupDeleted'))
  } catch (e) {
    toast.error(t('editor.groupDeleteFailed'), e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  } finally {
    saving.value = false
  }
}

async function handleUpdateGroupPosition(groupId: string, updates: { position?: { x: number; y: number }; size?: { width: number; height: number } }) {
  if (isReadonly.value || !project.value) return
  try {
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
    await blockGroupsApi.update(project.value.id, groupId, updates)
  } catch (e) {
    toast.error('Failed to update group', e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  }
}

async function handleGroupDrop(data: { type: BlockGroupType; name: string; position: { x: number; y: number } }) {
  if (!project.value || isReadonly.value) return
  try {
    const response = await blockGroupsApi.create(project.value.id, {
      name: data.name,
      type: data.type,
      position: data.position,
      size: { width: 400, height: 300 },
    })
    if (response?.data) {
      blockGroups.value = [...blockGroups.value, response.data]
    }
  } catch (e) {
    toast.error('Failed to create group', e instanceof Error ? e.message : undefined)
  }
}

// Handle group move complete
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
  if (!project.value || isReadonly.value) return

  try {
    const group = blockGroups.value.find(g => g.id === groupId)
    if (group) {
      group.position_x = data.position.x
      group.position_y = data.position.y
    }

    const stepsInGroup = project.value.steps?.filter(s => s.block_group_id === groupId) || []
    for (const step of stepsInGroup) {
      step.position_x += data.delta.x
      step.position_y += data.delta.y
    }

    for (const pushed of data.pushedBlocks) {
      const step = project.value.steps?.find(s => s.id === pushed.stepId)
      if (step) {
        step.position_x = pushed.position.x
        step.position_y = pushed.position.y
      }
    }

    const edgesToDelete: string[] = []
    for (const added of data.addedBlocks) {
      const step = project.value.steps?.find(s => s.id === added.stepId)
      if (step) {
        const connectedEdges = project.value.edges?.filter(e =>
          e.source_step_id === added.stepId || e.target_step_id === added.stepId
        ) || []
        for (const edge of connectedEdges) {
          edgesToDelete.push(edge.id)
        }
        step.block_group_id = groupId
        step.group_role = added.role
        step.position_x = added.position.x
        step.position_y = added.position.y
      }
    }

    if (edgesToDelete.length > 0 && project.value.edges) {
      project.value.edges = project.value.edges.filter(e => !edgesToDelete.includes(e.id))
    }

    const stepsInMovedGroups: Array<{ step: Step; delta: { x: number; y: number } }> = []
    for (const movedGroup of data.movedGroups) {
      const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
      if (targetGroup) {
        targetGroup.position_x = movedGroup.position.x
        targetGroup.position_y = movedGroup.position.y
      }
      const stepsInThisGroup = project.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
      for (const step of stepsInThisGroup) {
        step.position_x += movedGroup.delta.x
        step.position_y += movedGroup.delta.y
        stepsInMovedGroups.push({ step, delta: movedGroup.delta })
      }
    }

    const updatePromises: Promise<unknown>[] = []
    updatePromises.push(blockGroupsApi.update(project.value.id, groupId, { position: data.position }))

    for (const step of stepsInGroup) {
      updatePromises.push(projects.updateStep(project.value.id, step.id, {
        position: { x: step.position_x, y: step.position_y },
      }))
    }

    for (const pushed of data.pushedBlocks) {
      updatePromises.push(projects.updateStep(project.value.id, pushed.stepId, {
        position: pushed.position,
      }))
    }

    for (const edgeId of edgesToDelete) {
      updatePromises.push(projects.deleteEdge(project.value.id, edgeId).catch(() => {}))
    }

    for (const added of data.addedBlocks) {
      updatePromises.push(
        blockGroupsApi.addStep(project.value.id, groupId, {
          step_id: added.stepId,
          group_role: added.role,
        }).then(() => {
          return projects.updateStep(project.value!.id, added.stepId, {
            position: added.position,
          })
        })
      )
    }

    for (const movedGroup of data.movedGroups) {
      updatePromises.push(blockGroupsApi.update(project.value.id, movedGroup.groupId, {
        position: movedGroup.position,
      }))
    }

    for (const { step } of stepsInMovedGroups) {
      updatePromises.push(projects.updateStep(project.value.id, step.id, {
        position: { x: step.position_x, y: step.position_y },
      }))
    }

    await Promise.all(updatePromises)
  } catch (e) {
    toast.error('Failed to update group', e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  }
}

// Handle group resize complete
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
  if (!project.value || isReadonly.value) return

  try {
    for (const pushed of data.pushedBlocks) {
      const step = project.value.steps?.find(s => s.id === pushed.stepId)
      if (step) {
        step.block_group_id = undefined
        step.group_role = undefined
        step.position_x = pushed.position.x
        step.position_y = pushed.position.y
      }
    }

    const edgesToDelete: string[] = []
    for (const added of data.addedBlocks) {
      const step = project.value.steps?.find(s => s.id === added.stepId)
      if (step) {
        const connectedEdges = project.value.edges?.filter(e =>
          e.source_step_id === added.stepId || e.target_step_id === added.stepId
        ) || []
        for (const edge of connectedEdges) {
          edgesToDelete.push(edge.id)
        }
        step.block_group_id = groupId
        step.group_role = added.role
        step.position_x = added.position.x
        step.position_y = added.position.y
      }
    }

    if (edgesToDelete.length > 0 && project.value.edges) {
      project.value.edges = project.value.edges.filter(e => !edgesToDelete.includes(e.id))
    }

    const stepsInMovedGroups: Array<{ step: Step; delta: { x: number; y: number } }> = []
    for (const movedGroup of data.movedGroups) {
      const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
      if (targetGroup) {
        targetGroup.position_x = movedGroup.position.x
        targetGroup.position_y = movedGroup.position.y
      }
      const stepsInThisGroup = project.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
      for (const step of stepsInThisGroup) {
        step.position_x += movedGroup.delta.x
        step.position_y += movedGroup.delta.y
        stepsInMovedGroups.push({ step, delta: movedGroup.delta })
      }
    }

    const updatePromises: Promise<unknown>[] = []

    for (const pushed of data.pushedBlocks) {
      updatePromises.push(
        blockGroupsApi.removeStep(project.value.id, groupId, pushed.stepId).catch(() => {}).then(() => {
          return projects.updateStep(project.value!.id, pushed.stepId, {
            position: pushed.position,
          })
        })
      )
    }

    for (const edgeId of edgesToDelete) {
      updatePromises.push(projects.deleteEdge(project.value.id, edgeId).catch(() => {}))
    }

    for (const added of data.addedBlocks) {
      updatePromises.push(
        blockGroupsApi.addStep(project.value.id, groupId, {
          step_id: added.stepId,
          group_role: added.role,
        }).then(() => {
          return projects.updateStep(project.value!.id, added.stepId, {
            position: added.position,
          })
        })
      )
    }

    for (const movedGroup of data.movedGroups) {
      updatePromises.push(blockGroupsApi.update(project.value.id, movedGroup.groupId, {
        position: movedGroup.position,
      }))
    }

    for (const { step } of stepsInMovedGroups) {
      updatePromises.push(projects.updateStep(project.value.id, step.id, {
        position: { x: step.position_x, y: step.position_y },
      }))
    }

    await Promise.all(updatePromises)
  } catch (e) {
    toast.error('Failed to update after resize', e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  }
}

// Assign step to a group
async function handleStepAssignGroup(
  stepId: string,
  groupId: string | null,
  position: { x: number; y: number },
  role?: GroupRole,
  movedGroups?: Array<{ groupId: string; position: { x: number; y: number }; delta: { x: number; y: number } }>
) {
  if (!project.value || isReadonly.value) return

  try {
    const step = project.value.steps?.find(s => s.id === stepId)
    if (!step) return

    const currentGroupId = step.block_group_id

    if (currentGroupId !== groupId) {
      const edges = project.value.edges || []
      const connectedEdges = edges.filter(e =>
        e.source_step_id === stepId || e.target_step_id === stepId
      )

      if (project.value.edges) {
        project.value.edges = project.value.edges.filter(e =>
          e.source_step_id !== stepId && e.target_step_id !== stepId
        )
      }

      for (const edge of connectedEdges) {
        try {
          await projects.deleteEdge(project.value.id, edge.id)
        } catch {
          console.warn(`Failed to delete edge ${edge.id}`)
        }
      }
    }

    step.block_group_id = groupId || undefined
    step.group_role = role || undefined
    step.position_x = position.x
    step.position_y = position.y

    const updatePromises: Promise<unknown>[] = []

    if (currentGroupId) {
      await blockGroupsApi.removeStep(project.value.id, currentGroupId, stepId)
    }

    if (groupId) {
      await blockGroupsApi.addStep(project.value.id, groupId, {
        step_id: stepId,
        group_role: role || 'body',
      })
    }

    updatePromises.push(projects.updateStep(project.value.id, stepId, { position }))

    if (movedGroups && movedGroups.length > 0) {
      for (const movedGroup of movedGroups) {
        const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
        if (targetGroup) {
          targetGroup.position_x = movedGroup.position.x
          targetGroup.position_y = movedGroup.position.y
        }

        const stepsInThisGroup = project.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
        for (const groupStep of stepsInThisGroup) {
          groupStep.position_x += movedGroup.delta.x
          groupStep.position_y += movedGroup.delta.y
          updatePromises.push(projects.updateStep(project.value.id, groupStep.id, {
            position: { x: groupStep.position_x, y: groupStep.position_y },
          }))
        }

        updatePromises.push(blockGroupsApi.update(project.value.id, movedGroup.groupId, {
          position: movedGroup.position,
        }))
      }
    }

    await Promise.all(updatePromises)
  } catch (e) {
    toast.error('Failed to update step group', e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  }
}

// Update step position
async function handleUpdateStepPosition(
  stepId: string,
  position: { x: number; y: number },
  movedGroups?: Array<{ groupId: string; position: { x: number; y: number }; delta: { x: number; y: number } }>
) {
  if (!project.value || isReadonly.value) return

  try {
    const step = project.value.steps?.find(s => s.id === stepId)
    if (step) {
      step.position_x = position.x
      step.position_y = position.y
    }

    const updatePromises: Promise<unknown>[] = []
    updatePromises.push(projects.updateStep(project.value.id, stepId, { position }))

    if (movedGroups && movedGroups.length > 0) {
      for (const movedGroup of movedGroups) {
        const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
        if (targetGroup) {
          targetGroup.position_x = movedGroup.position.x
          targetGroup.position_y = movedGroup.position.y
        }

        const stepsInThisGroup = project.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
        for (const groupStep of stepsInThisGroup) {
          groupStep.position_x += movedGroup.delta.x
          groupStep.position_y += movedGroup.delta.y
          updatePromises.push(projects.updateStep(project.value.id, groupStep.id, {
            position: { x: groupStep.position_x, y: groupStep.position_y },
          }))
        }

        updatePromises.push(blockGroupsApi.update(project.value.id, movedGroup.groupId, {
          position: movedGroup.position,
        }))
      }
    }

    await Promise.all(updatePromises)
  } catch (e) {
    console.error('Failed to update step position:', e)
  }
}

// Add edge
async function handleAddEdge(source: string, target: string, sourcePort?: string, targetPort?: string) {
  if (!project.value || isReadonly.value) return

  const sourceInfo = parseNodeId(source)
  const targetInfo = parseNodeId(target)

  try {
    const edgeRequest: Parameters<typeof projects.createEdge>[1] = {
      source_port: sourcePort,
      target_port: targetPort,
    }

    if (sourceInfo.isGroup) {
      edgeRequest.source_block_group_id = sourceInfo.id
    } else {
      edgeRequest.source_step_id = sourceInfo.id
    }

    if (targetInfo.isGroup) {
      edgeRequest.target_block_group_id = targetInfo.id
    } else {
      edgeRequest.target_step_id = targetInfo.id
    }

    const response = await projects.createEdge(project.value.id, edgeRequest as Parameters<typeof projects.createEdge>[1])
    if (project.value && response?.data) {
      project.value.edges = [...(project.value.edges || []), response.data]
    }
  } catch (e) {
    toast.error('Failed to add edge', e instanceof Error ? e.message : undefined)
  }
}

// Delete edge
async function handleDeleteEdge(edgeId: string) {
  if (!project.value || isReadonly.value) return

  try {
    await projects.deleteEdge(project.value.id, edgeId)
    if (project.value?.edges) {
      project.value.edges = project.value.edges.filter(e => e.id !== edgeId)
    }
  } catch (e) {
    toast.error(t('editor.edgeDeleteFailed'), e instanceof Error ? e.message : undefined)
  }
}

// Prepare project data for save
function prepareProjectData() {
  if (!project.value) return null

  return {
    name: project.value.name,
    description: project.value.description,
    variables: project.value.variables,
    steps: (project.value.steps || []).map(s => ({
      id: s.id,
      name: s.name,
      type: s.type,
      config: s.config,
      position_x: s.position_x,
      position_y: s.position_y,
    })),
    edges: (project.value.edges || []).map(e => ({
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

// Save project (simple save, no version)
async function handleSave() {
  const data = prepareProjectData()
  if (!data || !project.value) return

  try {
    saving.value = true
    saveStatus.value = 'saving'
    const response = await projects.saveDraft(project.value.id, data)
    if (response?.data && project.value) {
      project.value.updated_at = response.data.updated_at
    }
    saveStatus.value = 'saved'
  } catch (e) {
    saveStatus.value = 'error'
    toast.error('Failed to save project', e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Open release modal
function handleOpenReleaseModal() {
  showReleaseModal.value = true
}

// Create release (versioned snapshot)
// Note: release_name and release_description would be supported by backend in future
async function handleCreateRelease(_name: string, _description: string) {
  const data = prepareProjectData()
  if (!data || !project.value) return

  try {
    saving.value = true
    // First save the current state
    await projects.saveDraft(project.value.id, data)

    // Then create a versioned release using the save endpoint
    const response = await projects.save(project.value.id, data)

    if (response?.data && project.value) {
      project.value.version = response.data.version
      project.value.status = response.data.status
      project.value.updated_at = response.data.updated_at
    }

    showReleaseModal.value = false
    toast.success(t('editor.releaseCreated'))
  } catch (e) {
    toast.error(t('editor.releaseCreateFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Run project (will be triggered from Start blocks in future)
function _handleRun() {
  if (!project.value) return
  showRunDialog.value = true
}

// Execute project from dialog
async function handleRunFromDialog(input: Record<string, unknown>) {
  if (!project.value) return

  const startStep = project.value.steps?.find(s => s.type === 'start')
  if (!startStep) {
    toast.error(t('execution.errors.noStartStep'))
    return
  }

  try {
    running.value = true
    const response = await runs.create(project.value.id, { triggered_by: 'manual', input, start_step_id: startStep.id })
    showRunDialog.value = false
    toast.success(t('projects.runStarted'))
    window.open(`/runs/${response.data.id}`, '_blank')
  } catch (e) {
    toast.error(t('projects.runFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    running.value = false
  }
}

// Update step name
function handleUpdateStepName(name: string) {
  const step = editorState.selectedStep.value
  if (!step || !project.value) return

  const stepIndex = project.value.steps?.findIndex(s => s.id === step.id)
  if (stepIndex !== undefined && stepIndex >= 0 && project.value.steps) {
    project.value.steps[stepIndex] = {
      ...project.value.steps[stepIndex],
      name,
    }
  }
}

// Update variables
async function handleUpdateVariables(variables: Record<string, unknown>) {
  if (!project.value || isReadonly.value) return

  try {
    (project.value as { variables?: Record<string, unknown> }).variables = variables
    await projects.update(project.value.id, { variables })
  } catch (e) {
    toast.error('Failed to update variables', e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  }
}

// Save step from context panel
async function handleSaveStep(formData: { name: string; type: string; config: Record<string, unknown> }) {
  const step = editorState.selectedStep.value
  if (!step || !project.value) return

  try {
    saving.value = true
    const stepId = step.id
    const stepIndex = project.value.steps?.findIndex(s => s.id === stepId)
    if (stepIndex !== undefined && stepIndex >= 0 && project.value.steps) {
      project.value.steps[stepIndex] = {
        ...project.value.steps[stepIndex],
        name: formData.name,
        config: formData.config as object,
      }
    }
    await projects.updateStep(project.value.id, stepId, formData)
    toast.success(t('editor.stepSaved'))
  } catch (e) {
    toast.error('Failed to save step', e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  } finally {
    saving.value = false
  }
}

// Delete step
async function handleDeleteStep() {
  const step = editorState.selectedStep.value
  if (!step || !project.value) return

  try {
    saving.value = true
    const stepId = step.id
    clearSelection()
    setSelectedRun(null)
    await projects.deleteStep(project.value.id, stepId)

    project.value.steps = (project.value.steps || []).filter(s => s.id !== stepId)
    project.value.edges = (project.value.edges || []).filter(
      e => e.source_step_id !== stepId && e.target_step_id !== stepId
    )
    toast.success(t('editor.stepDeleted'))
  } catch (e) {
    toast.error('Failed to delete step', e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Paste step
async function handlePasteStep(data: { type: StepType; name: string; config: Record<string, unknown> }) {
  if (!project.value || isReadonly.value) return

  try {
    const response = await projects.createStep(project.value.id, {
      name: data.name,
      type: data.type,
      config: data.config,
      position: { x: 200, y: 200 },
    })
    project.value.steps = [...(project.value.steps || []), response.data]
    selectStep(response.data.id)
  } catch (e) {
    toast.error('Failed to paste step', e instanceof Error ? e.message : undefined)
  }
}

// Get output ports for layout
function getOutputPortsForLayout(stepType: StepType, step?: Step): OutputPort[] {
  const config = step?.config as Record<string, unknown> | undefined

  if (stepType === 'switch' && config?.cases) {
    const cases = config.cases as Array<{ name: string; expression?: string; is_default?: boolean }>
    const dynamicPorts: OutputPort[] = []

    for (const caseItem of cases) {
      if (caseItem.is_default) {
        dynamicPorts.push({ name: 'default', label: 'Default', is_default: true })
      } else {
        dynamicPorts.push({
          name: caseItem.name || `case_${dynamicPorts.length + 1}`,
          label: caseItem.name || `Case ${dynamicPorts.length + 1}`,
          is_default: false,
        })
      }
    }

    dynamicPorts.push({ name: 'error', label: 'Error', is_default: false })
    return dynamicPorts
  }

  const blockDef = blockDefinitions.value.find(b => b.slug === stepType)
  let basePorts: OutputPort[] = blockDef?.output_ports || [{ name: 'out', label: 'Output', is_default: false }]

  if (config?.enable_error_port) {
    const hasErrorPort = basePorts.some(p => p.name === 'error')
    if (!hasErrorPort) {
      basePorts = [...basePorts, { name: 'error', label: 'Error', is_default: false }]
    }
  }

  return basePorts
}

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

// Auto-layout
async function handleAutoLayout() {
  if (!project.value || isReadonly.value) return
  const steps = project.value.steps || []
  const edges = project.value.edges || []

  if (steps.length === 0) return

  try {
    saving.value = true

    if (blockGroups.value.length > 0) {
      const layoutResults = calculateLayoutWithGroups(steps, edges, blockGroups.value, {
        getOutputPorts: getOutputPortsForLayout,
        getGroupOutputPorts: getGroupOutputPortsForLayout,
      })

      for (const result of layoutResults.steps) {
        const step = project.value.steps?.find(s => s.id === result.stepId)
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

      const stepUpdatePromises = layoutResults.steps.map(result =>
        projects.updateStep(project.value!.id, result.stepId, {
          position: { x: result.x, y: result.y },
        })
      )
      const groupUpdatePromises = layoutResults.groups.map(result =>
        blockGroupsApi.update(project.value!.id, result.groupId, {
          position: { x: result.x, y: result.y },
          size: { width: result.width, height: result.height },
        })
      )

      await Promise.all([...stepUpdatePromises, ...groupUpdatePromises])
    } else {
      const layoutResults = calculateLayout(steps, edges, {
        getOutputPorts: getOutputPortsForLayout,
      })

      for (const result of layoutResults) {
        const step = project.value.steps?.find(s => s.id === result.stepId)
        if (step) {
          step.position_x = result.x
          step.position_y = result.y
        }
      }

      const updatePromises = layoutResults.map(result =>
        projects.updateStep(project.value!.id, result.stepId, {
          position: { x: result.x, y: result.y },
        })
      )

      await Promise.all(updatePromises)
    }
    toast.success(t('editor.layoutApplied'))
  } catch (e) {
    toast.error('Failed to auto-layout', e instanceof Error ? e.message : undefined)
  } finally {
    saving.value = false
  }
}

// Zoom handlers
function handleZoomIn() {
  dagEditorRef.value?.zoomIn()
}

function handleZoomOut() {
  dagEditorRef.value?.zoomOut()
}

function handleZoomReset() {
  dagEditorRef.value?.zoomTo(1)
}

function handleSetZoom(level: number) {
  dagEditorRef.value?.zoomTo(level)
}

// Handle toggle slide out
function handleToggleSlideOut(panel: Exclude<SlideOutPanel, null>) {
  toggleSlideOut(panel)
}

// Handle open settings (placeholder - will open settings modal)
function handleOpenSettings() {
  // TODO: Implement settings modal
  console.log('Open settings')
}

// Keyboard shortcuts
useKeyboardShortcuts({
  selectedStep: editorState.selectedStep,
  selectedGroupId,
  isReadonly,
  onDelete: handleDeleteStep,
  onDeleteGroup: handleDeleteGroup,
  onCopy: () => {},
  onPaste: handlePasteStep,
  onClearSelection: () => {
    clearSelection()
    selectedGroupId.value = null
    setSelectedRun(null)
  },
})

// Load block definitions
async function loadBlockDefinitions() {
  try {
    const response = await blocksApi.list()
    blockDefinitions.value = response.blocks
  } catch (e) {
    console.error('Failed to load block definitions:', e)
  }
}

// Load latest run
async function loadLatestRun() {
  if (!project.value) return
  try {
    const response = await runs.list(project.value.id, { limit: 1 })
    if (response.data && response.data.length > 0) {
      latestRun.value = response.data[0]
    }
  } catch (e) {
    console.error('Failed to load latest run:', e)
  }
}

onMounted(async () => {
  await loadBlockDefinitions()
  await initializeProject()
  if (project.value) {
    await loadLatestRun()
  }
})
</script>

<template>
  <div class="miro-editor">
    <!-- Loading state (initializing) -->
    <div v-if="initializing" class="loading-overlay">
      <div class="loading-spinner" />
      <p class="loading-text">{{ t('editor.loading') }}</p>
    </div>

    <!-- Error state -->
    <div v-else-if="error && !project" class="error-overlay">
      <div class="error-content">
        <div class="error-icon">!</div>
        <h3>{{ t('editor.loadFailed') }}</h3>
        <p>{{ error }}</p>
        <button class="btn btn-primary" @click="initializeProject">{{ t('common.retry') }}</button>
      </div>
    </div>

    <!-- Main editor -->
    <template v-else-if="project">
      <!-- Editor Layout with Bottom Panel -->
      <div class="editor-layout">
        <!-- Top Area (DAG Canvas) -->
        <div class="canvas-container" :style="{ height: `calc(100% - ${bottomPanelHeight}px)` }">
          <DagEditor
            ref="dagEditorRef"
            :steps="project.steps || []"
            :edges="project.edges || []"
            :block-groups="blockGroups"
            :block-definitions="blockDefinitions"
            :step-runs="stepRunsForDag"
            :readonly="isReadonly"
            :selected-step-id="selectedStepId"
            :selected-group-id="selectedGroupId"
            :show-minimap="false"
            @step:select="handleSelectStep"
            @step:update="handleUpdateStepPosition"
            @step:drop="handleStepDrop"
            @step:assign-group="handleStepAssignGroup"
            @edge:add="handleAddEdge"
            @edge:delete="handleDeleteEdge"
            @pane:click="handlePaneClick"
            @group:select="handleSelectGroup"
            @group:update="handleUpdateGroupPosition"
            @group:drop="handleGroupDrop"
            @group:move-complete="handleGroupMoveComplete"
            @group:resize-complete="handleGroupResizeComplete"
            @auto-layout="handleAutoLayout"
          />
        </div>

        <!-- Bottom Panel (Run History) -->
        <BottomPanel
          :workflow-id="project.id"
          :selected-run-id="selectedRun?.id"
          @run:select="handleRunSelect"
          @height-change="handleBottomPanelHeightChange"
        />
      </div>

      <!-- Floating Header -->
      <FloatingHeader
        :project="project"
        :saving="saving"
        :save-status="saveStatus"
        @save="handleSave"
        @create-release="handleOpenReleaseModal"
        @open-history="handleToggleSlideOut('runs')"
        @select-project="handleSelectProject"
        @create-project="handleCreateProject"
      />

      <!-- Release Modal -->
      <ReleaseModal
        :show="showReleaseModal"
        :project-name="project?.name"
        @close="showReleaseModal = false"
        @create="handleCreateRelease"
      />

      <!-- Floating Account Menu (top right) -->
      <FloatingAccountMenu @open-settings="handleOpenSettings" />

      <!-- Floating Toolbar -->
      <FloatingToolbar :readonly="isReadonly" />

      <!-- Floating Zoom Control -->
      <FloatingZoomControl
        :zoom="zoomLevel"
        :panel-open="showRightPanel"
        @zoom-in="handleZoomIn"
        @zoom-out="handleZoomOut"
        @zoom-reset="handleZoomReset"
        @set-zoom="handleSetZoom"
      />

      <!-- Right Floating Panel (Primary) -->
      <FloatingRightPanel
        :show="showRightPanel"
        :title="rightPanelTitle"
        :shift-left="primaryPanelShift"
        :level="1"
        @close="handleCloseRightPanel"
      >
        <!-- Block Properties Panel -->
        <PropertiesPanel
          v-if="rightPanelMode === 'block' && editorState.selectedStep.value"
          :step="editorState.selectedStep.value"
          :workflow-id="project.id"
          :saving="saving"
          :steps="project.steps || []"
          :edges="project.edges || []"
          :block-definitions="blockDefinitions"
          @save="handleSaveStep"
          @delete="handleDeleteStep"
          @update:name="handleUpdateStepName"
          @run:created="handleRunCreated"
        />

        <!-- Run Details Panel -->
        <RunDetailPanel
          v-else-if="rightPanelMode === 'run' && selectedRun"
          :run="selectedRun"
          :selected-step-run-id="selectedStepRun?.id"
          @close="handleCloseRightPanel"
          @step:select="handleStepRunSelect"
          @refresh="handleRunRefresh"
        />
      </FloatingRightPanel>

      <!-- Nested Floating Panel (Step Details) -->
      <FloatingRightPanel
        :show="showNestedPanel"
        :title="t('execution.stepDetails')"
        :level="2"
        @close="handleCloseNestedPanel"
      >
        <StepDetailPanel
          v-if="selectedStepRun"
          :step-run="selectedStepRun"
        />
      </FloatingRightPanel>

      <!-- Quick Search Modal -->
      <QuickSearchModal
        :open="showQuickSearch"
        :blocks="blockDefinitions"
        @update:open="showQuickSearch = $event"
        @select-block="handleSelectBlock"
        @select-group="handleSelectGroupType"
      />

      <!-- Slide Out Panels -->
      <SlideOutPanel
        :show="activeSlideOut === 'runs'"
        :title="t('editor.runs')"
        :bottom-offset="bottomPanelHeight"
        :no-transition="bottomPanelResizing"
        @close="closeSlideOut"
      >
        <RunHistoryPanel :project-id="project.id" />
      </SlideOutPanel>

      <SlideOutPanel
        :show="activeSlideOut === 'schedules'"
        :title="t('editor.schedules')"
        :bottom-offset="bottomPanelHeight"
        :no-transition="bottomPanelResizing"
        @close="closeSlideOut"
      >
        <SchedulesPanel
          :project-id="project.id"
          :steps="project.steps || []"
        />
      </SlideOutPanel>

      <SlideOutPanel
        :show="activeSlideOut === 'variables'"
        :title="t('editor.variables')"
        :bottom-offset="bottomPanelHeight"
        :no-transition="bottomPanelResizing"
        @close="closeSlideOut"
      >
        <VariablesPanel
          :project-id="project.id"
          :variables="(project.variables as Record<string, unknown>) || {}"
          :readonly="isReadonly"
          @update:variables="handleUpdateVariables"
        />
      </SlideOutPanel>

      <!-- Run Dialog -->
      <RunDialog
        :show="showRunDialog"
        :workflow-id="project.id"
        :workflow-name="project.name"
        :steps="project.steps || []"
        :edges="project.edges || []"
        :blocks="blockDefinitions"
        @close="showRunDialog = false"
        @run="handleRunFromDialog"
      />
    </template>
  </div>
</template>

<style scoped>
.miro-editor {
  position: fixed;
  inset: 0;
  overflow: hidden;
  background: #f8f9fa;
}

.editor-layout {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
}

.canvas-container {
  position: relative;
  flex: 1;
  min-height: 0;
}

/* Loading overlay */
.loading-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #f8f9fa;
  z-index: 50;
}

.loading-spinner {
  width: 48px;
  height: 48px;
  border: 3px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.loading-text {
  margin-top: 1rem;
  color: #6b7280;
  font-size: 0.875rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Error overlay */
.error-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8f9fa;
  z-index: 50;
}

.error-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  max-width: 400px;
  padding: 2rem;
}

.error-icon {
  width: 48px;
  height: 48px;
  background: #dc2626;
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 1.5rem;
  margin-bottom: 1rem;
}

.error-content h3 {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
  color: #111827;
}

.error-content p {
  color: #6b7280;
  margin-bottom: 1.5rem;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
}

.btn-primary {
  background: #3b82f6;
  color: white;
}

.btn-primary:hover {
  background: #2563eb;
}
</style>
