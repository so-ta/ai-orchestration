<script setup lang="ts">
import { VueFlow, useVueFlow, Handle, Position, MarkerType, type Node } from '@vue-flow/core'
import { MiniMap } from '@vue-flow/minimap'
import { NodeResizer, type OnResizeStart, type OnResize, type OnResizeEnd } from '@vue-flow/node-resizer'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'
import '@vue-flow/minimap/dist/style.css'
import '@vue-flow/node-resizer/dist/style.css'
import type { Step, Edge, StepType, StepRun, BlockDefinition, BlockGroup, BlockGroupType, GroupRole } from '~/types/api'
import NodeIcon from './NodeIcon.vue'
import NodeStatusOverlay from './NodeStatusOverlay.vue'
import { useCopilotOffset } from '~/composables/useFloatingLayout'
import { useEditorState } from '~/composables/useEditorState'
import {
  GROUP_NODE_PREFIX,
  getGroupUuidFromNodeId,
  snapToGrid,
  determineRoleInGroup,
} from './utils/dagHelpers'

// Import composables
import { useDropZone } from './composables/useDropZone'
import { useFlowEdges, type PreviewState } from './composables/useFlowEdges'
import { useFlowNodes } from './composables/useFlowNodes'
import { useGroupResize, type PushedBlock, type AddedBlock } from './composables/useGroupResize'
import { useNodeDragHandler } from './composables/useNodeDragHandler'
import type { MovedGroup } from './composables/useCascadePush'

const props = defineProps<{
  steps: Step[]
  edges: Edge[]
  blockGroups?: BlockGroup[]
  readonly?: boolean
  selectedStepId?: string | null
  selectedGroupId?: string | null
  stepRuns?: StepRun[]
  blockDefinitions?: BlockDefinition[]
  showMinimap?: boolean
  previewState?: PreviewState | null
}>()

const emit = defineEmits<{
  (e: 'step:select', step: Step): void
  (e: 'step:update', stepId: string, position: { x: number; y: number }, movedGroups?: MovedGroup[]): void
  (e: 'step:drop', data: { type: StepType; name: string; position: { x: number; y: number }; groupId?: string; groupRole?: GroupRole }): void
  (e: 'step:assign-group', stepId: string, groupId: string | null, position: { x: number; y: number }, role?: GroupRole, movedGroups?: MovedGroup[]): void
  (e: 'edge:add', source: string, target: string, sourcePort?: string, targetPort?: string): void
  (e: 'edge:delete', edgeId: string): void
  (e: 'pane:click' | 'autoLayout'): void
  (e: 'step:showDetails', stepRun: StepRun): void
  (e: 'group:select', group: BlockGroup): void
  (e: 'group:update', groupId: string, updates: { position?: { x: number; y: number }; size?: { width: number; height: number } }): void
  (e: 'group:drop', data: { type: BlockGroupType; name: string; position: { x: number; y: number } }): void
  (e: 'group:move-complete', groupId: string, data: {
    position: { x: number; y: number }
    delta: { x: number; y: number }
    pushedBlocks: PushedBlock[]
    addedBlocks: AddedBlock[]
    movedGroups: MovedGroup[]
  }): void
  (e: 'group:resize-complete', groupId: string, data: {
    position: { x: number; y: number }
    size: { width: number; height: number }
    pushedBlocks: PushedBlock[]
    addedBlocks: AddedBlock[]
    movedGroups: MovedGroup[]
  }): void
}>()

// Vue Flow hooks
const { onConnect, onNodeDragStop, onPaneClick, onEdgeClick, project, updateNode, setNodes, getNodes, removeNodes, viewport, zoomIn, zoomOut, zoomTo } = useVueFlow()

// Copilot Sidebar offset
const { value: copilotOffset, isResizing: copilotResizing } = useCopilotOffset(12)
const { copilotSidebarOpen, toggleCopilotSidebar } = useEditorState()

// Reactive refs from props
const stepsRef = computed(() => props.steps)
const edgesRef = computed(() => props.edges)
const blockGroupsRef = computed(() => props.blockGroups)
const blockDefinitionsRef = computed(() => props.blockDefinitions)
const stepRunsRef = computed(() => props.stepRuns)
const selectedStepIdRef = computed(() => props.selectedStepId)
const selectedGroupIdRef = computed(() => props.selectedGroupId)
const previewStateRef = computed(() => props.previewState)
const readonlyRef = computed(() => props.readonly)

// Initialize composables
const { findDropZone, snapToValidPosition } = useDropZone({ blockGroups: blockGroupsRef })

const {
  flowEdges,
  getOutgoingEdgesFromStep,
  getOutgoingEdgesFromGroup,
  getDefaultSourcePort,
  getDefaultTargetPort,
} = useFlowEdges({
  edges: edgesRef,
  steps: stepsRef,
  stepRuns: stepRunsRef,
  blockDefinitions: blockDefinitionsRef,
  blockGroups: blockGroupsRef,
  selectedEdgeId: ref(null),
  previewState: previewStateRef,
})

const {
  nodes,
  getStepColor,
} = useFlowNodes({
  steps: stepsRef,
  blockGroups: blockGroupsRef,
  blockDefinitions: blockDefinitionsRef,
  stepRuns: stepRunsRef,
  selectedStepId: selectedStepIdRef,
  selectedGroupId: selectedGroupIdRef,
  previewState: previewStateRef,
})

const { onGroupResizeStart, onGroupResize, onGroupResizeEnd } = useGroupResize({
  steps: stepsRef,
  blockGroups: blockGroupsRef,
  readonly: readonlyRef,
  updateNode,
  emit: {
    groupUpdate: (groupId, updates) => emit('group:update', groupId, updates),
    groupResizeComplete: (groupId, data) => emit('group:resize-complete', groupId, data),
  },
})

const { onNodeDragStop: handleNodeDragStop } = useNodeDragHandler({
  steps: stepsRef,
  blockGroups: blockGroupsRef,
  readonly: readonlyRef,
  updateNode,
  emit: {
    stepUpdate: (stepId, position, movedGroups) => emit('step:update', stepId, position, movedGroups),
    stepAssignGroup: (stepId, groupId, position, role, movedGroups) => emit('step:assign-group', stepId, groupId, position, role, movedGroups),
    groupMoveComplete: (groupId, data) => emit('group:move-complete', groupId, data),
  },
})

// Right offset for auto-layout button
const autoLayoutRightOffset = computed(() => {
  if (props.selectedStepId || props.selectedGroupId) {
    return copilotOffset.value + 360 + 12
  }
  return copilotOffset.value
})

// Edge selection state
const selectedEdgeId = ref<string | null>(null)
const edgeClickFlowPosition = ref<{ x: number; y: number } | null>(null)
const dagEditorRef = ref<HTMLElement | null>(null)

// Compute button position from flow coordinates
const edgeDeleteButtonPosition = computed(() => {
  if (!edgeClickFlowPosition.value) return null
  const screenX = edgeClickFlowPosition.value.x * viewport.value.zoom + viewport.value.x
  const screenY = edgeClickFlowPosition.value.y * viewport.value.zoom + viewport.value.y
  return { x: screenX + 10, y: screenY - 30 }
})

// Drag state
const isDragOver = ref(false)
const dragEnterCounter = ref(0)

// Minimum size for group nodes
const MIN_GROUP_WIDTH = 160
const MIN_GROUP_HEIGHT = 150

// Sync Vue Flow's internal state when steps/groups change
function syncVueFlowNodes() {
  nextTick(() => {
    const currentVueFlowNodes = getNodes.value
    const newNodes = nodes.value
    const newNodeIds = new Set(newNodes.map(n => n.id))
    const nodesToRemove = currentVueFlowNodes.filter(n => !newNodeIds.has(n.id))
    if (nodesToRemove.length > 0) {
      removeNodes(nodesToRemove)
    }
    setNodes(newNodes)
  })
}

// Watch step and group IDs for changes
watch(() => props.steps.map(s => s.id).join(','), () => syncVueFlowNodes())
watch(() => (props.blockGroups || []).map(g => g.id).join(','), () => syncVueFlowNodes())

// Handle new connection
onConnect((params) => {
  if (!props.readonly) {
    const sourceNodeId = params.source
    const sourceStep = props.steps.find(s => s.id === sourceNodeId)
    const isSourceGroup = sourceNodeId?.startsWith(GROUP_NODE_PREFIX)

    if (isSourceGroup) {
      const groupId = getGroupUuidFromNodeId(sourceNodeId)
      const existingEdges = getOutgoingEdgesFromGroup(groupId)
      for (const edge of existingEdges) {
        emit('edge:delete', edge.id)
      }
    } else if (sourceStep) {
      const isBranchingBlock = sourceStep.type === 'condition' || sourceStep.type === 'switch'
      const existingEdges = getOutgoingEdgesFromStep(sourceNodeId)
      const allowMultiple = isBranchingBlock && sourceStep.block_group_id

      if (!allowMultiple && existingEdges.length > 0) {
        for (const edge of existingEdges) {
          emit('edge:delete', edge.id)
        }
      }
    }

    const sourcePort = params.sourceHandle || getDefaultSourcePort(sourceNodeId, sourceStep)
    const targetPort = params.targetHandle || getDefaultTargetPort(params.target)
    emit('edge:add', params.source, params.target, sourcePort, targetPort)
  }
})

// Handle edge click
onEdgeClick(({ edge, event }) => {
  if (!props.readonly) {
    const actualEdge = props.edges.find(e => e.id === edge.id || edge.id.includes(e.id))
    if (actualEdge && dagEditorRef.value) {
      selectedEdgeId.value = actualEdge.id
      const mouseEvent = event as MouseEvent
      const rect = dagEditorRef.value.getBoundingClientRect()
      const flowPos = project({ x: mouseEvent.clientX - rect.left, y: mouseEvent.clientY - rect.top })
      edgeClickFlowPosition.value = { x: flowPos.x, y: flowPos.y }
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
    handleNodeDragStop(event.node)
  }
})

// Handle node click
function onNodeClick(event: { node: Node }) {
  selectedEdgeId.value = null
  edgeClickFlowPosition.value = null

  if (event.node.type === 'group') {
    emit('group:select', event.node.data.group)
  } else {
    emit('step:select', event.node.data.step)
    if (event.node.data.stepRun) {
      emit('step:showDetails', event.node.data.stepRun)
    }
  }
}

// Get step run status color
function getStepRunStatusColor(status?: string): string {
  if (!status) return ''
  const colors: Record<string, string> = {
    pending: '#f59e0b',
    running: '#3b82f6',
    completed: '#22c55e',
    failed: '#ef4444',
    skipped: '#94a3b8',
  }
  return colors[status] || '#94a3b8'
}

// Drag and drop handlers
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

  const target = event.currentTarget as HTMLElement
  const rect = target.getBoundingClientRect()
  const flowPosition = project({ x: event.clientX - rect.left, y: event.clientY - rect.top })

  const groupType = event.dataTransfer.getData('group-type') as BlockGroupType
  const groupName = event.dataTransfer.getData('group-name')

  if (groupType) {
    const position = { x: snapToGrid(flowPosition.x - 200), y: snapToGrid(flowPosition.y - 150) }
    emit('group:drop', { type: groupType, name: groupName || 'New Group', position })
    return
  }

  const stepType = event.dataTransfer.getData('step-type') as StepType
  const stepName = event.dataTransfer.getData('step-name') || 'New Step'

  if (!stepType) return

  let positionX = snapToGrid(flowPosition.x - 75)
  let positionY = snapToGrid(flowPosition.y - 30)

  const dropZone = findDropZone(positionX, positionY)
  let targetGroupId: string | undefined
  let targetRole: GroupRole | undefined

  if (dropZone.zone === 'boundary' && dropZone.group) {
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

  emit('step:drop', { type: stepType, name: stepName, position: { x: positionX, y: positionY }, groupId: targetGroupId, groupRole: targetRole })
}

// Get port handle color based on port name
function getPortColor(portName: string): string {
  const portColors: Record<string, string> = {
    'true': '#22c55e',
    'false': '#ef4444',
    'approved': '#22c55e',
    'rejected': '#ef4444',
    'timeout': '#f59e0b',
    'matched': '#22c55e',
    'unmatched': '#94a3b8',
    'loop': '#3b82f6',
    'complete': '#22c55e',
    'item': '#3b82f6',
    'out': '#22c55e',
    'default': '#94a3b8',
    'output': '#94a3b8',
    'input': '#94a3b8',
  }
  return portColors[portName] || '#6366f1'
}

// Check if step type is a trigger block
const triggerBlockTypes = ['start', 'schedule_trigger', 'webhook_trigger']
function isStartNode(type: string): boolean {
  return triggerBlockTypes.includes(type)
}

// Expose zoom functions and viewport for parent components
defineExpose({ viewport, zoomIn, zoomOut, zoomTo })
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
          :class="['dag-group', { 'dag-group-selected': data.isSelected }]"
          :style="{ borderColor: data.color, '--group-color': data.color, '--group-height': `${data.height}px` }"
        >
          <Handle id="group-input" type="target" :position="Position.Left" class="dag-group-handle dag-group-handle-input" :style="{ top: '50%' }" />

          <div class="dag-group-header">
            <span class="dag-group-icon">{{ data.icon }}</span>
            <span class="dag-group-name">{{ data.label }}</span>
          </div>

          <div v-if="!data.hasMultipleZones" class="dag-group-entry" :style="{ color: data.color }">
            <span class="dag-group-entry-arrow">→</span>
            <span class="dag-group-entry-label">Start</span>
          </div>

          <template v-if="data.hasMultipleZones && data.zones">
            <template v-for="(zone, index) in data.zones" :key="zone.role">
              <div
                class="dag-group-section-label"
                :style="{ top: `${32 + (data.height - 32) * zone.top + 8}px`, left: zone.left === 0 ? '12px' : `${data.width * zone.left + 12}px`, color: data.color }"
              >
                <span class="dag-group-section-arrow">→</span>
                {{ zone.label }}
              </div>
              <div v-if="index > 0 && zone.left === 0" class="dag-group-divider-h" :style="{ top: `${32 + (data.height - 32) * zone.top}px`, backgroundColor: data.color }" />
              <div v-if="index > 0 && zone.left > 0" class="dag-group-divider-v" :style="{ left: `${data.width * zone.left - 2}px`, backgroundColor: data.color }" />
            </template>
          </template>

          <div class="dag-group-outputs">
            <Handle
              v-for="(port, index) in data.outputPorts"
              :id="port.name"
              :key="port.name"
              type="source"
              :position="Position.Right"
              class="dag-group-handle dag-group-handle-output"
              :style="{ top: `${50 + (index - (data.outputPorts.length - 1) / 2) * 40}%`, '--handle-color': port.color }"
            />
          </div>
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
          :style="{ '--node-color': getStepColor(data.type) }"
        >
          <div v-if="data.stepRun" class="dag-node-status-miro" :style="{ backgroundColor: getStepRunStatusColor(data.stepRun.status) }" :title="`${data.stepRun.status} - Click for details`" />

          <template v-if="!isStartNode(data.type) && data.inputPorts && data.inputPorts.length > 1">
            <Handle
              v-for="(port, index) in data.inputPorts"
              :id="port.name"
              :key="port.name"
              type="target"
              :position="Position.Left"
              :class="['dag-handle-miro', 'dag-handle-target']"
              :style="{ '--handle-top': `${24 + (index - (data.inputPorts.length - 1) / 2) * 12}px` }"
              :title="port.label"
            />
          </template>
          <Handle v-else-if="!isStartNode(data.type)" id="input" type="target" :position="Position.Left" class="dag-handle-miro dag-handle-target dag-handle-center" />

          <div class="dag-node-icon-box">
            <NodeIcon :icon="data.icon" :color="getStepColor(data.type)" />
            <NodeStatusOverlay
              v-if="data.stepRun"
              :status="data.stepRun.status"
              :output="data.stepRun.output"
            />
          </div>

          <div class="dag-node-label-miro">{{ data.label }}</div>

          <div v-if="data.stepRun?.duration_ms" class="dag-node-duration-miro">
            {{ data.stepRun.duration_ms < 1000 ? `${data.stepRun.duration_ms}ms` : `${(data.stepRun.duration_ms / 1000).toFixed(1)}s` }}
          </div>

          <template v-if="data.outputPorts && data.outputPorts.length > 1">
            <Handle
              v-for="(port, index) in data.outputPorts"
              :id="port.name"
              :key="port.name"
              type="source"
              :position="Position.Right"
              :class="['dag-handle-miro', 'dag-handle-source', 'dag-handle-colored']"
              :style="{ '--handle-top': `${24 + (index - (data.outputPorts.length - 1) / 2) * 12}px`, '--handle-bg': getPortColor(port.name), '--handle-border': getPortColor(port.name) }"
              :title="port.label"
            />
            <div class="dag-port-labels-miro">
              <div
                v-for="(port, index) in data.outputPorts"
                :key="`label-${port.name}`"
                class="dag-port-label-miro"
                :style="{ '--label-top': `${24 + (index - (data.outputPorts.length - 1) / 2) * 12}px`, color: getPortColor(port.name) }"
              >
                {{ port.label }}
              </div>
            </div>
          </template>
          <Handle v-else id="output" type="source" :position="Position.Right" class="dag-handle-miro dag-handle-source dag-handle-center" />
        </div>
      </template>

      <MiniMap
        v-if="props.showMinimap !== false"
        :pannable="true"
        :zoomable="true"
        :node-color="(node: Node) => node.type === 'group' ? node.data.color : getStepColor(node.data.type)"
        class="dag-minimap"
      />
    </VueFlow>

    <!-- Edge Delete Button -->
    <div
      v-if="edgeDeleteButtonPosition && selectedEdgeId && !readonly"
      class="edge-delete-button-container"
      :style="{ left: `${edgeDeleteButtonPosition.x}px`, top: `${edgeDeleteButtonPosition.y}px` }"
    >
      <button class="edge-delete-button-floating" title="エッジを削除 (Delete)" @click.stop="handleDeleteSelectedEdge">
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
    <div class="bottom-right-buttons" :class="{ 'no-transition': copilotResizing }" :style="{ right: autoLayoutRightOffset + 'px' }">
      <button v-if="!readonly" class="action-button" data-tooltip="整形する" @click="emit('autoLayout')">
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="7" height="7" rx="1"/>
          <rect x="14" y="3" width="7" height="7" rx="1"/>
          <rect x="14" y="14" width="7" height="7" rx="1"/>
          <rect x="3" y="14" width="7" height="7" rx="1"/>
        </svg>
      </button>

      <button class="action-button copilot-toggle" :class="{ active: copilotSidebarOpen }" data-tooltip="AI Copilot" @click="toggleCopilotSidebar">
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
  background-image: radial-gradient(circle, #e5e5e5 1px, transparent 1px);
  background-size: 24px 24px;
  position: relative;
  overflow: hidden;
  transition: background-color 0.2s;
}

.dag-editor.drag-over {
  background-color: rgba(59, 130, 246, 0.05);
}

/* Minimal Linear Design System */
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

.dag-node-running {
  animation: node-pulse 1.5s ease-in-out infinite;
  border-color: #3b82f6 !important;
}

@keyframes node-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
  50% { box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.2); }
}

@keyframes status-pulse {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.3); opacity: 0.8; }
}

.dag-node-completed {
  border-color: #22c55e !important;
  box-shadow: 0 0 0 1px #22c55e;
}

.dag-node-failed {
  border-color: #ef4444 !important;
  box-shadow: 0 0 0 1px #ef4444;
}

.dag-node-start {
  min-width: 160px;
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

/* Edge Styles */
:deep(.vue-flow__edge-path) {
  stroke: #d4d4d4;
  stroke-width: 1.5;
}

:deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke: #3b82f6;
  stroke-width: 2;
}

:deep(.vue-flow__connection-line) {
  stroke: #3b82f6;
  stroke-width: 2;
  stroke-dasharray: 5;
}

/* Block Group Styles */
.dag-group {
  width: 100%;
  height: 100%;
  border: none;
  border-radius: 12px;
  background: rgba(0, 0, 0, 0.01);
  position: relative;
  transition: border-color 0.15s;
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
  background: transparent;
  display: flex;
  align-items: center;
  padding: 0 14px;
  gap: 8px;
  font-size: 0.75rem;
  font-weight: 500;
}

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

/* Group Handles */
.dag-group-handle {
  width: 8px !important;
  height: 8px !important;
  border-radius: 50% !important;
  border: 1.5px solid !important;
  transition: all 0.15s ease;
  z-index: 10;
  opacity: 0;
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

/* Group Entry Point */
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

/* Group Outputs */
.dag-group-outputs {
  position: absolute;
  right: 8px;
  top: 0;
  height: var(--group-height, 100%);
  pointer-events: none;
}

/* Section Labels and Dividers */
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

.dag-group-divider-h {
  position: absolute;
  left: 12px;
  right: 12px;
  height: 1px;
  background: #e5e5e5 !important;
  opacity: 1;
  z-index: 5;
}

.dag-group-divider-v {
  position: absolute;
  top: 42px;
  bottom: 12px;
  width: 1px;
  background: #e5e5e5 !important;
  opacity: 1;
  z-index: 5;
}

/* Group Nodes */
:deep(.vue-flow__node-group) {
  cursor: grab;
}

:deep(.vue-flow__node-group:active) {
  cursor: grabbing;
}

/* Resize Handle Styles */
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

:deep(.dag-group-resize-line) {
  border-color: var(--group-color, #94a3b8) !important;
  border-width: 1px !important;
  opacity: 0;
  transition: opacity 0.15s ease;
}

:deep(.vue-flow__node-group:hover .dag-group-resize-line) {
  opacity: 0.5;
}

/* Edge Delete Button */
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

/* Bottom Right Button Group */
.bottom-right-buttons {
  position: absolute;
  bottom: 70px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 10;
  transition: right 0.3s ease;
}

.bottom-right-buttons.no-transition {
  transition: none;
}

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

.action-button.copilot-toggle.active {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.1) 0%, rgba(139, 92, 246, 0.1) 100%);
  color: #6366f1;
  border-color: #6366f1;
}

/* Tooltip */
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

/* Miro-style Icon Node Design */
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
  width: 48px;
  min-width: 48px;
  overflow: visible;
}

.dag-node-icon-box {
  position: relative;
  z-index: 1;
  flex-shrink: 0;
}

.dag-node-miro.dag-node-selected {
  box-shadow: none !important;
  border-color: transparent !important;
}

.dag-node-miro.dag-node-selected .dag-node-icon-box :deep(.node-icon-wrapper) {
  border-color: #3b82f6 !important;
  box-shadow: none !important;
}

.dag-node-running .dag-node-icon-box :deep(.node-icon-wrapper) {
  animation: miro-node-pulse 1.5s ease-in-out infinite;
  border-color: #3b82f6 !important;
}

@keyframes miro-node-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.4); }
  50% { box-shadow: 0 0 0 6px rgba(59, 130, 246, 0.1); }
}

.dag-node-completed .dag-node-icon-box :deep(.node-icon-wrapper) {
  border-color: #22c55e !important;
  box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.2);
}

.dag-node-failed .dag-node-icon-box :deep(.node-icon-wrapper) {
  border-color: #ef4444 !important;
  box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.2);
}

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

.dag-handle-miro.dag-handle-target {
  left: -5px !important;
  top: var(--handle-top, 24px) !important;
  transform: translateY(-50%);
}

.dag-handle-miro.dag-handle-source {
  right: -5px !important;
  top: var(--handle-top, 24px) !important;
  transform: translateY(-50%);
}

.dag-handle-miro.dag-handle-center {
  top: 24px !important;
}

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
  0%, 100% { outline-color: #22c55e; box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.3); }
  50% { outline-color: #16a34a; box-shadow: 0 0 8px 2px rgba(34, 197, 94, 0.4); }
}

@keyframes preview-pulse-blue {
  0%, 100% { outline-color: #3b82f6; box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.3); }
  50% { outline-color: #2563eb; box-shadow: 0 0 8px 2px rgba(59, 130, 246, 0.4); }
}

@keyframes preview-pulse-red {
  0%, 100% { outline-color: #ef4444; box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.3); }
  50% { outline-color: #dc2626; box-shadow: 0 0 8px 2px rgba(239, 68, 68, 0.4); }
}
</style>
