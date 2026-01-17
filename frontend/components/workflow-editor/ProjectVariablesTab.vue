<script setup lang="ts">
/**
 * ProjectVariablesTab.vue
 * プロジェクトレベル共有変数の編集タブ
 *
 * 機能:
 * - Key-Valueフォームでの変数編集
 * - JSONエディタでの一括編集
 * - 変数の追加/削除
 */

const { t } = useI18n()
const toast = useToast()

const props = defineProps<{
  projectId: string
  variables?: Record<string, unknown>
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:variables', variables: Record<string, unknown>): void
}>()

// Editor mode
const editorMode = ref<'form' | 'json'>('form')

// Local state for form mode
interface VariableEntry {
  key: string
  value: string
  type: 'string' | 'number' | 'boolean' | 'json'
}

const entries = ref<VariableEntry[]>([])

// Local state for JSON mode
const jsonContent = ref('')
const jsonError = ref<string | null>(null)

// Initialize from props
function initFromProps() {
  const vars = props.variables || {}
  entries.value = Object.entries(vars).map(([key, value]) => ({
    key,
    value: typeof value === 'object' ? JSON.stringify(value) : String(value),
    type: detectType(value),
  }))
  jsonContent.value = JSON.stringify(vars, null, 2)
  jsonError.value = null
}

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
  emitChanges()
}

// Update entry
function updateEntry(index: number, field: keyof VariableEntry, value: string) {
  entries.value[index][field] = value as never
  emitChanges()
}

// Emit changes (debounced)
let debounceTimer: ReturnType<typeof setTimeout> | null = null
function emitChanges() {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    if (editorMode.value === 'form') {
      emit('update:variables', buildVariablesFromEntries())
    } else {
      try {
        const parsed = JSON.parse(jsonContent.value)
        jsonError.value = null
        emit('update:variables', parsed)
      } catch (e) {
        jsonError.value = e instanceof Error ? e.message : 'Invalid JSON'
      }
    }
  }, 300)
}

// Watch for prop changes
watch(() => props.variables, () => {
  initFromProps()
}, { immediate: true, deep: true })

// Switch editor mode
function switchMode(mode: 'form' | 'json') {
  if (mode === 'json' && editorMode.value === 'form') {
    // Sync form to JSON
    jsonContent.value = JSON.stringify(buildVariablesFromEntries(), null, 2)
  } else if (mode === 'form' && editorMode.value === 'json') {
    // Sync JSON to form
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
  emitChanges()
}

// Type options
const typeOptions = [
  { value: 'string', label: t('variables.types.string') },
  { value: 'number', label: t('variables.types.number') },
  { value: 'boolean', label: t('variables.types.boolean') },
  { value: 'json', label: t('variables.types.json') },
]
</script>

<template>
  <div class="variables-tab">
    <!-- Header -->
    <div class="tab-header">
      <h3 class="tab-title">{{ t('variables.title') }}</h3>
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
            :disabled="readonly"
            @input="updateEntry(index, 'key', ($event.target as HTMLInputElement).value)"
          >
          <select
            :value="entry.type"
            class="entry-type"
            :disabled="readonly"
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
              :disabled="readonly"
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
              :disabled="readonly"
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
              :disabled="readonly"
              @input="updateEntry(index, 'value', ($event.target as HTMLInputElement).value)"
            >
          </template>
          <button
            v-if="!readonly"
            type="button"
            class="btn btn-ghost btn-icon btn-danger"
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
        v-if="!readonly"
        type="button"
        class="btn btn-ghost add-btn"
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
        :disabled="readonly"
        rows="15"
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
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="16" x2="12" y2="12" />
        <line x1="12" y1="8" x2="12.01" y2="8" />
      </svg>
      <span>{{ t('variables.usageHint') }}</span>
    </div>
  </div>
</template>

<style scoped>
.variables-tab {
  padding: 1rem;
}

.tab-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.tab-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

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
  transition: background-color 0.15s, color 0.15s;
}

.mode-btn:hover {
  background: var(--color-bg-hover);
}

.mode-btn.active {
  background: var(--color-surface);
  color: var(--color-text);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
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
  padding: 2rem 1rem;
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

.usage-hint {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1rem;
  padding: 0.75rem;
  background: var(--color-bg);
  border-radius: 6px;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

/* Buttons */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  border: none;
  border-radius: 6px;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s, color 0.15s;
}

.btn-ghost {
  background: transparent;
  color: var(--color-text-secondary);
}

.btn-ghost:hover {
  background: var(--color-bg-hover);
  color: var(--color-text);
}

.btn-icon {
  padding: 0.375rem;
}

.btn-danger {
  color: #ef4444;
}

.btn-danger:hover {
  background: rgba(239, 68, 68, 0.1);
}
</style>
