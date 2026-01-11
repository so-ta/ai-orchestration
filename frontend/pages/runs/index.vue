<script setup lang="ts">
import type { Run, Workflow } from '~/types/api'

const { t } = useI18n()
const { list: listWorkflows } = useWorkflows()
const { list: listRuns } = useRuns()

// State
const runs = ref<Run[]>([])
const workflows = ref<Map<string, Workflow>>(new Map())
const loading = ref(true)
const error = ref<string | null>(null)

// Filters
const statusFilter = ref<string>('all')
const modeFilter = ref<string>('all')
const workflowFilter = ref<string>('all')

// Fetch all runs
async function fetchData() {
  loading.value = true
  error.value = null
  try {
    // Get all workflows
    const wfResponse = await listWorkflows()
    wfResponse.data?.forEach(wf => {
      workflows.value.set(wf.id, wf)
    })

    // Get runs for each workflow
    const allRuns: Run[] = []
    for (const wf of wfResponse.data || []) {
      try {
        const runsResponse = await listRuns(wf.id)
        allRuns.push(...(runsResponse.data || []))
      } catch (e) {
        // Ignore errors for individual workflows
      }
    }

    // Sort by created_at descending
    allRuns.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    runs.value = allRuns.slice(0, 100) // Show last 100 runs
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to fetch runs'
  } finally {
    loading.value = false
  }
}

// Filtered runs
const filteredRuns = computed(() => {
  let result = runs.value

  // Filter by status
  if (statusFilter.value !== 'all') {
    result = result.filter(r => r.status === statusFilter.value)
  }

  // Filter by mode
  if (modeFilter.value !== 'all') {
    result = result.filter(r => r.mode === modeFilter.value)
  }

  // Filter by workflow
  if (workflowFilter.value !== 'all') {
    result = result.filter(r => r.workflow_id === workflowFilter.value)
  }

  return result
})

// Stats
const stats = computed(() => ({
  total: runs.value.length,
  running: runs.value.filter(r => ['pending', 'running'].includes(r.status)).length,
  completed: runs.value.filter(r => r.status === 'completed').length,
  failed: runs.value.filter(r => r.status === 'failed').length,
}))

// Available workflows for filter
const workflowOptions = computed(() => {
  return Array.from(workflows.value.values()).map(wf => ({
    id: wf.id,
    name: wf.name,
  }))
})

function getWorkflowName(workflowId: string) {
  return workflows.value.get(workflowId)?.name || 'Unknown Workflow'
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

// Auto-refresh for active runs
let refreshInterval: ReturnType<typeof setInterval> | null = null

function startAutoRefresh() {
  if (!refreshInterval) {
    refreshInterval = setInterval(() => {
      if (runs.value.some(r => ['pending', 'running'].includes(r.status))) {
        fetchData()
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
  fetchData()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('runs.title') }}</h1>
        <p class="page-subtitle">{{ t('runs.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <div v-if="stats.running > 0" class="running-indicator">
          <span class="running-dot"></span>
          {{ t('runs.activeCount', { count: stats.running }) }}
        </div>
        <button class="btn btn-outline" @click="fetchData" :disabled="loading">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="23 4 23 10 17 10"></polyline>
            <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
          </svg>
          {{ t('workflows.refresh') }}
        </button>
      </div>
    </div>

    <!-- Stats Cards -->
    <div v-if="!loading && !error" class="stats-row">
      <div class="mini-stat">
        <div class="mini-stat-value">{{ stats.total }}</div>
        <div class="mini-stat-label">{{ t('runs.stats.total') }}</div>
      </div>
      <div class="mini-stat">
        <div class="mini-stat-value text-primary">{{ stats.running }}</div>
        <div class="mini-stat-label">{{ t('runs.stats.active') }}</div>
      </div>
      <div class="mini-stat">
        <div class="mini-stat-value text-success">{{ stats.completed }}</div>
        <div class="mini-stat-label">{{ t('runs.stats.completed') }}</div>
      </div>
      <div class="mini-stat">
        <div class="mini-stat-value text-error">{{ stats.failed }}</div>
        <div class="mini-stat-label">{{ t('runs.stats.failed') }}</div>
      </div>
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
      <button class="btn btn-outline btn-sm" @click="fetchData">{{ t('common.retry') }}</button>
    </div>

    <!-- Content -->
    <template v-else>
      <!-- Filters -->
      <div class="filter-bar">
        <div class="filter-group">
          <label class="filter-label">{{ t('runs.filters.workflow') }}:</label>
          <select v-model="workflowFilter" class="filter-select">
            <option value="all">{{ t('runs.filters.allWorkflows') }}</option>
            <option v-for="wf in workflowOptions" :key="wf.id" :value="wf.id">
              {{ wf.name }}
            </option>
          </select>
        </div>
        <div class="filter-group">
          <label class="filter-label">{{ t('runs.filters.status') }}:</label>
          <select v-model="statusFilter" class="filter-select">
            <option value="all">{{ t('common.filter') }}</option>
            <option value="pending">{{ t('runs.status.pending') }}</option>
            <option value="running">{{ t('runs.status.running') }}</option>
            <option value="completed">{{ t('runs.status.completed') }}</option>
            <option value="failed">{{ t('runs.status.failed') }}</option>
            <option value="cancelled">{{ t('runs.status.cancelled') }}</option>
          </select>
        </div>
        <div class="filter-group">
          <label class="filter-label">{{ t('runs.filters.mode') }}:</label>
          <select v-model="modeFilter" class="filter-select">
            <option value="all">{{ t('common.filter') }}</option>
            <option value="test">{{ t('runs.mode.test') }}</option>
            <option value="production">{{ t('runs.mode.production') }}</option>
          </select>
        </div>
        <div class="filter-spacer"></div>
        <div class="filter-results">
          {{ t('runs.resultsCount', { filtered: filteredRuns.length, total: runs.length }) }}
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="runs.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
          </svg>
        </div>
        <h3 class="empty-title">{{ t('runs.noRunsYet') }}</h3>
        <p class="empty-subtitle">{{ t('runs.noRunsDesc') }}</p>
        <NuxtLink to="/workflows" class="btn btn-primary mt-4">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
          </svg>
          {{ t('runs.viewWorkflows') }}
        </NuxtLink>
      </div>

      <!-- No Results State -->
      <div v-else-if="filteredRuns.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
        </div>
        <h3 class="empty-title">{{ t('runs.noMatchingRuns') }}</h3>
        <p class="empty-subtitle">{{ t('runs.noMatchingDesc') }}</p>
        <button class="btn btn-outline mt-4" @click="statusFilter = 'all'; modeFilter = 'all'; workflowFilter = 'all'">
          {{ t('workflows.clearFilters') }}
        </button>
      </div>

      <!-- Runs Table -->
      <div v-else class="card table-card">
        <div class="table-container">
          <table class="table-enhanced">
            <thead>
              <tr>
                <th>{{ t('runs.table.workflow') }}</th>
                <th>{{ t('runs.table.status') }}</th>
                <th>{{ t('runs.table.mode') }}</th>
                <th>{{ t('runs.table.duration') }}</th>
                <th>{{ t('runs.table.triggeredBy') }}</th>
                <th>{{ t('runs.table.created') }}</th>
                <th class="text-right">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="run in filteredRuns" :key="run.id">
                <td>
                  <div class="workflow-cell">
                    <div class="workflow-icon">
                      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
                      </svg>
                    </div>
                    <div>
                      <NuxtLink :to="`/workflows/${run.workflow_id}`" class="workflow-link">
                        {{ getWorkflowName(run.workflow_id) }}
                      </NuxtLink>
                      <div class="run-id">
                        {{ run.id.substring(0, 8) }}...
                      </div>
                    </div>
                  </div>
                </td>
                <td>
                  <span :class="['badge', getStatusBadge(run.status)]">
                    <span class="status-dot" :class="getStatusDot(run.status)"></span>
                    {{ t(`runs.status.${run.status}`) }}
                  </span>
                </td>
                <td>
                  <span :class="['mode-tag', `mode-${run.mode}`]">
                    {{ t(`runs.mode.${run.mode}`) }}
                  </span>
                </td>
                <td>
                  <span class="duration">{{ calculateDuration(run) }}</span>
                </td>
                <td class="text-secondary text-sm">
                  {{ run.triggered_by }}
                </td>
                <td class="text-secondary text-sm">
                  {{ formatDate(run.created_at) }}
                </td>
                <td>
                  <div class="action-buttons">
                    <NuxtLink :to="`/runs/${run.id}`" class="btn btn-outline btn-sm">
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
  </div>
</template>

<style scoped>
/* Page Header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
}

.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--color-text);
  margin: 0;
}

.page-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header-actions .btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.running-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: var(--color-primary);
  font-weight: 500;
}

.running-dot {
  width: 8px;
  height: 8px;
  background: var(--color-primary);
  border-radius: 50%;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

/* Stats Row */
.stats-row {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.mini-stat {
  background: white;
  padding: 1rem 1.5rem;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  text-align: center;
  min-width: 100px;
}

.mini-stat-value {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--color-text);
}

.mini-stat-value.text-primary { color: var(--color-primary); }
.mini-stat-value.text-success { color: var(--color-success); }
.mini-stat-value.text-error { color: var(--color-error); }

.mini-stat-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-top: 0.25rem;
}

/* Loading */
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

/* Error */
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

/* Filter Bar */
.filter-bar {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem;
  background: white;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  margin-bottom: 1rem;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.filter-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.filter-select {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 0.875rem;
  background: white;
  min-width: 120px;
}

.filter-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

.filter-spacer {
  flex: 1;
}

.filter-results {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  background: white;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  text-align: center;
}

.empty-icon {
  color: var(--color-text-secondary);
  opacity: 0.5;
  margin-bottom: 1rem;
}

.empty-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.empty-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin-top: 0.5rem;
}

.empty-state .btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Table Card */
.table-card {
  padding: 0;
  overflow: hidden;
}

.table-container {
  overflow-x: auto;
}

/* Enhanced Table */
.table-enhanced {
  width: 100%;
  border-collapse: collapse;
}

.table-enhanced th {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.875rem 1rem;
  text-align: left;
  border-bottom: 2px solid var(--color-border);
  background: var(--color-surface);
}

.table-enhanced td {
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}

.table-enhanced tbody tr {
  transition: background 0.15s;
}

.table-enhanced tbody tr:hover {
  background: var(--color-surface);
}

.table-enhanced tbody tr:last-child td {
  border-bottom: none;
}

/* Workflow Cell */
.workflow-cell {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.workflow-icon {
  width: 36px;
  height: 36px;
  background: #dbeafe;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  flex-shrink: 0;
}

.workflow-link {
  font-weight: 500;
  color: var(--color-text);
  text-decoration: none;
  transition: color 0.15s;
}

.workflow-link:hover {
  color: var(--color-primary);
}

.run-id {
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

/* Badge with dot */
.badge .status-dot {
  width: 6px;
  height: 6px;
  margin-right: 0.375rem;
}

/* Mode Tag */
.mode-tag {
  font-size: 0.75rem;
  padding: 0.25rem 0.5rem;
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

/* Duration */
.duration {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.875rem;
  color: var(--color-text);
}

/* Action Buttons */
.action-buttons {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.action-buttons .btn {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}

/* Button Variants */
.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

/* Responsive */
@media (max-width: 1024px) {
  .filter-bar {
    flex-wrap: wrap;
  }

  .filter-spacer {
    display: none;
  }

  .filter-results {
    width: 100%;
    text-align: center;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 1rem;
  }

  .header-actions {
    width: 100%;
    justify-content: space-between;
  }

  .stats-row {
    flex-wrap: wrap;
  }

  .mini-stat {
    flex: 1;
    min-width: 80px;
  }

  .filter-select {
    min-width: 100px;
  }
}
</style>
