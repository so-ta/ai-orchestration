<script setup lang="ts">
import type { Run, WorkflowDefinition, StepRun } from '~/types/api'

const route = useRoute()
const runId = route.params.id as string

const runsApi = useRuns()
const toast = useToast()

const run = ref<Run | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

// UI state
const activeTab = ref<'overview' | 'workflow' | 'steps' | 'input' | 'output'>('overview')

// Step details modal state
const selectedStepRun = ref<StepRun | null>(null)
const showStepDetails = ref(false)

// Computed workflow definition from API response
const workflowDefinition = computed<WorkflowDefinition | null>(() => {
  return run.value?.workflow_definition || null
})
const expandedSteps = ref<Set<string>>(new Set())

// Auto-refresh for running status
let refreshInterval: ReturnType<typeof setInterval> | null = null

async function loadRun() {
  try {
    error.value = null
    const response = await runsApi.get(runId)
    run.value = response.data

    // Stop auto-refresh if run is complete
    if (run.value && !['pending', 'running'].includes(run.value.status)) {
      stopAutoRefresh()
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load run'
    stopAutoRefresh()
  } finally {
    loading.value = false
  }
}

function startAutoRefresh() {
  if (!refreshInterval) {
    refreshInterval = setInterval(loadRun, 2000)
  }
}

function stopAutoRefresh() {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

async function handleCancel() {
  if (!run.value) return

  if (!confirm('Are you sure you want to cancel this run?')) return

  try {
    await runsApi.cancel(runId)
    await loadRun()
  } catch (e) {
    toast.error('Failed to cancel run', e instanceof Error ? e.message : undefined)
  }
}

async function handleRerun() {
  if (!run.value) return

  try {
    const response = await runsApi.create(run.value.workflow_id, {
      mode: run.value.mode,
      input: run.value.input || {},
    })
    navigateTo(`/runs/${response.data.id}`)
  } catch (e) {
    toast.error('Failed to rerun workflow', e instanceof Error ? e.message : undefined)
  }
}

function toggleStepExpanded(stepId: string) {
  if (expandedSteps.value.has(stepId)) {
    expandedSteps.value.delete(stepId)
  } else {
    expandedSteps.value.add(stepId)
  }
}

function getStatusBadge(status: string) {
  const badges: Record<string, string> = {
    pending: 'badge-warning',
    running: 'badge-info',
    completed: 'badge-success',
    failed: 'badge-error',
    cancelled: 'badge-secondary',
  }
  return badges[status] || 'badge-info'
}

function getStatusIcon(status: string) {
  switch (status) {
    case 'completed':
      return '✓'
    case 'failed':
      return '✕'
    case 'running':
      return '●'
    case 'pending':
      return '○'
    default:
      return '○'
  }
}

function formatDuration(ms?: number) {
  if (!ms) return '-'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`
  return `${Math.floor(ms / 60000)}m ${((ms % 60000) / 1000).toFixed(0)}s`
}

function formatDateTime(dateStr?: string) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString()
}

function formatTimestamp(dateStr?: string) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleTimeString()
}

function formatJson(obj: any): string {
  if (!obj) return '-'
  return JSON.stringify(obj, null, 2)
}

function calculateDuration(start?: string, end?: string): number {
  if (!start || !end) return 0
  return new Date(end).getTime() - new Date(start).getTime()
}

const totalDuration = computed(() => {
  if (!run.value?.started_at) return 0
  const endTime = run.value.completed_at || new Date().toISOString()
  return calculateDuration(run.value.started_at, endTime)
})

const completedSteps = computed(() => {
  return run.value?.step_runs?.filter(s => s.status === 'completed').length || 0
})

const totalSteps = computed(() => {
  return run.value?.step_runs?.length || 0
})

// Copy to clipboard
function copyToClipboard(text: string) {
  if (typeof navigator !== 'undefined' && navigator.clipboard) {
    navigator.clipboard.writeText(text)
  }
}

// Handle step details from DAG click
function handleStepShowDetails(stepRun: StepRun) {
  selectedStepRun.value = stepRun
  showStepDetails.value = true
}

// Close step details modal
function closeStepDetails() {
  showStepDetails.value = false
  selectedStepRun.value = null
}

onMounted(() => {
  loadRun()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="text-secondary mt-2">Loading run details...</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-banner">
      <div class="error-icon">!</div>
      <div>
        <div class="error-title">Failed to load run</div>
        <div class="error-message">{{ error }}</div>
      </div>
      <button class="btn btn-outline btn-sm" @click="loadRun">Retry</button>
    </div>

    <!-- Run Details -->
    <div v-else-if="run">
      <!-- Header -->
      <div class="page-header">
        <div class="page-header-info">
          <div class="breadcrumb">
            <NuxtLink to="/runs" class="breadcrumb-link">Runs</NuxtLink>
            <span class="breadcrumb-separator">/</span>
            <span class="breadcrumb-current">{{ run.id.substring(0, 8) }}...</span>
          </div>
          <h1 class="page-title">Run Details</h1>
          <div class="run-id-display">
            <code>{{ run.id }}</code>
            <button class="copy-btn" @click="copyToClipboard(run.id)" title="Copy Run ID">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
              </svg>
            </button>
          </div>
        </div>
        <div class="page-header-actions">
          <div class="status-display">
            <span :class="['status-badge', `status-${run.status}`]">
              <span class="status-dot"></span>
              {{ run.status }}
            </span>
            <span class="mode-badge">{{ run.mode }}</span>
          </div>
        </div>
      </div>

      <!-- Quick Stats -->
      <div class="stats-bar">
        <div class="stat-item">
          <div class="stat-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ formatDuration(totalDuration) }}</div>
            <div class="stat-label">Duration</div>
          </div>
        </div>

        <div class="stat-item">
          <div class="stat-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 11 12 14 22 4"></polyline>
              <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"></path>
            </svg>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ completedSteps }} / {{ totalSteps }}</div>
            <div class="stat-label">Steps Completed</div>
          </div>
        </div>

        <div class="stat-item">
          <div class="stat-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
              <polyline points="14 2 14 8 20 8"></polyline>
            </svg>
          </div>
          <div class="stat-info">
            <div class="stat-value">v{{ run.workflow_version }}</div>
            <div class="stat-label">Workflow Version</div>
          </div>
        </div>

        <div class="stat-item">
          <div class="stat-icon">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
            </svg>
          </div>
          <div class="stat-info">
            <div class="stat-value">{{ run.triggered_by }}</div>
            <div class="stat-label">Triggered By</div>
          </div>
        </div>
      </div>

      <!-- Actions Bar -->
      <div class="actions-bar">
        <div class="actions-left">
          <NuxtLink :to="`/workflows/${run.workflow_id}`" class="btn btn-outline">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
            </svg>
            View Workflow
          </NuxtLink>
          <button
            v-if="!['pending', 'running'].includes(run.status)"
            class="btn btn-outline"
            @click="handleRerun"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="23 4 23 10 17 10"></polyline>
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
            </svg>
            Rerun
          </button>
          <button
            v-if="['pending', 'running'].includes(run.status)"
            class="btn btn-danger-outline"
            @click="handleCancel"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="15" y1="9" x2="9" y2="15"></line>
              <line x1="9" y1="9" x2="15" y2="15"></line>
            </svg>
            Cancel
          </button>
        </div>
        <div class="actions-right">
          <div v-if="['pending', 'running'].includes(run.status)" class="refresh-indicator">
            <div class="refresh-dot"></div>
            Auto-refreshing
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="tabs-container">
        <div class="tabs">
          <button
            :class="['tab', { active: activeTab === 'overview' }]"
            @click="activeTab = 'overview'"
          >
            Overview
          </button>
          <button
            :class="['tab', { active: activeTab === 'workflow' }]"
            @click="activeTab = 'workflow'"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
              <line x1="9" y1="3" x2="9" y2="21"></line>
            </svg>
            Workflow
          </button>
          <button
            :class="['tab', { active: activeTab === 'steps' }]"
            @click="activeTab = 'steps'"
          >
            Step Executions
            <span v-if="run.step_runs?.length" class="tab-badge">{{ run.step_runs.length }}</span>
          </button>
          <button
            :class="['tab', { active: activeTab === 'input' }]"
            @click="activeTab = 'input'"
          >
            Input
          </button>
          <button
            :class="['tab', { active: activeTab === 'output' }]"
            @click="activeTab = 'output'"
          >
            Output
          </button>
        </div>

        <!-- Tab Content -->
        <div class="tab-content">
          <!-- Overview Tab -->
          <div v-if="activeTab === 'overview'" class="overview-grid">
            <div class="info-card">
              <h3 class="info-card-title">Execution Timeline</h3>
              <div class="timeline">
                <div class="timeline-item">
                  <div class="timeline-marker completed"></div>
                  <div class="timeline-content">
                    <div class="timeline-label">Created</div>
                    <div class="timeline-value">{{ formatDateTime(run.created_at) }}</div>
                  </div>
                </div>
                <div class="timeline-item" :class="{ pending: !run.started_at }">
                  <div :class="['timeline-marker', run.started_at ? 'completed' : 'pending']"></div>
                  <div class="timeline-content">
                    <div class="timeline-label">Started</div>
                    <div class="timeline-value">{{ formatDateTime(run.started_at) }}</div>
                  </div>
                </div>
                <div class="timeline-item" :class="{ pending: !run.completed_at }">
                  <div :class="['timeline-marker', run.completed_at ? (run.status === 'failed' ? 'failed' : 'completed') : 'pending']"></div>
                  <div class="timeline-content">
                    <div class="timeline-label">{{ run.status === 'failed' ? 'Failed' : 'Completed' }}</div>
                    <div class="timeline-value">{{ formatDateTime(run.completed_at) }}</div>
                  </div>
                </div>
              </div>
            </div>

            <div class="info-card">
              <h3 class="info-card-title">Run Details</h3>
              <div class="details-list">
                <div class="detail-item">
                  <span class="detail-label">Workflow ID</span>
                  <code class="detail-value">{{ run.workflow_id }}</code>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Tenant ID</span>
                  <code class="detail-value">{{ run.tenant_id }}</code>
                </div>
                <div class="detail-item">
                  <span class="detail-label">Mode</span>
                  <span :class="['mode-tag', `mode-${run.mode}`]">{{ run.mode }}</span>
                </div>
              </div>
            </div>

            <div v-if="run.error" class="info-card error-card">
              <h3 class="info-card-title">
                <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="8" x2="12" y2="12"></line>
                  <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
                Error
              </h3>
              <pre class="error-message">{{ run.error }}</pre>
            </div>
          </div>

          <!-- Workflow Tab -->
          <div v-if="activeTab === 'workflow'" class="workflow-preview">
            <div v-if="workflowDefinition" class="workflow-preview-container">
              <div class="workflow-preview-header">
                <h3 class="workflow-preview-title">
                  {{ workflowDefinition.name }}
                  <span class="version-badge">v{{ run.workflow_version }}</span>
                </h3>
                <p v-if="workflowDefinition.description" class="workflow-preview-description">
                  {{ workflowDefinition.description }}
                </p>
              </div>
              <div class="workflow-dag-container">
                <DagEditor
                  :steps="workflowDefinition.steps || []"
                  :edges="workflowDefinition.edges || []"
                  :readonly="true"
                  :selected-step-id="selectedStepRun?.step_id || null"
                  :step-runs="run?.step_runs || []"
                  @step:showDetails="handleStepShowDetails"
                />
              </div>
              <p class="workflow-hint">Click on a step to view its input/output details</p>
            </div>
            <div v-else class="workflow-preview-fallback">
              <div class="empty-state">
                <div class="empty-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                    <polyline points="14 2 14 8 20 8"></polyline>
                  </svg>
                </div>
                <p class="empty-title">Workflow definition not available</p>
                <p class="empty-subtitle">This run was created before workflow versioning was enabled</p>
                <NuxtLink :to="`/workflows/${run.workflow_id}`" class="btn btn-outline">
                  View Current Workflow
                </NuxtLink>
              </div>
            </div>
          </div>

          <!-- Steps Tab -->
          <div v-if="activeTab === 'steps'">
            <div v-if="!run.step_runs?.length" class="empty-state">
              <div class="empty-icon">
                <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                  <polyline points="9 11 12 14 22 4"></polyline>
                  <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"></path>
                </svg>
              </div>
              <p class="empty-title">No step executions yet</p>
              <p class="empty-subtitle">Step executions will appear here once the run starts</p>
            </div>

            <div v-else class="steps-list">
              <div
                v-for="(stepRun, index) in run.step_runs"
                :key="stepRun.id"
                class="step-card"
              >
                <div class="step-header" @click="toggleStepExpanded(stepRun.id)">
                  <div class="step-info">
                    <span class="step-number">{{ index + 1 }}</span>
                    <span :class="['step-status-icon', `status-${stepRun.status}`]">
                      {{ getStatusIcon(stepRun.status) }}
                    </span>
                    <span class="step-name">{{ stepRun.step_name }}</span>
                    <span :class="['badge badge-sm', getStatusBadge(stepRun.status)]">
                      {{ stepRun.status }}
                    </span>
                  </div>
                  <div class="step-meta">
                    <span v-if="stepRun.attempt > 1" class="attempt-badge">
                      Attempt {{ stepRun.attempt }}
                    </span>
                    <span class="step-duration">{{ formatDuration(stepRun.duration_ms) }}</span>
                    <svg
                      :class="['expand-icon', { expanded: expandedSteps.has(stepRun.id) }]"
                      xmlns="http://www.w3.org/2000/svg"
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <polyline points="6 9 12 15 18 9"></polyline>
                    </svg>
                  </div>
                </div>

                <div v-if="expandedSteps.has(stepRun.id)" class="step-details">
                  <div class="step-detail-grid">
                    <div class="step-detail-item">
                      <span class="detail-label">Step ID</span>
                      <code class="detail-value">{{ stepRun.step_id }}</code>
                    </div>
                    <div class="step-detail-item">
                      <span class="detail-label">Started</span>
                      <span class="detail-value">{{ formatTimestamp(stepRun.started_at) }}</span>
                    </div>
                    <div class="step-detail-item">
                      <span class="detail-label">Completed</span>
                      <span class="detail-value">{{ formatTimestamp(stepRun.completed_at) }}</span>
                    </div>
                  </div>

                  <div v-if="stepRun.error" class="step-error">
                    <div class="step-error-title">Error</div>
                    <pre class="step-error-message">{{ stepRun.error }}</pre>
                  </div>

                  <div v-if="stepRun.input" class="step-data">
                    <div class="step-data-title">Input</div>
                    <pre class="step-data-content">{{ formatJson(stepRun.input) }}</pre>
                  </div>

                  <div v-if="stepRun.output" class="step-data">
                    <div class="step-data-title">Output</div>
                    <pre class="step-data-content">{{ formatJson(stepRun.output) }}</pre>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Input Tab -->
          <div v-if="activeTab === 'input'" class="data-view">
            <div class="data-header">
              <h3 class="data-title">Input Data</h3>
              <button
                v-if="run.input"
                class="btn btn-outline btn-sm"
                @click="copyToClipboard(formatJson(run.input))"
              >
                Copy
              </button>
            </div>
            <pre v-if="run.input && Object.keys(run.input).length > 0" class="data-content">{{ formatJson(run.input) }}</pre>
            <div v-else class="empty-data">No input data</div>
          </div>

          <!-- Output Tab -->
          <div v-if="activeTab === 'output'" class="data-view">
            <div class="data-header">
              <h3 class="data-title">Output Data</h3>
              <button
                v-if="run.output"
                class="btn btn-outline btn-sm"
                @click="copyToClipboard(formatJson(run.output))"
              >
                Copy
              </button>
            </div>
            <pre v-if="run.output && Object.keys(run.output).length > 0" class="data-content">{{ formatJson(run.output) }}</pre>
            <div v-else class="empty-data">No output data</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Step Details Modal -->
    <Teleport to="body">
      <div v-if="showStepDetails && selectedStepRun" class="step-modal-overlay" @click.self="closeStepDetails">
        <div class="step-modal">
          <div class="step-modal-header">
            <div class="step-modal-title-area">
              <span :class="['step-modal-status-icon', `status-${selectedStepRun.status}`]">
                {{ getStatusIcon(selectedStepRun.status) }}
              </span>
              <h3 class="step-modal-title">{{ selectedStepRun.step_name }}</h3>
              <span :class="['badge', getStatusBadge(selectedStepRun.status)]">
                {{ selectedStepRun.status }}
              </span>
            </div>
            <button class="step-modal-close" @click="closeStepDetails">
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"></line>
                <line x1="6" y1="6" x2="18" y2="18"></line>
              </svg>
            </button>
          </div>

          <div class="step-modal-body">
            <div class="step-modal-meta">
              <div class="step-meta-item">
                <span class="meta-label">Step ID</span>
                <code class="meta-value">{{ selectedStepRun.step_id.substring(0, 8) }}...</code>
              </div>
              <div class="step-meta-item">
                <span class="meta-label">Duration</span>
                <span class="meta-value">{{ formatDuration(selectedStepRun.duration_ms) }}</span>
              </div>
              <div v-if="selectedStepRun.attempt > 1" class="step-meta-item">
                <span class="meta-label">Attempt</span>
                <span class="meta-value attempt">{{ selectedStepRun.attempt }}</span>
              </div>
              <div class="step-meta-item">
                <span class="meta-label">Started</span>
                <span class="meta-value">{{ formatTimestamp(selectedStepRun.started_at) }}</span>
              </div>
              <div class="step-meta-item">
                <span class="meta-label">Completed</span>
                <span class="meta-value">{{ formatTimestamp(selectedStepRun.completed_at) }}</span>
              </div>
            </div>

            <div v-if="selectedStepRun.error" class="step-modal-error">
              <div class="error-header">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="8" x2="12" y2="12"></line>
                  <line x1="12" y1="16" x2="12.01" y2="16"></line>
                </svg>
                Error
              </div>
              <pre class="error-content">{{ selectedStepRun.error }}</pre>
            </div>

            <div class="step-modal-data-section">
              <div class="data-section">
                <div class="data-section-header">
                  <h4 class="data-section-title">Input</h4>
                  <button
                    v-if="selectedStepRun.input"
                    class="btn btn-outline btn-xs"
                    @click="copyToClipboard(formatJson(selectedStepRun.input))"
                  >
                    Copy
                  </button>
                </div>
                <pre v-if="selectedStepRun.input && Object.keys(selectedStepRun.input).length > 0" class="data-section-content">{{ formatJson(selectedStepRun.input) }}</pre>
                <div v-else class="data-section-empty">No input data</div>
              </div>

              <div class="data-section">
                <div class="data-section-header">
                  <h4 class="data-section-title">Output</h4>
                  <button
                    v-if="selectedStepRun.output"
                    class="btn btn-outline btn-xs"
                    @click="copyToClipboard(formatJson(selectedStepRun.output))"
                  >
                    Copy
                  </button>
                </div>
                <pre v-if="selectedStepRun.output && Object.keys(selectedStepRun.output).length > 0" class="data-section-content">{{ formatJson(selectedStepRun.output) }}</pre>
                <div v-else class="data-section-empty">No output data</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
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
  margin-bottom: 1.5rem;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  margin-bottom: 0.5rem;
}

.breadcrumb-link {
  color: var(--color-primary);
  text-decoration: none;
}

.breadcrumb-link:hover {
  text-decoration: underline;
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
  margin: 0 0 0.5rem 0;
}

.run-id-display {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.run-id-display code {
  font-size: 0.75rem;
  background: var(--color-surface);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  color: var(--color-text-secondary);
}

.copy-btn {
  background: none;
  border: none;
  padding: 0.25rem;
  cursor: pointer;
  color: var(--color-text-secondary);
  border-radius: 4px;
}

.copy-btn:hover {
  background: var(--color-surface);
  color: var(--color-primary);
}

.status-display {
  display: flex;
  gap: 0.5rem;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
  text-transform: capitalize;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.status-pending {
  background: #fef3c7;
  color: #d97706;
}

.status-running {
  background: #dbeafe;
  color: #2563eb;
}

.status-running .status-dot {
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.status-completed {
  background: #dcfce7;
  color: #16a34a;
}

.status-failed {
  background: #fee2e2;
  color: #dc2626;
}

.status-cancelled {
  background: #f3f4f6;
  color: #6b7280;
}

.mode-badge {
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
  background: #f3f4f6;
  color: #374151;
  text-transform: capitalize;
}

/* Stats Bar */
.stats-bar {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
  background: white;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
}

.stat-icon {
  color: var(--color-text-secondary);
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--color-text);
}

.stat-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Actions Bar */
.actions-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  padding: 1rem;
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

.refresh-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.refresh-dot {
  width: 8px;
  height: 8px;
  background: #10b981;
  border-radius: 50%;
  animation: pulse 1.5s ease-in-out infinite;
}

/* Tabs */
.tabs-container {
  background: white;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  overflow: hidden;
}

.tabs {
  display: flex;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}

.tab {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem 1.5rem;
  background: none;
  border: none;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition: all 0.2s;
}

.tab:hover {
  color: var(--color-text);
  background: rgba(0, 0, 0, 0.02);
}

.tab.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
  background: white;
}

.tab-badge {
  background: var(--color-primary);
  color: white;
  font-size: 0.625rem;
  padding: 0.125rem 0.375rem;
  border-radius: 10px;
}

.tab-content {
  padding: 1.5rem;
}

/* Overview Grid */
.overview-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
}

.info-card {
  padding: 1.5rem;
  background: var(--color-surface);
  border-radius: var(--radius);
}

.info-card.error-card {
  grid-column: 1 / -1;
  background: #fef2f2;
  border: 1px solid #fecaca;
}

.info-card-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 1rem 0;
}

.error-card .info-card-title {
  color: var(--color-error);
}

/* Timeline */
.timeline {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.timeline-item {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.timeline-marker {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  margin-top: 4px;
  flex-shrink: 0;
}

.timeline-marker.completed {
  background: #10b981;
}

.timeline-marker.pending {
  background: #d1d5db;
}

.timeline-marker.failed {
  background: var(--color-error);
}

.timeline-content {
  flex: 1;
}

.timeline-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.timeline-value {
  font-size: 0.875rem;
  color: var(--color-text);
  margin-top: 0.125rem;
}

/* Details List */
.details-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.detail-label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.detail-value {
  font-size: 0.75rem;
  background: white;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mode-tag {
  font-size: 0.75rem;
  padding: 0.25rem 0.75rem;
  border-radius: 20px;
  font-weight: 500;
}

.mode-test {
  background: #fef3c7;
  color: #d97706;
}

.mode-production {
  background: #dcfce7;
  color: #16a34a;
}

/* Steps List */
.steps-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.step-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  overflow: hidden;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.25rem;
  cursor: pointer;
  background: white;
  transition: background 0.2s;
}

.step-header:hover {
  background: var(--color-surface);
}

.step-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.step-number {
  width: 24px;
  height: 24px;
  background: var(--color-surface);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.step-status-icon {
  font-size: 0.875rem;
}

.step-status-icon.status-completed {
  color: #10b981;
}

.step-status-icon.status-failed {
  color: var(--color-error);
}

.step-status-icon.status-running {
  color: var(--color-primary);
  animation: pulse 1.5s ease-in-out infinite;
}

.step-status-icon.status-pending {
  color: var(--color-text-secondary);
}

.step-name {
  font-weight: 500;
}

.step-meta {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.attempt-badge {
  font-size: 0.75rem;
  padding: 0.25rem 0.5rem;
  background: #fef3c7;
  color: #d97706;
  border-radius: 4px;
}

.step-duration {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  font-family: monospace;
}

.expand-icon {
  color: var(--color-text-secondary);
  transition: transform 0.2s;
}

.expand-icon.expanded {
  transform: rotate(180deg);
}

.step-details {
  padding: 1rem 1.25rem;
  background: var(--color-surface);
  border-top: 1px solid var(--color-border);
}

.step-detail-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1rem;
  margin-bottom: 1rem;
}

.step-detail-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.step-error {
  margin-bottom: 1rem;
  padding: 1rem;
  background: #fef2f2;
  border-radius: var(--radius);
}

.step-error-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-error);
  margin-bottom: 0.5rem;
}

.step-error-message {
  font-size: 0.75rem;
  color: var(--color-error);
  margin: 0;
  white-space: pre-wrap;
  font-family: monospace;
}

.step-data {
  margin-bottom: 1rem;
}

.step-data:last-child {
  margin-bottom: 0;
}

.step-data-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 0.5rem;
}

.step-data-content {
  font-size: 0.75rem;
  background: white;
  padding: 1rem;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  margin: 0;
  overflow-x: auto;
  font-family: 'SF Mono', Monaco, monospace;
}

/* Data View */
.data-view {
  min-height: 300px;
}

.data-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.data-title {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
}

.data-content {
  background: var(--color-surface);
  padding: 1.5rem;
  border-radius: var(--radius);
  margin: 0;
  overflow-x: auto;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.875rem;
  line-height: 1.6;
}

.empty-data {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  background: var(--color-surface);
  border-radius: var(--radius);
  color: var(--color-text-secondary);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  text-align: center;
}

.empty-icon {
  color: var(--color-text-secondary);
  opacity: 0.5;
  margin-bottom: 1rem;
}

.empty-title {
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.empty-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

/* Button Variants */
.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

.btn-danger-outline {
  background: white;
  border: 1px solid #fecaca;
  color: var(--color-error);
}

.btn-danger-outline:hover {
  background: #fef2f2;
  border-color: var(--color-error);
}

.badge-sm {
  font-size: 0.625rem;
  padding: 0.125rem 0.375rem;
}

.badge-secondary {
  background: #e5e7eb;
  color: #6b7280;
}

/* Workflow Preview */
.workflow-preview {
  min-height: 400px;
}

.workflow-preview-container {
  height: 100%;
}

.workflow-preview-header {
  margin-bottom: 1rem;
}

.workflow-preview-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.workflow-preview-description {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin-top: 0.5rem;
}

.version-badge {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.25rem 0.5rem;
  background: #e0e7ff;
  color: #4f46e5;
  border-radius: 4px;
  font-family: 'SF Mono', Monaco, monospace;
}

.workflow-dag-container {
  height: 500px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--color-surface);
}

.workflow-preview-fallback {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  background: var(--color-surface);
  border-radius: var(--radius);
}

.workflow-hint {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  text-align: center;
  margin-top: 0.75rem;
  margin-bottom: 0;
}

/* Step Details Modal */
.step-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
  animation: fadeIn 0.15s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.step-modal {
  background: white;
  border-radius: var(--radius-lg, 12px);
  max-width: 700px;
  width: 100%;
  max-height: 85vh;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  animation: slideUp 0.2s ease-out;
}

@keyframes slideUp {
  from { transform: translateY(10px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.step-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}

.step-modal-title-area {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.step-modal-status-icon {
  font-size: 1rem;
}

.step-modal-status-icon.status-completed {
  color: #10b981;
}

.step-modal-status-icon.status-failed {
  color: var(--color-error);
}

.step-modal-status-icon.status-running {
  color: var(--color-primary);
}

.step-modal-status-icon.status-pending {
  color: var(--color-text-secondary);
}

.step-modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.step-modal-close {
  background: none;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  color: var(--color-text-secondary);
  border-radius: 6px;
  transition: all 0.15s;
}

.step-modal-close:hover {
  background: rgba(0, 0, 0, 0.05);
  color: var(--color-text);
}

.step-modal-body {
  padding: 1.5rem;
  overflow-y: auto;
  max-height: calc(85vh - 60px);
}

.step-modal-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  padding: 1rem;
  background: var(--color-surface);
  border-radius: var(--radius);
  margin-bottom: 1.5rem;
}

.step-meta-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 100px;
}

.meta-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.meta-value {
  font-size: 0.875rem;
  font-weight: 500;
}

.meta-value.attempt {
  color: #d97706;
}

.step-modal-error {
  margin-bottom: 1.5rem;
  padding: 1rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: var(--radius);
}

.step-modal-error .error-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-error);
  margin-bottom: 0.75rem;
}

.step-modal-error .error-content {
  font-size: 0.75rem;
  color: #b91c1c;
  margin: 0;
  white-space: pre-wrap;
  font-family: 'SF Mono', Monaco, monospace;
}

.step-modal-data-section {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.data-section {
  background: var(--color-surface);
  border-radius: var(--radius);
  overflow: hidden;
}

.data-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: rgba(0, 0, 0, 0.02);
  border-bottom: 1px solid var(--color-border);
}

.data-section-title {
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.data-section-content {
  margin: 0;
  padding: 1rem;
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  overflow-x: auto;
  max-height: 300px;
  overflow-y: auto;
  background: white;
  line-height: 1.5;
}

.data-section-empty {
  padding: 2rem;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.btn-xs {
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
}

/* Responsive */
@media (max-width: 1024px) {
  .stats-bar {
    grid-template-columns: repeat(2, 1fr);
  }

  .overview-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .stats-bar {
    grid-template-columns: 1fr;
  }

  .step-detail-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    flex-direction: column;
    gap: 1rem;
  }

  .workflow-dag-container {
    height: 350px;
  }
}
</style>
