<script setup lang="ts">
import type { Step, BlockDefinition } from '~/types/api'
import type { ConfigSchema } from '../config/types/config-schema'
import type { SchemaField } from '~/composables/execution'
import DynamicConfigForm from '../config/DynamicConfigForm.vue'

const { t } = useI18n()

defineProps<{
  step: Step
  effectiveStep: Step | null
  selectedStepBlock: BlockDefinition | null
  previousStep: Step | null
  hasStepInputFields: boolean
  stepInputSchema: ConfigSchema | null
  stepSchemaFields: SchemaField[]
  suggestedFields: Array<{ name: string; value: unknown; type: string }>
  templateVariables: string[]
  templatePreview: Array<{ variable: string; resolved: string; isResolved: boolean }>
  latestStepRunOutput: unknown
  executing: boolean
  pollingRunId: string | null | undefined
  useJsonMode: boolean
  customInputJson: string
  inputError: string | null
  schemaValidationErrors: Array<{ field: string; message: string }>
  stepInputValues: Record<string, unknown>
  stepFormValid: boolean
}>()

const emit = defineEmits<{
  'update:customInputJson': [value: string]
  'update:stepInputValues': [value: Record<string, unknown>]
  'update:stepFormValid': [value: boolean]
  'toggle-mode': []
  'use-previous-output': []
  'clear-input': []
  'insert-suggested-field': [name: string, value: unknown]
  'execute-step-only': []
  'execute-from-step': []
}>()

function getTypeBadgeClass(type: string): string {
  const typeMap: Record<string, string> = {
    string: 'type-string',
    number: 'type-number',
    integer: 'type-number',
    boolean: 'type-boolean',
    array: 'type-array',
    object: 'type-object',
  }
  return typeMap[type] || 'type-any'
}

function generateExampleJson(fields: SchemaField[]): string {
  const example: Record<string, unknown> = {}
  for (const field of fields) {
    switch (field.type) {
      case 'string':
        example[field.name] = ''
        break
      case 'number':
      case 'integer':
        example[field.name] = 0
        break
      case 'boolean':
        example[field.name] = false
        break
      case 'array':
        example[field.name] = []
        break
      case 'object':
        example[field.name] = {}
        break
      default:
        example[field.name] = null
    }
  }
  return JSON.stringify(example, null, 2)
}

function formatTemplateVar(variable: string): string {
  return `{{${variable}}}`
}
</script>

<template>
  <div class="step-input-section">
    <!-- Input Section (only shown when step has input fields) -->
    <div v-if="hasStepInputFields" class="input-section">
      <div class="input-header">
        <label class="input-label">{{ t('execution.customInput') }}</label>
        <div class="input-actions">
          <!-- Use Previous Output Button -->
          <button
            v-if="latestStepRunOutput"
            class="btn-icon"
            :title="t('execution.usePreviousOutput')"
            @click="emit('use-previous-output')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="1 4 1 10 7 10"/>
              <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/>
            </svg>
          </button>
          <!-- Clear Button -->
          <button class="btn-icon" :title="t('execution.clearInput')" @click="emit('clear-input')">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 6h18"/>
              <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
            </svg>
          </button>
          <!-- Toggle Form/JSON Mode -->
          <button
            v-if="hasStepInputFields"
            class="btn-icon"
            :class="{ active: useJsonMode }"
            :title="useJsonMode ? t('execution.switchToForm') : t('execution.switchToJson')"
            @click="emit('toggle-mode')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 18 22 12 16 6"/>
              <polyline points="8 6 2 12 8 18"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Dynamic Form (when input_schema is available and not in JSON mode) -->
      <template v-if="hasStepInputFields && !useJsonMode">
        <p class="input-description">
          {{ t('execution.inputDescription', { stepName: effectiveStep?.name || step.name }) }}
        </p>
        <DynamicConfigForm
          :model-value="stepInputValues"
          :schema="stepInputSchema"
          :disabled="executing"
          @update:model-value="emit('update:stepInputValues', $event)"
          @validation-change="emit('update:stepFormValid', $event)"
        />
      </template>

      <!-- JSON Textarea (when no schema, or JSON mode is active) -->
      <template v-else>
        <!-- Schema Preview (if we have schema info) -->
        <div v-if="stepSchemaFields.length > 0" class="schema-preview">
          <div class="schema-preview-header">
            <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
              <polyline points="14 2 14 8 20 8"/>
            </svg>
            <span>{{ t('execution.expectedFields') }}</span>
          </div>
          <div class="schema-fields">
            <div v-for="field in stepSchemaFields" :key="field.name" class="schema-field">
              <div class="field-header">
                <code class="field-name">{{ field.name }}</code>
                <span :class="['type-badge', getTypeBadgeClass(field.type)]">{{ field.type }}</span>
                <span v-if="field.required" class="required-badge">{{ t('execution.required') }}</span>
              </div>
              <p v-if="field.description" class="field-description">{{ field.description }}</p>
            </div>
          </div>
        </div>

        <textarea
          :value="customInputJson"
          class="json-input"
          rows="4"
          :placeholder="stepSchemaFields.length > 0 ? generateExampleJson(stepSchemaFields) : t('execution.inputPlaceholder')"
          @input="emit('update:customInputJson', ($event.target as HTMLTextAreaElement).value)"
        />

        <p v-if="stepSchemaFields.length > 0" class="json-hint">
          {{ t('execution.jsonHint') }}
        </p>
      </template>

      <!-- Schema Validation Errors -->
      <div v-if="schemaValidationErrors.length > 0" class="validation-errors">
        <div v-for="error in schemaValidationErrors" :key="error.field" class="validation-error">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
          <span>{{ error.message }}</span>
        </div>
      </div>

      <p v-if="inputError" class="input-error">{{ inputError }}</p>

      <!-- Suggested Fields from Previous Step -->
      <div v-if="suggestedFields.length > 0 && (useJsonMode || !hasStepInputFields)" class="suggested-fields">
        <div class="suggested-header">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
          </svg>
          <span>{{ t('execution.suggestedFields') }} ({{ previousStep?.name }})</span>
        </div>
        <div class="suggested-chips">
          <button
            v-for="field in suggestedFields"
            :key="field.name"
            class="suggested-chip"
            :title="JSON.stringify(field.value, null, 2)"
            @click="emit('insert-suggested-field', field.name, field.value)"
          >
            <code>{{ field.name }}</code>
            <span :class="['chip-type', `type-${field.type}`]">{{ field.type }}</span>
          </button>
        </div>
      </div>

      <!-- Template Preview -->
      <div v-if="templateVariables.length > 0" class="template-preview">
        <div class="template-preview-header">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
            <polyline points="14 2 14 8 20 8"/>
            <path d="M12 18v-6"/>
            <path d="M8 15h8"/>
          </svg>
          <span>{{ t('execution.templatePreview') }}</span>
        </div>
        <div class="template-items">
          <div v-for="item in templatePreview" :key="item.variable" class="template-item">
            <code class="template-variable" v-text="formatTemplateVar(item.variable)"/>
            <span class="template-arrow">→</span>
            <span :class="['template-resolved', { 'not-resolved': !item.isResolved }]">
              {{ item.resolved }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Execution Buttons -->
    <div class="execution-buttons">
      <button
        class="btn btn-primary"
        :disabled="executing || !!pollingRunId"
        @click="emit('execute-step-only')"
      >
        <svg v-if="!pollingRunId && !executing" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="5 3 19 12 5 21 5 3"/>
        </svg>
        <svg v-else class="spinning" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
        </svg>
        {{ pollingRunId ? t('execution.waitingForResult') : (executing ? t('execution.executing') : t('execution.executeThisStepOnly')) }}
      </button>
      <button
        class="btn btn-outline"
        :disabled="executing || !!pollingRunId"
        @click="emit('execute-from-step')"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
        </svg>
        {{ t('execution.executeFromThisStep') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.step-input-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.input-section {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.input-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.375rem;
}

.input-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text);
}

.input-actions {
  display: flex;
  gap: 0.25rem;
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: white;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.btn-icon:hover {
  background: var(--color-surface);
  color: var(--color-text);
  border-color: var(--color-text-secondary);
}

.btn-icon.active {
  background: #eff6ff;
  color: #3b82f6;
  border-color: #3b82f6;
}

.json-input {
  width: 100%;
  padding: 0.5rem;
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  resize: vertical;
  min-height: 60px;
}

.json-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.input-description {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.5rem 0;
}

.input-error {
  font-size: 0.6875rem;
  color: var(--color-error);
  margin: 0;
}

.schema-preview {
  background: #f8fafc;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 0.75rem;
  margin-bottom: 0.5rem;
}

.schema-preview-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.625rem;
}

.schema-fields {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.schema-field {
  padding: 0.5rem;
  background: white;
  border-radius: 4px;
  border: 1px solid var(--color-border);
}

.field-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  flex-wrap: wrap;
}

.field-name {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  background: #f1f5f9;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.field-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0.375rem 0 0 0;
  line-height: 1.4;
}

.type-badge {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.type-string { background: #dcfce7; color: #16a34a; }
.type-number { background: #dbeafe; color: #2563eb; }
.type-boolean { background: #fef3c7; color: #d97706; }
.type-array { background: #f3e8ff; color: #9333ea; }
.type-object { background: #fce7f3; color: #db2777; }
.type-any { background: #f3f4f6; color: #6b7280; }

.required-badge {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
  background: #fee2e2;
  color: #dc2626;
}

.json-hint {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0.375rem 0 0 0;
}

.validation-errors {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-top: 0.5rem;
}

.validation-error {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.5rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 4px;
  font-size: 0.6875rem;
  color: #dc2626;
}

.validation-error svg { flex-shrink: 0; }

.suggested-fields {
  margin-top: 0.75rem;
  padding: 0.625rem;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 6px;
}

.suggested-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.6875rem;
  font-weight: 600;
  color: #16a34a;
  margin-bottom: 0.5rem;
}

.suggested-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
}

.suggested-chip {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  background: white;
  border: 1px solid #86efac;
  border-radius: 4px;
  font-size: 0.6875rem;
  cursor: pointer;
  transition: all 0.15s;
}

.suggested-chip:hover {
  background: #dcfce7;
  border-color: #22c55e;
}

.suggested-chip code {
  font-family: 'SF Mono', Monaco, monospace;
  font-weight: 500;
  color: var(--color-text);
}

.chip-type {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.0625rem 0.25rem;
  border-radius: 2px;
}

.chip-type.type-string { background: #dcfce7; color: #16a34a; }
.chip-type.type-number { background: #dbeafe; color: #2563eb; }
.chip-type.type-boolean { background: #fef3c7; color: #d97706; }
.chip-type.type-array { background: #f3e8ff; color: #9333ea; }
.chip-type.type-object { background: #fce7f3; color: #db2777; }

.template-preview {
  margin-top: 0.75rem;
  padding: 0.625rem;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
}

.template-preview-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.6875rem;
  font-weight: 600;
  color: #2563eb;
  margin-bottom: 0.5rem;
}

.template-items {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.template-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.5rem;
  background: white;
  border-radius: 4px;
  font-size: 0.6875rem;
}

.template-variable {
  font-family: 'SF Mono', Monaco, monospace;
  font-weight: 500;
  color: #7c3aed;
  background: #f5f3ff;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.template-arrow { color: var(--color-text-secondary); }

.template-resolved {
  font-family: 'SF Mono', Monaco, monospace;
  color: #16a34a;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.template-resolved.not-resolved {
  color: #d97706;
  font-style: italic;
}

.execution-buttons {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.execution-buttons .btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
