<script setup lang="ts">
import type { BlockDefinition, BlockCategory } from '~/types/api'
import { categoryConfig } from '~/composables/useBlocks'

const { t } = useI18n()

definePageMeta({
  layout: 'default',
})

const blocksApi = useBlocks()

// State
const blocks = ref<BlockDefinition[]>([])
const loading = ref(true)
const message = ref<{ type: 'success' | 'error'; text: string } | null>(null)

// Modal state
const showCreateModal = ref(false)
const showDeleteModal = ref(false)
const selectedBlock = ref<BlockDefinition | null>(null)

// Filter state
const selectedCategory = ref<BlockCategory | ''>('')

// Form state
const formData = reactive({
  slug: '',
  name: '',
  description: '',
  category: 'integration' as BlockCategory,
  icon: '',
  executor_type: 'builtin',
  enabled: true,
})

// Computed
const filteredBlocks = computed(() => {
  if (!selectedCategory.value) return blocks.value
  return blocks.value.filter(b => b.category === selectedCategory.value)
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
    const response = await blocksApi.list()
    blocks.value = response.blocks || []
  } catch (err) {
    showMessageToast('error', t('errors.loadFailed'))
  } finally {
    loading.value = false
  }
}

function resetForm() {
  formData.slug = ''
  formData.name = ''
  formData.description = ''
  formData.category = 'integration'
  formData.icon = ''
  formData.executor_type = 'builtin'
  formData.enabled = true
}

function openCreateModal() {
  resetForm()
  selectedBlock.value = null
  showCreateModal.value = true
}

function openEditModal(block: BlockDefinition) {
  selectedBlock.value = block
  formData.slug = block.slug
  formData.name = block.name
  formData.description = block.description || ''
  formData.category = block.category
  formData.icon = block.icon || ''
  formData.executor_type = block.executor_type
  formData.enabled = block.enabled
  showCreateModal.value = true
}

function openDeleteModal(block: BlockDefinition) {
  selectedBlock.value = block
  showDeleteModal.value = true
}

async function submitForm() {
  try {
    if (selectedBlock.value) {
      // Update existing
      await blocksApi.update(selectedBlock.value.slug, {
        name: formData.name,
        description: formData.description || undefined,
        icon: formData.icon || undefined,
        enabled: formData.enabled,
      })
      showMessageToast('success', t('admin.blocks.messages.updated'))
    } else {
      // Create new
      await blocksApi.create({
        slug: formData.slug,
        name: formData.name,
        description: formData.description || undefined,
        category: formData.category,
        icon: formData.icon || undefined,
        executor_type: formData.executor_type,
      })
      showMessageToast('success', t('admin.blocks.messages.created'))
    }
    showCreateModal.value = false
    resetForm()
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
  } catch (err) {
    showMessageToast('error', t('admin.blocks.messages.deleteFailed'))
  }
}

function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleDateString()
}

function generateSlug() {
  if (!selectedBlock.value && formData.name) {
    formData.slug = formData.name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_+|_+$/g, '')
  }
}

function getCategoryName(category: BlockCategory): string {
  const config = categoryConfig[category]
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
              <span :class="['status-badge', block.enabled ? 'status-enabled' : 'status-disabled']">
                {{ block.enabled ? $t('common.enabled') : $t('common.disabled') }}
              </span>
            </td>
            <td>{{ formatDate(block.updated_at) }}</td>
            <td>
              <div class="flex gap-2">
                <button
                  class="btn btn-sm btn-secondary"
                  @click="openEditModal(block)"
                >
                  {{ $t('common.edit') }}
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

    <!-- Create/Edit Modal -->
    <UiModal
      :show="showCreateModal"
      :title="selectedBlock ? $t('admin.blocks.editBlock') : $t('admin.blocks.newBlock')"
      size="lg"
      @close="showCreateModal = false"
    >
      <form @submit.prevent="submitForm">
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.name') }} *</label>
            <input
              v-model="formData.name"
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
              v-model="formData.slug"
              type="text"
              class="form-input"
              placeholder="e.g., slack_message"
              required
              pattern="[a-z0-9_]+"
              :disabled="!!selectedBlock"
            />
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('admin.blocks.form.description') }}</label>
          <textarea
            v-model="formData.description"
            class="form-input"
            :placeholder="$t('admin.blocks.form.descriptionPlaceholder')"
            rows="2"
          />
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.category') }} *</label>
            <select
              v-model="formData.category"
              class="form-input"
              required
              :disabled="!!selectedBlock"
            >
              <option v-for="cat in categories" :key="cat" :value="cat">
                {{ getCategoryName(cat) }}
              </option>
            </select>
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.blocks.form.icon') }}</label>
            <input
              v-model="formData.icon"
              type="text"
              class="form-input"
              placeholder="e.g., message-circle"
            />
          </div>
        </div>

        <div class="form-group">
          <label class="form-label flex items-center gap-2">
            <input
              v-model="formData.enabled"
              type="checkbox"
              class="form-checkbox"
            />
            {{ $t('admin.blocks.table.enabled') }}
          </label>
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false">
          {{ $t('common.cancel') }}
        </button>
        <button class="btn btn-primary" :disabled="loading" @click="submitForm">
          {{ loading ? $t('common.saving') : $t('common.save') }}
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

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.form-checkbox {
  width: 1rem;
  height: 1rem;
  cursor: pointer;
}
</style>
