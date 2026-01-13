<script setup lang="ts">
import type { Run, StepRun } from '~/types/api'

const props = defineProps<{
  workflowId: string
}>()

const { t } = useI18n()
const { list: listRuns, get: getRun } = useRuns()

const runs = ref<Run[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Flattened step runs from all runs
interface StepRunWithRunInfo extends StepRun {
  run_id: string
  workflow_version: number
  run_mode: string
  run_status: string
}

const allStepRuns = computed<StepRunWithRunInfo[]>(() => {
  const stepRuns: StepRunWithRunInfo[] = []
  for (const run of runs.value) {
    if (run.step_runs) {
      for (const stepRun of run.step_runs) {
        stepRuns.push({
          ...stepRun,
          run_id: run.id,
          workflow_version: run.workflow_version,
          run_mode: run.mode,
          run_status: run.status,
        })
      }
    }
  }
  // Sort by created_at descending (newest first)
  return stepRuns.sort((a, b) => {
    const dateA = new Date(a.completed_at || a.started_at || a.created_at).getTime()
    const dateB = new Date(b.completed_at || b.started_at || b.created_at).getTime()
    return dateB - dateA
  })
})

async function fetchRuns() {
  loading.value = true
  error.value = null
  try {
    const response = await listRuns(props.workflowId, { limit: 50 })
    const runList = response.data || []

    // Fetch detailed run data with step_runs for each run
    const detailedRuns: Run[] = []
    for (const run of runList) {
      try {
        const detailedResponse = await getRun(run.id)
        detailedRuns.push(detailedResponse.data)
      } catch {
        // If detail fetch fails, use the basic run info
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

function getStatusBadge(status: string) {
  const badges: Record<string, string> = {
    pending: 'badge-warning',
    running: 'badge-info',
    completed: 'badge-success',
    failed: 'badge-error',
    cancelled: 'badge-warning',
  }
  return badges[status] || 'badge-info'
}

function getStatusDot(status: string) {
  const dots: Record<string, string> = {
    pending: 'status-dot-pending',
    running: 'status-dot-running',
    completed: 'status-dot-completed',
    failed: 'status-dot-failed',
    cancelled: 'status-dot-cancelled',
  }
  return dots[status] || ''
}

function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  return date.toLocaleString()
}

function calculateDuration(run: Run): string {
  if (!run.started_at) return '-'
  const start = new Date(run.started_at).getTime()
  const end = run.completed_at ? new Date(run.completed_at).getTime() : Date.now()
  const ms = end - start

  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

function calculateStepDuration(stepRun: StepRunWithRunInfo): string {
  if (!stepRun.started_at) return '-'
  const start = new Date(stepRun.started_at).getTime()
  const end = stepRun.completed_at ? new Date(stepRun.completed_at).getTime() : Date.now()
  const ms = end - start

  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
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

onMounted(() => {
  fetchRuns()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})

// Watch for workflowId changes
watch(() => props.workflowId, () => {
  fetchRuns()
})
</script>

<template>
  <div class="run-history">
    <!-- Header -->
    <div class="history-header">
      <h3 class="history-title">{{ t('workflows.runHistory.title') }}</h3>
      <button class="btn btn-outline btn-sm" @click="fetchRuns" :disabled="loading">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="23 4 23 10 17 10"></polyline>
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
        </svg>
        {{ t('workflows.refresh') }}
      </button>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="text-secondary mt-2">{{ t('runs.loading') }}</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-banner">
      <div class="error-icon">!</div>
      <div>
        <div class="error-title">{{ t('runs.loadFailed') }}</div>
        <div class="error-message">{{ error }}</div>
      </div>
      <button class="btn btn-outline btn-sm" @click="fetchRuns">{{ t('common.retry') }}</button>
    </div>

    <!-- Empty State -->
    <div v-else-if="allStepRuns.length === 0" class="empty-state">
      <div class="empty-icon">
        <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
        </svg>
      </div>
      <h4 class="empty-title">{{ t('workflows.runHistory.noRuns') }}</h4>
      <p class="empty-subtitle">{{ t('workflows.runHistory.noRunsDesc') }}</p>
    </div>

    <!-- Step Runs Table -->
    <div v-else class="table-container">
      <table class="history-table">
        <thead>
          <tr>
            <th>{{ t('runs.table.step') }}</th>
            <th>{{ t('runs.table.status') }}</th>
            <th>{{ t('workflows.runHistory.version') }}</th>
            <th>{{ t('runs.table.duration') }}</th>
            <th>{{ t('runs.table.created') }}</th>
            <th class="text-right">{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="stepRun in allStepRuns" :key="stepRun.id">
            <td>
              <span class="step-name">{{ stepRun.step_name }}</span>
            </td>
            <td>
              <span :class="['badge', getStatusBadge(stepRun.status)]">
                <span class="status-dot" :class="getStatusDot(stepRun.status)"></span>
                {{ t(`runs.status.${stepRun.status}`) }}
              </span>
            </td>
            <td>
              <span class="version-badge">v{{ stepRun.workflow_version }}</span>
            </td>
            <td>
              <span class="duration">{{ calculateStepDuration(stepRun) }}</span>
            </td>
            <td class="text-secondary text-sm">
              {{ formatDate(stepRun.created_at) }}
            </td>
            <td>
              <div class="action-buttons">
                <NuxtLink :to="`/runs/${stepRun.run_id}`" class="btn btn-outline btn-sm">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                    <circle cx="12" cy="12" r="3"></circle>
                  </svg>
                  {{ t('runs.view') }}
                </NuxtLink>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.run-history {
  padding: 1.5rem;
  background: white;
  height: 100%;
  overflow-y: auto;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.history-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.history-header .btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

/* Loading */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 2rem;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Error */
.error-banner {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: var(--radius);
}

.error-icon {
  width: 28px;
  height: 28px;
  background: var(--color-error);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 0.875rem;
  flex-shrink: 0;
}

.error-title {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--color-error);
}

.error-message {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 2rem;
  text-align: center;
}

.empty-icon {
  color: var(--color-text-secondary);
  opacity: 0.4;
  margin-bottom: 1rem;
}

.empty-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.empty-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin-top: 0.5rem;
}

/* Table */
.table-container {
  overflow-x: auto;
}

.history-table {
  width: 100%;
  border-collapse: collapse;
}

.history-table th {
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.75rem 0.75rem;
  text-align: left;
  border-bottom: 2px solid var(--color-border);
  background: var(--color-surface);
}

.history-table td {
  padding: 0.75rem;
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}

.history-table tbody tr {
  transition: background 0.15s;
}

.history-table tbody tr:hover {
  background: var(--color-surface);
}

.history-table tbody tr:last-child td {
  border-bottom: none;
}

/* Badge with dot */
.badge .status-dot {
  width: 6px;
  height: 6px;
  margin-right: 0.375rem;
}

/* Version Badge */
.version-badge {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.25rem 0.5rem;
  background: #e0e7ff;
  color: #4f46e5;
  border-radius: 4px;
  font-family: 'SF Mono', Monaco, monospace;
}

/* Mode Tag */
.mode-tag {
  font-size: 0.7rem;
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
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

/* Step Name */
.step-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}

/* Duration */
.duration {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.8rem;
  color: var(--color-text);
}

/* Action Buttons */
.action-buttons {
  display: flex;
  justify-content: flex-end;
}

.action-buttons .btn {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

/* Button */
.btn-sm {
  padding: 0.375rem 0.625rem;
  font-size: 0.7rem;
}

.text-right {
  text-align: right;
}

.text-secondary {
  color: var(--color-text-secondary);
}

.text-sm {
  font-size: 0.75rem;
}
</style>
