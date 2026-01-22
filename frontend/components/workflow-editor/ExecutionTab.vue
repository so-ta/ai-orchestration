<script setup lang="ts">
import type { Step, Run, BlockDefinition } from '~/types/api'
import type { ConfigSchema } from './config/types/config-schema'
import { useTemplateVariables } from '~/composables/useTemplateVariables'
import { useSchemaFields, useStepNavigation, useWorkflowExecution, useExecutionInput } from '~/composables/execution'
import StepInputSection from './execution/StepInputSection.vue'
import WorkflowInputSection from './execution/WorkflowInputSection.vue'

const { t } = useI18n()
const toast = useToast()

const props = defineProps<{
  step: Step | null
  workflowId: string
  latestRun: Run | null
  isActive?: boolean
  steps: Step[]
  edges: Array<{ id: string; source_step_id?: string | null; target_step_id?: string | null }>
  blocks: BlockDefinition[]
}>()

const emit = defineEmits<{
  (e: 'execute', data: { stepId: string; input: object; triggered_by: 'test' | 'manual' }): void
  (e: 'execute-workflow', triggered_by: 'test' | 'manual', input: object): void
  (e: 'run:created', run: Run): void
}>()

// Convert props to refs for composables
const stepRef = computed(() => props.step)
const stepsRef = computed(() => props.steps)
const edgesRef = computed(() => props.edges)
const blocksRef = computed(() => props.blocks)

// Use schema fields composable
const { getSchemaFields, toConfigSchema, hasFields } = useSchemaFields()

// Use step navigation composable
const {
  startStep,
  firstExecutableStep,
  isStartStep,
  selectedStepBlock,
  effectiveStep,
  previousStep,
  suggestedFields,
  latestStepRunOutput,
} = useStepNavigation({
  step: stepRef,
  steps: stepsRef,
  edges: edgesRef,
  blocks: blocksRef,
  testRuns: computed(() => testRuns.value),
})

// Use workflow execution composable
const {
  executing,
  testRuns,
  pollingRunId,
  loadTestRuns,
  executeWorkflow: execWorkflow,
  executeThisStepOnly: execThisStep,
  executeFromThisStep: execFromStep,
} = useWorkflowExecution({
  workflowId: props.workflowId,
  onRunCreated: (run) => emit('run:created', run),
  onTestRunsLoaded: () => {},
})

// Schema computed properties
const workflowInputSchema = computed<ConfigSchema | null>(() => {
  if (!startStep.value?.config) return null
  const config = startStep.value.config as Record<string, unknown>
  const schema = config.input_schema as Record<string, unknown> | undefined
  return toConfigSchema(schema)
})

const stepInputSchema = computed<ConfigSchema | null>(() => {
  if (isStartStep.value) {
    return workflowInputSchema.value
  }
  const schema = selectedStepBlock.value?.config_schema as Record<string, unknown> | undefined
  return toConfigSchema(schema)
})

const hasWorkflowInputFields = computed(() => hasFields(workflowInputSchema.value))
const hasStepInputFields = computed(() => {
  if (isStartStep.value) {
    return hasWorkflowInputFields.value
  }
  return hasFields(stepInputSchema.value)
})

// Use execution input composable
const {
  useJsonMode,
  useWorkflowJsonMode,
  customInputJson,
  inputError,
  schemaValidationErrors,
  workflowInputValues,
  stepInputValues,
  workflowFormValid,
  stepFormValid,
  getWorkflowInput,
  getStepInput,
  toggleInputMode,
  toggleWorkflowInputMode,
  usePreviousOutput,
  clearInput,
  insertSuggestedField,
} = useExecutionInput({
  workflowId: props.workflowId,
  stepId: computed(() => props.step?.id),
  hasStepInputFields,
  hasWorkflowInputFields,
  selectedStepBlock,
})

// Schema fields for preview
const workflowSchemaFields = computed(() => {
  if (!startStep.value?.config) return []
  const config = startStep.value.config as Record<string, unknown>
  const schema = config.input_schema as Record<string, unknown> | undefined
  return getSchemaFields(schema)
})

const stepSchemaFields = computed(() => {
  const schema = selectedStepBlock.value?.config_schema as Record<string, unknown> | undefined
  return getSchemaFields(schema)
})

// Template variables detection and preview using composable
const stepConfigRef = computed(() => props.step?.config as Record<string, unknown> | undefined)
const {
  variables: templateVariables,
  createPreview: createTemplatePreview,
} = useTemplateVariables(stepConfigRef)

// Template preview with resolved values
const templatePreview = computed(() => {
  const input = useJsonMode.value || !hasStepInputFields.value
    ? (() => { try { return JSON.parse(customInputJson.value) } catch { return {} } })()
    : stepInputValues.value
  return createTemplatePreview(input)
})

// Execution handlers
async function executeWorkflow() {
  const input = getWorkflowInput()
  if (input === null) return

  if (!startStep.value) {
    toast.error(t('execution.errors.noStartStep'))
    return
  }

  await execWorkflow(input, startStep.value.id)
}

async function executeThisStepOnly() {
  if (!props.step) {
    toast.error(t('execution.errors.noStepSelected'))
    return
  }

  const input = getStepInput()
  if (input === null) return

  await execThisStep(props.step, input, props.latestRun)
}

async function executeFromThisStep() {
  if (!props.step) {
    toast.error(t('execution.errors.noStepSelected'))
    return
  }

  const input = getStepInput()
  if (input === null) return

  await execFromStep(props.step, input)
}

// Handle use previous output
function handleUsePreviousOutput() {
  if (latestStepRunOutput.value) {
    usePreviousOutput(latestStepRunOutput.value as Record<string, unknown>)
  }
}

// Load test runs when tab becomes active or step changes while active
watch(() => props.isActive, (isActive) => {
  if (isActive) {
    loadTestRuns()
  }
}, { immediate: true })

watch(() => props.step, () => {
  if (props.isActive) {
    loadTestRuns()
  }
})
</script>

<template>
  <div class="execution-tab">
    <div class="execution-content">
      <!-- Step Selected (not Start): Show step-specific execution controls -->
      <StepInputSection
        v-if="step && !isStartStep"
        :step="step"
        :effective-step="effectiveStep"
        :selected-step-block="selectedStepBlock"
        :previous-step="previousStep"
        :has-step-input-fields="hasStepInputFields"
        :step-input-schema="stepInputSchema"
        :step-schema-fields="stepSchemaFields"
        :suggested-fields="suggestedFields"
        :template-variables="templateVariables"
        :template-preview="templatePreview"
        :latest-step-run-output="latestStepRunOutput"
        :executing="executing"
        :polling-run-id="pollingRunId"
        :use-json-mode="useJsonMode"
        :custom-input-json="customInputJson"
        :input-error="inputError"
        :schema-validation-errors="schemaValidationErrors"
        :step-input-values="stepInputValues"
        :step-form-valid="stepFormValid"
        @update:custom-input-json="customInputJson = $event"
        @update:step-input-values="stepInputValues = $event"
        @update:step-form-valid="stepFormValid = $event"
        @toggle-mode="toggleInputMode"
        @use-previous-output="handleUsePreviousOutput"
        @clear-input="clearInput"
        @insert-suggested-field="insertSuggestedField"
        @execute-step-only="executeThisStepOnly"
        @execute-from-step="executeFromThisStep"
      />

      <!-- No Step Selected OR Start Step: Show workflow execution controls -->
      <WorkflowInputSection
        v-else
        :first-executable-step="firstExecutableStep"
        :has-workflow-input-fields="hasWorkflowInputFields"
        :workflow-input-schema="workflowInputSchema"
        :workflow-schema-fields="workflowSchemaFields"
        :executing="executing"
        :use-workflow-json-mode="useWorkflowJsonMode"
        :custom-input-json="customInputJson"
        :input-error="inputError"
        :workflow-input-values="workflowInputValues"
        :workflow-form-valid="workflowFormValid"
        @update:custom-input-json="customInputJson = $event"
        @update:workflow-input-values="workflowInputValues = $event"
        @update:workflow-form-valid="workflowFormValid = $event"
        @toggle-mode="toggleWorkflowInputMode"
        @clear-input="clearInput"
        @execute-workflow="executeWorkflow"
      />
    </div>
  </div>
</template>

<style scoped>
.execution-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.execution-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.execution-content::-webkit-scrollbar {
  width: 6px;
}

.execution-content::-webkit-scrollbar-track {
  background: transparent;
}

.execution-content::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}
</style>
