<script setup lang="ts">
/**
 * EnvironmentVariablesModal.vue
 * 環境変数設定モーダル（3スコープ: 組織/プロジェクト/個人）
 *
 * 機能:
 * - タブ切り替えで3種類の変数を管理
 * - Form/JSON の切り替え編集
 * - 変数の参照方法ガイド表示
 */

const { t } = useI18n()
const toast = useToast()

const props = defineProps<{
  show: boolean
  projectId?: string
  projectVariables?: Record<string, unknown>
}>()

const emit = defineEmits<{
  close: []
  'update:project-variables': [variables: Record<string, unknown>]
}>()

// Tab state
type TabType = 'organization' | 'project' | 'personal'
const activeTab = ref<TabType>('organization')

// Composables
const tenantVars = useTenantVariables()
const userVars = useUserVariables()

// Loading state
const loading = computed(() => tenantVars.loading.value || userVars.loading.value)
const saving = ref(false)

// Local state for each scope
const orgVariables = ref<Record<string, unknown>>({})
const personalVariables = ref<Record<string, unknown>>({})
const localProjectVariables = ref<Record<string, unknown>>({})

// Editor mode
const editorMode = ref<'form' | 'json'>('form')

// Variable entries for form mode
interface VariableEntry {
  key: string
  value: string
  type: 'string' | 'number' | 'boolean' | 'json'
}

const entries = ref<VariableEntry[]>([])
const jsonContent = ref('')
const jsonError = ref<string | null>(null)

// Get current variables based on active tab
const currentVariables = computed((): Record<string, unknown> => {
  switch (activeTab.value) {
    case 'organization':
      return orgVariables.value
    case 'project':
      return localProjectVariables.value
    case 'personal':
      return personalVariables.value
    default:
      return {}
  }
})

// Initialize data when modal opens
watch(() => props.show, async (show) => {
  if (show) {
    activeTab.value = 'organization'
    await loadAllVariables()
  }
})

// Load all variables
async function loadAllVariables() {
  // Load each scope independently to handle partial failures gracefully
  const results = await Promise.allSettled([
    tenantVars.fetchVariables(),
    userVars.fetchVariables(),
  ])

  // Check for any failures and log them, but continue with available data
  const hasErrors = results.some(r => r.status === 'rejected')
  if (hasErrors) {
    console.warn('Some variables failed to load:', results)
  }

  // Use loaded values or empty objects for failed requests
  orgVariables.value = { ...tenantVars.variables.value }
  personalVariables.value = { ...userVars.variables.value }
  localProjectVariables.value = { ...(props.projectVariables || {}) }
  initEntriesFromCurrentTab()
}

// Initialize entries from current tab
function initEntriesFromCurrentTab() {
  const vars = currentVariables.value
  entries.value = Object.entries(vars).map(([key, value]) => ({
    key,
    value: typeof value === 'object' ? JSON.stringify(value) : String(value),
    type: detectType(value),
  }))
  jsonContent.value = JSON.stringify(vars, null, 2)
  jsonError.value = null
}

// Watch tab changes
watch(activeTab, () => {
  initEntriesFromCurrentTab()
})

// Detect value type
function detectType(value: unknown): VariableEntry['type'] {
  if (typeof value === 'boolean') return 'boolean'
  if (typeof value === 'number') return 'number'
  if (typeof value === 'object') return 'json'
  return 'string'
}

// Convert entry to proper value
function convertValue(entry: VariableEntry): unknown {
  switch (entry.type) {
    case 'number':
      return parseFloat(entry.value) || 0
    case 'boolean':
      return entry.value === 'true'
    case 'json':
      try {
        return JSON.parse(entry.value)
      } catch {
        return entry.value
      }
    default:
      return entry.value
  }
}

// Build variables object from entries
function buildVariablesFromEntries(): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const entry of entries.value) {
    if (entry.key.trim()) {
      result[entry.key.trim()] = convertValue(entry)
    }
  }
  return result
}

// Add new entry
function addEntry() {
  entries.value.push({
    key: '',
    value: '',
    type: 'string',
  })
}

// Remove entry
function removeEntry(index: number) {
  entries.value.splice(index, 1)
  updateLocalState()
}

// Update entry
function updateEntry(index: number, field: keyof VariableEntry, value: string) {
  entries.value[index][field] = value as never
  updateLocalState()
}

// Update local state based on current tab
function updateLocalState() {
  const vars = editorMode.value === 'form'
    ? buildVariablesFromEntries()
    : (() => {
        try {
          return JSON.parse(jsonContent.value)
        } catch {
          return currentVariables.value
        }
      })()

  switch (activeTab.value) {
    case 'organization':
      orgVariables.value = vars
      break
    case 'project':
      localProjectVariables.value = vars
      break
    case 'personal':
      personalVariables.value = vars
      break
  }
}

// Switch editor mode
function switchMode(mode: 'form' | 'json') {
  if (mode === 'json' && editorMode.value === 'form') {
    jsonContent.value = JSON.stringify(buildVariablesFromEntries(), null, 2)
  } else if (mode === 'form' && editorMode.value === 'json') {
    try {
      const parsed = JSON.parse(jsonContent.value)
      entries.value = Object.entries(parsed).map(([key, value]) => ({
        key,
        value: typeof value === 'object' ? JSON.stringify(value) : String(value),
        type: detectType(value),
      }))
      jsonError.value = null
    } catch {
      toast.error(t('variables.invalidJson'))
      return
    }
  }
  editorMode.value = mode
}

// Handle JSON input
function handleJsonInput(e: Event) {
  jsonContent.value = (e.target as HTMLTextAreaElement).value
  try {
    JSON.parse(jsonContent.value)
    jsonError.value = null
    updateLocalState()
  } catch (err) {
    jsonError.value = err instanceof Error ? err.message : 'Invalid JSON'
  }
}

// Save all changes
async function save() {
  saving.value = true
  try {
    // Ensure local state is up to date
    updateLocalState()

    // Save organization variables
    if (JSON.stringify(orgVariables.value) !== JSON.stringify(tenantVars.variables.value)) {
      await tenantVars.updateVariables(orgVariables.value)
    }

    // Save personal variables
    if (JSON.stringify(personalVariables.value) !== JSON.stringify(userVars.variables.value)) {
      await userVars.updateVariables(personalVariables.value)
    }

    // Emit project variables update
    if (JSON.stringify(localProjectVariables.value) !== JSON.stringify(props.projectVariables || {})) {
      emit('update:project-variables', localProjectVariables.value)
    }

    toast.success(t('variables.saveSuccess'))
    emit('close')
  } catch {
    toast.error(t('variables.saveError'))
  } finally {
    saving.value = false
  }
}

// Type options
const typeOptions = [
  { value: 'string', label: t('variables.types.string') },
  { value: 'number', label: t('variables.types.number') },
  { value: 'boolean', label: t('variables.types.boolean') },
  { value: 'json', label: t('variables.types.json') },
]

// Tab info
const tabs = computed(() => [
  { id: 'organization' as const, label: t('variables.tabs.organization'), prefix: '$org' },
  { id: 'project' as const, label: t('variables.tabs.project'), prefix: '$project' },
  { id: 'personal' as const, label: t('variables.tabs.personal'), prefix: '$personal' },
])

const currentTabPrefix = computed(() => {
  const tab = tabs.value.find(t => t.id === activeTab.value)
  return tab?.prefix || ''
})

// Example syntax for current tab
const currentExample = computed(() => {
  return `{{${currentTabPrefix.value}.${t('variables.exampleKey')}}}`
})

// Example syntax for each scope
function getScopeExample(prefix: string): string {
  return `{{${prefix}.KEY}}`
}
</script>

<template>
  <UiModal :show="show" :title="t('variables.modalTitle')" size="lg" @close="emit('close')">
    <div class="env-vars-modal">
      <!-- Tabs -->
      <div class="tabs">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab"
          :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id"
        >
          {{ tab.label }}
        </button>
        <div class="tab-spacer" />
        <!-- Mode switch -->
        <div class="mode-switch">
          <button
            class="mode-btn"
            :class="{ active: editorMode === 'form' }"
            @click="switchMode('form')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="8" y1="6" x2="21" y2="6" />
              <line x1="8" y1="12" x2="21" y2="12" />
              <line x1="8" y1="18" x2="21" y2="18" />
              <line x1="3" y1="6" x2="3.01" y2="6" />
              <line x1="3" y1="12" x2="3.01" y2="12" />
              <line x1="3" y1="18" x2="3.01" y2="18" />
            </svg>
            {{ t('variables.formMode') }}
          </button>
          <button
            class="mode-btn"
            :class="{ active: editorMode === 'json' }"
            @click="switchMode('json')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 18 22 12 16 6" />
              <polyline points="8 6 2 12 8 18" />
            </svg>
            {{ t('variables.jsonMode') }}
          </button>
        </div>
      </div>

      <!-- Loading state -->
      <div v-if="loading" class="loading-state">
        <div class="spinner" />
        <span>{{ t('common.loading') }}</span>
      </div>

      <!-- Content -->
      <div v-else class="content">
        <!-- Form Mode -->
        <div v-if="editorMode === 'form'" class="form-editor">
          <!-- Empty state -->
          <div v-if="entries.length === 0" class="empty-state">
            <svg xmlns="http://www.w3.org/2000/svg" width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2" />
              <line x1="3" y1="9" x2="21" y2="9" />
              <line x1="9" y1="21" x2="9" y2="9" />
            </svg>
            <p class="empty-title">{{ t('variables.noVariables') }}</p>
            <p class="empty-desc">{{ t('variables.noVariablesDesc') }}</p>
          </div>

          <!-- Entries list -->
          <div v-else class="entries-list">
            <div v-for="(entry, index) in entries" :key="index" class="entry-row">
              <input
                :value="entry.key"
                type="text"
                class="entry-key"
                :placeholder="t('variables.keyPlaceholder')"
                @input="updateEntry(index, 'key', ($event.target as HTMLInputElement).value)"
              >
              <select
                :value="entry.type"
                class="entry-type"
                @change="updateEntry(index, 'type', ($event.target as HTMLSelectElement).value)"
              >
                <option v-for="opt in typeOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </option>
              </select>
              <template v-if="entry.type === 'boolean'">
                <select
                  :value="entry.value"
                  class="entry-value"
                  @change="updateEntry(index, 'value', ($event.target as HTMLSelectElement).value)"
                >
                  <option value="true">true</option>
                  <option value="false">false</option>
                </select>
              </template>
              <template v-else-if="entry.type === 'json'">
                <textarea
                  :value="entry.value"
                  class="entry-value entry-value-json"
                  :placeholder="t('variables.jsonPlaceholder')"
                  rows="2"
                  @input="updateEntry(index, 'value', ($event.target as HTMLTextAreaElement).value)"
                />
              </template>
              <template v-else>
                <input
                  :value="entry.value"
                  :type="entry.type === 'number' ? 'number' : 'text'"
                  class="entry-value"
                  :placeholder="t('variables.valuePlaceholder')"
                  @input="updateEntry(index, 'value', ($event.target as HTMLInputElement).value)"
                >
              </template>
              <button
                type="button"
                class="btn-icon btn-danger"
                :title="t('common.delete')"
                @click="removeEntry(index)"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>
          </div>

          <!-- Add button -->
          <button
            type="button"
            class="btn-ghost add-btn"
            @click="addEntry"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
            {{ t('variables.addVariable') }}
          </button>
        </div>

        <!-- JSON Mode -->
        <div v-else class="json-editor">
          <textarea
            :value="jsonContent"
            class="json-textarea"
            :placeholder="t('variables.jsonPlaceholder')"
            rows="12"
            @input="handleJsonInput"
          />
          <div v-if="jsonError" class="json-error">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" />
              <line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
            {{ jsonError }}
          </div>
        </div>

        <!-- Usage hint -->
        <div class="usage-hint">
          <div class="hint-title">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="16" x2="12" y2="12" />
              <line x1="12" y1="8" x2="12.01" y2="8" />
            </svg>
            {{ t('variables.referenceTitle') }}
          </div>
          <div class="hint-examples">
            <code>{{ currentExample }}</code>
          </div>
          <div class="hint-all-scopes">
            <span v-for="tab in tabs" :key="tab.id" class="scope-example">
              <strong>{{ tab.label }}:</strong>
              <code>{{ getScopeExample(tab.prefix) }}</code>
            </span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <button class="btn btn-secondary" @click="emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button class="btn btn-primary" :disabled="saving || loading" @click="save">
        <span v-if="saving" class="spinner-small" />
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.env-vars-modal {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Tabs */
.tabs {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--color-border);
}

.tab {
  padding: 0.5rem 1rem;
  font-size: 0.875rem;
  font-weight: 500;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.tab:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.tab.active {
  background: var(--color-primary);
  color: white;
}

.tab-spacer {
  flex: 1;
}

/* Mode switch */
.mode-switch {
  display: flex;
  gap: 0.25rem;
  background: var(--color-bg);
  padding: 0.25rem;
  border-radius: 6px;
}

.mode-btn {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.375rem 0.5rem;
  font-size: 0.75rem;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.mode-btn:hover {
  background: var(--color-bg-hover);
}

.mode-btn.active {
  background: var(--color-surface);
  color: var(--color-text);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* Loading state */
.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 3rem;
  color: var(--color-text-secondary);
}

.spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.spinner-small {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Content */
.content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  min-height: 300px;
}

/* Form editor */
.form-editor {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 1rem;
  text-align: center;
  color: var(--color-text-secondary);
}

.empty-state svg {
  margin-bottom: 0.75rem;
  opacity: 0.5;
}

.empty-title {
  font-weight: 500;
  margin: 0 0 0.25rem;
  font-size: 0.875rem;
}

.empty-desc {
  font-size: 0.8125rem;
  margin: 0;
}

.entries-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.entry-row {
  display: grid;
  grid-template-columns: 1fr 90px 1fr 32px;
  gap: 0.5rem;
  align-items: start;
}

.entry-key,
.entry-type,
.entry-value {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg);
  color: var(--color-text);
  font-size: 0.8125rem;
}

.entry-key:focus,
.entry-type:focus,
.entry-value:focus {
  outline: none;
  border-color: var(--color-primary);
}

.entry-value-json {
  font-family: 'SF Mono', Monaco, monospace;
  resize: vertical;
  min-height: 60px;
}

.add-btn {
  align-self: flex-start;
  margin-top: 0.5rem;
}

/* JSON editor */
.json-editor {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.json-textarea {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg);
  color: var(--color-text);
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.8125rem;
  resize: vertical;
  min-height: 200px;
}

.json-textarea:focus {
  outline: none;
  border-color: var(--color-primary);
}

.json-error {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 6px;
  color: #ef4444;
  font-size: 0.8125rem;
}

/* Usage hint */
.usage-hint {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem;
  background: var(--color-bg);
  border-radius: 6px;
}

.hint-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.hint-examples code {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  background: var(--color-surface);
  border-radius: 4px;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.8125rem;
  color: var(--color-primary);
}

.hint-all-scopes {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 0.25rem;
}

.scope-example {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.scope-example code {
  padding: 0.125rem 0.375rem;
  background: var(--color-surface);
  border-radius: 3px;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.6875rem;
}

/* Buttons */
.btn-ghost {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border: none;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 500;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.btn-ghost:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.btn-icon:hover {
  background: var(--color-bg-hover);
}

.btn-danger {
  color: #ef4444;
}

.btn-danger:hover {
  background: rgba(239, 68, 68, 0.1);
}

/* Footer buttons */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-bg);
  color: var(--color-text);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-hover);
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  filter: brightness(1.1);
}
</style>
