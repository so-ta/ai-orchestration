<script setup lang="ts">
import type { Workflow, Run } from '~/types/api'

const { t } = useI18n()
const workflows = useWorkflows()
const runsApi = useRuns()

// Stats data
const stats = ref({
  totalWorkflows: 0,
  publishedWorkflows: 0,
  activeRuns: 0,
  completedToday: 0,
  failedToday: 0,
  avgDuration: 0,
})

// Run group interface
interface RunGroup {
  workflowId: string
  workflowName: string
  triggerType: 'manual' | 'schedule' | 'webhook'
  runs: Run[]
}

const runGroups = ref<RunGroup[]>([])
const recentWorkflows = ref<Workflow[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Load dashboard data
async function loadDashboard() {
  try {
    loading.value = true
    error.value = null

    // Get workflows first
    const workflowsRes = await workflows.list()
    const allWorkflows = workflowsRes.data || []

    // Create workflow name map
    const workflowMap = new Map<string, string>()
    for (const wf of allWorkflows) {
      workflowMap.set(wf.id, wf.name)
    }

    // Get runs for each workflow
    const allRuns: Run[] = []
    for (const wf of allWorkflows) {
      try {
        const runsRes = await runsApi.list(wf.id)
        allRuns.push(...(runsRes.data || []))
      } catch {
        // Ignore errors for individual workflows
      }
    }

    // Sort runs by created_at descending
    allRuns.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())

    // Calculate stats
    const today = new Date()
    today.setHours(0, 0, 0, 0)

    const todayRuns = allRuns.filter(r => new Date(r.created_at) >= today)
    const completedRuns = allRuns.filter(r => r.status === 'completed' && r.started_at && r.completed_at)

    stats.value = {
      totalWorkflows: allWorkflows.length,
      publishedWorkflows: allWorkflows.filter(w => w.status === 'published').length,
      activeRuns: allRuns.filter(r => ['pending', 'running'].includes(r.status)).length,
      completedToday: todayRuns.filter(r => r.status === 'completed').length,
      failedToday: todayRuns.filter(r => r.status === 'failed').length,
      avgDuration: completedRuns.length > 0
        ? Math.round(completedRuns.reduce((sum, r) => {
            const start = new Date(r.started_at!).getTime()
            const end = new Date(r.completed_at!).getTime()
            return sum + (end - start)
          }, 0) / completedRuns.length)
        : 0,
    }

    // Group runs by workflow + trigger type
    const groupMap = new Map<string, RunGroup>()
    for (const run of allRuns) {
      const key = `${run.workflow_id}:${run.triggered_by}`
      if (!groupMap.has(key)) {
        groupMap.set(key, {
          workflowId: run.workflow_id,
          workflowName: workflowMap.get(run.workflow_id) || 'Unknown Workflow',
          triggerType: run.triggered_by,
          runs: [],
        })
      }
      const group = groupMap.get(key)!
      // Keep only 3 most recent runs per group
      if (group.runs.length < 3) {
        group.runs.push(run)
      }
    }

    // Convert to array and sort by most recent run in each group
    runGroups.value = Array.from(groupMap.values())
      .sort((a, b) => {
        const aTime = new Date(a.runs[0]?.created_at || 0).getTime()
        const bTime = new Date(b.runs[0]?.created_at || 0).getTime()
        return bTime - aTime
      })
      .slice(0, 10) // Limit to 10 groups

    recentWorkflows.value = allWorkflows.slice(0, 5)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load dashboard'
  } finally {
    loading.value = false
  }
}

function getStatusBadge(status: string) {
  const badges: Record<string, string> = {
    completed: 'badge-success',
    running: 'badge-info',
    pending: 'badge-warning',
    failed: 'badge-error',
    cancelled: 'badge-secondary',
    published: 'badge-success',
    draft: 'badge-warning',
  }
  return badges[status] || 'badge-info'
}

function formatDuration(ms: number) {
  if (ms === 0) return '-'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

function formatTime(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  return date.toLocaleDateString()
}

function getTriggerIcon(trigger: string) {
  switch (trigger) {
    case 'manual':
      return '👤'
    case 'webhook':
      return '🔗'
    case 'schedule':
      return '⏰'
    default:
      return '▶'
  }
}

onMounted(() => {
  loadDashboard()
})
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('dashboard.title') }}</h1>
        <p class="page-subtitle">{{ t('dashboard.welcomeDesc') }}</p>
      </div>
      <div class="flex gap-2">
        <NuxtLink to="/workflows/new" class="btn btn-primary">
          <span class="btn-icon">+</span>
          {{ t('workflows.newWorkflow') }}
        </NuxtLink>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="text-secondary mt-2">{{ t('common.loading') }}</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-banner">
      <div class="error-icon">!</div>
      <div>
        <div class="error-title">{{ t('errors.loadFailed') }}</div>
        <div class="error-message">{{ error }}</div>
      </div>
      <button class="btn btn-outline btn-sm" @click="loadDashboard">{{ t('common.retry') }}</button>
    </div>

    <template v-else>
      <!-- Stats Grid -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon stat-icon-blue">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
              <polyline points="14 2 14 8 20 8"></polyline>
              <line x1="16" y1="13" x2="8" y2="13"></line>
              <line x1="16" y1="17" x2="8" y2="17"></line>
            </svg>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ stats.totalWorkflows }}</div>
            <div class="stat-label">{{ t('dashboard.stats.totalWorkflows') }}</div>
            <div class="stat-detail">{{ stats.publishedWorkflows }} {{ t('dashboard.stats.publishedWorkflows') }}</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon stat-icon-green">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
            </svg>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ stats.activeRuns }}</div>
            <div class="stat-label">{{ t('dashboard.stats.activeRuns') }}</div>
            <div class="stat-detail">{{ t('dashboard.stats.currentlyExecuting') }}</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon stat-icon-emerald">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
              <polyline points="22 4 12 14.01 9 11.01"></polyline>
            </svg>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ stats.completedToday }}</div>
            <div class="stat-label">{{ t('dashboard.stats.completedToday') }}</div>
            <div class="stat-detail stat-detail-error" v-if="stats.failedToday > 0">{{ stats.failedToday }} {{ t('dashboard.stats.failed') }}</div>
            <div class="stat-detail" v-else>{{ t('dashboard.stats.allSuccessful') }}</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon stat-icon-purple">
            <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
          </div>
          <div class="stat-content">
            <div class="stat-value">{{ formatDuration(stats.avgDuration) }}</div>
            <div class="stat-label">{{ t('dashboard.stats.avgDuration') }}</div>
            <div class="stat-detail">{{ t('dashboard.stats.perWorkflowRun') }}</div>
          </div>
        </div>
      </div>

      <!-- Main Content Grid -->
      <div class="dashboard-grid">
        <!-- Recent Runs (Grouped) -->
        <div class="card">
          <div class="card-header">
            <h2 class="card-title">{{ t('dashboard.recentRuns') }}</h2>
            <NuxtLink to="/runs" class="btn btn-outline btn-sm">{{ t('dashboard.viewRuns') }}</NuxtLink>
          </div>
          <div v-if="runGroups.length === 0" class="empty-state">
            <div class="empty-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
              </svg>
            </div>
            <p class="empty-title">{{ t('runs.noRuns') }}</p>
            <p class="empty-subtitle">{{ t('dashboard.executeWorkflowHint') }}</p>
          </div>
          <div v-else class="run-groups">
            <div v-for="group in runGroups" :key="`${group.workflowId}-${group.triggerType}`" class="run-group">
              <div class="run-group-header">
                <NuxtLink :to="`/workflows/${group.workflowId}`" class="run-group-title">
                  {{ group.workflowName }}
                </NuxtLink>
                <span class="run-group-trigger">
                  <span class="trigger-icon">{{ getTriggerIcon(group.triggerType) }}</span>
                  {{ t(`dashboard.triggers.${group.triggerType}`) }}
                </span>
              </div>
              <div class="run-group-runs">
                <NuxtLink
                  v-for="run in group.runs"
                  :key="run.id"
                  :to="`/runs/${run.id}`"
                  class="run-item-compact"
                >
                  <span :class="['status-dot', `status-${run.status}`]"></span>
                  <span class="run-id-compact">{{ run.id.substring(0, 8) }}</span>
                  <span class="run-time-compact">{{ formatTime(run.created_at) }}</span>
                </NuxtLink>
              </div>
            </div>
          </div>
        </div>

        <!-- Quick Actions & Workflows -->
        <div class="sidebar-content">
          <!-- Quick Actions -->
          <div class="card">
            <h2 class="card-title mb-3">{{ t('dashboard.quickActions') }}</h2>
            <div class="quick-actions">
              <NuxtLink to="/workflows/new" class="quick-action">
                <div class="quick-action-icon">+</div>
                <span>{{ t('dashboard.createWorkflow') }}</span>
              </NuxtLink>
              <NuxtLink to="/workflows" class="quick-action">
                <div class="quick-action-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                  </svg>
                </div>
                <span>{{ t('dashboard.manageWorkflows') }}</span>
              </NuxtLink>
              <NuxtLink to="/runs" class="quick-action">
                <div class="quick-action-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
                  </svg>
                </div>
                <span>{{ t('dashboard.viewRuns') }}</span>
              </NuxtLink>
              <NuxtLink to="/settings" class="quick-action">
                <div class="quick-action-icon">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="3"></circle>
                    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
                  </svg>
                </div>
                <span>{{ t('nav.settings') }}</span>
              </NuxtLink>
            </div>
          </div>

          <!-- Recent Workflows -->
          <div class="card">
            <div class="card-header">
              <h2 class="card-title">{{ t('dashboard.recentWorkflows') }}</h2>
            </div>
            <div v-if="recentWorkflows.length === 0" class="empty-state-sm">
              <p class="text-secondary">{{ t('workflows.noWorkflows') }}</p>
            </div>
            <div v-else class="workflow-list">
              <NuxtLink
                v-for="wf in recentWorkflows"
                :key="wf.id"
                :to="`/workflows/${wf.id}`"
                class="workflow-item"
              >
                <div class="workflow-info">
                  <div class="workflow-name">{{ wf.name }}</div>
                  <div class="workflow-meta">
                    <span :class="['badge badge-sm', getStatusBadge(wf.status)]">{{ wf.status }}</span>
                    <span class="workflow-version">v{{ wf.version }}</span>
                  </div>
                </div>
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-secondary">
                  <polyline points="9 18 15 12 9 6"></polyline>
                </svg>
              </NuxtLink>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 2rem;
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

.btn-icon {
  font-size: 1.25rem;
  margin-right: 0.5rem;
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
  margin-bottom: 1.5rem;
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

/* Stats Grid */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.stat-card {
  background: white;
  border-radius: var(--radius);
  padding: 1.5rem;
  display: flex;
  gap: 1rem;
  box-shadow: var(--shadow);
  border: 1px solid var(--color-border);
  transition: box-shadow 0.2s, transform 0.2s;
}

.stat-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-2px);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon-blue {
  background: #dbeafe;
  color: #2563eb;
}

.stat-icon-green {
  background: #dcfce7;
  color: #16a34a;
}

.stat-icon-emerald {
  background: #d1fae5;
  color: #059669;
}

.stat-icon-purple {
  background: #ede9fe;
  color: #7c3aed;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1;
}

.stat-label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.stat-detail {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 0.5rem;
}

.stat-detail-error {
  color: var(--color-error);
}

/* Dashboard Grid */
.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 350px;
  gap: 1.5rem;
}

.sidebar-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

/* Card */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.card-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
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

.empty-state-sm {
  padding: 2rem 1rem;
  text-align: center;
}

/* Run Groups */
.run-groups {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.run-group {
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  overflow: hidden;
}

.run-group-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.run-group-title {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--color-text);
  text-decoration: none;
  transition: color 0.2s;
}

.run-group-title:hover {
  color: var(--color-primary);
}

.run-group-trigger {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  background: var(--color-bg);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.trigger-icon {
  font-size: 0.875rem;
}

.run-group-runs {
  display: flex;
  flex-direction: column;
}

.run-item-compact {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 1rem;
  text-decoration: none;
  color: var(--color-text);
  transition: background 0.2s;
  border-bottom: 1px solid var(--color-border);
}

.run-item-compact:last-child {
  border-bottom: none;
}

.run-item-compact:hover {
  background: var(--color-surface);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.status-completed {
  background: #22c55e;
}

.status-dot.status-running {
  background: #3b82f6;
  animation: pulse 1.5s infinite;
}

.status-dot.status-pending {
  background: #f59e0b;
}

.status-dot.status-failed {
  background: #ef4444;
}

.status-dot.status-cancelled {
  background: #6b7280;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.run-id-compact {
  font-family: monospace;
  font-size: 0.8125rem;
  color: var(--color-text);
}

.run-time-compact {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Quick Actions */
.quick-actions {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.quick-action {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  text-decoration: none;
  color: var(--color-text);
  font-size: 0.875rem;
  transition: all 0.2s;
}

.quick-action:hover {
  background: var(--color-surface);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.quick-action-icon {
  width: 32px;
  height: 32px;
  background: var(--color-surface);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.25rem;
  color: var(--color-text-secondary);
}

.quick-action:hover .quick-action-icon {
  background: #dbeafe;
  color: var(--color-primary);
}

/* Workflow List */
.workflow-list {
  display: flex;
  flex-direction: column;
}

.workflow-item {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--color-border);
  text-decoration: none;
  color: var(--color-text);
  transition: all 0.2s;
}

.workflow-item:last-child {
  border-bottom: none;
}

.workflow-item:hover {
  color: var(--color-primary);
}

.workflow-info {
  flex: 1;
}

.workflow-name {
  font-weight: 500;
  font-size: 0.875rem;
}

.workflow-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.25rem;
}

.workflow-version {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Button variants */
.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

.btn-ghost {
  background: transparent;
  border: none;
  padding: 0.5rem;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-radius: var(--radius);
}

.btn-ghost:hover {
  background: var(--color-surface);
  color: var(--color-primary);
}

.badge-sm {
  font-size: 0.625rem;
  padding: 0.125rem 0.375rem;
}

.badge-secondary {
  background: #e5e7eb;
  color: #6b7280;
}

/* Responsive */
@media (max-width: 1200px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .sidebar-content {
    flex-direction: row;
  }

  .sidebar-content > .card {
    flex: 1;
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .sidebar-content {
    flex-direction: column;
  }

  .quick-actions {
    grid-template-columns: 1fr;
  }
}
</style>
