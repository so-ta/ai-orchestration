<script setup lang="ts">
/**
 * BottomPanel.vue
 * DAGエディタ下部のリサイズ可能な実行履歴パネル
 */
import type { Run } from '~/types/api'
import { ChevronDown, ChevronRight, RefreshCw } from 'lucide-vue-next'

const props = defineProps<{
  workflowId: string
  selectedRunId?: string | null
}>()

const emit = defineEmits<{
  (e: 'run:select', run: Run): void
  (e: 'height-change', height: number): void
}>()

const { t } = useI18n()
const runsApi = useRuns()

// Get global bottom panel state (singleton)
const {
  bottomPanelCollapsed,
  bottomPanelHeight,
  bottomPanelResizing,
  setBottomPanelCollapsed,
  setBottomPanelHeight,
  setBottomPanelResizing,
} = useEditorState()

// Use global state aliases for template readability
const isCollapsed = bottomPanelCollapsed
const panelHeight = bottomPanelHeight
const filter = ref<'all' | 'test' | 'production'>('all')

// Run data
const runs = ref<Run[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Resize state (local)
const startY = ref(0)
const startHeight = ref(0)

// Filtered runs
const filteredRuns = computed(() => {
  if (filter.value === 'all') return runs.value
  if (filter.value === 'test') return runs.value.filter(r => r.triggered_by === 'test')
  return runs.value.filter(r => r.triggered_by !== 'test')
})

// Fetch runs
async function fetchRuns() {
  if (!props.workflowId) return

  loading.value = true
  error.value = null

  try {
    const response = await runsApi.list(props.workflowId, { limit: 50 })
    const runList = response.data || []

    // Fetch detailed run data with step_runs
    const detailedRuns: Run[] = []
    for (const run of runList) {
      try {
        const detailedResponse = await runsApi.get(run.id)
        detailedRuns.push(detailedResponse.data)
      } catch {
        detailedRuns.push(run)
      }
    }
    runs.value = detailedRuns
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to fetch runs'
  } finally {
    loading.value = false
  }
}

// Toggle collapse
function toggleCollapse() {
  setBottomPanelCollapsed(!isCollapsed.value)
  emit('height-change', isCollapsed.value ? 40 : panelHeight.value)
}

// Resize handlers
function startResize(e: MouseEvent) {
  if (isCollapsed.value) return
  setBottomPanelResizing(true)
  startY.value = e.clientY
  startHeight.value = panelHeight.value
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
}

function handleResize(e: MouseEvent) {
  if (!bottomPanelResizing.value) return
  const delta = startY.value - e.clientY
  const newHeight = Math.max(100, Math.min(400, startHeight.value + delta))
  setBottomPanelHeight(newHeight)
  emit('height-change', newHeight)
}

function stopResize() {
  setBottomPanelResizing(false)
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
}

// Run selection
function handleRunSelect(run: Run) {
  emit('run:select', run)
}

// Auto-refresh for active runs
let refreshInterval: ReturnType<typeof setInterval> | null = null

function startAutoRefresh() {
  if (!refreshInterval) {
    refreshInterval = setInterval(() => {
      if (runs.value.some(r => ['pending', 'running'].includes(r.status))) {
        fetchRuns()
      }
    }, 5000)
  }
}

function stopAutoRefresh() {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

// Lifecycle
onMounted(() => {
  fetchRuns()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

watch(() => props.workflowId, () => {
  fetchRuns()
})
</script>

<template>
  <div
    class="bottom-panel"
    :class="{ collapsed: isCollapsed, resizing: bottomPanelResizing }"
    :style="{ height: isCollapsed ? '40px' : `${panelHeight}px` }"
  >
    <!-- Resize Handle -->
    <div class="resize-handle" @mousedown="startResize">
      <div class="resize-line" />
    </div>

    <!-- Header -->
    <div class="bottom-panel-header">
      <div class="header-left">
        <button class="collapse-button" @click="toggleCollapse">
          <ChevronDown v-if="!isCollapsed" :size="16" />
          <ChevronRight v-else :size="16" />
        </button>
        <span class="panel-title">{{ t('execution.runHistory') }}</span>
        <span class="run-count">({{ filteredRuns.length }})</span>
      </div>

      <div class="header-right">
        <!-- Filter Tabs -->
        <div class="filter-tabs">
          <button :class="{ active: filter === 'all' }" @click="filter = 'all'">
            {{ t('execution.filter.all') }}
          </button>
          <button :class="{ active: filter === 'test' }" @click="filter = 'test'">
            {{ t('execution.filter.test') }}
          </button>
          <button :class="{ active: filter === 'production' }" @click="filter = 'production'">
            {{ t('execution.filter.production') }}
          </button>
        </div>

        <button class="refresh-button" :disabled="loading" @click="fetchRuns">
          <RefreshCw :size="14" :class="{ spinning: loading }" />
        </button>
      </div>
    </div>

    <!-- Content -->
    <div v-if="!isCollapsed" class="bottom-panel-content">
      <RunHistoryList
        :runs="filteredRuns"
        :selected-run-id="selectedRunId"
        :loading="loading"
        :error="error"
        @run:select="handleRunSelect"
        @retry="fetchRuns"
      />
    </div>
  </div>
</template>

<style scoped>
.bottom-panel {
  position: relative;
  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(12px);
  border-top: 1px solid rgba(0, 0, 0, 0.08);
  display: flex;
  flex-direction: column;
  transition: height 0.2s ease;
  flex-shrink: 0;
}

.bottom-panel.collapsed {
  overflow: hidden;
}

.bottom-panel.resizing {
  transition: none;
  user-select: none;
}

/* Resize Handle */
.resize-handle {
  position: absolute;
  top: -4px;
  left: 0;
  right: 0;
  height: 8px;
  cursor: ns-resize;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: center;
}

.resize-handle:hover .resize-line,
.bottom-panel.resizing .resize-line {
  background: #3b82f6;
  height: 3px;
}

.resize-line {
  width: 40px;
  height: 2px;
  background: rgba(0, 0, 0, 0.1);
  border-radius: 2px;
  transition: all 0.15s;
}

/* Header */
.bottom-panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 1rem;
  background: rgba(248, 250, 252, 0.8);
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.collapse-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  color: #64748b;
  transition: all 0.15s;
}

.collapse-button:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #1e293b;
}

.panel-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #1e293b;
}

.run-count {
  font-size: 0.75rem;
  color: #64748b;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

/* Filter Tabs */
.filter-tabs {
  display: flex;
  gap: 2px;
  background: rgba(0, 0, 0, 0.03);
  padding: 2px;
  border-radius: 6px;
}

.filter-tabs button {
  padding: 0.25rem 0.625rem;
  font-size: 0.6875rem;
  font-weight: 500;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  color: #64748b;
  transition: all 0.15s;
}

.filter-tabs button:hover {
  color: #1e293b;
}

.filter-tabs button.active {
  background: white;
  color: #1e293b;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

/* Refresh Button */
.refresh-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  background: white;
  border-radius: 6px;
  cursor: pointer;
  color: #64748b;
  transition: all 0.15s;
}

.refresh-button:hover {
  background: #f8fafc;
  color: #1e293b;
  border-color: rgba(0, 0, 0, 0.12);
}

.refresh-button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.refresh-button .spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Content */
.bottom-panel-content {
  flex: 1;
  overflow: hidden;
  min-height: 0;
}
</style>
