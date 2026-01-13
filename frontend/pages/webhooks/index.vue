<script setup lang="ts">
import type { Webhook, CreateWebhookRequest, UpdateWebhookRequest, Workflow } from '~/types/api'

const { t } = useI18n()
const webhooksApi = useWebhooks()
const toast = useToast()
const { confirm } = useConfirm()

// State
const webhooks = ref<Webhook[]>([])
const workflows = ref<Workflow[]>([])
const loading = ref(false)
const showModal = ref(false)
const editingWebhook = ref<Webhook | null>(null)
const showSecretFor = ref<string | null>(null)

// Filters
const filterWorkflow = ref('')

// Form
const formData = ref({
  name: '',
  description: '',
  workflow_id: '',
  input_mapping: '{}',
})

// Fetch data
const fetchWebhooks = async () => {
  loading.value = true
  try {
    const result = await webhooksApi.list()
    webhooks.value = result
  } catch (e) {
    toast.error(t('errors.loadFailed'))
  } finally {
    loading.value = false
  }
}

const fetchWorkflows = async () => {
  try {
    const config = useRuntimeConfig()
    const baseUrl = config.public.apiBase || 'http://localhost:8080'
    const tenantId = import.meta.client
      ? (localStorage.getItem('tenant_id') || '00000000-0000-0000-0000-000000000001')
      : '00000000-0000-0000-0000-000000000001'
    const response = await fetch(`${baseUrl}/workflows`, {
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    })
    if (response.ok) {
      const result = await response.json()
      workflows.value = result.data
    }
  } catch (e) {
    console.error('Failed to fetch workflows:', e)
  }
}

onMounted(() => {
  fetchWebhooks()
  fetchWorkflows()
})

// Computed
const filteredWebhooks = computed(() => {
  return webhooks.value.filter((w) => {
    if (filterWorkflow.value && w.workflow_id !== filterWorkflow.value) return false
    return true
  })
})

// Methods
const getWorkflowName = (workflowId: string): string => {
  const workflow = workflows.value.find((w) => w.id === workflowId)
  return workflow?.name || workflowId
}

const formatDate = (date: string | undefined): string => {
  if (!date) return '-'
  return new Date(date).toLocaleString()
}

const getWebhookUrl = (webhook: Webhook): string => {
  return webhooksApi.getWebhookUrl(webhook)
}

const copyToClipboard = async (text: string, type: 'url' | 'secret') => {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(t(type === 'url' ? 'webhooks.messages.urlCopied' : 'webhooks.messages.secretCopied'))
  } catch (e) {
    console.error('Failed to copy:', e)
  }
}

const toggleSecret = (webhookId: string) => {
  showSecretFor.value = showSecretFor.value === webhookId ? null : webhookId
}

const openCreateModal = () => {
  editingWebhook.value = null
  formData.value = {
    name: '',
    description: '',
    workflow_id: '',
    input_mapping: '{}',
  }
  showModal.value = true
}

const openEditModal = (webhook: Webhook) => {
  editingWebhook.value = webhook
  formData.value = {
    name: webhook.name,
    description: webhook.description || '',
    workflow_id: webhook.workflow_id,
    input_mapping: webhook.input_mapping ? JSON.stringify(webhook.input_mapping, null, 2) : '{}',
  }
  showModal.value = true
}

const handleSubmit = async () => {
  try {
    let parsedMapping = {}
    try {
      parsedMapping = JSON.parse(formData.value.input_mapping)
    } catch {
      toast.error(t('webhooks.messages.invalidJson'))
      return
    }

    if (editingWebhook.value) {
      const updateData: UpdateWebhookRequest = {
        name: formData.value.name,
        description: formData.value.description || undefined,
        input_mapping: parsedMapping,
      }
      await webhooksApi.update(editingWebhook.value.id, updateData)
      toast.success(t('webhooks.messages.updated'))
    } else {
      const createData: CreateWebhookRequest = {
        workflow_id: formData.value.workflow_id,
        name: formData.value.name,
        description: formData.value.description || undefined,
        input_mapping: parsedMapping,
      }
      await webhooksApi.create(createData)
      toast.success(t('webhooks.messages.created'))
    }

    showModal.value = false
    fetchWebhooks()
  } catch (e) {
    toast.error(
      editingWebhook.value
        ? t('webhooks.messages.updateFailed')
        : t('webhooks.messages.createFailed'),
    )
  }
}

const handleDelete = async (webhook: Webhook) => {
  const confirmed = await confirm({
    title: t('webhooks.deleteTitle'),
    message: t('webhooks.confirmDelete'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    variant: 'danger',
  })
  if (!confirmed) return

  try {
    await webhooksApi.remove(webhook.id)
    toast.success(t('webhooks.messages.deleted'))
    fetchWebhooks()
  } catch (e) {
    toast.error(t('webhooks.messages.deleteFailed'))
  }
}

const handleEnable = async (webhook: Webhook) => {
  try {
    await webhooksApi.enable(webhook.id)
    toast.success(t('webhooks.messages.enabled'))
    fetchWebhooks()
  } catch (e) {
    toast.error(t('webhooks.messages.enableFailed'))
  }
}

const handleDisable = async (webhook: Webhook) => {
  try {
    await webhooksApi.disable(webhook.id)
    toast.success(t('webhooks.messages.disabled'))
    fetchWebhooks()
  } catch (e) {
    toast.error(t('webhooks.messages.disableFailed'))
  }
}

const handleRegenerateSecret = async (webhook: Webhook) => {
  const confirmed = await confirm({
    title: t('webhooks.regenerateTitle'),
    message: t('webhooks.confirmRegenerate'),
    confirmText: t('webhooks.regenerate'),
    cancelText: t('common.cancel'),
    variant: 'danger',
  })
  if (!confirmed) return

  try {
    await webhooksApi.regenerateSecret(webhook.id)
    toast.success(t('webhooks.messages.secretRegenerated'))
    fetchWebhooks()
  } catch (e) {
    toast.error(t('webhooks.messages.regenerateFailed'))
  }
}
</script>

<template>
  <div>
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ $t('webhooks.title') }}</h1>
        <p class="text-secondary">{{ $t('webhooks.subtitle') }}</p>
      </div>
      <button class="btn btn-primary" @click="openCreateModal">
        + {{ $t('webhooks.newWebhook') }}
      </button>
    </div>

    <!-- Filters -->
    <div class="filters-bar">
      <select v-model="filterWorkflow" class="filter-select">
        <option value="">{{ $t('webhooks.allWorkflows') }}</option>
        <option v-for="wf in workflows" :key="wf.id" :value="wf.id">
          {{ wf.name }}
        </option>
      </select>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      {{ $t('common.loading') }}
    </div>

    <!-- Empty State -->
    <div v-else-if="filteredWebhooks.length === 0" class="empty-state">
      <p>{{ $t('webhooks.noWebhooks') }}</p>
      <p class="text-secondary">{{ $t('webhooks.noWebhooksDesc') }}</p>
    </div>

    <!-- Webhooks List -->
    <div v-else class="webhooks-list">
      <div v-for="webhook in filteredWebhooks" :key="webhook.id" class="webhook-card">
        <div class="webhook-header">
          <div class="webhook-info">
            <h3>{{ webhook.name }}</h3>
            <p v-if="webhook.description" class="text-secondary">{{ webhook.description }}</p>
          </div>
          <div class="webhook-status">
            <span :class="['status-indicator', webhook.enabled ? 'enabled' : 'disabled']">
              {{ webhook.enabled ? $t('common.enabled') : $t('common.disabled') }}
            </span>
          </div>
        </div>

        <div class="webhook-details">
          <div class="detail-row">
            <span class="detail-label">{{ $t('webhooks.table.workflow') }}:</span>
            <NuxtLink :to="`/workflows/${webhook.workflow_id}`" class="workflow-link">
              {{ getWorkflowName(webhook.workflow_id) }}
            </NuxtLink>
          </div>

          <div class="detail-row">
            <span class="detail-label">{{ $t('webhooks.webhookUrl') }}:</span>
            <div class="url-container">
              <code class="url-code">{{ getWebhookUrl(webhook) }}</code>
              <button class="copy-btn" @click="copyToClipboard(getWebhookUrl(webhook), 'url')">
                {{ $t('webhooks.copyUrl') }}
              </button>
            </div>
          </div>

          <div class="detail-row">
            <span class="detail-label">{{ $t('webhooks.secret') }}:</span>
            <div class="secret-container">
              <code v-if="showSecretFor === webhook.id" class="secret-code">{{ webhook.secret }}</code>
              <code v-else class="secret-code">••••••••••••</code>
              <button class="copy-btn" @click="toggleSecret(webhook.id)">
                {{ showSecretFor === webhook.id ? 'Hide' : 'Show' }}
              </button>
              <button class="copy-btn" @click="copyToClipboard(webhook.secret, 'secret')">
                {{ $t('webhooks.copySecret') }}
              </button>
              <button class="copy-btn danger" @click="handleRegenerateSecret(webhook)">
                {{ $t('webhooks.regenerateSecret') }}
              </button>
            </div>
          </div>

          <div class="detail-row">
            <span class="detail-label">{{ $t('webhooks.triggerCount') }}:</span>
            <span>{{ webhook.trigger_count }}</span>
          </div>

          <div class="detail-row">
            <span class="detail-label">{{ $t('webhooks.lastTriggered') }}:</span>
            <span>{{ formatDate(webhook.last_triggered_at) }}</span>
          </div>
        </div>

        <div class="webhook-actions">
          <button v-if="webhook.enabled" class="btn btn-sm btn-secondary" @click="handleDisable(webhook)">
            {{ $t('webhooks.actions.disable') }}
          </button>
          <button v-else class="btn btn-sm btn-secondary" @click="handleEnable(webhook)">
            {{ $t('webhooks.actions.enable') }}
          </button>
          <button class="btn btn-sm btn-secondary" @click="openEditModal(webhook)">
            {{ $t('common.edit') }}
          </button>
          <button class="btn btn-sm btn-danger" @click="handleDelete(webhook)">
            {{ $t('common.delete') }}
          </button>
        </div>

        <!-- Usage Example -->
        <details class="usage-section">
          <summary>{{ $t('webhooks.usage.title') }}</summary>
          <div class="usage-content">
            <p class="usage-method">{{ $t('webhooks.usage.method') }}</p>
            <p class="usage-headers">{{ $t('webhooks.usage.headers') }}:</p>
            <pre class="usage-code">Content-Type: application/json
X-Webhook-Secret: {{ webhook.secret }}</pre>
            <p class="usage-body">{{ $t('webhooks.usage.body') }}:</p>
            <pre class="usage-code">{
  "your_data": "here"
}</pre>
            <p class="usage-curl">{{ $t('webhooks.usage.curlExample') }}:</p>
            <pre class="usage-code">curl -X POST "{{ getWebhookUrl(webhook) }}" \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Secret: {{ webhook.secret }}" \
  -d '{"your_data": "here"}'</pre>
          </div>
        </details>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h2>{{ editingWebhook ? $t('webhooks.editWebhook') : $t('webhooks.newWebhook') }}</h2>
          <button class="modal-close" @click="showModal = false">&times;</button>
        </div>
        <form @submit.prevent="handleSubmit">
          <div class="form-group">
            <label>{{ $t('webhooks.form.name') }}</label>
            <input
              v-model="formData.name"
              type="text"
              class="form-input"
              :placeholder="$t('webhooks.form.namePlaceholder')"
              required
            />
          </div>

          <div class="form-group">
            <label>{{ $t('webhooks.form.description') }}</label>
            <textarea
              v-model="formData.description"
              class="form-input"
              :placeholder="$t('webhooks.form.descriptionPlaceholder')"
              rows="2"
            />
          </div>

          <div v-if="!editingWebhook" class="form-group">
            <label>{{ $t('webhooks.form.workflow') }}</label>
            <select v-model="formData.workflow_id" class="form-input" required>
              <option value="">{{ $t('webhooks.form.workflowPlaceholder') }}</option>
              <option v-for="wf in workflows" :key="wf.id" :value="wf.id">
                {{ wf.name }}
              </option>
            </select>
          </div>

          <div class="form-group">
            <label>{{ $t('webhooks.form.inputMapping') }}</label>
            <textarea
              v-model="formData.input_mapping"
              class="form-input code-input"
              :placeholder="$t('webhooks.form.inputMappingPlaceholder')"
              rows="4"
            />
            <p class="form-hint">{{ $t('webhooks.form.inputMappingHint') }}</p>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showModal = false">
              {{ $t('common.cancel') }}
            </button>
            <button type="submit" class="btn btn-primary">
              {{ editingWebhook ? $t('common.save') : $t('common.create') }}
            </button>
          </div>
        </form>
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
}

.filter-select {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  min-width: 160px;
}

.loading-state,
.empty-state {
  text-align: center;
  padding: 3rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}

.webhooks-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.webhook-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 1.5rem;
}

.webhook-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.webhook-info h3 {
  margin: 0;
  font-size: 1.125rem;
}

.webhook-info p {
  margin: 0.25rem 0 0;
  font-size: 0.875rem;
}

.status-indicator {
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 500;
}

.status-indicator.enabled {
  background: #dcfce7;
  color: #166534;
}

.status-indicator.disabled {
  background: #f3f4f6;
  color: #6b7280;
}

.webhook-details {
  margin-bottom: 1rem;
}

.detail-row {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
}

.detail-label {
  color: var(--color-text-secondary);
  min-width: 120px;
  flex-shrink: 0;
}

.workflow-link {
  color: var(--color-primary);
  text-decoration: none;
}

.workflow-link:hover {
  text-decoration: underline;
}

.url-container,
.secret-container {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.url-code,
.secret-code {
  background: var(--color-background);
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  word-break: break-all;
}

.copy-btn {
  padding: 0.25rem 0.5rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  cursor: pointer;
}

.copy-btn:hover {
  background: var(--color-surface);
}

.copy-btn.danger {
  color: #ef4444;
  border-color: #ef4444;
}

.webhook-actions {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.usage-section {
  background: var(--color-background);
  border-radius: var(--radius);
  padding: 0.5rem 1rem;
}

.usage-section summary {
  cursor: pointer;
  font-weight: 500;
  font-size: 0.875rem;
}

.usage-content {
  padding-top: 0.5rem;
}

.usage-method,
.usage-headers,
.usage-body,
.usage-curl {
  margin: 0.5rem 0 0.25rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.usage-code {
  background: var(--color-surface);
  padding: 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  overflow-x: auto;
  margin: 0;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 0.875rem;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-secondary {
  background: var(--color-background);
  border: 1px solid var(--color-border);
}

.btn-danger {
  background: #ef4444;
  color: white;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
}

/* Modal */
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
  z-index: 1000;
}

.modal-content {
  background: var(--color-surface);
  border-radius: var(--radius);
  width: 90%;
  max-width: 600px;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.modal-header h2 {
  margin: 0;
  font-size: 1.25rem;
}

.modal-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: var(--color-text-secondary);
}

.modal-content form {
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.form-input {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-background);
}

.form-hint {
  margin-top: 0.25rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.code-input {
  font-family: monospace;
  font-size: 0.875rem;
}

.modal-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}
</style>
