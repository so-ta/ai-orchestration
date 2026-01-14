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
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: object): void
  (e: 'blur'): void
}>()

// Field types
const fieldTypes = [
  { value: 'string', label: '文字列' },
  { value: 'number', label: '数値' },
  { value: 'boolean', label: '真偽値' },
  { value: 'object', label: 'オブジェクト' },
  { value: 'array', label: '配列' },
]

// Schema field interface
interface SchemaField {
  id: string
  name: string
  type: string
  title: string
  description: string
  required: boolean
}

// Internal state
const fields = ref<SchemaField[]>([])
const showJsonEditor = ref(false)
const jsonText = ref('')
const parseError = ref<string | null>(null)
const isInternalUpdate = ref(false)

// Get display title
const title = computed(() => {
  return props.property.title || props.name
})

// Get description
const description = computed(() => {
  return props.property.description
})

// Parse schema to fields
function parseSchemaToFields(schema: any): SchemaField[] {
  if (!schema || schema.type !== 'object' || !schema.properties) {
    return []
  }

  const required = schema.required || []
  return Object.entries(schema.properties).map(([name, prop]: [string, any]) => ({
    id: crypto.randomUUID(),
    name,
    type: prop.type || 'string',
    title: prop.title || '',
    description: prop.description || '',
    required: required.includes(name)
  }))
}

// Convert fields to schema
function fieldsToSchema(fields: SchemaField[]): object {
  if (fields.length === 0) {
    return {}
  }

  const properties: Record<string, any> = {}
  const required: string[] = []

  for (const field of fields) {
    if (!field.name.trim()) continue

    properties[field.name] = {
      type: field.type,
      ...(field.title && { title: field.title }),
      ...(field.description && { description: field.description })
    }

    if (field.required) {
      required.push(field.name)
    }
  }

  return {
    type: 'object',
    properties,
    ...(required.length > 0 && { required })
  }
}

// Initialize from modelValue
function initFromModel() {
  if (props.modelValue && typeof props.modelValue === 'object') {
    fields.value = parseSchemaToFields(props.modelValue)
    jsonText.value = JSON.stringify(props.modelValue, null, 2)
  } else {
    fields.value = []
    jsonText.value = ''
  }
  parseError.value = null
}

// Watch for external changes (skip if change came from internal update)
watch(() => props.modelValue, (newValue, oldValue) => {
  if (isInternalUpdate.value) {
    isInternalUpdate.value = false
    return
  }
  if (JSON.stringify(newValue) !== JSON.stringify(oldValue)) {
    initFromModel()
  }
}, { immediate: true, deep: true })

// Emit changes
function emitChanges() {
  const schema = fieldsToSchema(fields.value)
  jsonText.value = Object.keys(schema).length > 0 ? JSON.stringify(schema, null, 2) : ''
  isInternalUpdate.value = true
  emit('update:modelValue', schema)
}

// Add new field
function addField() {
  fields.value.push({
    id: crypto.randomUUID(),
    name: '',
    type: 'string',
    title: '',
    description: '',
    required: false
  })
}

// Remove field
function removeField(index: number) {
  fields.value.splice(index, 1)
  emitChanges()
}

// Update field
function updateField(index: number, key: keyof SchemaField, value: any) {
  if (fields.value[index]) {
    (fields.value[index] as any)[key] = value
    emitChanges()
  }
}

// Handle JSON text change
function handleJsonInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  jsonText.value = target.value

  if (!target.value.trim()) {
    parseError.value = null
    fields.value = []
    isInternalUpdate.value = true
    emit('update:modelValue', {})
    return
  }

  try {
    const parsed = JSON.parse(target.value)
    parseError.value = null
    fields.value = parseSchemaToFields(parsed)
    isInternalUpdate.value = true
    emit('update:modelValue', parsed)
  } catch (e) {
    parseError.value = (e as Error).message
  }
}

// Toggle JSON editor
function toggleJsonEditor() {
  if (!showJsonEditor.value) {
    // Sync JSON text before showing
    const schema = fieldsToSchema(fields.value)
    jsonText.value = Object.keys(schema).length > 0 ? JSON.stringify(schema, null, 2) : ''
  }
  showJsonEditor.value = !showJsonEditor.value
}

// Handle blur
function handleBlur() {
  emit('blur')
}
</script>

<template>
  <div class="output-schema-widget">
    <div class="widget-header">
      <label class="widget-label">
        {{ title }}
      </label>
      <button
        type="button"
        class="toggle-button"
        :class="{ active: showJsonEditor }"
        :disabled="disabled"
        @click="toggleJsonEditor"
      >
        <svg v-if="!showJsonEditor" xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="16 18 22 12 16 6"/>
          <polyline points="8 6 2 12 8 18"/>
        </svg>
        <svg v-else xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <rect x="3" y="3" width="7" height="7"/>
          <rect x="14" y="3" width="7" height="7"/>
          <rect x="14" y="14" width="7" height="7"/>
          <rect x="3" y="14" width="7" height="7"/>
        </svg>
        {{ showJsonEditor ? 'ビジュアル' : 'JSON' }}
      </button>
    </div>

    <p v-if="description" class="widget-description">
      {{ description }}
    </p>

    <!-- Visual Editor -->
    <div v-if="!showJsonEditor" class="visual-editor">
      <div v-if="fields.length === 0" class="empty-state">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="12" y1="18" x2="12" y2="12"/>
          <line x1="9" y1="15" x2="15" y2="15"/>
        </svg>
        <p>出力フィールドが定義されていません</p>
        <p class="empty-hint">フィールドを追加すると、定義されたフィールドのみが次のステップに渡されます</p>
      </div>

      <div v-else class="fields-list">
        <div v-for="(field, index) in fields" :key="field.id" class="field-item">
          <div class="field-header">
            <span class="field-index">#{{ index + 1 }}</span>
            <button
              type="button"
              class="remove-button"
              :disabled="disabled"
              @click="removeField(index)"
              title="フィールドを削除"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>

          <div class="field-row">
            <div class="field-group name-group">
              <label class="field-label">フィールド名 *</label>
              <input
                type="text"
                class="field-input"
                :value="field.name"
                :disabled="disabled"
                placeholder="field_name"
                @input="updateField(index, 'name', ($event.target as HTMLInputElement).value)"
                @blur="handleBlur"
              />
            </div>
            <div class="field-group type-group">
              <label class="field-label">型</label>
              <select
                class="field-select"
                :value="field.type"
                :disabled="disabled"
                @change="updateField(index, 'type', ($event.target as HTMLSelectElement).value)"
              >
                <option v-for="t in fieldTypes" :key="t.value" :value="t.value">
                  {{ t.label }}
                </option>
              </select>
            </div>
            <div class="field-group required-group">
              <label class="field-label">必須</label>
              <label class="checkbox-wrapper">
                <input
                  type="checkbox"
                  :checked="field.required"
                  :disabled="disabled"
                  @change="updateField(index, 'required', ($event.target as HTMLInputElement).checked)"
                />
                <span class="checkbox-mark"></span>
              </label>
            </div>
          </div>

          <div class="field-row">
            <div class="field-group">
              <label class="field-label">表示名</label>
              <input
                type="text"
                class="field-input"
                :value="field.title"
                :disabled="disabled"
                placeholder="日本語の表示名"
                @input="updateField(index, 'title', ($event.target as HTMLInputElement).value)"
                @blur="handleBlur"
              />
            </div>
          </div>

          <div class="field-row">
            <div class="field-group">
              <label class="field-label">説明</label>
              <input
                type="text"
                class="field-input"
                :value="field.description"
                :disabled="disabled"
                placeholder="このフィールドの説明"
                @input="updateField(index, 'description', ($event.target as HTMLInputElement).value)"
                @blur="handleBlur"
              />
            </div>
          </div>
        </div>
      </div>

      <button
        type="button"
        class="add-field-button"
        :disabled="disabled"
        @click="addField"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"/>
          <line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
        フィールドを追加
      </button>
    </div>

    <!-- JSON Editor -->
    <div v-else class="json-editor">
      <textarea
        class="json-textarea"
        :value="jsonText"
        :disabled="disabled"
        placeholder='{"type": "object", "properties": {...}}'
        spellcheck="false"
        @input="handleJsonInput"
        @blur="handleBlur"
      ></textarea>
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
          JSON Schema
        </span>
      </div>
    </div>

    <p v-if="error" class="widget-error">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.output-schema-widget {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.widget-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.widget-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text);
}

.toggle-button {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.toggle-button:hover:not(:disabled) {
  color: var(--color-primary);
  border-color: var(--color-primary);
}

.toggle-button.active {
  color: var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
  border-color: var(--color-primary);
}

.toggle-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.widget-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.4;
}

/* Visual Editor */
.visual-editor {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 1.5rem;
  background: var(--color-background);
  border: 1px dashed var(--color-border);
  border-radius: 6px;
  text-align: center;
}

.empty-state svg {
  color: var(--color-text-secondary);
  opacity: 0.5;
}

.empty-state p {
  margin: 0;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.empty-hint {
  font-size: 0.6875rem !important;
  color: var(--color-text-secondary);
  opacity: 0.7;
}

.fields-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.field-item {
  padding: 0.75rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.field-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.field-index {
  font-size: 0.625rem;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.remove-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.remove-button:hover:not(:disabled) {
  color: var(--color-error);
  background: rgba(239, 68, 68, 0.1);
}

.remove-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.field-row {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.field-row:last-child {
  margin-bottom: 0;
}

.field-group {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.name-group {
  flex: 2;
}

.type-group {
  flex: 1;
}

.required-group {
  flex: 0 0 auto;
  width: 50px;
  align-items: center;
}

.field-label {
  font-size: 0.625rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.field-input,
.field-select {
  padding: 0.375rem 0.5rem;
  font-size: 0.75rem;
  color: var(--color-text);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  outline: none;
  transition: border-color 0.15s ease;
}

.field-input:focus,
.field-select:focus {
  border-color: var(--color-primary);
}

.field-input:disabled,
.field-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.field-input::placeholder {
  color: var(--color-text-secondary);
  opacity: 0.5;
}

.checkbox-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.checkbox-wrapper input {
  position: absolute;
  opacity: 0;
  cursor: pointer;
}

.checkbox-mark {
  width: 18px;
  height: 18px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  transition: all 0.15s ease;
}

.checkbox-wrapper input:checked ~ .checkbox-mark {
  background: var(--color-primary);
  border-color: var(--color-primary);
}

.checkbox-wrapper input:checked ~ .checkbox-mark::after {
  content: '';
  position: absolute;
  left: 6px;
  top: 2px;
  width: 5px;
  height: 10px;
  border: solid white;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}

.checkbox-wrapper input:disabled ~ .checkbox-mark {
  opacity: 0.5;
  cursor: not-allowed;
}

.add-field-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  padding: 0.5rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-primary);
  background: transparent;
  border: 1px dashed var(--color-primary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.add-field-button:hover:not(:disabled) {
  background: rgba(59, 130, 246, 0.05);
}

.add-field-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* JSON Editor */
.json-editor {
  border: 1px solid var(--color-border);
  border-radius: 6px;
  overflow: hidden;
  background: #1e1e1e;
}

.json-textarea {
  width: 100%;
  min-height: 150px;
  padding: 0.75rem;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
  line-height: 1.5;
  color: #d4d4d4;
  background: transparent;
  border: none;
  outline: none;
  resize: vertical;
}

.json-textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.json-textarea::placeholder {
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

.widget-error {
  font-size: 0.6875rem;
  color: var(--color-error);
  margin: 0;
}
</style>
