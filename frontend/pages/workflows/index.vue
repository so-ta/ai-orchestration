<script setup lang="ts">
import type { Workflow } from '~/types/api'

const { t } = useI18n()
const { list: listWorkflows } = useWorkflows()
const toast = useToast()

// State
const workflows = ref<Workflow[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Run modal state
const showRunModal = ref(false)
const selectedWorkflow = ref<Workflow | null>(null)

// Filters
const statusFilter = ref<string>('all')
const searchQuery = ref('')

// Fetch workflows
async function fetchWorkflows() {
  loading.value = true
  error.value = null
  try {
    const response = await listWorkflows()
    workflows.value = response.data || []
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to fetch workflows'
  } finally {
    loading.value = false
  }
}

// Filtered workflows
const filteredWorkflows = computed(() => {
  let result = workflows.value

  // Filter by status
  if (statusFilter.value !== 'all') {
    result = result.filter(w => w.status === statusFilter.value)
  }

  // Filter by search query
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(w =>
      w.name.toLowerCase().includes(query) ||
      w.description?.toLowerCase().includes(query)
    )
  }

  return result
})

// Stats
const stats = computed(() => ({
  total: workflows.value.length,
  published: workflows.value.filter(w => w.status === 'published').length,
  draft: workflows.value.filter(w => w.status === 'draft').length,
}))

// Open run modal
function openRunModal(workflow: Workflow) {
  if (workflow.status !== 'published') {
    return
  }
  selectedWorkflow.value = workflow
  showRunModal.value = true
}

// Handle successful run
function handleRunSuccess(runId: string) {
  showRunModal.value = false
  selectedWorkflow.value = null
  navigateTo(`/runs/${runId}`)
}

// Close run modal
function closeRunModal() {
  showRunModal.value = false
  selectedWorkflow.value = null
}

// Format date
function formatDate(dateStr: string) {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}d ago`
  return date.toLocaleDateString()
}

function getStatusBadge(status: string) {
  return status === 'published' ? 'badge-success' : 'badge-warning'
}

// Fetch on mount
onMounted(fetchWorkflows)
</script>

<template>
  <div>
    <!-- Page Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('workflows.title') }}</h1>
        <p class="page-subtitle">{{ t('workflows.subtitle') }}</p>
      </div>
      <NuxtLink to="/workflows/new" class="btn btn-primary">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        {{ t('workflows.newWorkflow') }}
      </NuxtLink>
    </div>

    <!-- Stats Cards -->
    <div v-if="!loading && !error" class="stats-row">
      <div class="mini-stat">
        <div class="mini-stat-value">{{ stats.total }}</div>
        <div class="mini-stat-label">{{ t('workflows.stats.total') }}</div>
      </div>
      <div class="mini-stat">
        <div class="mini-stat-value text-success">{{ stats.published }}</div>
        <div class="mini-stat-label">{{ t('workflows.stats.published') }}</div>
      </div>
      <div class="mini-stat">
        <div class="mini-stat-value text-warning">{{ stats.draft }}</div>
        <div class="mini-stat-label">{{ t('workflows.stats.draft') }}</div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="loading-container">
      <div class="loading-spinner"></div>
      <p class="text-secondary mt-2">{{ t('workflows.loading') }}</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-banner">
      <div class="error-icon">!</div>
      <div>
        <div class="error-title">{{ t('workflows.loadFailed') }}</div>
        <div class="error-message">{{ error }}</div>
      </div>
      <button class="btn btn-outline btn-sm" @click="fetchWorkflows">{{ t('common.retry') }}</button>
    </div>

    <!-- Content -->
    <template v-else>
      <!-- Filters -->
      <div class="filter-bar">
        <div class="filter-group">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="text-secondary">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            class="filter-input"
            :placeholder="t('workflows.searchPlaceholder')"
          >
        </div>
        <div class="filter-group">
          <label class="filter-label">{{ t('common.status') }}:</label>
          <select v-model="statusFilter" class="filter-select">
            <option value="all">{{ t('common.filter') }}</option>
            <option value="published">{{ t('workflows.status.published') }}</option>
            <option value="draft">{{ t('workflows.status.draft') }}</option>
          </select>
        </div>
        <div class="filter-actions">
          <button class="btn btn-outline btn-sm" @click="fetchWorkflows" :disabled="loading">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="23 4 23 10 17 10"></polyline>
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
            </svg>
            {{ t('workflows.refresh') }}
          </button>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="workflows.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
            <polyline points="14 2 14 8 20 8"></polyline>
            <line x1="12" y1="18" x2="12" y2="12"></line>
            <line x1="9" y1="15" x2="15" y2="15"></line>
          </svg>
        </div>
        <h3 class="empty-title">{{ t('workflows.noWorkflowsYet') }}</h3>
        <p class="empty-subtitle">{{ t('workflows.createFirstDesc') }}</p>
        <NuxtLink to="/workflows/new" class="btn btn-primary mt-4">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
          {{ t('workflows.createWorkflow') }}
        </NuxtLink>
      </div>

      <!-- No Results State -->
      <div v-else-if="filteredWorkflows.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
        </div>
        <h3 class="empty-title">{{ t('workflows.noMatchingWorkflows') }}</h3>
        <p class="empty-subtitle">{{ t('workflows.noMatchingDesc') }}</p>
        <button class="btn btn-outline mt-4" @click="searchQuery = ''; statusFilter = 'all'">
          {{ t('workflows.clearFilters') }}
        </button>
      </div>

      <!-- Workflows Table -->
      <div v-else class="card table-card">
        <div class="table-container">
          <table class="table-enhanced">
            <thead>
              <tr>
                <th>{{ t('workflows.table.name') }}</th>
                <th>{{ t('workflows.table.status') }}</th>
                <th>{{ t('workflows.table.version') }}</th>
                <th>{{ t('workflows.table.lastUpdated') }}</th>
                <th class="text-right">{{ t('workflows.table.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="workflow in filteredWorkflows" :key="workflow.id">
                <td>
                  <div class="workflow-cell">
                    <div class="workflow-icon">
                      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                        <polyline points="14 2 14 8 20 8"></polyline>
                      </svg>
                    </div>
                    <div>
                      <NuxtLink :to="`/workflows/${workflow.id}`" class="workflow-name">
                        {{ workflow.name }}
                      </NuxtLink>
                      <div v-if="workflow.description" class="workflow-desc">
                        {{ workflow.description }}
                      </div>
                    </div>
                  </div>
                </td>
                <td>
                  <span :class="['badge', getStatusBadge(workflow.status)]">
                    <span class="status-dot" :class="`status-dot-${workflow.status === 'published' ? 'completed' : 'pending'}`"></span>
                    {{ t(`workflows.status.${workflow.status}`) }}
                  </span>
                </td>
                <td>
                  <span class="version-tag">v{{ workflow.version }}</span>
                </td>
                <td class="text-secondary text-sm">
                  {{ formatDate(workflow.updated_at) }}
                </td>
                <td>
                  <div class="action-buttons">
                    <NuxtLink :to="`/workflows/${workflow.id}`" class="btn btn-outline btn-sm">
                      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
                      </svg>
                      {{ t('common.edit') }}
                    </NuxtLink>
                    <button
                      class="btn btn-primary btn-sm"
                      :disabled="workflow.status !== 'published'"
                      @click="openRunModal(workflow)"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polygon points="5 3 19 12 5 21 5 3"></polygon>
                      </svg>
                      {{ t('workflows.run') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- Run Modal -->
    <WorkflowRunModal
      :show="showRunModal"
      :workflow-id="selectedWorkflow?.id ?? ''"
      :workflow-name="selectedWorkflow?.name ?? ''"
      @close="closeRunModal"
      @success="handleRunSuccess"
    />
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

.page-header .btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
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

.mini-stat-value.text-success { color: var(--color-success); }
.mini-stat-value.text-warning { color: var(--color-warning); }

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

.filter-input {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 0.875rem;
  width: 250px;
}

.filter-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
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
}

.filter-select:focus {
  outline: none;
  border-color: var(--color-primary);
}

.filter-actions {
  margin-left: auto;
}

.filter-actions .btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
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

.workflow-name {
  font-weight: 500;
  color: var(--color-text);
  text-decoration: none;
  transition: color 0.15s;
}

.workflow-name:hover {
  color: var(--color-primary);
}

.workflow-desc {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Badge with dot */
.badge .status-dot {
  width: 6px;
  height: 6px;
  margin-right: 0.375rem;
}

/* Version Tag */
.version-tag {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.75rem;
  background: var(--color-surface);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  color: var(--color-text-secondary);
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

/* Button Spinner */
.btn-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

/* Responsive */
@media (max-width: 1024px) {
  .filter-input {
    width: 180px;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 1rem;
  }

  .stats-row {
    flex-wrap: wrap;
  }

  .filter-bar {
    flex-wrap: wrap;
  }

  .filter-input {
    width: 100%;
  }

  .filter-actions {
    width: 100%;
    margin-left: 0;
  }

  .workflow-desc {
    display: none;
  }
}
</style>
