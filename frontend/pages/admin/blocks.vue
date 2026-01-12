<script setup lang="ts">
import type { BlockDefinition, BlockCategory } from '~/types/api'
import { useAdminBlocks, type BlockVersion, categoryConfig } from '~/composables/useBlocks'

const { t } = useI18n()

definePageMeta({
  layout: 'default',
  middleware: ['admin'],
})

const blocksApi = useBlocks()
const adminBlocks = useAdminBlocks()

// State
const blocks = ref<BlockDefinition[]>([])
const loading = ref(true)
const message = ref<{ type: 'success' | 'error'; text: string } | null>(null)

// Modal state
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const showVersionModal = ref(false)
const selectedBlock = ref<BlockDefinition | null>(null)

// Filter state
const selectedCategory = ref<BlockCategory | ''>('')

// Create form state
const createForm = reactive({
  slug: '',
  name: '',
  description: '',
  category: 'integration' as BlockCategory,
  icon: '',
})

// Edit form state (includes code editing)
const editForm = reactive({
  name: '',
  description: '',
  icon: '',
  enabled: true,
  code: '',
  configSchema: '{}',
  uiConfig: '{}',
  changeSummary: '',
})

// Version state
const versions = ref<BlockVersion[]>([])
const loadingVersions = ref(false)

// Computed
const filteredBlocks = computed(() => {
  let result = blocks.value
  if (selectedCategory.value) {
    result = result.filter(b => b.category === selectedCategory.value)
  }
  // Sort by category then name
  return [...result].sort((a, b) => {
    if (a.category !== b.category) {
      return a.category.localeCompare(b.category)
    }
    return a.name.localeCompare(b.name)
  })
})

const categories: BlockCategory[] = ['ai', 'logic', 'data', 'integration', 'control', 'utility']

// Functions
function showMessageToast(type: 'success' | 'error', text: string) {
  message.value = { type, text }
  setTimeout(() => {
    message.value = null
  }, 3000)
}

async function fetchBlocks() {
  try {
    loading.value = true
    // Use admin API to get all blocks with code
    const response = await adminBlocks.listSystemBlocks()
    blocks.value = response.blocks || []
  } catch (err) {
    // Fallback to regular API
    try {
      const response = await blocksApi.list()
      blocks.value = response.blocks || []
    } catch {
      showMessageToast('error', t('errors.loadFailed'))
    }
  } finally {
    loading.value = false
  }
}

function resetCreateForm() {
  createForm.slug = ''
  createForm.name = ''
  createForm.description = ''
  createForm.category = 'integration'
  createForm.icon = ''
}

function openCreateModal() {
  resetCreateForm()
  showCreateModal.value = true
}

function openEditModal(block: BlockDefinition) {
  selectedBlock.value = block
  editForm.name = block.name
  editForm.description = block.description || ''
  editForm.icon = block.icon || ''
  editForm.enabled = block.enabled
  editForm.code = block.code || ''
  editForm.configSchema = JSON.stringify(block.config_schema || {}, null, 2)
  editForm.uiConfig = JSON.stringify(block.ui_config || {}, null, 2)
  editForm.changeSummary = ''
  showEditModal.value = true
}

function openDeleteModal(block: BlockDefinition) {
  selectedBlock.value = block
  showDeleteModal.value = true
}

async function openVersionModal(block: BlockDefinition) {
  selectedBlock.value = block
  versions.value = []
  loadingVersions.value = true
  showVersionModal.value = true

  try {
    const response = await adminBlocks.listVersions(block.id)
    versions.value = response.versions || []
  } catch (err) {
    showMessageToast('error', t('admin.blocks.messages.versionLoadFailed'))
    console.error('Error loading versions:', err)
  } finally {
    loadingVersions.value = false
  }
}

async function submitCreate() {
  try {
    await blocksApi.create({
      slug: createForm.slug,
      name: createForm.name,
      description: createForm.description || undefined,
      category: createForm.category,
      icon: createForm.icon || undefined,
    })
    showMessageToast('success', t('admin.blocks.messages.created'))
    showCreateModal.value = false
    resetCreateForm()
    await fetchBlocks()
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : t('errors.generic')
    showMessageToast('error', errorMessage)
  }
}

async function submitEdit() {
  if (!selectedBlock.value) return

  try {
    // Parse JSON fields
    let configSchema
    let uiConfig
    try {
      configSchema = JSON.parse(editForm.configSchema)
      uiConfig = JSON.parse(editForm.uiConfig)
    } catch {
      showMessageToast('error', t('admin.blocks.messages.invalidJson'))
      return
    }

    await adminBlocks.updateSystemBlock(selectedBlock.value.id, {
      name: editForm.name,
      description: editForm.description || undefined,
      code: editForm.code,
      config_schema: configSchema,
      ui_config: uiConfig,
      change_summary: editForm.changeSummary,
    })

    showMessageToast('success', t('admin.blocks.messages.updated'))
    showEditModal.value = false
    await fetchBlocks()
  } catch (err) {
    showMessageToast('error', t('admin.blocks.messages.updateFailed'))
    console.error('Error updating block:', err)
  }
}

async function confirmDelete() {
  if (!selectedBlock.value) return

  try {
    await blocksApi.remove(selectedBlock.value.slug)
    showMessageToast('success', t('admin.blocks.messages.deleted'))
    showDeleteModal.value = false
    selectedBlock.value = null
    await fetchBlocks()
  } catch (err) {
    showMessageToast('error', t('admin.blocks.messages.deleteFailed'))
  }
}

async function rollbackToVersion(version: BlockVersion) {
  if (!selectedBlock.value) return

  if (!confirm(t('admin.blocks.confirmRollback', { version: version.version }))) {
    return
  }

  try {
    await adminBlocks.rollback(selectedBlock.value.id, version.version)
    showMessageToast('success', t('admin.blocks.messages.rolledBack', { version: version.version }))
    showVersionModal.value = false
    await fetchBlocks()
  } catch (err) {
    showMessageToast('error', t('admin.blocks.messages.rollbackFailed'))
    console.error('Error rolling back:', err)
  }
}

function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleString()
}

function generateSlug() {
  if (createForm.name) {
    createForm.slug = createForm.name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_+|_+$/g, '')
  }
}

function getCategoryName(category: BlockCategory): string {
  const config = categoryConfig[category]
  return config ? t(config.nameKey) : category
}

function truncateCode(code: string | undefined, maxLength: number = 50): string {
  if (!code) return '-'
  return code.length > maxLength ? code.substring(0, maxLength) + '...' : code
}

onMounted(() => {
  fetchBlocks()
})
</script>

<template>
  <div>
    <!-- Breadcrumb -->
    <div class="breadcrumb mb-4">
      <NuxtLink to="/admin" class="breadcrumb-link">
        {{ $t('admin.title') }}
      </NuxtLink>
      <span class="breadcrumb-separator">/</span>
      <span>{{ $t('admin.blocks.title') }}</span>
    </div>

    <div class="flex justify-between items-center mb-4">
      <div>
        <h1 style="font-size: 1.5rem; font-weight: 600;">
          {{ $t('admin.blocks.title') }}
        </h1>
        <p class="text-secondary" style="margin-top: 0.25rem;">
          {{ $t('admin.blocks.subtitle') }}
        </p>
      </div>
      <button class="btn btn-primary" @click="openCreateModal">
        {{ $t('admin.blocks.newBlock') }}
      </button>
    </div>

    <!-- Success/Error message -->
    <div
      v-if="message"
      :class="['message-toast', message.type === 'success' ? 'bg-success' : 'bg-error']"
    >
      {{ message.text }}
    </div>

    <!-- Filter -->
    <div class="filter-bar mb-4">
      <label class="filter-label">{{ $t('admin.blocks.table.category') }}:</label>
      <select v-model="selectedCategory" class="filter-select">
        <option value="">{{ $t('common.all') }}</option>
        <option v-for="cat in categories" :key="cat" :value="cat">
          {{ getCategoryName(cat) }}
        </option>
      </select>
    </div>

    <!-- Loading state -->
    <div v-if="loading && blocks.length === 0" class="card" style="padding: 2rem; text-align: center;">
      <p class="text-secondary">{{ $t('common.loading') }}</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="blocks.length === 0" class="card" style="padding: 3rem; text-align: center;">
      <p class="text-secondary" style="font-size: 1.125rem; margin-bottom: 0.5rem;">
        {{ $t('admin.blocks.noBlocks') }}
      </p>
      <p class="text-secondary" style="margin-bottom: 1.5rem;">
        {{ $t('admin.blocks.noBlocksDesc') }}
      </p>
      <button class="btn btn-primary" @click="openCreateModal">
        {{ $t('admin.blocks.createFirst') }}
      </button>
    </div>

    <!-- Block list -->
    <div v-else class="card">
      <table class="table">
        <thead>
          <tr>
            <th>{{ $t('admin.blocks.table.name') }}</th>
            <th>{{ $t('admin.blocks.table.category') }}</th>
            <th>{{ $t('admin.blocks.table.code') }}</th>
            <th>{{ $t('admin.blocks.table.version') }}</th>
            <th>{{ $t('admin.blocks.table.enabled') }}</th>
            <th>{{ $t('admin.blocks.table.updatedAt') }}</th>
            <th>{{ $t('admin.blocks.table.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="block in filteredBlocks" :key="block.id">
            <td>
              <div>
                <strong>{{ block.name }}</strong>
                <code class="slug-code">{{ block.slug }}</code>
                <p v-if="block.description" class="text-secondary description-text">
                  {{ block.description }}
                </p>
              </div>
            </td>
            <td>
              <span class="badge badge-category">
                {{ getCategoryName(block.category) }}
              </span>
            </td>
            <td>
              <div class="code-preview" :title="block.code">
                {{ truncateCode(block.code) }}
              </div>
            </td>
            <td>
              <span class="version-badge">v{{ block.version || 1 }}</span>
            </td>
            <td>
              <span :class="['status-badge', block.enabled ? 'status-enabled' : 'status-disabled']">
                {{ block.enabled ? $t('common.enabled') : $t('common.disabled') }}
              </span>
            </td>
            <td>{{ formatDate(block.updated_at) }}</td>
            <td>
              <div class="flex gap-2">
                <button
                  class="btn btn-sm btn-primary"
                  @click="openEditModal(block)"
                >
                  {{ $t('common.edit') }}
                </button>
                <button
                  class="btn btn-sm btn-secondary"
                  @click="openVersionModal(block)"
                >
                  {{ $t('admin.blocks.versions') }}
                </button>
                <button
                  class="btn btn-sm btn-danger"
                  @click="openDeleteModal(block)"
                >
                  {{ $t('common.delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <UiModal
      :show="showCreateModal"
      :title="$t('admin.blocks.newBlock')"
      size="lg"
      @close="showCreateModal = false"
    >
      <form @submit.prevent="submitCreate">
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.name') }} *</label>
            <input
              v-model="createForm.name"
              type="text"
              class="form-input"
              :placeholder="$t('admin.blocks.form.namePlaceholder')"
              required
              @blur="generateSlug"
            />
          </div>
          <div class="form-group">
            <label class="form-label">Slug *</label>
            <input
              v-model="createForm.slug"
              type="text"
              class="form-input"
              placeholder="e.g., slack_message"
              required
              pattern="[a-z0-9_]+"
            />
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('admin.blocks.form.description') }}</label>
          <textarea
            v-model="createForm.description"
            class="form-input"
            :placeholder="$t('admin.blocks.form.descriptionPlaceholder')"
            rows="2"
          />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.category') }} *</label>
            <select
              v-model="createForm.category"
              class="form-input"
              required
            >
              <option v-for="cat in categories" :key="cat" :value="cat">
                {{ getCategoryName(cat) }}
              </option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.icon') }}</label>
            <input
              v-model="createForm.icon"
              type="text"
              class="form-input"
              placeholder="e.g., message-circle"
            />
          </div>
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="loading" @click="submitCreate">
          {{ $t('common.create') }}
        </button>
      </template>
    </UiModal>

    <!-- Edit Modal (with code editing) -->
    <UiModal
      :show="showEditModal"
      :title="$t('admin.blocks.editBlock') + ': ' + (selectedBlock?.name || '')"
      size="xl"
      @close="showEditModal = false"
    >
      <form @submit.prevent="submitEdit">
        <!-- Basic Info Section -->
        <div class="section-title">{{ $t('admin.blocks.sections.basicInfo') }}</div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.name') }} *</label>
            <input
              v-model="editForm.name"
              type="text"
              class="form-input"
              required
            />
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.icon') }}</label>
            <input
              v-model="editForm.icon"
              type="text"
              class="form-input"
              placeholder="e.g., message-circle"
            />
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('admin.blocks.form.description') }}</label>
          <input
            v-model="editForm.description"
            type="text"
            class="form-input"
          />
        </div>

        <!-- Code Section -->
        <div class="section-title">{{ $t('admin.blocks.sections.code') }}</div>

        <div class="form-group">
          <label class="form-label">{{ $t('admin.blocks.form.code') }} (JavaScript)</label>
          <textarea
            v-model="editForm.code"
            class="form-input code-editor"
            rows="12"
            spellcheck="false"
          />
        </div>

        <!-- Schema Section -->
        <div class="section-title">{{ $t('admin.blocks.sections.schemas') }}</div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.configSchema') }} (JSON)</label>
            <textarea
              v-model="editForm.configSchema"
              class="form-input code-editor"
              rows="6"
              spellcheck="false"
            />
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.uiConfig') }} (JSON)</label>
            <textarea
              v-model="editForm.uiConfig"
              class="form-input code-editor"
              rows="6"
              spellcheck="false"
            />
          </div>
        </div>

        <!-- Change Summary -->
        <div class="form-group">
          <label class="form-label">{{ $t('admin.blocks.form.changeSummary') }}</label>
          <input
            v-model="editForm.changeSummary"
            type="text"
            class="form-input"
            :placeholder="$t('admin.blocks.form.changeSummaryPlaceholder')"
          />
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showEditModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="loading" @click="submitEdit">
          {{ $t('common.save') }}
        </button>
      </template>
    </UiModal>

    <!-- Delete Confirmation Modal -->
    <UiModal
      :show="showDeleteModal"
      :title="$t('admin.blocks.deleteBlock')"
      size="sm"
      @close="showDeleteModal = false"
    >
      <p>{{ $t('admin.blocks.confirmDelete') }}</p>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-danger" :disabled="loading" @click="confirmDelete">
          {{ $t('common.delete') }}
        </button>
      </template>
    </UiModal>

    <!-- Versions Modal -->
    <UiModal
      :show="showVersionModal"
      :title="$t('admin.blocks.versionHistory') + ': ' + (selectedBlock?.name || '')"
      size="lg"
      @close="showVersionModal = false"
    >
      <div v-if="loadingVersions" style="text-align: center; padding: 2rem;">
        {{ $t('common.loading') }}
      </div>

      <div v-else-if="versions.length === 0" style="text-align: center; padding: 2rem;">
        <p class="text-secondary">{{ $t('admin.blocks.noVersions') }}</p>
      </div>

      <div v-else class="version-list">
        <div v-for="version in versions" :key="version.id" class="version-item">
          <div class="version-header">
            <span class="version-badge">v{{ version.version }}</span>
            <span class="version-date">{{ formatDate(version.created_at) }}</span>
          </div>
          <p v-if="version.change_summary" class="version-summary">
            {{ version.change_summary }}
          </p>
          <div class="version-actions">
            <button
              class="btn btn-sm btn-secondary"
              @click="rollbackToVersion(version)"
            >
              {{ $t('admin.blocks.rollbackTo') }}
            </button>
          </div>
        </div>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showVersionModal = false">
          {{ $t('common.close') }}
        </button>
      </template>
    </UiModal>
  </div>
</template>

<style scoped>
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
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

.message-toast {
  padding: 0.75rem 1rem;
  margin-bottom: 1rem;
  border-radius: var(--radius);
}

.bg-success {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.bg-error {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.filter-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.filter-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.filter-select {
  padding: 0.375rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-background);
  font-size: 0.875rem;
  min-width: 150px;
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
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.slug-code {
  display: inline-block;
  margin-left: 0.5rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.75rem;
  background: var(--color-background);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  color: var(--color-text-secondary);
}

.description-text {
  font-size: 0.75rem;
  margin-top: 0.25rem;
}

.badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
}

.badge-category {
  background: rgba(99, 102, 241, 0.1);
  color: var(--color-primary);
}

.version-badge {
  display: inline-block;
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
  background: rgba(99, 102, 241, 0.1);
  color: var(--color-primary);
}

.code-preview {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
}

.status-enabled {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.status-disabled {
  background: rgba(107, 114, 128, 0.1);
  color: #6b7280;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

.btn-danger {
  background: #ef4444;
  color: white;
}

.btn-danger:hover {
  background: #dc2626;
}

.section-title {
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--color-text);
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--color-border);
}

.section-title:first-child {
  margin-top: 0;
}

.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-weight: 500;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  background: var(--color-background);
  color: var(--color-text);
  font-size: 0.875rem;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.form-input:disabled {
  background: var(--color-surface);
  color: var(--color-text-secondary);
  cursor: not-allowed;
}

textarea.form-input {
  resize: vertical;
  min-height: 60px;
}

.code-editor {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
  resize: vertical;
  min-height: 100px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.version-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.version-item {
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
}

.version-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.25rem;
}

.version-date {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.version-summary {
  font-size: 0.875rem;
  color: var(--color-text);
  margin-bottom: 0.5rem;
}

.version-actions {
  margin-top: 0.5rem;
}
</style>
