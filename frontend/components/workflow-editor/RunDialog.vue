<script setup lang="ts">
/**
 * RunDialog - ワークフロー実行ダイアログ
 *
 * 開始ステップの input_schema を基に入力フォームを生成し、
 * ユーザーの入力を受け付けてワークフローを実行する。
 */
import { ref, computed, watch } from 'vue'
import type { Step, BlockDefinition } from '~/types/api'
import type { ConfigSchema } from './config/types/config-schema'
import DynamicConfigForm from './config/DynamicConfigForm.vue'

const props = defineProps<{
  show: boolean
  workflowId: string
  workflowName: string
  steps: Step[]
  edges: Array<{ source_step_id: string; target_step_id: string }>
  blocks: BlockDefinition[]
}>()

const emit = defineEmits<{
  close: []
  run: [input: Record<string, unknown>]
}>()

const { t } = useI18n()
const inputValues = ref<Record<string, unknown>>({})
const isValid = ref(true)
const loading = ref(false)

// Find the start step and its next step
const startStep = computed(() => props.steps.find(s => s.type === 'start'))

const firstExecutableStep = computed(() => {
  if (!startStep.value) return null

  // Find edge from start step
  const edge = props.edges.find(e => e.source_step_id === startStep.value!.id)
  if (!edge) return null

  // Find target step
  return props.steps.find(s => s.id === edge.target_step_id) || null
})

// Get the block definition for the first executable step
const firstStepBlock = computed(() => {
  if (!firstExecutableStep.value) return null
  return props.blocks.find(b => b.slug === firstExecutableStep.value!.type) || null
})

// Convert input_schema to ConfigSchema format
const inputSchema = computed<ConfigSchema | null>(() => {
  const schema = firstStepBlock.value?.input_schema as Record<string, unknown> | undefined
  if (!schema || schema.type !== 'object') return null

  return {
    type: 'object',
    properties: (schema.properties as Record<string, unknown>) || {},
    required: (schema.required as string[]) || [],
  } as ConfigSchema
})

// Check if there are any input fields
const hasInputFields = computed(() => {
  if (!inputSchema.value?.properties) return false
  return Object.keys(inputSchema.value.properties).length > 0
})

// Reset form when dialog opens
watch(() => props.show, (show) => {
  if (show) {
    inputValues.value = {}
    loading.value = false
  }
})

function handleValidationChange(valid: boolean) {
  isValid.value = valid
}

async function handleRun() {
  loading.value = true
  emit('run', inputValues.value)
}

function handleClose() {
  emit('close')
}
</script>

<template>
  <UiModal
    :show="show"
    :title="t('workflows.runDialog.title')"
    size="md"
    @close="handleClose"
  >
    <div class="run-dialog-content">
      <!-- Workflow info -->
      <div class="workflow-info">
        <span class="workflow-name">{{ workflowName }}</span>
      </div>

      <!-- Input form -->
      <div v-if="hasInputFields" class="input-section">
        <h3 class="input-title">{{ t('workflows.runDialog.inputTitle') }}</h3>
        <p class="input-description">
          {{ t('workflows.runDialog.inputDescription', { stepName: firstExecutableStep?.name }) }}
        </p>

        <DynamicConfigForm
          v-model="inputValues"
          :schema="inputSchema"
          :disabled="loading"
          @validation-change="handleValidationChange"
        />
      </div>

      <!-- No input fields message -->
      <div v-else class="no-input-message">
        <p>{{ t('workflows.runDialog.noInputRequired') }}</p>
      </div>

      <!-- Schema info (for debugging) -->
      <div v-if="firstStepBlock?.input_schema" class="schema-info">
        <details>
          <summary>{{ t('workflows.runDialog.schemaDetails') }}</summary>
          <pre>{{ JSON.stringify(firstStepBlock.input_schema, null, 2) }}</pre>
        </details>
      </div>
    </div>

    <template #footer>
      <button
        class="btn btn-secondary"
        :disabled="loading"
        @click="handleClose"
      >
        {{ t('common.cancel') }}
      </button>
      <button
        class="btn btn-primary"
        :disabled="loading || (hasInputFields && !isValid)"
        @click="handleRun"
      >
        <span v-if="loading" class="spinner" />
        {{ t('workflows.runDialog.run') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.run-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.workflow-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: var(--color-bg-secondary);
  border-radius: 0.375rem;
}

.workflow-name {
  font-weight: 500;
  color: var(--color-text);
}

.input-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.input-title {
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.input-description {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin: 0;
}

.no-input-message {
  padding: 1rem;
  background: var(--color-bg-secondary);
  border-radius: 0.375rem;
  text-align: center;
}

.no-input-message p {
  margin: 0;
  color: var(--color-text-secondary);
}

.schema-info {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.schema-info summary {
  cursor: pointer;
  user-select: none;
}

.schema-info pre {
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: var(--color-bg-secondary);
  border-radius: 0.25rem;
  overflow-x: auto;
  font-size: 0.6875rem;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  transition: all 0.15s ease;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  color: var(--color-text);
}

.btn-secondary:hover:not(:disabled) {
  background: var(--color-bg-hover);
}

.btn-primary {
  background: var(--color-primary);
  border: 1px solid var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-hover);
}

.spinner {
  width: 1rem;
  height: 1rem;
  border: 2px solid transparent;
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
