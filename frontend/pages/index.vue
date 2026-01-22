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

import type { Project, Step, StepType, BlockDefinition, BlockGroup, BlockGroupType, Run, GroupRole, OutputPort, StepRun, AgentConfig } from '~/types/api'
import type DagEditor from '~/components/dag-editor/DagEditor.vue'
import type { SlideOutPanel } from '~/composables/useEditorState'
import { calculateLayout, calculateLayoutWithGroups, parseNodeId } from '~/utils/graph-layout'
import { useCommandHistory } from '~/composables/useCommandHistory'
import {
  CreateStepCommand,
  DeleteStepCommand,
} from '~/composables/commands/StepCommands'
import {
  CreateEdgeCommand,
  DeleteEdgeCommand,
} from '~/composables/commands/EdgeCommands'
import {
  CreateGroupCommand,
  DeleteGroupCommand,
} from '~/composables/commands/GroupCommands'
import {
  WorkflowChangeCommand,
  type WorkflowChanges,
  type StepChange,
  type GroupChange,
  type EdgeChange,
} from '~/composables/commands/WorkflowChangeCommand'
import type { ProposalChange } from '~/components/workflow-editor/CopilotProposalCard.vue'

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

// Command history for Undo/Redo
const commandHistory = useCommandHistory()

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

// Welcome dialog state (shows on new project creation)
const showWelcomeDialog = ref(false)

// Right panel mode: 'block' for properties, 'run' for run details, 'group' for group properties
type RightPanelMode = 'block' | 'run' | 'group'

// Quick search modal
const showQuickSearch = ref(false)

// Release modal state
const showReleaseModal = ref(false)

// Environment variables modal state
const showVariablesModal = ref(false)

// Settings modal state
const showSettingsModal = ref(false)
const settingsInitialTab = ref<string | undefined>(undefined)

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

// Selected group computed
const selectedGroup = computed(() => {
  if (!selectedGroupId.value) return null
  return blockGroups.value.find(g => g.id === selectedGroupId.value) || null
})

// Child steps for selected group
const childStepsForSelectedGroup = computed(() => {
  if (!selectedGroupId.value || !project.value?.steps) return []
  return project.value.steps.filter(s => s.block_group_id === selectedGroupId.value)
})

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
async function createNewProject(skipWelcome = false) {
  try {
    const response = await projects.create({
      name: t('editor.untitledProject'),
      description: '',
    })
    project.value = response.data
    blockGroups.value = []
    setCurrentProjectId(response.data.id)
    setProjectInUrl(response.data.id)

    // Show welcome dialog for new projects (unless skipped)
    if (!skipWelcome) {
      showWelcomeDialog.value = true
    }
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

// Trigger block mapping: block slug -> trigger_type
const triggerBlockMap: Record<string, 'manual' | 'webhook' | 'schedule'> = {
  'manual_trigger': 'manual',
  'schedule_trigger': 'schedule',
  'webhook_trigger': 'webhook',
  'start': 'manual',
}

// Default trigger configs by trigger type
function getDefaultTriggerConfig(triggerType: string): object {
  switch (triggerType) {
    case 'schedule':
      return { cron_expression: '0 9 * * *', timezone: 'Asia/Tokyo', enabled: true }
    case 'webhook':
      return { secret: '', allowed_ips: [] }
    case 'manual':
    default:
      return {}
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

  // Check if this is a trigger block
  const triggerType = triggerBlockMap[data.type]

  try {
    const cmd = new CreateStepCommand(
      project.value.id,
      {
        name: data.name,
        type: data.type,
        config: defaultConfigs[data.type] || {},
        position: data.position,
        trigger_type: triggerType,
        trigger_config: triggerType ? getDefaultTriggerConfig(triggerType) : undefined,
      },
      projects,
      () => project.value
    )
    await commandHistory.execute(cmd)

    const createdStepId = cmd.getCreatedStepId()

    // If dropped inside a group, add the step to the group
    if (data.groupId && createdStepId) {
      await blockGroupsApi.addStep(project.value.id, data.groupId, {
        step_id: createdStepId,
        group_role: data.groupRole || 'body',
      })
    }

    if (createdStepId) {
      selectStep(createdStepId)
    }
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
  return editorState.selectedStep.value !== null || selectedRun.value !== null || selectedGroupId.value !== null
})

const rightPanelMode = computed<RightPanelMode>(() => {
  // Block editing takes priority
  if (editorState.selectedStep.value !== null) return 'block'
  // Group editing second
  if (selectedGroupId.value !== null) return 'group'
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

// Handle open settings from credential bindings - open settings modal with credentials tab
function handleOpenSettingsCredentials() {
  settingsInitialTab.value = 'credentials'
  showSettingsModal.value = true
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
  selectedGroupId.value = null
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

  const group = blockGroups.value.find(g => g.id === selectedGroupId.value)
  if (!group) return

  try {
    saving.value = true
    selectedGroupId.value = null

    const cmd = new DeleteGroupCommand(
      project.value.id,
      group,
      blockGroupsApi,
      () => blockGroups.value,
      (groups) => { blockGroups.value = groups },
      () => project.value
    )
    await commandHistory.execute(cmd)
  } catch (e) {
    toast.error(t('editor.groupDeleteFailed'), e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  } finally {
    saving.value = false
  }
}

// Update group config (for agent groups)
async function handleUpdateGroupConfig(config: AgentConfig) {
  if (!selectedGroupId.value || isReadonly.value || !project.value) return

  try {
    // Update local state
    const group = blockGroups.value.find(g => g.id === selectedGroupId.value)
    if (group) {
      group.config = config
    }

    // Save to API
    await blockGroupsApi.update(project.value.id, selectedGroupId.value, { config })
  } catch (e) {
    toast.error('Failed to update group config', e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  }
}

async function handleUpdateGroupPosition(groupId: string, updates: { position?: { x: number; y: number }; size?: { width: number; height: number } }) {
  if (isReadonly.value || !project.value) return

  const group = blockGroups.value.find(g => g.id === groupId)
  if (!group) return

  // Capture before state
  const beforeState: GroupChange['before'] = {
    position_x: group.position_x,
    position_y: group.position_y,
    width: group.width,
    height: group.height,
  }

  // Compute after state
  const afterState: GroupChange['after'] = {
    position_x: updates.position?.x ?? group.position_x,
    position_y: updates.position?.y ?? group.position_y,
    width: updates.size?.width ?? group.width,
    height: updates.size?.height ?? group.height,
  }

  // Skip if nothing changed
  if (beforeState.position_x === afterState.position_x &&
      beforeState.position_y === afterState.position_y &&
      beforeState.width === afterState.width &&
      beforeState.height === afterState.height) {
    return
  }

  const changes: WorkflowChanges = {
    groups: [{
      id: groupId,
      before: beforeState,
      after: afterState,
    }],
  }

  try {
    const cmd = new WorkflowChangeCommand(
      project.value.id,
      changes,
      `Update group: ${group.name}`,
      projects,
      blockGroupsApi,
      () => project.value,
      () => blockGroups.value
    )
    await commandHistory.execute(cmd)
  } catch (e) {
    toast.error('Failed to update group', e instanceof Error ? e.message : undefined)
    await loadProject(project.value.id)
  }
}

async function handleGroupDrop(data: { type: BlockGroupType; name: string; position: { x: number; y: number } }) {
  if (!project.value || isReadonly.value) return
  try {
    const cmd = new CreateGroupCommand(
      project.value.id,
      {
        name: data.name,
        type: data.type,
        position: data.position,
        size: { width: 400, height: 300 },
      },
      blockGroupsApi,
      () => blockGroups.value,
      (groups) => { blockGroups.value = groups }
    )
    await commandHistory.execute(cmd)
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

  const group = blockGroups.value.find(g => g.id === groupId)
  if (!group) return

  // Build changes for undo/redo
  const stepChanges: StepChange[] = []
  const groupChanges: GroupChange[] = []
  const edgeChanges: EdgeChange[] = []

  // 1. Main group position change
  groupChanges.push({
    id: groupId,
    before: {
      position_x: data.position.x - data.delta.x,
      position_y: data.position.y - data.delta.y,
    },
    after: {
      position_x: data.position.x,
      position_y: data.position.y,
    },
  })

  // 2. Steps inside the main group (move with delta)
  const stepsInGroup = project.value.steps?.filter(s => s.block_group_id === groupId) || []
  for (const step of stepsInGroup) {
    stepChanges.push({
      id: step.id,
      before: {
        position_x: step.position_x,
        position_y: step.position_y,
      },
      after: {
        position_x: step.position_x + data.delta.x,
        position_y: step.position_y + data.delta.y,
      },
    })
  }

  // 3. Pushed blocks (steps pushed out of group's way)
  for (const pushed of data.pushedBlocks) {
    const step = project.value.steps?.find(s => s.id === pushed.stepId)
    if (step) {
      stepChanges.push({
        id: pushed.stepId,
        before: {
          position_x: step.position_x,
          position_y: step.position_y,
        },
        after: {
          position_x: pushed.position.x,
          position_y: pushed.position.y,
        },
      })
    }
  }

  // 4. Added blocks (steps added to the group)
  for (const added of data.addedBlocks) {
    const step = project.value.steps?.find(s => s.id === added.stepId)
    if (step) {
      // Record edges to delete
      const connectedEdges = project.value.edges?.filter(e =>
        e.source_step_id === added.stepId || e.target_step_id === added.stepId
      ) || []
      for (const edge of connectedEdges) {
        edgeChanges.push({
          id: edge.id,
          action: 'delete',
          data: { ...edge },
        })
      }

      stepChanges.push({
        id: added.stepId,
        before: {
          position_x: step.position_x,
          position_y: step.position_y,
          block_group_id: step.block_group_id ?? null,
          group_role: step.group_role ?? null,
        },
        after: {
          position_x: added.position.x,
          position_y: added.position.y,
          block_group_id: groupId,
          group_role: added.role,
        },
      })
    }
  }

  // 5. Moved groups (linked groups that moved together)
  for (const movedGroup of data.movedGroups) {
    const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
    if (targetGroup) {
      groupChanges.push({
        id: movedGroup.groupId,
        before: {
          position_x: movedGroup.position.x - movedGroup.delta.x,
          position_y: movedGroup.position.y - movedGroup.delta.y,
        },
        after: {
          position_x: movedGroup.position.x,
          position_y: movedGroup.position.y,
        },
      })

      // Steps in moved groups
      const stepsInThisGroup = project.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
      for (const step of stepsInThisGroup) {
        stepChanges.push({
          id: step.id,
          before: {
            position_x: step.position_x,
            position_y: step.position_y,
          },
          after: {
            position_x: step.position_x + movedGroup.delta.x,
            position_y: step.position_y + movedGroup.delta.y,
          },
        })
      }
    }
  }

  const changes: WorkflowChanges = {
    steps: stepChanges.length > 0 ? stepChanges : undefined,
    groups: groupChanges.length > 0 ? groupChanges : undefined,
    edges: edgeChanges.length > 0 ? edgeChanges : undefined,
  }

  try {
    const cmd = new WorkflowChangeCommand(
      project.value.id,
      changes,
      `Move group: ${group.name}`,
      projects,
      blockGroupsApi,
      () => project.value,
      () => blockGroups.value
    )
    await commandHistory.execute(cmd)
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

  const group = blockGroups.value.find(g => g.id === groupId)
  if (!group) return

  // Build changes for undo/redo
  const stepChanges: StepChange[] = []
  const groupChanges: GroupChange[] = []
  const edgeChanges: EdgeChange[] = []

  // 1. Main group position/size change
  groupChanges.push({
    id: groupId,
    before: {
      position_x: group.position_x,
      position_y: group.position_y,
      width: group.width,
      height: group.height,
    },
    after: {
      position_x: data.position.x,
      position_y: data.position.y,
      width: data.size.width,
      height: data.size.height,
    },
  })

  // 2. Pushed blocks (steps removed from group due to resize)
  for (const pushed of data.pushedBlocks) {
    const step = project.value.steps?.find(s => s.id === pushed.stepId)
    if (step) {
      stepChanges.push({
        id: pushed.stepId,
        before: {
          position_x: step.position_x,
          position_y: step.position_y,
          block_group_id: step.block_group_id ?? null,
          group_role: step.group_role ?? null,
        },
        after: {
          position_x: pushed.position.x,
          position_y: pushed.position.y,
          block_group_id: null,
          group_role: null,
        },
      })
    }
  }

  // 3. Added blocks (steps added to group due to resize)
  for (const added of data.addedBlocks) {
    const step = project.value.steps?.find(s => s.id === added.stepId)
    if (step) {
      // Record edges to delete
      const connectedEdges = project.value.edges?.filter(e =>
        e.source_step_id === added.stepId || e.target_step_id === added.stepId
      ) || []
      for (const edge of connectedEdges) {
        edgeChanges.push({
          id: edge.id,
          action: 'delete',
          data: { ...edge },
        })
      }

      stepChanges.push({
        id: added.stepId,
        before: {
          position_x: step.position_x,
          position_y: step.position_y,
          block_group_id: step.block_group_id ?? null,
          group_role: step.group_role ?? null,
        },
        after: {
          position_x: added.position.x,
          position_y: added.position.y,
          block_group_id: groupId,
          group_role: added.role,
        },
      })
    }
  }

  // 4. Moved groups (linked groups that moved during resize)
  for (const movedGroup of data.movedGroups) {
    const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
    if (targetGroup) {
      groupChanges.push({
        id: movedGroup.groupId,
        before: {
          position_x: movedGroup.position.x - movedGroup.delta.x,
          position_y: movedGroup.position.y - movedGroup.delta.y,
        },
        after: {
          position_x: movedGroup.position.x,
          position_y: movedGroup.position.y,
        },
      })

      // Steps in moved groups
      const stepsInThisGroup = project.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
      for (const step of stepsInThisGroup) {
        stepChanges.push({
          id: step.id,
          before: {
            position_x: step.position_x,
            position_y: step.position_y,
          },
          after: {
            position_x: step.position_x + movedGroup.delta.x,
            position_y: step.position_y + movedGroup.delta.y,
          },
        })
      }
    }
  }

  // Skip if no meaningful changes
  if (stepChanges.length === 0 && groupChanges.length === 0 && edgeChanges.length === 0) {
    return
  }

  const changes: WorkflowChanges = {
    steps: stepChanges.length > 0 ? stepChanges : undefined,
    groups: groupChanges.length > 0 ? groupChanges : undefined,
    edges: edgeChanges.length > 0 ? edgeChanges : undefined,
  }

  try {
    const cmd = new WorkflowChangeCommand(
      project.value.id,
      changes,
      `Resize group: ${group.name}`,
      projects,
      blockGroupsApi,
      () => project.value,
      () => blockGroups.value
    )
    await commandHistory.execute(cmd)
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

  const step = project.value.steps?.find(s => s.id === stepId)
  if (!step) return

  // Build changes for undo/redo
  const stepChanges: StepChange[] = []
  const groupChanges: GroupChange[] = []

  // 1. Main step position change
  stepChanges.push({
    id: stepId,
    before: {
      position_x: step.position_x,
      position_y: step.position_y,
    },
    after: {
      position_x: position.x,
      position_y: position.y,
    },
  })

  // 2. Moved groups (linked groups that moved due to step collision)
  if (movedGroups && movedGroups.length > 0) {
    for (const movedGroup of movedGroups) {
      const targetGroup = blockGroups.value.find(g => g.id === movedGroup.groupId)
      if (targetGroup) {
        groupChanges.push({
          id: movedGroup.groupId,
          before: {
            position_x: movedGroup.position.x - movedGroup.delta.x,
            position_y: movedGroup.position.y - movedGroup.delta.y,
          },
          after: {
            position_x: movedGroup.position.x,
            position_y: movedGroup.position.y,
          },
        })

        // Steps in moved groups
        const stepsInThisGroup = project.value.steps?.filter(s => s.block_group_id === movedGroup.groupId) || []
        for (const groupStep of stepsInThisGroup) {
          stepChanges.push({
            id: groupStep.id,
            before: {
              position_x: groupStep.position_x,
              position_y: groupStep.position_y,
            },
            after: {
              position_x: groupStep.position_x + movedGroup.delta.x,
              position_y: groupStep.position_y + movedGroup.delta.y,
            },
          })
        }
      }
    }
  }

  const changes: WorkflowChanges = {
    steps: stepChanges.length > 0 ? stepChanges : undefined,
    groups: groupChanges.length > 0 ? groupChanges : undefined,
  }

  try {
    const cmd = new WorkflowChangeCommand(
      project.value.id,
      changes,
      `Move step: ${step.name}`,
      projects,
      blockGroupsApi,
      () => project.value,
      () => blockGroups.value
    )
    await commandHistory.execute(cmd)
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
    const edgeData: Parameters<typeof projects.createEdge>[1] = {
      source_port: sourcePort,
      target_port: targetPort,
    }

    if (sourceInfo.isGroup) {
      edgeData.source_block_group_id = sourceInfo.id
    } else {
      edgeData.source_step_id = sourceInfo.id
    }

    if (targetInfo.isGroup) {
      edgeData.target_block_group_id = targetInfo.id
    } else {
      edgeData.target_step_id = targetInfo.id
    }

    const cmd = new CreateEdgeCommand(project.value.id, edgeData, projects, () => project.value)
    await commandHistory.execute(cmd)
  } catch (e) {
    toast.error('Failed to add edge', e instanceof Error ? e.message : undefined)
  }
}

// Delete edge
async function handleDeleteEdge(edgeId: string) {
  if (!project.value || isReadonly.value) return

  const edge = project.value.edges?.find(e => e.id === edgeId)
  if (!edge) return

  try {
    const cmd = new DeleteEdgeCommand(project.value.id, edge, projects, () => project.value)
    await commandHistory.execute(cmd)
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
async function handleRunFromDialog(input: Record<string, unknown>, startStepId: string) {
  if (!project.value) return

  try {
    running.value = true
    const response = await runs.create(project.value.id, { triggered_by: 'manual', input, start_step_id: startStepId })
    showRunDialog.value = false
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

  // Capture before state
  const beforeState: StepChange['before'] = {
    name: step.name,
    config: step.config,
  }

  // Compute after state
  const afterState: StepChange['after'] = {
    name: formData.name,
    config: formData.config as object,
  }

  // Skip if nothing changed
  if (beforeState.name === afterState.name &&
      JSON.stringify(beforeState.config) === JSON.stringify(afterState.config)) {
    return
  }

  const changes: WorkflowChanges = {
    steps: [{
      id: step.id,
      before: beforeState,
      after: afterState,
    }],
  }

  try {
    saving.value = true
    const cmd = new WorkflowChangeCommand(
      project.value.id,
      changes,
      `Update step: ${step.name}`,
      projects,
      blockGroupsApi,
      () => project.value,
      () => blockGroups.value
    )
    await commandHistory.execute(cmd)
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
    clearSelection()
    setSelectedRun(null)

    const cmd = new DeleteStepCommand(project.value.id, step, projects, () => project.value)
    await commandHistory.execute(cmd)
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
  agent: [
    { name: 'out', label: 'Response', is_default: true },
    { name: 'error', label: 'Error', is_default: false },
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

// Welcome dialog handlers
function handleWelcomeSubmit(prompt: string) {
  showWelcomeDialog.value = false
  // Open Copilot sidebar and send the prompt
  const editorState = useEditorState()
  editorState.openCopilotSidebar()
  // Emit a custom event to send the prompt to CopilotTab
  // We use nextTick to ensure the sidebar is mounted
  nextTick(() => {
    window.dispatchEvent(new CustomEvent('copilot-send-prompt', { detail: { prompt } }))
  })
}

function handleWelcomeSkip() {
  showWelcomeDialog.value = false
}

// Handle Copilot changes applied
async function handleCopilotChangesApplied(changes: ProposalChange[]) {
  if (!project.value) return

  const projectId = project.value.id
  const tempIdToRealId = new Map<string, string>()

  try {
    // Process changes in order: creates first, then updates, then edges, then deletes
    const creates = changes.filter(c => c.type === 'step:create')
    const updates = changes.filter(c => c.type === 'step:update')
    const edgeCreates = changes.filter(c => c.type === 'edge:create')
    const edgeDeletes = changes.filter(c => c.type === 'edge:delete')
    const stepDeletes = changes.filter(c => c.type === 'step:delete')

    // 1. Create steps
    for (const change of creates) {
      const response = await projects.createStep(projectId, {
        name: change.name || 'New Step',
        type: change.step_type as StepType,
        config: change.config as Record<string, unknown>,
        position: change.position || { x: 200, y: 200 },
      })

      // Map temp ID to real ID for edge creation
      if (change.temp_id) {
        tempIdToRealId.set(change.temp_id, response.data.id)
      }

      // Update local state
      project.value.steps = [...(project.value.steps || []), response.data]
    }

    // 2. Update steps
    for (const change of updates) {
      if (!change.step_id || !change.patch) continue

      const patch = change.patch as Record<string, unknown>
      const updateData: Parameters<typeof projects.updateStep>[2] = {}

      if (patch.name !== undefined) updateData.name = patch.name as string
      if (patch.config !== undefined) updateData.config = patch.config as Record<string, unknown>
      if (patch.position !== undefined) updateData.position = patch.position as { x: number; y: number }

      await projects.updateStep(projectId, change.step_id, updateData)

      // Update local state
      const step = project.value.steps?.find(s => s.id === change.step_id)
      if (step) {
        if (patch.name !== undefined) step.name = patch.name as string
        if (patch.config !== undefined && patch.config !== null) step.config = patch.config as object
        if (patch.position !== undefined) {
          const pos = patch.position as { x: number; y: number }
          step.position_x = pos.x
          step.position_y = pos.y
        }
      }
    }

    // 3. Create edges (resolve temp IDs)
    for (const change of edgeCreates) {
      let sourceId = change.source_id || ''
      let targetId = change.target_id || ''

      // Resolve temp IDs
      if (sourceId && tempIdToRealId.has(sourceId)) {
        sourceId = tempIdToRealId.get(sourceId)!
      }
      if (targetId && tempIdToRealId.has(targetId)) {
        targetId = tempIdToRealId.get(targetId)!
      }

      const response = await projects.createEdge(projectId, {
        source_step_id: sourceId,
        target_step_id: targetId,
        source_port: change.source_port,
        target_port: change.target_port,
      })

      // Update local state
      project.value.edges = [...(project.value.edges || []), response.data]
    }

    // 4. Delete edges
    for (const change of edgeDeletes) {
      if (!change.edge_id) continue

      await projects.deleteEdge(projectId, change.edge_id)

      // Update local state
      if (project.value.edges) {
        project.value.edges = project.value.edges.filter(e => e.id !== change.edge_id)
      }
    }

    // 5. Delete steps
    for (const change of stepDeletes) {
      if (!change.step_id) continue

      await projects.deleteStep(projectId, change.step_id)

      // Update local state
      if (project.value.steps) {
        project.value.steps = project.value.steps.filter(s => s.id !== change.step_id)
      }
    }

  } catch (e) {
    console.error('Failed to apply Copilot changes:', e)
    toast.error('変更の適用に失敗しました')
    // Reload to restore consistent state
    await loadProject(projectId)
  }
}

// Handle Copilot changes preview
function handleCopilotChangesPreview() {
  // Optional: highlight preview in DAG editor
  // useCopilotDraft().previewState already provides this
}

// Handle workflow updated by Copilot (refresh data from API)
async function handleWorkflowUpdated() {
  if (!project.value) return
  try {
    // Reload project data from API to get any backend changes
    await loadProject(project.value.id)
  } catch (e) {
    console.error('Failed to refresh workflow after Copilot update:', e)
  }
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
  onUndo: async () => {
    try {
      await commandHistory.undo()
    } catch (e) {
      toast.error(t('editor.undoFailed'), e instanceof Error ? e.message : undefined)
    }
  },
  onRedo: async () => {
    try {
      await commandHistory.redo()
    } catch (e) {
      toast.error(t('editor.redoFailed'), e instanceof Error ? e.message : undefined)
    }
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
        @open-variables="showVariablesModal = true"
        @select-project="handleSelectProject"
        @create-project="handleCreateProject"
      />

      <!-- Release Modal (with integrated publish checklist) -->
      <ReleaseModal
        :show="showReleaseModal"
        :project-name="project?.name"
        :steps="project.steps || []"
        :edges="project.edges || []"
        :block-definitions="blockDefinitions"
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
          :steps="project.steps || []"
          :edges="project.edges || []"
          :block-definitions="blockDefinitions"
          @save="handleSaveStep"
          @delete="handleDeleteStep"
          @update:name="handleUpdateStepName"
          @run:created="handleRunCreated"
          @open-settings="handleOpenSettingsCredentials"
        />

        <!-- Agent Group Panel -->
        <AgentGroupPanel
          v-else-if="rightPanelMode === 'group' && selectedGroup"
          :group="selectedGroup"
          :child-steps="childStepsForSelectedGroup"
          :readonly="isReadonly"
          @update:config="handleUpdateGroupConfig"
          @close="handleCloseRightPanel"
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

      <!-- Copilot Sidebar (Fixed Right Panel) -->
      <CopilotSidebar
        :workflow-id="project.id"
        @changes:applied="handleCopilotChangesApplied"
        @changes:preview="handleCopilotChangesPreview"
        @workflow:updated="handleWorkflowUpdated"
      />

      <!-- Environment Variables Modal -->
      <EnvironmentVariablesModal
        :show="showVariablesModal"
        :project-id="project.id"
        :project-variables="(project.variables as Record<string, unknown>) || {}"
        :readonly="isReadonly"
        @close="showVariablesModal = false"
        @update:project-variables="handleUpdateVariables"
      />

      <!-- Settings Modal -->
      <SettingsModal
        :show="showSettingsModal"
        :initial-tab="settingsInitialTab"
        @close="showSettingsModal = false"
      />

      <!-- Run Dialog -->
      <RunDialog
        :show="showRunDialog"
        :workflow-id="project.id"
        :workflow-name="project.name"
        :steps="project.steps || []"
        :edges="project.edges || []"
        :blocks="blockDefinitions"
        :selected-start-step-id="editorState.selectedStep.value?.type === 'start' ? editorState.selectedStep.value.id : null"
        @close="showRunDialog = false"
        @run="handleRunFromDialog"
      />

      <!-- Welcome Dialog (Copilot-first onboarding) -->
      <WelcomeDialog
        :show="showWelcomeDialog"
        @submit="handleWelcomeSubmit"
        @skip-to-canvas="handleWelcomeSkip"
        @close="handleWelcomeSkip"
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
