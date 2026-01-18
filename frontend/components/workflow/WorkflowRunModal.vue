<script setup lang="ts">
/**
 * WorkflowRunModal - ワークフロー実行モーダル
 *
 * ワークフローIDを受け取り、必要なデータをフェッチして
 * DynamicConfigFormで入力フォームを表示し、ワークフローを実行する。
 */
import { ref, computed, watch } from 'vue'
import type { Project } from '~/types/api'
import type { ConfigSchema } from '~/components/workflow-editor/config/types/config-schema'
import DynamicConfigForm from '~/components/workflow-editor/config/DynamicConfigForm.vue'

const props = defineProps<{
  show: boolean
  workflowId: string
  workflowName: string
}>()

const emit = defineEmits<{
  close: []
  success: [runId: string]
}>()

const { t } = useI18n()
const projectsApi = useProjects()
const runsApi = useRuns()
const toast = useToast()

// Data state
const project = ref<Project | null>(null)
const loading = ref(false)
const dataLoading = ref(false)
const dataError = ref<string | null>(null)

// Input state
const inputValues = ref<Record<string, unknown>>({})
const isValid = ref(true)

// Derive input schema from Start step's trigger_config.input_schema
const inputSchema = computed<ConfigSchema | null>(() => {
  // Find Start step and get its trigger_config.input_schema
  const startStep = project.value?.steps?.find(s => s.type === 'start')
  if (!startStep) return null

  const triggerConfig = startStep.trigger_config as Record<string, unknown> | undefined
  const schema = triggerConfig?.input_schema as Record<string, unknown> | undefined
  if (!schema || schema.type !== 'object') return null

  const properties = schema.properties as Record<string, unknown> | undefined
  if (!properties || Object.keys(properties).length === 0) return null

  return {
    type: 'object',
    properties: properties,
    required: (schema.required as string[]) || [],
  } as ConfigSchema
})

// Check if there are any input fields
const hasInputFields = computed(() => {
  if (!inputSchema.value?.properties) return false
  return Object.keys(inputSchema.value.properties).length > 0
})

// Fetch project data when dialog opens
watch(() => props.show, async (show) => {
  if (show && props.workflowId) {
    await loadProjectData()
  } else {
    // Reset state when closed
    project.value = null
    inputValues.value = {}
    dataError.value = null
    isValid.value = true // Reset validation state
  }
})

async function loadProjectData() {
  dataLoading.value = true
  dataError.value = null

  try {
    const projectRes = await projectsApi.get(props.workflowId)
    project.value = projectRes.data
  } catch (e) {
    dataError.value = e instanceof Error ? e.message : 'Failed to load project data'
  } finally {
    dataLoading.value = false
  }
}

function handleValidationChange(valid: boolean) {
  isValid.value = valid
}

async function handleRun() {
  const startStep = project.value?.steps?.find(s => s.type === 'start')
  if (!startStep) {
    toast.error(t('execution.errors.noStartStep'))
    return
  }

  loading.value = true

  try {
    const response = await runsApi.create(props.workflowId, {
      input: inputValues.value,
      triggered_by: 'manual',
      start_step_id: startStep.id,
    })

    toast.success(t('workflows.runDialog.started'))
    emit('success', response.data.id)
    emit('close')
  } catch (e) {
    toast.error(t('workflows.runDialog.failed'), e instanceof Error ? e.message : undefined)
  } finally {
    loading.value = false
  }
}

function handleClose() {
  if (!loading.value) {
    emit('close')
  }
}
</script>

<template>
  <UiModal
    :show="show"
    :title="t('workflows.runDialog.title')"
    size="md"
    @close="handleClose"
  >
    <div class="run-modal-content">
      <!-- Workflow info -->
      <div class="workflow-info">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="5 3 19 12 5 21 5 3"/>
        </svg>
        <span class="workflow-name">{{ workflowName }}</span>
      </div>

      <!-- Loading state -->
      <div v-if="dataLoading" class="loading-state">
        <span class="spinner" />
        <span>{{ t('common.loading') }}</span>
      </div>

      <!-- Error state -->
      <div v-else-if="dataError" class="error-state">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
        <span>{{ dataError }}</span>
      </div>

      <!-- Input form -->
      <template v-else>
        <div v-if="hasInputFields" class="input-section">
          <div class="input-header">
            <label class="input-label">{{ t('execution.customInput') }}</label>
          </div>
          <p class="input-description">
            {{ t('workflows.runDialog.inputDescription') }}
          </p>

          <DynamicConfigForm
            v-model="inputValues"
            :schema="inputSchema"
            :disabled="loading"
            @validation-change="handleValidationChange"
          />
        </div>
      </template>
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
        :disabled="loading || dataLoading || !!dataError || (hasInputFields && !isValid)"
        @click="handleRun"
      >
        <span v-if="loading" class="spinner spinner-sm" />
        <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="5 3 19 12 5 21 5 3"/>
        </svg>
        {{ t('workflows.runDialog.run') }}
      </button>
    </template>
  </UiModal>
</template>

<style scoped>
.run-modal-content {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.workflow-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  background: var(--color-bg-secondary);
  border-radius: 0.375rem;
  color: var(--color-text-secondary);
}

.workflow-info svg {
  color: var(--color-primary);
  flex-shrink: 0;
}

.workflow-name {
  font-weight: 500;
  color: var(--color-text);
}

.loading-state,
.error-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2rem;
  color: var(--color-text-secondary);
}

.error-state {
  color: var(--color-danger);
}

.input-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.input-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.input-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text);
}

.input-description {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin: 0;
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

.spinner-sm {
  width: 0.875rem;
  height: 0.875rem;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
