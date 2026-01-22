<script setup lang="ts">
import type { Step } from '~/types/api'
import type { ConfigSchema } from '../config/types/config-schema'
import type { SchemaField } from '~/composables/execution'
import DynamicConfigForm from '../config/DynamicConfigForm.vue'

const { t } = useI18n()

defineProps<{
  firstExecutableStep: Step | null
  hasWorkflowInputFields: boolean
  workflowInputSchema: ConfigSchema | null
  workflowSchemaFields: SchemaField[]
  executing: boolean
  useWorkflowJsonMode: boolean
  customInputJson: string
  inputError: string | null
  workflowInputValues: Record<string, unknown>
  workflowFormValid: boolean
}>()

const emit = defineEmits<{
  'update:customInputJson': [value: string]
  'update:workflowInputValues': [value: Record<string, unknown>]
  'update:workflowFormValid': [value: boolean]
  'toggle-mode': []
  'clear-input': []
  'execute-workflow': []
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
</script>

<template>
  <div class="workflow-execution-section">
    <!-- Input Section (only shown when workflow has input fields) -->
    <div v-if="hasWorkflowInputFields" class="input-section">
      <div class="input-header">
        <label class="input-label">{{ t('execution.customInput') }}</label>
        <div class="input-actions">
          <button class="btn-icon" :title="t('execution.clearInput')" @click="emit('clear-input')">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 6h18"/>
              <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
            </svg>
          </button>
          <button
            v-if="hasWorkflowInputFields"
            class="btn-icon"
            :class="{ active: useWorkflowJsonMode }"
            :title="useWorkflowJsonMode ? t('execution.switchToForm') : t('execution.switchToJson')"
            @click="emit('toggle-mode')"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 18 22 12 16 6"/>
              <polyline points="8 6 2 12 8 18"/>
            </svg>
          </button>
        </div>
      </div>

      <!-- Dynamic Form -->
      <template v-if="hasWorkflowInputFields && !useWorkflowJsonMode">
        <p class="input-description">
          {{ t('execution.inputDescription', { stepName: firstExecutableStep?.name }) }}
        </p>
        <DynamicConfigForm
          :model-value="workflowInputValues"
          :schema="workflowInputSchema"
          :disabled="executing"
          @update:model-value="emit('update:workflowInputValues', $event)"
          @validation-change="emit('update:workflowFormValid', $event)"
        />
      </template>

      <!-- JSON Textarea -->
      <template v-else>
        <div v-if="workflowSchemaFields.length > 0" class="schema-preview">
          <div class="schema-preview-header">
            <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
              <polyline points="14 2 14 8 20 8"/>
            </svg>
            <span>{{ t('execution.expectedFields') }}</span>
          </div>
          <div class="schema-fields">
            <div v-for="field in workflowSchemaFields" :key="field.name" class="schema-field">
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
          :placeholder="workflowSchemaFields.length > 0 ? generateExampleJson(workflowSchemaFields) : t('execution.inputPlaceholder')"
          @input="emit('update:customInputJson', ($event.target as HTMLTextAreaElement).value)"
        />

        <p v-if="workflowSchemaFields.length > 0" class="json-hint">
          {{ t('execution.jsonHint') }}
        </p>
      </template>

      <p v-if="inputError" class="input-error">{{ inputError }}</p>
    </div>

    <!-- Execute Workflow Button -->
    <button
      class="btn btn-primary full-width"
      :disabled="executing"
      @click="emit('execute-workflow')"
    >
      <svg v-if="!executing" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="5 3 19 12 5 21 5 3"/>
      </svg>
      <svg v-else class="spinning" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
      </svg>
      {{ executing ? t('execution.executing') : t('execution.executeWorkflow') }}
    </button>
  </div>
</template>

<style scoped>
.workflow-execution-section {
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--color-border);
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

.full-width {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
