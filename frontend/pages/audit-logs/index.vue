<script setup lang="ts">
import type { AuditLog, AuditAction } from '~/types/api'
import type { AuditLogFilter } from '~/composables/useAuditLogs'

const { t } = useI18n()
const auditLogsApi = useAuditLogs()
const toast = useToast()

// State
const auditLogs = ref<AuditLog[]>([])
const total = ref(0)
const loading = ref(false)
const page = ref(1)
const limit = ref(50)

// Filters
const filterResourceType = ref('')
const filterAction = ref<AuditAction | ''>('')
const filterFromDate = ref('')
const filterToDate = ref('')

// Resource types available
const resourceTypes = [
  'workflow',
  'step',
  'edge',
  'run',
  'schedule',
  'webhook',
  'credential',
  'block',
  'tenant',
]

const actions: AuditAction[] = ['create', 'update', 'delete', 'publish', 'execute', 'cancel', 'approve', 'reject']

// Fetch data
const fetchAuditLogs = async () => {
  loading.value = true
  try {
    const filter: AuditLogFilter = {
      page: page.value,
      limit: limit.value,
    }
    if (filterResourceType.value) {
      filter.resource_type = filterResourceType.value
    }
    if (filterAction.value) {
      filter.action = filterAction.value
    }
    if (filterFromDate.value) {
      filter.from_date = filterFromDate.value
    }
    if (filterToDate.value) {
      filter.to_date = filterToDate.value
    }

    const result = await auditLogsApi.list(filter)
    auditLogs.value = result.data
    total.value = result.total
  } catch (e) {
    toast.error(t('errors.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchAuditLogs()
})

// Watch filters
watch([filterResourceType, filterAction, filterFromDate, filterToDate], () => {
  page.value = 1
  fetchAuditLogs()
})

watch(page, () => {
  fetchAuditLogs()
})

// Computed
const totalPages = computed(() => Math.ceil(total.value / limit.value))

// Methods
const formatDate = (date: string): string => {
  return new Date(date).toLocaleString()
}

const getActionBadgeClass = (action: AuditAction): string => {
  switch (action) {
    case 'create':
    case 'approve':
      return 'badge-success'
    case 'update':
    case 'publish':
      return 'badge-info'
    case 'delete':
    case 'reject':
      return 'badge-danger'
    case 'execute':
      return 'badge-warning'
    case 'cancel':
      return 'badge-secondary'
    default:
      return 'badge-secondary'
  }
}

const formatChanges = (changes: object | undefined): string => {
  if (!changes || Object.keys(changes).length === 0) {
    return t('auditLogs.details.noChanges')
  }
  return JSON.stringify(changes, null, 2)
}

const clearFilters = () => {
  filterResourceType.value = ''
  filterAction.value = ''
  filterFromDate.value = ''
  filterToDate.value = ''
}

const refresh = () => {
  fetchAuditLogs()
}
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ $t('auditLogs.title') }}</h1>
        <p class="text-secondary">{{ $t('auditLogs.subtitle') }}</p>
      </div>
      <button class="btn btn-secondary" @click="refresh">
        {{ $t('auditLogs.refresh') }}
      </button>
    </div>

    <!-- Filters -->
    <div class="filters-bar">
      <select v-model="filterResourceType" class="filter-select">
        <option value="">{{ $t('auditLogs.allResources') }}</option>
        <option v-for="type in resourceTypes" :key="type" :value="type">
          {{ $t(`auditLogs.resourceType.${type}`) }}
        </option>
      </select>
      <select v-model="filterAction" class="filter-select">
        <option value="">{{ $t('auditLogs.allActions') }}</option>
        <option v-for="action in actions" :key="action" :value="action">
          {{ $t(`auditLogs.action.${action}`) }}
        </option>
      </select>
      <div class="date-filter">
        <label>{{ $t('auditLogs.from') }}:</label>
        <input v-model="filterFromDate" type="date" class="filter-input" />
      </div>
      <div class="date-filter">
        <label>{{ $t('auditLogs.to') }}:</label>
        <input v-model="filterToDate" type="date" class="filter-input" />
      </div>
      <button v-if="filterResourceType || filterAction || filterFromDate || filterToDate" class="btn btn-secondary btn-sm" @click="clearFilters">
        {{ $t('workflows.clearFilters') }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      {{ $t('common.loading') }}
    </div>

    <!-- Empty State -->
    <div v-else-if="auditLogs.length === 0" class="empty-state">
      <p>{{ $t('auditLogs.noLogs') }}</p>
      <p class="text-secondary">{{ $t('auditLogs.noLogsDesc') }}</p>
    </div>

    <!-- Audit Logs Table -->
    <div v-else class="table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th>{{ $t('auditLogs.table.timestamp') }}</th>
            <th>{{ $t('auditLogs.table.action') }}</th>
            <th>{{ $t('auditLogs.table.resource') }}</th>
            <th>{{ $t('auditLogs.table.user') }}</th>
            <th>{{ $t('auditLogs.table.ipAddress') }}</th>
            <th>{{ $t('auditLogs.table.changes') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in auditLogs" :key="log.id">
            <td>{{ formatDate(log.created_at) }}</td>
            <td>
              <span :class="['action-badge', getActionBadgeClass(log.action)]">
                {{ $t(`auditLogs.action.${log.action}`) }}
              </span>
            </td>
            <td>
              <div class="resource-info">
                <span class="resource-type">{{ $t(`auditLogs.resourceType.${log.resource_type}`) }}</span>
                <code class="resource-id">{{ log.resource_id.substring(0, 8) }}...</code>
              </div>
            </td>
            <td>{{ log.user_id || '-' }}</td>
            <td>{{ log.ip_address || '-' }}</td>
            <td>
              <details v-if="log.changes && Object.keys(log.changes).length > 0" class="changes-details">
                <summary>{{ $t('auditLogs.details.title') }}</summary>
                <pre class="changes-content">{{ formatChanges(log.changes) }}</pre>
              </details>
              <span v-else class="no-changes">{{ $t('auditLogs.details.noChanges') }}</span>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="pagination">
        <button class="btn btn-sm btn-secondary" :disabled="page === 1" @click="page--">
          &lt;
        </button>
        <span class="page-info">{{ page }} / {{ totalPages }}</span>
        <button class="btn btn-sm btn-secondary" :disabled="page >= totalPages" @click="page++">
          &gt;
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0;
}

.filters-bar {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
  align-items: center;
}

.filter-select {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  min-width: 140px;
}

.date-filter {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.date-filter label {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.filter-input {
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 3rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.data-table th,
.data-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.data-table th {
  background: var(--color-background);
  font-weight: 600;
  font-size: 0.875rem;
}

.action-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 500;
}

.badge-success {
  background: #dcfce7;
  color: #166534;
}

.badge-info {
  background: #dbeafe;
  color: #1e40af;
}

.badge-warning {
  background: #fef3c7;
  color: #92400e;
}

.badge-danger {
  background: #fee2e2;
  color: #991b1b;
}

.badge-secondary {
  background: #f3f4f6;
  color: #374151;
}

.resource-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.resource-type {
  font-weight: 500;
}

.resource-id {
  background: var(--color-background);
  padding: 0.125rem 0.25rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
}

.changes-details {
  cursor: pointer;
}

.changes-details summary {
  color: var(--color-primary);
  font-size: 0.875rem;
}

.changes-content {
  background: var(--color-background);
  padding: 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  max-width: 300px;
  overflow-x: auto;
  margin: 0.5rem 0 0;
}

.no-changes {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  margin-top: 1rem;
  padding: 1rem;
}

.page-info {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 0.875rem;
}

.btn-secondary {
  background: var(--color-background);
  border: 1px solid var(--color-border);
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
