<script setup lang="ts">
import type { StepTestResult } from '~/composables/test'

type ViewMode = 'table' | 'json' | 'schema'

const props = defineProps<{
  result: StepTestResult | null
  executing?: boolean
  error?: string | null
  stepName?: string
}>()

const emit = defineEmits<{
  'pin-output': [output: unknown]
}>()

const { t } = useI18n()

// View mode state
const viewMode = ref<ViewMode>('table')

// Computed: Status info
const status = computed(() => {
  if (props.executing) return 'running'
  if (!props.result) return null
  return props.result.stepRun.status
})

const statusColor = computed(() => {
  const colors: Record<string, string> = {
    running: '#3b82f6',
    completed: '#22c55e',
    failed: '#ef4444',
    pending: '#f59e0b',
    skipped: '#94a3b8',
  }
  return status.value ? colors[status.value] || '#94a3b8' : '#94a3b8'
})

const statusLabel = computed(() => {
  if (props.executing) return t('test.executing')
  if (!props.result) return ''
  return props.result.stepRun.status === 'completed'
    ? t('test.success')
    : t('test.failed')
})

// Computed: Duration display
const durationDisplay = computed(() => {
  if (!props.result) return ''
  const ms = props.result.duration
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
})

// Computed: Timestamp display
const timestampDisplay = computed(() => {
  if (!props.result) return ''
  const date = new Date(props.result.executedAt)
  return date.toLocaleString('ja-JP', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
})

// Computed: Output data
const outputData = computed(() => {
  return props.result?.stepRun.output || null
})

// Computed: Table rows for table view
const tableRows = computed<Array<{ key: string; value: string; type: string }>>(() => {
  const data = outputData.value
  if (!data || typeof data !== 'object') return []

  const rows: Array<{ key: string; value: string; type: string }> = []

  function flatten(obj: object, prefix = ''): void {
    for (const [key, value] of Object.entries(obj)) {
      const fullKey = prefix ? `${prefix}.${key}` : key
      if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
        flatten(value as object, fullKey)
      } else {
        rows.push({
          key: fullKey,
          value: formatValue(value),
          type: getValueType(value),
        })
      }
    }
  }

  flatten(data as object)
  return rows.slice(0, 20) // Limit to 20 rows
})

// Computed: Schema fields
const schemaFields = computed<Array<{ key: string; type: string; nullable: boolean }>>(() => {
  const data = outputData.value
  if (!data || typeof data !== 'object') return []

  const fields: Array<{ key: string; type: string; nullable: boolean }> = []

  function extractSchema(obj: object, prefix = ''): void {
    for (const [key, value] of Object.entries(obj)) {
      const fullKey = prefix ? `${prefix}.${key}` : key
      if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
        extractSchema(value as object, fullKey)
      } else {
        fields.push({
          key: fullKey,
          type: getValueType(value),
          nullable: value === null,
        })
      }
    }
  }

  extractSchema(data as object)
  return fields
})

// Computed: Has more rows
const hasMoreRows = computed(() => {
  const data = outputData.value
  if (!data || typeof data !== 'object') return false
  let count = 0
  function countFields(obj: object): void {
    for (const [_, value] of Object.entries(obj)) {
      if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
        countFields(value as object)
      } else {
        count++
      }
    }
  }
  countFields(data as object)
  return count > 20
})

// Helpers
function formatValue(value: unknown): string {
  if (value === null) return 'null'
  if (value === undefined) return 'undefined'
  if (typeof value === 'string') return value.length > 50 ? `${value.substring(0, 50)}...` : value
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function getValueType(value: unknown): string {
  if (value === null) return 'null'
  if (Array.isArray(value)) return 'array'
  return typeof value
}

function getTypeClass(type: string): string {
  const classes: Record<string, string> = {
    string: 'type-string',
    number: 'type-number',
    boolean: 'type-boolean',
    object: 'type-object',
    array: 'type-array',
    null: 'type-null',
  }
  return classes[type] || 'type-any'
}

function handlePinOutput() {
  if (outputData.value) {
    emit('pin-output', outputData.value)
  }
}
</script>

<template>
  <div class="test-result-panel">
    <!-- Header with status -->
    <div v-if="result || executing" class="result-header">
      <div class="status-info">
        <!-- Status badge -->
        <div class="status-badge" :style="{ backgroundColor: statusColor }">
          <svg
            v-if="executing"
            class="spin"
            xmlns="http://www.w3.org/2000/svg"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          <svg
            v-else-if="status === 'completed'"
            xmlns="http://www.w3.org/2000/svg"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          <svg
            v-else-if="status === 'failed'"
            xmlns="http://www.w3.org/2000/svg"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
          <span>{{ statusLabel }}</span>
        </div>

        <!-- Duration & timestamp -->
        <span v-if="result && !executing" class="result-meta">
          ({{ durationDisplay }})
          <span class="timestamp">{{ timestampDisplay }}</span>
        </span>
      </div>

      <!-- View mode tabs -->
      <div v-if="result && !executing && outputData" class="view-tabs">
        <button
          class="view-tab"
          :class="{ active: viewMode === 'table' }"
          @click="viewMode = 'table'"
        >
          {{ t('test.viewMode.table') }}
        </button>
        <button
          class="view-tab"
          :class="{ active: viewMode === 'json' }"
          @click="viewMode = 'json'"
        >
          {{ t('test.viewMode.json') }}
        </button>
        <button
          class="view-tab"
          :class="{ active: viewMode === 'schema' }"
          @click="viewMode = 'schema'"
        >
          {{ t('test.viewMode.schema') }}
        </button>
      </div>
    </div>

    <!-- Running state -->
    <div v-if="executing" class="result-running">
      <div class="spinner" />
      <span>{{ t('test.executing') }}</span>
    </div>

    <!-- Error display -->
    <div v-else-if="error || (result && result.stepRun.status === 'failed')" class="result-error">
      <div class="error-icon">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
      </div>
      <span class="error-message">{{ error || result?.stepRun.error }}</span>
    </div>

    <!-- Success: Content display -->
    <div v-else-if="result && outputData" class="result-content">
      <!-- Table view -->
      <div v-if="viewMode === 'table'" class="table-view">
        <table class="result-table">
          <tbody>
            <tr v-for="row in tableRows" :key="row.key">
              <td class="key-cell"><code>{{ row.key }}</code></td>
              <td class="value-cell">
                <span :class="['value', getTypeClass(row.type)]">{{ row.value }}</span>
              </td>
            </tr>
          </tbody>
        </table>
        <div v-if="hasMoreRows" class="more-rows">
          {{ t('test.moreRows') }}
        </div>
      </div>

      <!-- JSON view -->
      <div v-else-if="viewMode === 'json'" class="json-view">
        <pre class="json-content">{{ JSON.stringify(outputData, null, 2) }}</pre>
      </div>

      <!-- Schema view -->
      <div v-else-if="viewMode === 'schema'" class="schema-view">
        <div v-for="field in schemaFields" :key="field.key" class="schema-field">
          <code class="field-name">{{ field.key }}</code>
          <span :class="['field-type', getTypeClass(field.type)]">{{ field.type }}</span>
          <span v-if="field.nullable" class="nullable">?</span>
        </div>
      </div>

      <!-- Pin button -->
      <button class="pin-button" @click="handlePinOutput">
        <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="m12 17 2 5 2-5"/>
          <path d="m6 12 1.09 4.36a2 2 0 0 0 1.94 1.64h5.94a2 2 0 0 0 1.94-1.64L18 12"/>
          <path d="M6 8a6 6 0 1 1 12 0c0 5-6 6-6 12h0c0-6-6-7-6-12Z"/>
        </svg>
        {{ t('test.pinData.pin') }}
      </button>
    </div>

    <!-- No output -->
    <div v-else-if="result && !outputData" class="no-output">
      <span>{{ t('test.noOutput') }}</span>
    </div>

    <!-- Empty state -->
    <div v-else class="empty-state">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <polygon points="5 3 19 12 5 21 5 3"/>
      </svg>
      <span>{{ t('test.noResult') }}</span>
    </div>
  </div>
</template>

<style scoped>
.test-result-panel {
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  overflow: hidden;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.625rem 0.75rem;
  background: #f8fafc;
  border-bottom: 1px solid var(--color-border);
  gap: 0.5rem;
  flex-wrap: wrap;
}

.status-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.625rem;
  border-radius: 12px;
  font-size: 0.6875rem;
  font-weight: 600;
  color: white;
}

.result-meta {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}

.timestamp {
  margin-left: 0.5rem;
  opacity: 0.8;
}

.view-tabs {
  display: flex;
  gap: 0.25rem;
}

.view-tab {
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.view-tab:hover {
  color: var(--color-text);
  background: white;
}

.view-tab.active {
  color: var(--color-primary);
  background: white;
  border-color: var(--color-primary);
}

.result-running {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 1.5rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.result-error {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.75rem;
  background: #fef2f2;
  border-bottom: 1px solid #fecaca;
}

.error-icon {
  color: #dc2626;
  flex-shrink: 0;
}

.error-message {
  font-size: 0.75rem;
  color: #dc2626;
  word-break: break-word;
}

.result-content {
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.table-view {
  max-height: 200px;
  overflow-y: auto;
}

.result-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.6875rem;
}

.result-table td {
  padding: 0.375rem 0.5rem;
  border-bottom: 1px solid var(--color-border);
}

.key-cell {
  width: 40%;
  color: var(--color-text-secondary);
}

.key-cell code {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.6875rem;
  background: #f1f5f9;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.value-cell {
  color: var(--color-text);
}

.value {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.6875rem;
}

.more-rows {
  padding: 0.5rem;
  text-align: center;
  font-size: 0.625rem;
  color: var(--color-text-secondary);
  background: #f8fafc;
  border-radius: 4px;
  margin-top: 0.5rem;
}

.json-view {
  max-height: 200px;
  overflow-y: auto;
}

.json-content {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.6875rem;
  line-height: 1.5;
  margin: 0;
  padding: 0.5rem;
  background: #f8fafc;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-word;
}

.schema-view {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.schema-field {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.5rem;
  background: #f8fafc;
  border-radius: 4px;
}

.field-name {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.6875rem;
  color: var(--color-text);
}

.field-type {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.nullable {
  font-size: 0.625rem;
  color: #f59e0b;
  font-weight: 600;
}

.type-string { background: #dcfce7; color: #16a34a; }
.type-number { background: #dbeafe; color: #2563eb; }
.type-boolean { background: #fef3c7; color: #d97706; }
.type-object { background: #fce7f3; color: #db2777; }
.type-array { background: #f3e8ff; color: #9333ea; }
.type-null { background: #f3f4f6; color: #6b7280; }
.type-any { background: #f3f4f6; color: #6b7280; }

.pin-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  padding: 0.5rem;
  font-size: 0.6875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  margin-top: 0.5rem;
}

.pin-button:hover {
  color: var(--color-primary);
  border-color: var(--color-primary);
  background: #eff6ff;
}

.no-output {
  padding: 1rem;
  text-align: center;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 1.5rem;
  color: var(--color-text-secondary);
}

.empty-state svg {
  opacity: 0.5;
}

.empty-state span {
  font-size: 0.75rem;
}
</style>
