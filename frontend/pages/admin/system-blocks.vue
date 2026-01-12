<script setup lang="ts">
import type { BlockDefinition } from '~/types/api'
import { useAdminBlocks, type BlockVersion, categoryConfig } from '~/composables/useBlocks'

const { t } = useI18n()

definePageMeta({
  layout: 'default',
})

const adminBlocks = useAdminBlocks()

// State
const blocks = ref<BlockDefinition[]>([])
const loading = ref(true)
const message = ref<{ type: 'success' | 'error'; text: string } | null>(null)

// Edit modal state
const showEditModal = ref(false)
const selectedBlock = ref<BlockDefinition | null>(null)
const editForm = reactive({
  name: '',
  description: '',
  code: '',
  configSchema: '{}',
  uiConfig: '{}',
  changeSummary: '',
})

// Version modal state
const showVersionModal = ref(false)
const versions = ref<BlockVersion[]>([])
const loadingVersions = ref(false)

// Computed
const sortedBlocks = computed(() => {
  return [...blocks.value].sort((a, b) => {
    // Sort by category first, then by name
    if (a.category !== b.category) {
      return a.category.localeCompare(b.category)
    }
    return a.name.localeCompare(b.name)
  })
})

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
    const response = await adminBlocks.listSystemBlocks()
    blocks.value = response.blocks || []
  } catch (err) {
    showMessageToast('error', 'Failed to load system blocks')
    console.error('Error loading system blocks:', err)
  } finally {
    loading.value = false
  }
}

function openEditModal(block: BlockDefinition) {
  selectedBlock.value = block
  editForm.name = block.name
  editForm.description = block.description || ''
  editForm.code = block.code || ''
  editForm.configSchema = JSON.stringify(block.config_schema || {}, null, 2)
  editForm.uiConfig = JSON.stringify(block.ui_config || {}, null, 2)
  editForm.changeSummary = ''
  showEditModal.value = true
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
    } catch (e) {
      showMessageToast('error', 'Invalid JSON in schema fields')
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

    showMessageToast('success', 'Block updated successfully')
    showEditModal.value = false
    await fetchBlocks()
  } catch (err) {
    showMessageToast('error', 'Failed to update block')
    console.error('Error updating block:', err)
  }
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
    showMessageToast('error', 'Failed to load versions')
    console.error('Error loading versions:', err)
  } finally {
    loadingVersions.value = false
  }
}

async function rollbackToVersion(version: BlockVersion) {
  if (!selectedBlock.value) return

  if (!confirm(`Rollback to version ${version.version}? This will create a new version.`)) {
    return
  }

  try {
    await adminBlocks.rollback(selectedBlock.value.id, version.version)
    showMessageToast('success', `Rolled back to version ${version.version}`)
    showVersionModal.value = false
    await fetchBlocks()
  } catch (err) {
    showMessageToast('error', 'Failed to rollback')
    console.error('Error rolling back:', err)
  }
}

function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleString()
}

function getCategoryName(category: string): string {
  const config = categoryConfig[category as keyof typeof categoryConfig]
  return config ? t(config.nameKey) : category
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
      <span>System Blocks</span>
    </div>

    <div class="flex justify-between items-center mb-4">
      <div>
        <h1 style="font-size: 1.5rem; font-weight: 600;">
          System Blocks
        </h1>
        <p class="text-secondary" style="margin-top: 0.25rem;">
          Edit system block code and configuration
        </p>
      </div>
    </div>

    <!-- Success/Error message -->
    <div
      v-if="message"
      :class="['message-toast', message.type === 'success' ? 'bg-success' : 'bg-error']"
    >
      {{ message.text }}
    </div>

    <!-- Loading state -->
    <div v-if="loading && blocks.length === 0" class="card" style="padding: 2rem; text-align: center;">
      <p class="text-secondary">{{ $t('common.loading') }}</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="blocks.length === 0" class="card" style="padding: 3rem; text-align: center;">
      <p class="text-secondary" style="font-size: 1.125rem;">
        No system blocks found
      </p>
    </div>

    <!-- Block list -->
    <div v-else class="card">
      <table class="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Category</th>
            <th>Code</th>
            <th>Version</th>
            <th>Updated</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="block in sortedBlocks" :key="block.id">
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
                {{ block.code ? block.code.substring(0, 50) + (block.code.length > 50 ? '...' : '') : '-' }}
              </div>
            </td>
            <td>
              <span class="version-badge">v{{ block.version || 1 }}</span>
            </td>
            <td>{{ formatDate(block.updated_at) }}</td>
            <td>
              <div class="flex gap-2">
                <button
                  class="btn btn-sm btn-primary"
                  @click="openEditModal(block)"
                >
                  Edit Code
                </button>
                <button
                  class="btn btn-sm btn-secondary"
                  @click="openVersionModal(block)"
                >
                  Versions
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Edit Modal -->
    <UiModal
      :show="showEditModal"
      :title="`Edit: ${selectedBlock?.name}`"
      size="xl"
      @close="showEditModal = false"
    >
      <form @submit.prevent="submitEdit">
        <div class="form-group">
          <label class="form-label">Name</label>
          <input
            v-model="editForm.name"
            type="text"
            class="form-input"
            required
          />
        </div>

        <div class="form-group">
          <label class="form-label">Description</label>
          <input
            v-model="editForm.description"
            type="text"
            class="form-input"
          />
        </div>

        <div class="form-group">
          <label class="form-label">Code (JavaScript)</label>
          <textarea
            v-model="editForm.code"
            class="form-input code-editor"
            rows="12"
            spellcheck="false"
          />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Config Schema (JSON)</label>
            <textarea
              v-model="editForm.configSchema"
              class="form-input code-editor"
              rows="6"
              spellcheck="false"
            />
          </div>
          <div class="form-group">
            <label class="form-label">UI Config (JSON)</label>
            <textarea
              v-model="editForm.uiConfig"
              class="form-input code-editor"
              rows="6"
              spellcheck="false"
            />
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">Change Summary</label>
          <input
            v-model="editForm.changeSummary"
            type="text"
            class="form-input"
            placeholder="Describe what you changed..."
          />
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showEditModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="loading" @click="submitEdit">
          Save Changes
        </button>
      </template>
    </UiModal>

    <!-- Versions Modal -->
    <UiModal
      :show="showVersionModal"
      :title="`Versions: ${selectedBlock?.name}`"
      size="lg"
      @close="showVersionModal = false"
    >
      <div v-if="loadingVersions" style="text-align: center; padding: 2rem;">
        {{ $t('common.loading') }}
      </div>

      <div v-else-if="versions.length === 0" style="text-align: center; padding: 2rem;">
        <p class="text-secondary">No version history available</p>
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
              Rollback to this version
            </button>
          </div>
        </div>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showVersionModal = false">
          Close
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
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
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
