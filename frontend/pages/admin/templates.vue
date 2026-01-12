<script setup lang="ts">
import type { BlockTemplate, TemplateExecutorType, CreateBlockTemplateRequest } from '~/types/api'

const { t } = useI18n()

definePageMeta({
  layout: 'default',
})

const {
  templates,
  builtinTemplates,
  customTemplates,
  loading,
  fetchTemplates,
  createTemplate,
  updateTemplate,
  deleteTemplate,
} = useBlockTemplates()

// Modal state
const showCreateModal = ref(false)
const showDeleteModal = ref(false)
const showCodeModal = ref(false)
const selectedTemplate = ref<BlockTemplate | null>(null)

// Form state
const formData = reactive({
  slug: '',
  name: '',
  description: '',
  executor_type: 'javascript' as TemplateExecutorType,
  executor_code: '',
  config_schema: '{}',
})

// Message state
const message = ref<{ type: 'success' | 'error'; text: string } | null>(null)

function showMessage(type: 'success' | 'error', text: string) {
  message.value = { type, text }
  setTimeout(() => {
    message.value = null
  }, 3000)
}

// Fetch templates on mount
onMounted(() => {
  fetchTemplates()
})

// Reset form
function resetForm() {
  formData.slug = ''
  formData.name = ''
  formData.description = ''
  formData.executor_type = 'javascript'
  formData.executor_code = defaultJsCode
  formData.config_schema = '{}'
}

// Default JavaScript code
const defaultJsCode = `// Executor function for the block template
// Available context:
// - input: The input data from the previous step
// - config: The block configuration
// - context.http: HTTP client for API calls
// - context.credentials: Resolved credentials

async function execute(input, config, context) {
  // Your logic here
  return {
    result: input
  };
}
`

// Open create modal
function openCreateModal() {
  resetForm()
  formData.executor_code = defaultJsCode
  selectedTemplate.value = null
  showCreateModal.value = true
}

// Open edit modal
function openEditModal(template: BlockTemplate) {
  if (template.is_builtin) {
    showMessage('error', t('admin.templates.cannotEditBuiltin'))
    return
  }
  selectedTemplate.value = template
  formData.slug = template.slug
  formData.name = template.name
  formData.description = template.description || ''
  formData.executor_type = template.executor_type
  formData.executor_code = template.executor_code || ''
  formData.config_schema = JSON.stringify(template.config_schema || {}, null, 2)
  showCreateModal.value = true
}

// View code modal
function viewCode(template: BlockTemplate) {
  selectedTemplate.value = template
  showCodeModal.value = true
}

// Submit form
async function submitForm() {
  try {
    // Validate JSON
    let configSchema
    try {
      configSchema = JSON.parse(formData.config_schema)
    } catch {
      showMessage('error', t('admin.templates.invalidJson'))
      return
    }

    if (selectedTemplate.value) {
      // Update existing
      await updateTemplate(selectedTemplate.value.id, {
        slug: formData.slug,
        name: formData.name,
        description: formData.description || undefined,
        executor_type: formData.executor_type,
        executor_code: formData.executor_code || undefined,
        config_schema: configSchema,
      })
      showMessage('success', t('admin.templates.messages.updated'))
    } else {
      // Create new
      const request: CreateBlockTemplateRequest = {
        slug: formData.slug,
        name: formData.name,
        description: formData.description || undefined,
        executor_type: formData.executor_type,
        executor_code: formData.executor_code || undefined,
        config_schema: configSchema,
      }
      await createTemplate(request)
      showMessage('success', t('admin.templates.messages.created'))
    }
    showCreateModal.value = false
    resetForm()
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : t('errors.generic')
    showMessage('error', selectedTemplate.value ? t('admin.templates.messages.updateFailed') + ': ' + errorMessage : t('admin.templates.messages.createFailed') + ': ' + errorMessage)
  }
}

// Open delete confirmation
function openDeleteModal(template: BlockTemplate) {
  if (template.is_builtin) {
    showMessage('error', t('admin.templates.cannotDeleteBuiltin'))
    return
  }
  selectedTemplate.value = template
  showDeleteModal.value = true
}

// Confirm delete
async function confirmDelete() {
  if (!selectedTemplate.value) return

  try {
    await deleteTemplate(selectedTemplate.value.id)
    showMessage('success', t('admin.templates.messages.deleted'))
    showDeleteModal.value = false
    selectedTemplate.value = null
  } catch (err) {
    showMessage('error', t('admin.templates.messages.deleteFailed'))
  }
}

// Format date
function formatDate(date: string | undefined): string {
  if (!date) return '-'
  return new Date(date).toLocaleDateString()
}

// Generate slug from name
function generateSlug() {
  if (!selectedTemplate.value && formData.name) {
    formData.slug = formData.name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '_')
      .replace(/^_+|_+$/g, '')
  }
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
      <span>{{ $t('admin.templates.title') }}</span>
    </div>

    <div class="flex justify-between items-center mb-4">
      <div>
        <h1 style="font-size: 1.5rem; font-weight: 600;">
          {{ $t('admin.templates.title') }}
        </h1>
        <p class="text-secondary" style="margin-top: 0.25rem;">
          {{ $t('admin.templates.subtitle') }}
        </p>
      </div>
      <button class="btn btn-primary" @click="openCreateModal">
        {{ $t('admin.templates.newTemplate') }}
      </button>
    </div>

    <!-- Success/Error message -->
    <div
      v-if="message"
      :class="['card', message.type === 'success' ? 'bg-success' : 'bg-error']"
      style="padding: 0.75rem 1rem; margin-bottom: 1rem;"
    >
      {{ message.text }}
    </div>

    <!-- Loading state -->
    <div v-if="loading && templates.length === 0" class="card" style="padding: 2rem; text-align: center;">
      <p class="text-secondary">{{ $t('common.loading') }}</p>
    </div>

    <!-- Empty state -->
    <div v-else-if="templates.length === 0" class="card" style="padding: 3rem; text-align: center;">
      <p class="text-secondary" style="font-size: 1.125rem; margin-bottom: 0.5rem;">
        {{ $t('admin.templates.noTemplates') }}
      </p>
      <p class="text-secondary" style="margin-bottom: 1.5rem;">
        {{ $t('admin.templates.noTemplatesDesc') }}
      </p>
      <button class="btn btn-primary" @click="openCreateModal">
        {{ $t('admin.templates.createFirst') }}
      </button>
    </div>

    <template v-else>
      <!-- Builtin Templates Section -->
      <div v-if="builtinTemplates.length > 0" class="section mb-6">
        <h2 class="section-title">{{ $t('admin.templates.builtinSection') }}</h2>
        <p class="text-secondary section-desc">{{ $t('admin.templates.builtinDesc') }}</p>
        <div class="card">
          <table class="table">
            <thead>
              <tr>
                <th>{{ $t('admin.templates.table.name') }}</th>
                <th>{{ $t('admin.templates.table.slug') }}</th>
                <th>{{ $t('admin.templates.table.executorType') }}</th>
                <th>{{ $t('admin.templates.table.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="template in builtinTemplates" :key="template.id">
                <td>
                  <div>
                    <strong>{{ template.name }}</strong>
                    <span class="badge badge-builtin ml-2">{{ $t('admin.templates.builtin') }}</span>
                    <p v-if="template.description" class="text-secondary" style="font-size: 0.75rem; margin-top: 0.125rem;">
                      {{ template.description }}
                    </p>
                  </div>
                </td>
                <td><code>{{ template.slug }}</code></td>
                <td>{{ template.executor_type }}</td>
                <td>
                  <button
                    class="btn btn-sm btn-secondary"
                    @click="viewCode(template)"
                  >
                    {{ $t('admin.templates.viewCode') }}
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Custom Templates Section -->
      <div class="section">
        <h2 class="section-title">{{ $t('admin.templates.customSection') }}</h2>
        <p class="text-secondary section-desc">{{ $t('admin.templates.customDesc') }}</p>

        <div v-if="customTemplates.length === 0" class="card" style="padding: 2rem; text-align: center;">
          <p class="text-secondary">{{ $t('admin.templates.noCustomTemplates') }}</p>
          <button class="btn btn-primary mt-4" @click="openCreateModal">
            {{ $t('admin.templates.createFirst') }}
          </button>
        </div>

        <div v-else class="card">
          <table class="table">
            <thead>
              <tr>
                <th>{{ $t('admin.templates.table.name') }}</th>
                <th>{{ $t('admin.templates.table.slug') }}</th>
                <th>{{ $t('admin.templates.table.executorType') }}</th>
                <th>{{ $t('admin.templates.table.updatedAt') }}</th>
                <th>{{ $t('admin.templates.table.actions') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="template in customTemplates" :key="template.id">
                <td>
                  <div>
                    <strong>{{ template.name }}</strong>
                    <p v-if="template.description" class="text-secondary" style="font-size: 0.75rem; margin-top: 0.125rem;">
                      {{ template.description }}
                    </p>
                  </div>
                </td>
                <td><code>{{ template.slug }}</code></td>
                <td>{{ template.executor_type }}</td>
                <td>{{ formatDate(template.updated_at) }}</td>
                <td>
                  <div class="flex gap-2">
                    <button
                      class="btn btn-sm btn-secondary"
                      @click="viewCode(template)"
                    >
                      {{ $t('admin.templates.viewCode') }}
                    </button>
                    <button
                      class="btn btn-sm btn-secondary"
                      @click="openEditModal(template)"
                    >
                      {{ $t('common.edit') }}
                    </button>
                    <button
                      class="btn btn-sm btn-danger"
                      @click="openDeleteModal(template)"
                    >
                      {{ $t('common.delete') }}
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <!-- Create/Edit Modal -->
    <UiModal
      :show="showCreateModal"
      :title="selectedTemplate ? $t('admin.templates.editTemplate') : $t('admin.templates.newTemplate')"
      size="xl"
      @close="showCreateModal = false"
    >
      <form @submit.prevent="submitForm">
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ $t('admin.templates.form.name') }} *</label>
            <input
              v-model="formData.name"
              type="text"
              class="form-input"
              :placeholder="$t('admin.templates.form.namePlaceholder')"
              required
              @blur="generateSlug"
            />
          </div>
          <div class="form-group">
            <label class="form-label">{{ $t('admin.templates.form.slug') }} *</label>
            <input
              v-model="formData.slug"
              type="text"
              class="form-input"
              :placeholder="$t('admin.templates.form.slugPlaceholder')"
              required
              pattern="[a-z0-9_]+"
            />
            <p class="text-secondary" style="font-size: 0.75rem; margin-top: 0.25rem;">
              {{ $t('admin.templates.form.slugHint') }}
            </p>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('admin.templates.form.description') }}</label>
          <textarea
            v-model="formData.description"
            class="form-input"
            :placeholder="$t('admin.templates.form.descriptionPlaceholder')"
            rows="2"
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('admin.templates.form.executorType') }} *</label>
          <div class="executor-type-options">
            <label :class="['executor-type-option', { active: formData.executor_type === 'javascript' }]">
              <input
                v-model="formData.executor_type"
                type="radio"
                value="javascript"
                class="sr-only"
              />
              <strong>JavaScript</strong>
              <span class="text-secondary" style="font-size: 0.75rem;">{{ $t('admin.templates.executorTypes.javascriptDesc') }}</span>
            </label>
            <label :class="['executor-type-option', { active: formData.executor_type === 'builtin' }]">
              <input
                v-model="formData.executor_type"
                type="radio"
                value="builtin"
                class="sr-only"
              />
              <strong>Builtin</strong>
              <span class="text-secondary" style="font-size: 0.75rem;">{{ $t('admin.templates.executorTypes.builtinDesc') }}</span>
            </label>
          </div>
        </div>

        <div v-if="formData.executor_type === 'javascript'" class="form-group">
          <label class="form-label">{{ $t('admin.templates.form.executorCode') }} *</label>
          <textarea
            v-model="formData.executor_code"
            class="form-input code-input"
            :placeholder="$t('admin.templates.form.executorCodePlaceholder')"
            rows="15"
            required
          />
        </div>

        <div class="form-group">
          <label class="form-label">{{ $t('admin.templates.form.configSchema') }}</label>
          <textarea
            v-model="formData.config_schema"
            class="form-input code-input"
            placeholder="{}"
            rows="8"
          />
          <p class="text-secondary" style="font-size: 0.75rem; margin-top: 0.25rem;">
            {{ $t('admin.templates.form.configSchemaHint') }}
          </p>
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

    <!-- View Code Modal -->
    <UiModal
      :show="showCodeModal"
      :title="selectedTemplate?.name || ''"
      size="xl"
      @close="showCodeModal = false"
    >
      <div v-if="selectedTemplate">
        <div class="code-section mb-4">
          <h4 class="code-section-title">{{ $t('admin.templates.form.executorCode') }}</h4>
          <pre class="code-block">{{ selectedTemplate.executor_code || 'N/A (Builtin executor)' }}</pre>
        </div>
        <div class="code-section">
          <h4 class="code-section-title">{{ $t('admin.templates.form.configSchema') }}</h4>
          <pre class="code-block">{{ JSON.stringify(selectedTemplate.config_schema, null, 2) }}</pre>
        </div>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showCodeModal = false">
          {{ $t('common.close') }}
        </button>
      </template>
    </UiModal>

    <!-- Delete Confirmation Modal -->
    <UiModal
      :show="showDeleteModal"
      :title="$t('admin.templates.deleteTemplate')"
      size="sm"
      @close="showDeleteModal = false"
    >
      <p>{{ $t('admin.templates.confirmDelete') }}</p>

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

.section {
  margin-bottom: 2rem;
}

.section-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.section-desc {
  font-size: 0.875rem;
  margin-bottom: 1rem;
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

.badge {
  display: inline-block;
  padding: 0.125rem 0.5rem;
  border-radius: 9999px;
  font-size: 0.625rem;
  font-weight: 500;
  text-transform: uppercase;
}

.badge-builtin {
  background: rgba(99, 102, 241, 0.1);
  color: var(--color-primary);
}

.ml-2 {
  margin-left: 0.5rem;
}

.mt-4 {
  margin-top: 1rem;
}

code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  background: var(--color-background);
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
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

textarea.form-input {
  resize: vertical;
  min-height: 60px;
}

.code-input {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

.executor-type-options {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.75rem;
}

.executor-type-option {
  display: flex;
  flex-direction: column;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  cursor: pointer;
  transition: all 0.2s;
}

.executor-type-option:hover {
  border-color: var(--color-primary);
}

.executor-type-option.active {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.1);
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}

.bg-success {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

.bg-error {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.code-section {
  margin-bottom: 1.5rem;
}

.code-section-title {
  font-size: 0.875rem;
  font-weight: 600;
  margin-bottom: 0.5rem;
}

.code-block {
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  padding: 1rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.8125rem;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 400px;
  overflow-y: auto;
}
</style>
