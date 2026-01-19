<script setup lang="ts">
import type { CustomBlockPackage, BlockPackageStatus } from '~/types/api'

definePageMeta({
  layout: 'default'
})

const { t } = useI18n()
const toast = useToast()
const { confirm } = useConfirm()

const {
  packages,
  loading,
  error,
  fetchPackages,
  createPackage,
  publishPackage,
  deprecatePackage,
  deletePackage
} = useBlockPackages()

// Filter state
const searchQuery = ref('')
const statusFilter = ref<BlockPackageStatus | ''>('')

// Modal state
const showCreateModal = ref(false)
const createForm = ref({
  name: '',
  version: '1.0.0',
  description: '',
  blocks: [] as Array<{
    slug: string
    name: string
    description?: string
    category: string
    icon?: string
    config_schema: object
    code: string
  }>,
  dependencies: [] as Array<{ name: string; version: string }>
})

// Fetch data on mount
onMounted(() => {
  loadPackages()
})

// Watch filters
watch([searchQuery, statusFilter], () => {
  loadPackages()
})

async function loadPackages() {
  await fetchPackages({
    search: searchQuery.value || undefined,
    status: statusFilter.value || undefined
  })
}

function openCreateModal() {
  createForm.value = {
    name: '',
    version: '1.0.0',
    description: '',
    blocks: [],
    dependencies: []
  }
  showCreateModal.value = true
}

async function handleCreate() {
  if (!createForm.value.name) {
    toast.error('パッケージ名は必須です')
    return
  }

  try {
    await createPackage({
      name: createForm.value.name,
      version: createForm.value.version,
      description: createForm.value.description,
      blocks: createForm.value.blocks,
      dependencies: createForm.value.dependencies
    })
    toast.success(t('blockPackages.messages.createSuccess'))
    showCreateModal.value = false
    await loadPackages()
  } catch {
    toast.error(t('blockPackages.messages.createFailed'))
  }
}

async function handlePublish(pkg: CustomBlockPackage) {
  try {
    await publishPackage(pkg.id)
    toast.success(t('blockPackages.messages.publishSuccess'))
    await loadPackages()
  } catch {
    toast.error(t('blockPackages.messages.publishFailed'))
  }
}

async function handleDeprecate(pkg: CustomBlockPackage) {
  const confirmed = await confirm({
    title: t('blockPackages.deprecate'),
    message: `パッケージ「${pkg.name}」を非推奨にしますか？`,
    confirmText: t('blockPackages.deprecate'),
    cancelText: t('common.cancel'),
    variant: 'danger'
  })

  if (confirmed) {
    try {
      await deprecatePackage(pkg.id)
      toast.success(t('blockPackages.messages.deprecateSuccess'))
      await loadPackages()
    } catch {
      toast.error('非推奨化に失敗しました')
    }
  }
}

async function handleDelete(pkg: CustomBlockPackage) {
  const confirmed = await confirm({
    title: t('common.delete'),
    message: `パッケージ「${pkg.name}」を削除しますか？`,
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    variant: 'danger'
  })

  if (confirmed) {
    try {
      await deletePackage(pkg.id)
      toast.success(t('blockPackages.messages.deleteSuccess'))
      await loadPackages()
    } catch {
      toast.error('削除に失敗しました')
    }
  }
}

function getStatusBadgeClass(status: BlockPackageStatus): string {
  switch (status) {
    case 'published': return 'badge-success'
    case 'deprecated': return 'badge-warning'
    default: return 'badge-secondary'
  }
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('ja-JP')
}
</script>

<template>
  <div class="packages-page">
    <!-- Header -->
    <div class="page-header">
      <div class="header-content">
        <h1 class="page-title">{{ t('blockPackages.title') }}</h1>
        <p class="page-subtitle">{{ t('blockPackages.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <button class="btn btn-primary" @click="openCreateModal">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
          </svg>
          {{ t('blockPackages.createPackage') }}
        </button>
      </div>
    </div>

    <!-- Filters -->
    <div class="filters">
      <div class="search-box">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/>
        </svg>
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="t('blockPackages.searchPlaceholder')"
          class="search-input"
        >
      </div>

      <select v-model="statusFilter" class="filter-select">
        <option value="">{{ t('common.all') }}</option>
        <option value="draft">{{ t('blockPackages.status.draft') }}</option>
        <option value="published">{{ t('blockPackages.status.published') }}</option>
        <option value="deprecated">{{ t('blockPackages.status.deprecated') }}</option>
      </select>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <div class="spinner" />
      <span>{{ t('common.loading') }}</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-state">
      <p>{{ error }}</p>
      <button class="btn btn-primary" @click="loadPackages">{{ t('common.retry') }}</button>
    </div>

    <!-- Empty State -->
    <div v-else-if="packages.length === 0" class="empty-state">
      <div class="empty-icon">
        <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
          <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
          <line x1="12" y1="22.08" x2="12" y2="12"/>
        </svg>
      </div>
      <h3>{{ t('blockPackages.noPackages') }}</h3>
      <p>{{ t('blockPackages.noPackagesDesc') }}</p>
      <button class="btn btn-primary" @click="openCreateModal">
        {{ t('blockPackages.createPackage') }}
      </button>
    </div>

    <!-- Package List -->
    <div v-else class="packages-table-wrapper">
      <table class="packages-table">
        <thead>
          <tr>
            <th>{{ t('common.name') }}</th>
            <th>{{ t('blockPackages.version') }}</th>
            <th>{{ t('common.status') }}</th>
            <th>{{ t('blockPackages.blocks') }}</th>
            <th>{{ t('common.createdAt') }}</th>
            <th>{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="pkg in packages" :key="pkg.id">
            <td>
              <div class="package-name-cell">
                <span class="package-name">{{ pkg.name }}</span>
                <span v-if="pkg.description" class="package-description">{{ pkg.description }}</span>
              </div>
            </td>
            <td>
              <code class="version-badge">v{{ pkg.version }}</code>
            </td>
            <td>
              <span :class="['status-badge', getStatusBadgeClass(pkg.status)]">
                {{ t(`blockPackages.status.${pkg.status}`) }}
              </span>
            </td>
            <td>
              <span class="blocks-count">{{ pkg.blocks?.length || 0 }}</span>
            </td>
            <td>
              <span class="date">{{ formatDate(pkg.created_at) }}</span>
            </td>
            <td>
              <div class="actions-cell">
                <button
                  v-if="pkg.status === 'draft'"
                  class="btn btn-sm btn-success"
                  @click="handlePublish(pkg)"
                >
                  {{ t('blockPackages.publish') }}
                </button>
                <button
                  v-if="pkg.status === 'published'"
                  class="btn btn-sm btn-warning-outline"
                  @click="handleDeprecate(pkg)"
                >
                  {{ t('blockPackages.deprecate') }}
                </button>
                <button
                  class="btn btn-sm btn-danger-outline"
                  @click="handleDelete(pkg)"
                >
                  {{ t('common.delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <UiModal :show="showCreateModal" :title="t('blockPackages.createPackage')" size="md" @close="showCreateModal = false">
      <div class="create-form">
        <div class="form-group">
          <label class="form-label">{{ t('blockPackages.form.name') }} *</label>
          <input
            v-model="createForm.name"
            type="text"
            class="form-input"
            :placeholder="t('blockPackages.form.namePlaceholder')"
          >
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('blockPackages.form.version') }}</label>
          <input
            v-model="createForm.version"
            type="text"
            class="form-input"
            :placeholder="t('blockPackages.form.versionPlaceholder')"
          >
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('blockPackages.form.description') }}</label>
          <textarea
            v-model="createForm.description"
            class="form-textarea"
            rows="3"
            :placeholder="t('blockPackages.form.descriptionPlaceholder')"
          />
        </div>

        <div class="form-note">
          ブロックの追加は作成後に行えます。
        </div>

        <div class="form-actions">
          <button class="btn btn-secondary" @click="showCreateModal = false">
            {{ t('common.cancel') }}
          </button>
          <button class="btn btn-primary" @click="handleCreate">
            {{ t('common.create') }}
          </button>
        </div>
      </div>
    </UiModal>
  </div>
</template>

<style scoped>
.packages-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 2rem;
}

.page-title {
  font-size: 1.75rem;
  font-weight: 700;
  margin: 0 0 0.25rem 0;
  color: var(--color-text);
}

.page-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.header-actions .btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.filters {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  flex: 1;
  max-width: 300px;
}

.search-box svg {
  color: var(--color-text-secondary);
}

.search-input {
  border: none;
  outline: none;
  font-size: 0.875rem;
  width: 100%;
}

.filter-select {
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: white;
}

.loading-state,
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  gap: 1rem;
}

.spinner {
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

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 4rem 2rem;
  text-align: center;
}

.empty-icon {
  color: var(--color-primary);
  opacity: 0.5;
  margin-bottom: 1rem;
}

.empty-state h3 {
  font-size: 1.125rem;
  margin: 0 0 0.5rem 0;
}

.empty-state p {
  color: var(--color-text-secondary);
  margin: 0 0 1.5rem 0;
}

.packages-table-wrapper {
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
}

.packages-table {
  width: 100%;
  border-collapse: collapse;
}

.packages-table th,
.packages-table td {
  padding: 0.875rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.packages-table th {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--color-surface);
}

.packages-table tbody tr:last-child td {
  border-bottom: none;
}

.packages-table tbody tr:hover {
  background: var(--color-surface);
}

.package-name-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.package-name {
  font-weight: 500;
  color: var(--color-text);
}

.package-description {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.version-badge {
  font-size: 0.75rem;
  padding: 0.25rem 0.5rem;
  background: var(--color-surface);
  border-radius: 4px;
  font-family: 'SF Mono', Monaco, monospace;
}

.status-badge {
  font-size: 0.6875rem;
  padding: 0.25rem 0.625rem;
  border-radius: 4px;
  font-weight: 500;
}

.badge-success {
  background: #dcfce7;
  color: #15803d;
}

.badge-warning {
  background: #fef3c7;
  color: #b45309;
}

.badge-secondary {
  background: #f1f5f9;
  color: #64748b;
}

.blocks-count {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.date {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.actions-cell {
  display: flex;
  gap: 0.5rem;
}

.btn-sm {
  padding: 0.375rem 0.625rem;
  font-size: 0.75rem;
}

.btn-success {
  background: #10b981;
  color: white;
  border: none;
}

.btn-success:hover {
  background: #059669;
}

.btn-warning-outline {
  background: white;
  border: 1px solid #fcd34d;
  color: #b45309;
}

.btn-warning-outline:hover {
  background: #fef3c7;
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

/* Create Modal Form */
.create-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.form-input,
.form-textarea {
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: white;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-textarea {
  resize: vertical;
  font-family: inherit;
}

.form-note {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  padding: 0.75rem;
  background: var(--color-surface);
  border-radius: 6px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}
</style>
