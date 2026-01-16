<script setup lang="ts">
import type { Tenant, TenantStatus, TenantPlan, CreateTenantRequest, TenantOverviewStats } from '~/types/api'

const { t } = useI18n()

definePageMeta({
  layout: 'default',
  middleware: ['admin'],
})

const {
  tenants,
  loading,
  pagination,
  fetchTenants,
  createTenant,
  updateTenant,
  deleteTenant,
  suspendTenant,
  activateTenant,
  getOverviewStats,
  getStatusColor,
  getPlanColor,
} = useTenants()

// Overview stats
const overviewStats = ref<TenantOverviewStats | null>(null)

// Filters
const filters = reactive({
  status: '' as TenantStatus | '',
  plan: '' as TenantPlan | '',
  search: '',
})

// Modal state
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const showSuspendModal = ref(false)
const selectedTenant = ref<Tenant | null>(null)

// Form state
const formData = reactive({
  name: '',
  slug: '',
  plan: 'free' as TenantPlan,
  owner_email: '',
  owner_name: '',
  billing_email: '',
})

// Suspend form
const suspendReason = ref('')

// Message state
const message = ref<{ type: 'success' | 'error'; text: string } | null>(null)

function showMessage(type: 'success' | 'error', text: string) {
  message.value = { type, text }
  setTimeout(() => {
    message.value = null
  }, 3000)
}

// Status options
const statusOptions = [
  { value: '', label: t('admin.tenants.allStatuses') },
  { value: 'active', label: t('admin.tenants.status.active') },
  { value: 'suspended', label: t('admin.tenants.status.suspended') },
  { value: 'pending', label: t('admin.tenants.status.pending') },
  { value: 'inactive', label: t('admin.tenants.status.inactive') },
]

// Plan options
const planOptions = [
  { value: '', label: t('admin.tenants.allPlans') },
  { value: 'free', label: t('admin.tenants.plan.free') },
  { value: 'starter', label: t('admin.tenants.plan.starter') },
  { value: 'professional', label: t('admin.tenants.plan.professional') },
  { value: 'enterprise', label: t('admin.tenants.plan.enterprise') },
]

// Fetch tenants on mount
onMounted(async () => {
  await loadTenants()
  try {
    overviewStats.value = await getOverviewStats()
  } catch {
    // Stats are optional, don't show error
  }
})

// Load tenants with filters
async function loadTenants() {
  try {
    await fetchTenants({
      status: filters.status || undefined,
      plan: filters.plan || undefined,
      search: filters.search || undefined,
      page: pagination.value.page,
      limit: pagination.value.limit,
    })
  } catch {
    showMessage('error', t('admin.tenants.messages.fetchFailed'))
  }
}

// Watch for filter changes
watch(filters, () => {
  pagination.value.page = 1
  loadTenants()
}, { deep: true })

// Reset form
function resetForm() {
  formData.name = ''
  formData.slug = ''
  formData.plan = 'free'
  formData.owner_email = ''
  formData.owner_name = ''
  formData.billing_email = ''
}

// Generate slug from name
function generateSlug() {
  if (formData.name && !selectedTenant.value) {
    formData.slug = formData.name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '')
  }
}

// Open create modal
function openCreateModal() {
  resetForm()
  selectedTenant.value = null
  showCreateModal.value = true
}

// Open edit modal
function openEditModal(tenant: Tenant) {
  selectedTenant.value = tenant
  formData.name = tenant.name
  formData.slug = tenant.slug
  formData.plan = tenant.plan
  formData.owner_email = tenant.owner_email || ''
  formData.owner_name = tenant.owner_name || ''
  formData.billing_email = tenant.billing_email || ''
  showEditModal.value = true
}

// Submit create form
async function submitCreate() {
  try {
    const request: CreateTenantRequest = {
      name: formData.name,
      slug: formData.slug,
      plan: formData.plan,
      owner_email: formData.owner_email || undefined,
      owner_name: formData.owner_name || undefined,
      billing_email: formData.billing_email || undefined,
    }
    await createTenant(request)
    showMessage('success', t('admin.tenants.messages.created'))
    showCreateModal.value = false
    resetForm()
    // Refresh overview stats
    overviewStats.value = await getOverviewStats()
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : t('errors.generic')
    showMessage('error', t('admin.tenants.messages.createFailed') + ': ' + errorMessage)
  }
}

// Submit edit form
async function submitEdit() {
  if (!selectedTenant.value) return

  try {
    await updateTenant(selectedTenant.value.id, {
      name: formData.name,
      slug: formData.slug,
      plan: formData.plan,
      owner_email: formData.owner_email || undefined,
      owner_name: formData.owner_name || undefined,
      billing_email: formData.billing_email || undefined,
    })
    showMessage('success', t('admin.tenants.messages.updated'))
    showEditModal.value = false
    selectedTenant.value = null
    resetForm()
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : t('errors.generic')
    showMessage('error', t('admin.tenants.messages.updateFailed') + ': ' + errorMessage)
  }
}

// Open delete confirmation
function openDeleteModal(tenant: Tenant) {
  selectedTenant.value = tenant
  showDeleteModal.value = true
}

// Confirm delete
async function confirmDelete() {
  if (!selectedTenant.value) return

  try {
    await deleteTenant(selectedTenant.value.id)
    showMessage('success', t('admin.tenants.messages.deleted'))
    showDeleteModal.value = false
    selectedTenant.value = null
    // Refresh overview stats
    overviewStats.value = await getOverviewStats()
  } catch (err) {
    showMessage('error', t('admin.tenants.messages.deleteFailed'))
  }
}

// Open suspend modal
function openSuspendModal(tenant: Tenant) {
  selectedTenant.value = tenant
  suspendReason.value = ''
  showSuspendModal.value = true
}

// Confirm suspend
async function confirmSuspend() {
  if (!selectedTenant.value || !suspendReason.value) return

  try {
    await suspendTenant(selectedTenant.value.id, suspendReason.value)
    showMessage('success', t('admin.tenants.messages.suspended'))
    showSuspendModal.value = false
    selectedTenant.value = null
    suspendReason.value = ''
    // Refresh overview stats
    overviewStats.value = await getOverviewStats()
  } catch (err) {
    showMessage('error', t('admin.tenants.messages.suspendFailed'))
  }
}

// Activate tenant
async function handleActivate(tenant: Tenant) {
  try {
    await activateTenant(tenant.id)
    showMessage('success', t('admin.tenants.messages.activated'))
    // Refresh overview stats
    overviewStats.value = await getOverviewStats()
  } catch (err) {
    showMessage('error', t('admin.tenants.messages.activateFailed'))
  }
}

// Pagination
function changePage(page: number) {
  pagination.value.page = page
  loadTenants()
}

// Format date
function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleDateString()
}

// Format cost
function formatCost(cost: number): string {
  return `$${cost.toFixed(2)}`
}

// Get status badge class
function getStatusBadgeClass(status: TenantStatus): string {
  const colorMap: Record<TenantStatus, string> = {
    active: 'badge-success',
    suspended: 'badge-error',
    pending: 'badge-warning',
    inactive: 'badge-secondary',
  }
  return colorMap[status] || 'badge-secondary'
}

// Get plan badge class
function getPlanBadgeClass(plan: TenantPlan): string {
  const colorMap: Record<TenantPlan, string> = {
    enterprise: 'badge-primary',
    professional: 'badge-info',
    starter: 'badge-success',
    free: 'badge-secondary',
  }
  return colorMap[plan] || 'badge-secondary'
}
</script>

<template>
  <div>
    <!-- Breadcrumb -->
    <div class="breadcrumb mb-4">
      <NuxtLink to="/admin" class="breadcrumb-link">
        {{ $t('admin.title') }}
      </NuxtLink>
      <span class="breadcrumb-separator">/</span>
      <span>{{ $t('admin.tenants.title') }}</span>
    </div>

    <!-- Header -->
    <div class="page-header mb-6">
      <div>
        <h1 class="page-title">{{ $t('admin.tenants.title') }}</h1>
        <p class="page-description text-secondary">
          {{ $t('admin.tenants.subtitle') }}
        </p>
      </div>
      <button class="btn btn-primary" @click="openCreateModal">
        + {{ $t('admin.tenants.createTenant') }}
      </button>
    </div>

    <!-- Message toast -->
    <div v-if="message" :class="['toast', message.type === 'success' ? 'toast-success' : 'toast-error']">
      {{ message.text }}
    </div>

    <!-- Overview Stats -->
    <div v-if="overviewStats" class="stats-grid mb-6">
      <div class="stat-card">
        <div class="stat-label">{{ $t('admin.tenants.stats.totalTenants') }}</div>
        <div class="stat-value">{{ overviewStats.total_tenants }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ $t('admin.tenants.stats.activeCount') }}</div>
        <div class="stat-value text-success">{{ overviewStats.status_counts?.active || 0 }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ $t('admin.tenants.stats.totalWorkflows') }}</div>
        <div class="stat-value">{{ overviewStats.total_projects }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ $t('admin.tenants.stats.runsThisMonth') }}</div>
        <div class="stat-value">{{ overviewStats.total_runs_this_month }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-label">{{ $t('admin.tenants.stats.costThisMonth') }}</div>
        <div class="stat-value">{{ formatCost(overviewStats.cost_this_month) }}</div>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters mb-4">
      <select v-model="filters.status" class="filter-select">
        <option v-for="opt in statusOptions" :key="opt.value" :value="opt.value">
          {{ opt.label }}
        </option>
      </select>
      <select v-model="filters.plan" class="filter-select">
        <option v-for="opt in planOptions" :key="opt.value" :value="opt.value">
          {{ opt.label }}
        </option>
      </select>
      <input
        v-model="filters.search"
        type="text"
        :placeholder="$t('admin.tenants.searchPlaceholder')"
        class="filter-input"
      >
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="loading-state">
      {{ $t('common.loading') }}
    </div>

    <!-- Tenants Table -->
    <div v-else class="table-container">
      <table class="table">
        <thead>
          <tr>
            <th>{{ $t('admin.tenants.table.name') }}</th>
            <th>{{ $t('admin.tenants.table.status') }}</th>
            <th>{{ $t('admin.tenants.table.plan') }}</th>
            <th>{{ $t('admin.tenants.table.owner') }}</th>
            <th>{{ $t('admin.tenants.table.workflows') }}</th>
            <th>{{ $t('admin.tenants.table.runs') }}</th>
            <th>{{ $t('admin.tenants.table.cost') }}</th>
            <th>{{ $t('admin.tenants.table.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="tenant in tenants" :key="tenant.id">
            <td>
              <div class="tenant-name">{{ tenant.name }}</div>
              <div class="tenant-slug text-secondary">{{ tenant.slug }}</div>
            </td>
            <td>
              <span :class="['badge', getStatusBadgeClass(tenant.status)]">
                {{ $t(`admin.tenants.status.${tenant.status}`) }}
              </span>
            </td>
            <td>
              <span :class="['badge', getPlanBadgeClass(tenant.plan)]">
                {{ $t(`admin.tenants.plan.${tenant.plan}`) }}
              </span>
            </td>
            <td>
              <div v-if="tenant.owner_email">{{ tenant.owner_email }}</div>
              <div v-else class="text-secondary">-</div>
            </td>
            <td>{{ tenant.stats?.project_count || 0 }}</td>
            <td>{{ tenant.stats?.runs_this_month || 0 }}</td>
            <td>{{ formatCost(tenant.stats?.cost_this_month || 0) }}</td>
            <td>
              <div class="action-buttons">
                <button class="btn btn-sm btn-ghost" @click="openEditModal(tenant)">
                  {{ $t('common.edit') }}
                </button>
                <button
                  v-if="tenant.status === 'active'"
                  class="btn btn-sm btn-ghost text-warning"
                  @click="openSuspendModal(tenant)"
                >
                  {{ $t('admin.tenants.suspend') }}
                </button>
                <button
                  v-else-if="tenant.status === 'suspended'"
                  class="btn btn-sm btn-ghost text-success"
                  @click="handleActivate(tenant)"
                >
                  {{ $t('admin.tenants.activate') }}
                </button>
                <button class="btn btn-sm btn-ghost text-error" @click="openDeleteModal(tenant)">
                  {{ $t('common.delete') }}
                </button>
              </div>
            </td>
          </tr>
          <tr v-if="tenants.length === 0">
            <td colspan="8" class="text-center text-secondary py-8">
              {{ $t('admin.tenants.noTenants') }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div v-if="pagination.total > pagination.limit" class="pagination mt-4">
      <button
        class="btn btn-sm"
        :disabled="pagination.page <= 1"
        @click="changePage(pagination.page - 1)"
      >
        {{ $t('common.previous') }}
      </button>
      <span class="pagination-info">
        {{ pagination.page }} / {{ Math.ceil(pagination.total / pagination.limit) }}
      </span>
      <button
        class="btn btn-sm"
        :disabled="pagination.page >= Math.ceil(pagination.total / pagination.limit)"
        @click="changePage(pagination.page + 1)"
      >
        {{ $t('common.next') }}
      </button>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
      <div class="modal">
        <div class="modal-header">
          <h2 class="modal-title">{{ $t('admin.tenants.createTenant') }}</h2>
          <button class="modal-close" @click="showCreateModal = false">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.name') }} *</label>
            <input
              v-model="formData.name"
              type="text"
              class="form-input"
              :placeholder="$t('admin.tenants.form.namePlaceholder')"
              @input="generateSlug"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.slug') }} *</label>
            <input
              v-model="formData.slug"
              type="text"
              class="form-input"
              :placeholder="$t('admin.tenants.form.slugPlaceholder')"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.plan') }}</label>
            <select v-model="formData.plan" class="form-input">
              <option value="free">{{ $t('admin.tenants.plan.free') }}</option>
              <option value="starter">{{ $t('admin.tenants.plan.starter') }}</option>
              <option value="professional">{{ $t('admin.tenants.plan.professional') }}</option>
              <option value="enterprise">{{ $t('admin.tenants.plan.enterprise') }}</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.ownerEmail') }}</label>
            <input
              v-model="formData.owner_email"
              type="email"
              class="form-input"
              :placeholder="$t('admin.tenants.form.ownerEmailPlaceholder')"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.ownerName') }}</label>
            <input
              v-model="formData.owner_name"
              type="text"
              class="form-input"
              :placeholder="$t('admin.tenants.form.ownerNamePlaceholder')"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.billingEmail') }}</label>
            <input
              v-model="formData.billing_email"
              type="email"
              class="form-input"
              :placeholder="$t('admin.tenants.form.billingEmailPlaceholder')"
            >
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost" @click="showCreateModal = false">
            {{ $t('common.cancel') }}
          </button>
          <button
            class="btn btn-primary"
            :disabled="!formData.name || !formData.slug || loading"
            @click="submitCreate"
          >
            {{ $t('common.create') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Edit Modal -->
    <div v-if="showEditModal" class="modal-overlay" @click.self="showEditModal = false">
      <div class="modal">
        <div class="modal-header">
          <h2 class="modal-title">{{ $t('admin.tenants.editTenant') }}</h2>
          <button class="modal-close" @click="showEditModal = false">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.name') }} *</label>
            <input
              v-model="formData.name"
              type="text"
              class="form-input"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.slug') }} *</label>
            <input
              v-model="formData.slug"
              type="text"
              class="form-input"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.plan') }}</label>
            <select v-model="formData.plan" class="form-input">
              <option value="free">{{ $t('admin.tenants.plan.free') }}</option>
              <option value="starter">{{ $t('admin.tenants.plan.starter') }}</option>
              <option value="professional">{{ $t('admin.tenants.plan.professional') }}</option>
              <option value="enterprise">{{ $t('admin.tenants.plan.enterprise') }}</option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.ownerEmail') }}</label>
            <input
              v-model="formData.owner_email"
              type="email"
              class="form-input"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.ownerName') }}</label>
            <input
              v-model="formData.owner_name"
              type="text"
              class="form-input"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.tenants.form.billingEmail') }}</label>
            <input
              v-model="formData.billing_email"
              type="email"
              class="form-input"
            >
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost" @click="showEditModal = false">
            {{ $t('common.cancel') }}
          </button>
          <button
            class="btn btn-primary"
            :disabled="!formData.name || !formData.slug || loading"
            @click="submitEdit"
          >
            {{ $t('common.save') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click.self="showDeleteModal = false">
      <div class="modal modal-sm">
        <div class="modal-header">
          <h2 class="modal-title">{{ $t('admin.tenants.deleteConfirm.title') }}</h2>
          <button class="modal-close" @click="showDeleteModal = false">&times;</button>
        </div>
        <div class="modal-body">
          <p>{{ $t('admin.tenants.deleteConfirm.message', { name: selectedTenant?.name }) }}</p>
          <p class="text-warning mt-2">{{ $t('admin.tenants.deleteConfirm.warning') }}</p>
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost" @click="showDeleteModal = false">
            {{ $t('common.cancel') }}
          </button>
          <button class="btn btn-error" :disabled="loading" @click="confirmDelete">
            {{ $t('common.delete') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Suspend Modal -->
    <div v-if="showSuspendModal" class="modal-overlay" @click.self="showSuspendModal = false">
      <div class="modal modal-sm">
        <div class="modal-header">
          <h2 class="modal-title">{{ $t('admin.tenants.suspendConfirm.title') }}</h2>
          <button class="modal-close" @click="showSuspendModal = false">&times;</button>
        </div>
        <div class="modal-body">
          <p>{{ $t('admin.tenants.suspendConfirm.message', { name: selectedTenant?.name }) }}</p>
          <div class="form-group mt-4">
            <label class="form-label">{{ $t('admin.tenants.suspendConfirm.reason') }} *</label>
            <textarea
              v-model="suspendReason"
              class="form-input"
              rows="3"
              :placeholder="$t('admin.tenants.suspendConfirm.reasonPlaceholder')"
            />
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn btn-ghost" @click="showSuspendModal = false">
            {{ $t('common.cancel') }}
          </button>
          <button
            class="btn btn-warning"
            :disabled="!suspendReason || loading"
            @click="confirmSuspend"
          >
            {{ $t('admin.tenants.suspend') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.breadcrumb-separator {
  color: var(--color-text-secondary);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0;
}

.page-description {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
}

.stat-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 1rem;
}

.stat-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-bottom: 0.25rem;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 600;
}

.filters {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.filter-select,
.filter-input {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  font-size: 0.875rem;
}

.filter-input {
  min-width: 200px;
}

.loading-state {
  text-align: center;
  padding: 3rem;
  color: var(--color-text-secondary);
}

.table-container {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  overflow-x: auto;
}

.table {
  width: 100%;
  border-collapse: collapse;
}

.table th,
.table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.table th {
  font-weight: 600;
  font-size: 0.75rem;
  text-transform: uppercase;
  color: var(--color-text-secondary);
  background: var(--color-background);
}

.table tbody tr:hover {
  background: var(--color-background);
}

.tenant-name {
  font-weight: 500;
}

.tenant-slug {
  font-size: 0.75rem;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius);
  font-size: 0.75rem;
  font-weight: 500;
}

.badge-success {
  background: #dcfce7;
  color: #166534;
}

.badge-error {
  background: #fef2f2;
  color: #dc2626;
}

.badge-warning {
  background: #fef3c7;
  color: #92400e;
}

.badge-secondary {
  background: var(--color-background);
  color: var(--color-text-secondary);
}

.badge-primary {
  background: #dbeafe;
  color: #1d4ed8;
}

.badge-info {
  background: #e0f2fe;
  color: #0369a1;
}

.action-buttons {
  display: flex;
  gap: 0.25rem;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 1rem;
}

.pagination-info {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.toast {
  position: fixed;
  top: 1rem;
  right: 1rem;
  padding: 0.75rem 1rem;
  border-radius: var(--radius);
  z-index: 1000;
  animation: slideIn 0.3s ease;
}

.toast-success {
  background: #dcfce7;
  color: #166534;
  border: 1px solid #86efac;
}

.toast-error {
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}

.modal {
  background: var(--color-surface);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 500px;
  max-height: 90vh;
  overflow: auto;
}

.modal-sm {
  max-width: 400px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: var(--color-text-secondary);
}

.modal-body {
  padding: 1.5rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--color-border);
}

.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  font-size: 0.875rem;
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 0.875rem;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

textarea.form-input {
  resize: vertical;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: var(--radius);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-ghost {
  background: transparent;
  color: var(--color-text);
}

.btn-ghost:hover {
  background: var(--color-background);
}

.btn-error {
  background: #dc2626;
  color: white;
}

.btn-warning {
  background: #f59e0b;
  color: white;
}

.text-secondary {
  color: var(--color-text-secondary);
}

.text-success {
  color: #16a34a;
}

.text-warning {
  color: #f59e0b;
}

.text-error {
  color: #dc2626;
}

.text-center {
  text-align: center;
}

.mt-2 {
  margin-top: 0.5rem;
}

.mt-4 {
  margin-top: 1rem;
}

.mb-4 {
  margin-bottom: 1rem;
}

.mb-6 {
  margin-bottom: 1.5rem;
}

.py-8 {
  padding-top: 2rem;
  padding-bottom: 2rem;
}
</style>
