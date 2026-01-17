<script setup lang="ts">
/**
 * Admin Blocks Page - ブロック管理ページ
 *
 * システムブロックとカスタムブロックの作成・編集・削除・バージョン管理を行う。
 * 新しいBlockEditorコンポーネントを使用した改善版UI。
 */
import type { BlockDefinition, BlockCategory } from '~/types/api'
import { useAdminBlocks, type BlockVersion, categoryConfig } from '~/composables/useBlocks'
import type { BlockFormData } from '~/composables/useBlockEditor'

const { t } = useI18n()
const { confirm } = useConfirm()

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
const showCreateWizard = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const showVersionModal = ref(false)
const selectedBlock = ref<BlockDefinition | null>(null)

// Creation wizard state
const creationType = ref<'scratch' | 'inherit' | 'template'>('scratch')
const selectedTemplate = ref<string | null>(null)
const selectedParentBlock = ref<BlockDefinition | null>(null)

// Version state
const versions = ref<BlockVersion[]>([])
const loadingVersions = ref(false)

// Filter state
const searchQuery = ref('')
const selectedCategory = ref<BlockCategory | ''>('')
const viewMode = ref<'grid' | 'table'>('grid')

// Computed
const filteredBlocks = computed(() => {
  let result = blocks.value

  // Category filter
  if (selectedCategory.value) {
    result = result.filter(b => b.category === selectedCategory.value)
  }

  // Search filter
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(b =>
      b.name.toLowerCase().includes(query) ||
      b.slug.toLowerCase().includes(query) ||
      b.description?.toLowerCase().includes(query)
    )
  }

  // Sort by category then name
  return [...result].sort((a, b) => {
    if (a.category !== b.category) {
      return a.category.localeCompare(b.category)
    }
    return a.name.localeCompare(b.name)
  })
})

const categories: BlockCategory[] = ['ai', 'flow', 'apps', 'custom']

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
    try {
      const response = await adminBlocks.listSystemBlocks()
      blocks.value = response.blocks || []
    } catch {
      // Fallback to regular API
      const response = await blocksApi.list()
      blocks.value = response.blocks || []
    }
  } catch {
    showMessageToast('error', t('errors.loadFailed'))
  } finally {
    loading.value = false
  }
}

function openCreateWizard() {
  creationType.value = 'scratch'
  selectedTemplate.value = null
  selectedParentBlock.value = null
  showCreateWizard.value = true
}

function handleWizardSelect(type: 'scratch' | 'inherit' | 'template', data?: { templateId?: string; parentBlock?: BlockDefinition }) {
  creationType.value = type
  if (data?.templateId) {
    selectedTemplate.value = data.templateId
  }
  if (data?.parentBlock) {
    selectedParentBlock.value = data.parentBlock
  }
  showCreateWizard.value = false
  showEditModal.value = true
  selectedBlock.value = null
}

function openEditModal(block: BlockDefinition) {
  selectedBlock.value = block
  creationType.value = block.parent_block_id ? 'inherit' : 'scratch'
  selectedParentBlock.value = block.parent_block_id
    ? blocks.value.find(b => b.id === block.parent_block_id) || null
    : null
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

async function handleFormSubmit(formData: BlockFormData) {
  try {
    // Parse JSON fields
    let configSchema, uiConfig
    try {
      configSchema = JSON.parse(formData.config_schema || '{}')
      uiConfig = JSON.parse(formData.ui_config || '{}')
    } catch {
      showMessageToast('error', t('blockEditor.errors.invalidJson'))
      return
    }

    // Parse optional JSON fields
    if (formData.config_defaults) {
      try {
        JSON.parse(formData.config_defaults)
      } catch {
        showMessageToast('error', t('blockEditor.errors.invalidJson'))
        return
      }
    }

    if (selectedBlock.value) {
      // Update existing block
      await adminBlocks.updateSystemBlock(selectedBlock.value.id, {
        name: formData.name,
        description: formData.description || undefined,
        code: formData.code,
        config_schema: configSchema,
        ui_config: uiConfig,
        change_summary: formData.change_summary,
      })
      showMessageToast('success', t('admin.blocks.messages.updated'))
    } else {
      // Create new block
      // Note: parent_block_id, config_defaults, pre_process, post_process would need backend API support
      await blocksApi.create({
        slug: formData.slug,
        name: formData.name,
        description: formData.description || undefined,
        category: formData.category,
        icon: formData.icon || undefined,
        code: formData.code || undefined,
        config_schema: configSchema,
        ui_config: uiConfig,
      })
      showMessageToast('success', t('admin.blocks.messages.created'))
    }

    showEditModal.value = false
    selectedBlock.value = null
    await fetchBlocks()
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : t('errors.generic')
    showMessageToast('error', errorMessage)
  }
}

async function handleDuplicate(block: BlockDefinition) {
  try {
    let configSchema, uiConfig
    try {
      configSchema = typeof block.config_schema === 'string'
        ? JSON.parse(block.config_schema)
        : block.config_schema || {}
      uiConfig = typeof block.ui_config === 'string'
        ? JSON.parse(block.ui_config)
        : block.ui_config || {}
    } catch {
      configSchema = {}
      uiConfig = {}
    }

    await blocksApi.create({
      slug: block.slug + '_copy',
      name: block.name + ' (Copy)',
      description: block.description || undefined,
      category: block.category,
      icon: block.icon || undefined,
      code: block.code || undefined,
      config_schema: configSchema,
      ui_config: uiConfig,
    })
    showMessageToast('success', t('admin.blocks.messages.created'))
    await fetchBlocks()
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : t('errors.generic')
    showMessageToast('error', errorMessage)
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
  } catch {
    showMessageToast('error', t('admin.blocks.messages.deleteFailed'))
  }
}

async function rollbackToVersion(version: BlockVersion) {
  if (!selectedBlock.value) return

  const confirmed = await confirm({
    title: t('admin.blocks.rollbackTitle'),
    message: t('admin.blocks.confirmRollback', { version: version.version }),
    confirmText: t('admin.blocks.rollback'),
    cancelText: t('common.cancel'),
    variant: 'danger',
  })
  if (!confirmed) return

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

function getCategoryName(category: BlockCategory): string {
  const config = categoryConfig[category]
  return config ? t(config.nameKey) : category
}

function blockToFormData(block: BlockDefinition): BlockFormData {
  return {
    slug: block.slug,
    name: block.name,
    description: block.description || '',
    category: block.category,
    icon: block.icon || '',
    code: block.code || '',
    config_schema: JSON.stringify(block.config_schema || {}, null, 2),
    ui_config: JSON.stringify(block.ui_config || {}, null, 2),
    change_summary: '',
    parent_block_id: block.parent_block_id,
    config_defaults: JSON.stringify(block.config_defaults || {}, null, 2),
    pre_process: block.pre_process || '',
    post_process: block.post_process || '',
  }
}

function getInitialFormData(): BlockFormData {
  // Template defaults
  const templates: Record<string, Partial<BlockFormData>> = {
    discord_notify: {
      name: 'Discord Notification',
      slug: 'discord_notify',
      description: 'Send notification to Discord webhook',
      category: 'apps',
      icon: 'message-circle',
      code: `// Discord notification
const webhookUrl = config.webhook_url;
const message = input.message || 'Notification from workflow';

const response = ctx.http.post(webhookUrl, {
  body: JSON.stringify({ content: message }),
  headers: { 'Content-Type': 'application/json' }
});

return { success: response.status === 204 };`,
      config_schema: JSON.stringify({
        type: 'object',
        properties: {
          webhook_url: { type: 'string', title: 'Webhook URL' }
        },
        required: ['webhook_url']
      }, null, 2),
    },
    slack_notify: {
      name: 'Slack Notification',
      slug: 'slack_notify',
      description: 'Send notification to Slack webhook',
      category: 'apps',
      icon: 'hash',
      code: `// Slack notification
const webhookUrl = config.webhook_url;
const message = input.message || 'Notification from workflow';

const response = ctx.http.post(webhookUrl, {
  body: JSON.stringify({ text: message }),
  headers: { 'Content-Type': 'application/json' }
});

return { success: response.status === 200 };`,
      config_schema: JSON.stringify({
        type: 'object',
        properties: {
          webhook_url: { type: 'string', title: 'Webhook URL' }
        },
        required: ['webhook_url']
      }, null, 2),
    },
    json_transformer: {
      name: 'JSON Transformer',
      slug: 'json_transformer',
      description: 'Transform JSON data with custom logic',
      category: 'flow',
      icon: 'code',
      code: `// Transform input JSON
const data = input.data || input;
const result = {};

// Add your transformation logic here
for (const key in data) {
  result[key] = data[key];
}

return result;`,
    },
    api_request: {
      name: 'API Request',
      slug: 'api_request',
      description: 'Make HTTP API request',
      category: 'apps',
      icon: 'globe',
      code: `// API request
const url = config.url;
const method = config.method || 'GET';
const headers = config.headers || {};
const body = input.body;

const response = ctx.http[method.toLowerCase()](url, {
  headers: headers,
  body: body ? JSON.stringify(body) : undefined
});

return {
  status: response.status,
  body: response.json()
};`,
      config_schema: JSON.stringify({
        type: 'object',
        properties: {
          url: { type: 'string', title: 'URL' },
          method: { type: 'string', enum: ['GET', 'POST', 'PUT', 'DELETE'], default: 'GET' },
          headers: { type: 'object', title: 'Headers' }
        },
        required: ['url']
      }, null, 2),
    },
  }

  if (selectedTemplate.value && templates[selectedTemplate.value]) {
    const template = templates[selectedTemplate.value]
    return {
      slug: template.slug || '',
      name: template.name || '',
      description: template.description || '',
      category: template.category || 'custom',
      icon: template.icon || '',
      code: template.code || '',
      config_schema: template.config_schema || '{}',
      ui_config: '{}',
      change_summary: '',
      config_defaults: '{}',
      pre_process: '',
      post_process: '',
    }
  }

  if (creationType.value === 'inherit' && selectedParentBlock.value) {
    return {
      slug: '',
      name: '',
      description: '',
      category: selectedParentBlock.value.category,
      icon: selectedParentBlock.value.icon || '',
      code: '',
      config_schema: JSON.stringify(selectedParentBlock.value.config_schema || {}, null, 2),
      ui_config: JSON.stringify(selectedParentBlock.value.ui_config || {}, null, 2),
      change_summary: '',
      parent_block_id: selectedParentBlock.value.id,
      config_defaults: '{}',
      pre_process: '',
      post_process: '',
    }
  }

  return {
    slug: '',
    name: '',
    description: '',
    category: 'custom',
    icon: '',
    code: '',
    config_schema: '{}',
    ui_config: '{}',
    change_summary: '',
    config_defaults: '{}',
    pre_process: '',
    post_process: '',
  }
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

    <!-- Header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          {{ $t('admin.blocks.title') }}
        </h1>
        <p class="page-subtitle">
          {{ $t('admin.blocks.subtitle') }}
        </p>
      </div>
      <button class="btn btn-primary" @click="openCreateWizard">
        <span class="btn-icon">+</span>
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

    <!-- Filter Bar -->
    <div class="filter-bar">
      <div class="filter-left">
        <div class="search-input-wrapper">
          <input
            v-model="searchQuery"
            type="text"
            class="search-input"
            :placeholder="$t('blockEditor.searchPlaceholder')"
          >
        </div>
        <select v-model="selectedCategory" class="filter-select">
          <option value="">{{ $t('common.all') }}</option>
          <option v-for="cat in categories" :key="cat" :value="cat">
            {{ getCategoryName(cat) }}
          </option>
        </select>
      </div>
      <div class="view-toggle">
        <button
          :class="['view-btn', { active: viewMode === 'grid' }]"
          :title="$t('blockEditor.viewGrid')"
          @click="viewMode = 'grid'"
        >
          <span class="view-icon">&#9638;</span>
        </button>
        <button
          :class="['view-btn', { active: viewMode === 'table' }]"
          :title="$t('blockEditor.viewTable')"
          @click="viewMode = 'table'"
        >
          <span class="view-icon">&#9776;</span>
        </button>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="loading && blocks.length === 0" class="card loading-card">
      <p class="text-secondary">{{ $t('common.loading') }}</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="blocks.length === 0" class="card empty-card">
      <div class="empty-icon">&#128230;</div>
      <p class="empty-title">
        {{ $t('admin.blocks.noBlocks') }}
      </p>
      <p class="empty-description">
        {{ $t('admin.blocks.noBlocksDesc') }}
      </p>
      <button class="btn btn-primary" @click="openCreateWizard">
        {{ $t('admin.blocks.createFirst') }}
      </button>
    </div>

    <!-- Block Grid -->
    <div v-else-if="viewMode === 'grid'" class="block-grid">
      <BlockEditorBlockCard
        v-for="block in filteredBlocks"
        :key="block.id"
        :block="block"
        @edit="openEditModal"
        @delete="openDeleteModal"
        @duplicate="handleDuplicate"
        @view-versions="openVersionModal"
      />
    </div>

    <!-- Block Table -->
    <div v-else class="card">
      <table class="table">
        <thead>
          <tr>
            <th>{{ $t('admin.blocks.table.name') }}</th>
            <th>{{ $t('admin.blocks.table.category') }}</th>
            <th>{{ $t('admin.blocks.table.version') }}</th>
            <th>{{ $t('admin.blocks.table.enabled') }}</th>
            <th>{{ $t('admin.blocks.table.updatedAt') }}</th>
            <th>{{ $t('admin.blocks.table.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="block in filteredBlocks" :key="block.id">
            <td>
              <div class="block-name-cell">
                <div class="block-icon-small">{{ block.icon || '&#128230;' }}</div>
                <div>
                  <strong>{{ block.name }}</strong>
                  <code class="slug-code">{{ block.slug }}</code>
                  <div class="block-badges">
                    <span v-if="block.is_system" class="badge badge-system">System</span>
                    <span v-if="block.parent_block_id" class="badge badge-inherited">Inherited</span>
                  </div>
                </div>
              </div>
            </td>
            <td>
              <span class="badge badge-category">
                {{ getCategoryName(block.category) }}
              </span>
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
              <div class="action-buttons">
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

    <!-- Creation Wizard Modal -->
    <UiModal
      :show="showCreateWizard"
      :title="$t('blockEditor.wizard.title')"
      size="lg"
      @close="showCreateWizard = false"
    >
      <BlockEditorBlockCreationWizard
        :blocks="blocks"
        @select="handleWizardSelect"
        @close="showCreateWizard = false"
      />
    </UiModal>

    <!-- Block Form Modal (Create/Edit) -->
    <UiModal
      :show="showEditModal"
      :title="selectedBlock ? $t('admin.blocks.editBlock') + ': ' + selectedBlock.name : $t('admin.blocks.newBlock')"
      size="xl"
      @close="showEditModal = false"
    >
      <BlockEditorBlockForm
        :initial-data="selectedBlock ? blockToFormData(selectedBlock) : getInitialFormData()"
        :is-edit="!!selectedBlock"
        :blocks="blocks"
        :use-inheritance="creationType === 'inherit'"
        :parent-block="selectedParentBlock"
        @submit="handleFormSubmit"
        @cancel="showEditModal = false"
      />
    </UiModal>

    <!-- Delete Confirmation Modal -->
    <UiModal
      :show="showDeleteModal"
      :title="$t('admin.blocks.deleteBlock')"
      size="sm"
      @close="showDeleteModal = false"
    >
      <div class="delete-confirm">
        <p>{{ $t('admin.blocks.confirmDelete') }}</p>
        <p v-if="selectedBlock" class="delete-block-name">
          {{ selectedBlock.name }} <code>{{ selectedBlock.slug }}</code>
        </p>
      </div>

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
      <div v-if="loadingVersions" class="loading-state">
        {{ $t('common.loading') }}
      </div>

      <div v-else-if="versions.length === 0" class="empty-state">
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

.breadcrumb-separator {
  color: var(--color-text-secondary);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.page-title {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0;
}

.page-subtitle {
  margin-top: 0.25rem;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.btn-icon {
  margin-right: 0.5rem;
  font-weight: bold;
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
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
  gap: 1rem;
}

.filter-left {
  display: flex;
  gap: 0.75rem;
  flex: 1;
}

.search-input-wrapper {
  flex: 1;
  max-width: 300px;
}

.search-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-background);
  font-size: 0.875rem;
}

.search-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

.filter-select {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-background);
  font-size: 0.875rem;
  min-width: 150px;
}

.view-toggle {
  display: flex;
  gap: 0.25rem;
  background: var(--color-surface);
  padding: 0.25rem;
  border-radius: var(--radius);
}

.view-btn {
  padding: 0.375rem 0.5rem;
  border: none;
  background: transparent;
  border-radius: calc(var(--radius) - 2px);
  cursor: pointer;
  color: var(--color-text-secondary);
  transition: all 0.15s;
}

.view-btn:hover {
  color: var(--color-text);
}

.view-btn.active {
  background: var(--color-background);
  color: var(--color-primary);
}

.view-icon {
  font-size: 1rem;
}

.loading-card,
.empty-card {
  padding: 3rem;
  text-align: center;
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  opacity: 0.5;
}

.empty-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.empty-description {
  color: var(--color-text-secondary);
  margin-bottom: 1.5rem;
}

.block-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1rem;
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

.block-name-cell {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.block-icon-small {
  width: 2rem;
  height: 2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-surface);
  border-radius: 0.375rem;
  font-size: 1rem;
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

.block-badges {
  display: flex;
  gap: 0.25rem;
  margin-top: 0.25rem;
}

.badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.625rem;
  font-weight: 500;
  text-transform: uppercase;
}

.badge-category {
  background: rgba(99, 102, 241, 0.1);
  color: var(--color-primary);
}

.badge-system {
  background: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
}

.badge-inherited {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
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

.action-buttons {
  display: flex;
  gap: 0.5rem;
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

.delete-confirm {
  text-align: center;
  padding: 1rem 0;
}

.delete-block-name {
  font-weight: 600;
  margin-top: 0.5rem;
}

.delete-block-name code {
  font-weight: normal;
  margin-left: 0.5rem;
  color: var(--color-text-secondary);
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 2rem;
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
