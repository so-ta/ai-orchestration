<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { JSONSchemaProperty, FieldOverride } from '../types/config-schema'

const props = defineProps<{
  name: string
  property: JSONSchemaProperty
  modelValue: object | undefined
  override?: FieldOverride
  error?: string
  disabled?: boolean
  required?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: object): void
  (e: 'blur'): void
}>()

// Internal string representation of JSON
const jsonText = ref('')
const parseError = ref<string | null>(null)

// Get display title
const title = computed(() => {
  return props.property.title || props.name
})

// Get description
const description = computed(() => {
  return props.property.description
})

// Initialize JSON text from modelValue
function initJsonText() {
  if (props.modelValue !== undefined && props.modelValue !== null) {
    try {
      jsonText.value = JSON.stringify(props.modelValue, null, 2)
      parseError.value = null
    } catch {
      jsonText.value = ''
      parseError.value = null
    }
  } else {
    jsonText.value = ''
    parseError.value = null
  }
}

// Watch for external changes to modelValue
watch(() => props.modelValue, (newValue, oldValue) => {
  // Only update if the value is actually different
  // (avoid loop when we emit changes)
  if (JSON.stringify(newValue) !== JSON.stringify(oldValue)) {
    initJsonText()
  }
}, { immediate: true, deep: true })

// Handle text input
function handleInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  jsonText.value = target.value

  // Try to parse and emit
  if (target.value.trim() === '') {
    parseError.value = null
    emit('update:modelValue', {})
    return
  }

  try {
    const parsed = JSON.parse(target.value)
    parseError.value = null
    emit('update:modelValue', parsed)
  } catch (e) {
    parseError.value = (e as Error).message
  }
}

// Handle blur
function handleBlur() {
  emit('blur')
}

// Format JSON
function formatJson() {
  if (!jsonText.value.trim()) return

  try {
    const parsed = JSON.parse(jsonText.value)
    jsonText.value = JSON.stringify(parsed, null, 2)
    parseError.value = null
    emit('update:modelValue', parsed)
  } catch (e) {
    parseError.value = (e as Error).message
  }
}

// Generate sample schema
function generateSample() {
  const sample = {
    type: 'object',
    properties: {
      field1: { type: 'string', title: 'フィールド1', description: '説明' },
      field2: { type: 'number', title: 'フィールド2' }
    },
    required: ['field1']
  }
  jsonText.value = JSON.stringify(sample, null, 2)
  parseError.value = null
  emit('update:modelValue', sample)
}

// Number of rows
const rows = computed(() => {
  const lineCount = (jsonText.value || '').split('\n').length
  return Math.min(Math.max(lineCount + 2, 6), 20)
})

// Highlighted JSON
const highlightedJson = computed(() => {
  const json = jsonText.value || ''
  if (!json) return ''

  // Escape HTML
  let html = json
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // Highlight JSON syntax
  // Keys (in quotes followed by colon)
  html = html.replace(/"([^"]+)"(\s*:)/g, '<span class="json-key">"$1"</span>$2')
  // String values (in quotes not followed by colon)
  html = html.replace(/:\s*"([^"]*)"/g, ': <span class="json-string">"$1"</span>')
  // Numbers
  html = html.replace(/:\s*(\d+\.?\d*)/g, ': <span class="json-number">$1</span>')
  // Booleans and null
  html = html.replace(/:\s*(true|false|null)/g, ': <span class="json-boolean">$1</span>')
  // Array brackets
  html = html.replace(/(\[|\])/g, '<span class="json-bracket">$1</span>')
  // Object braces
  html = html.replace(/(\{|\})/g, '<span class="json-brace">$1</span>')

  return html
})
</script>

<template>
  <div class="json-widget">
    <div class="json-widget-header">
      <label :for="name" class="json-widget-label">
        {{ title }}
        <span v-if="required" class="field-required">*</span>
      </label>
      <div class="header-actions">
        <button
          type="button"
          class="action-button"
          :disabled="disabled"
          @click="generateSample"
          title="サンプルを挿入"
        >
          サンプル
        </button>
        <button
          type="button"
          class="action-button"
          :disabled="disabled || !jsonText"
          @click="formatJson"
          title="JSONを整形"
        >
          整形
        </button>
      </div>
    </div>

    <p v-if="description" class="json-widget-description">
      {{ description }}
    </p>

    <div class="json-widget-container" :class="{ 'has-error': !!parseError }">
      <div class="json-editor">
        <div class="line-numbers">
          <span v-for="i in jsonText.split('\n').length" :key="i">{{ i }}</span>
        </div>
        <div class="code-area">
          <!-- Highlighted code display -->
          <pre class="code-highlight" v-html="highlightedJson"></pre>
          <!-- Actual textarea for input -->
          <textarea
            :id="name"
            :value="jsonText"
            :rows="rows"
            :disabled="disabled"
            class="code-input"
            spellcheck="false"
            placeholder='{"type": "object", "properties": {...}}'
            @input="handleInput"
            @blur="handleBlur"
          ></textarea>
        </div>
      </div>
      <div class="json-footer">
        <span v-if="parseError" class="parse-error">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
          {{ parseError }}
        </span>
        <span v-else class="json-valid">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          JSON
        </span>
      </div>
    </div>

    <p v-if="error" class="json-widget-error">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.json-widget {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.json-widget-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.json-widget-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text);
}

.field-required {
  color: var(--color-error, #ef4444);
  margin-left: 2px;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.action-button {
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
  font-weight: 500;
  color: var(--color-primary, #3b82f6);
  background: transparent;
  border: 1px solid var(--color-primary, #3b82f6);
  border-radius: 3px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.action-button:hover:not(:disabled) {
  background: var(--color-primary, #3b82f6);
  color: white;
}

.action-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.json-widget-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.4;
}

.json-widget-container {
  border: 1px solid var(--color-border);
  border-radius: 6px;
  overflow: hidden;
  background: #1e1e1e;
}

.json-widget-container.has-error {
  border-color: var(--color-error, #ef4444);
}

.json-editor {
  display: flex;
  min-height: 120px;
  max-height: 300px;
  overflow: auto;
}

.line-numbers {
  display: flex;
  flex-direction: column;
  padding: 0.75rem 0.5rem;
  background: #2d2d2d;
  color: #6e7681;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  text-align: right;
  user-select: none;
  min-width: 2.5rem;
  border-right: 1px solid #404040;
}

.code-area {
  position: relative;
  flex: 1;
  overflow: auto;
}

.code-highlight,
.code-input {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0.75rem;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
  overflow: hidden;
}

.code-highlight {
  color: #d4d4d4;
  pointer-events: none;
  z-index: 1;
}

.code-input {
  background: transparent;
  color: transparent;
  caret-color: #d4d4d4;
  border: none;
  outline: none;
  resize: none;
  z-index: 2;
}

.code-input:focus {
  outline: none;
}

.code-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.code-input::placeholder {
  color: #6e7681;
}

.json-footer {
  display: flex;
  justify-content: flex-end;
  padding: 0.375rem 0.75rem;
  background: #2d2d2d;
  border-top: 1px solid #404040;
}

.parse-error {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.625rem;
  color: #f87171;
}

.json-valid {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.625rem;
  color: #4ade80;
}

.json-widget-error {
  font-size: 0.6875rem;
  color: var(--color-error);
  margin: 0;
}

/* JSON syntax highlighting */
:deep(.json-key) {
  color: #9cdcfe;
}

:deep(.json-string) {
  color: #ce9178;
}

:deep(.json-number) {
  color: #b5cea8;
}

:deep(.json-boolean) {
  color: #569cd6;
}

:deep(.json-bracket) {
  color: #ffd700;
}

:deep(.json-brace) {
  color: #da70d6;
}
</style>
