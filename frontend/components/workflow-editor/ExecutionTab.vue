<script setup lang="ts">
import type { Step, Run, BlockDefinition } from '~/types/api'
import type { ConfigSchema } from './config/types/config-schema'
import { validateConfig } from './config/composables/useValidation'
import { useStoredInput } from '~/composables/useStoredInput'
import { usePolling } from '~/composables/usePolling'
import { useTemplateVariables } from '~/composables/useTemplateVariables'
import DynamicConfigForm from './config/DynamicConfigForm.vue'

const { t } = useI18n()
const runsApi = useRuns()
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

// Execution state
const executing = ref(false)

// Input mode toggle (form vs json)
const useJsonMode = ref(false)
const useWorkflowJsonMode = ref(false)

// Custom input state (always shown, no toggle)
const customInputJson = ref('{}')
const inputError = ref<string | null>(null)
const schemaValidationErrors = ref<Array<{ field: string; message: string }>>([])

// Input values for DynamicConfigForm
const workflowInputValues = ref<Record<string, unknown>>({})
const stepInputValues = ref<Record<string, unknown>>({})
const workflowFormValid = ref(true)
const stepFormValid = ref(true)

// Stored input composable for persisting step inputs
const storedInput = useStoredInput({ keyPrefix: `aio:input:${props.workflowId}` })

// Load saved input when step changes
watch(() => props.step?.id, (newStepId) => {
  if (newStepId) {
    const saved = storedInput.load(newStepId)
    if (saved) {
      stepInputValues.value = saved
      customInputJson.value = JSON.stringify(saved, null, 2)
    } else {
      stepInputValues.value = {}
      customInputJson.value = '{}'
    }
  }
}, { immediate: true })

// Get the latest step run output for current step
const latestStepRunOutput = computed(() => {
  if (!props.step || !testRuns.value.length) return null

  // Find the latest completed step run for this step
  for (const run of testRuns.value) {
    if (run.step_runs) {
      const stepRun = run.step_runs.find(sr =>
        sr.step_id === props.step!.id &&
        sr.status === 'completed' &&
        sr.output
      )
      if (stepRun?.output) {
        return stepRun.output
      }
    }
  }
  return null
})

// Use previous output as input
function usePreviousOutput() {
  if (latestStepRunOutput.value) {
    const output = latestStepRunOutput.value as Record<string, unknown>
    stepInputValues.value = output
    customInputJson.value = JSON.stringify(output, null, 2)
    toast.success(t('execution.usedPreviousOutput'))
  }
}

// Clear input
function clearInput() {
  stepInputValues.value = {}
  workflowInputValues.value = {}
  customInputJson.value = '{}'
  inputError.value = null
  if (props.step?.id) {
    storedInput.clear(props.step.id)
  }
}

// Toggle between form and JSON mode (for step execution)
function toggleInputMode() {
  if (useJsonMode.value) {
    // Switching from JSON to Form - parse JSON and set form values
    try {
      const parsed = JSON.parse(customInputJson.value)
      stepInputValues.value = parsed
      inputError.value = null
    } catch {
      inputError.value = t('execution.errors.invalidJson')
      return // Don't switch if JSON is invalid
    }
  } else {
    // Switching from Form to JSON - serialize form values
    customInputJson.value = JSON.stringify(stepInputValues.value, null, 2)
  }
  useJsonMode.value = !useJsonMode.value
}

// Toggle between form and JSON mode (for workflow execution)
function toggleWorkflowInputMode() {
  if (useWorkflowJsonMode.value) {
    // Switching from JSON to Form - parse JSON and set form values
    try {
      const parsed = JSON.parse(customInputJson.value)
      workflowInputValues.value = parsed
      inputError.value = null
    } catch {
      inputError.value = t('execution.errors.invalidJson')
      return // Don't switch if JSON is invalid
    }
  } else {
    // Switching from Form to JSON - serialize form values
    customInputJson.value = JSON.stringify(workflowInputValues.value, null, 2)
  }
  useWorkflowJsonMode.value = !useWorkflowJsonMode.value
}

// Get previous step in the workflow (for autocomplete)
const previousStep = computed(() => {
  if (!props.step) return null
  // Find edge that targets current step
  const incomingEdge = props.edges.find(e => e.target_step_id === props.step!.id)
  if (!incomingEdge) return null
  return props.steps.find(s => s.id === incomingEdge.source_step_id) || null
})

// Get previous step's output from latest run (for autocomplete)
const previousStepOutput = computed(() => {
  if (!previousStep.value || !testRuns.value.length) return null

  for (const run of testRuns.value) {
    if (run.step_runs) {
      const stepRun = run.step_runs.find(sr =>
        sr.step_id === previousStep.value!.id &&
        sr.status === 'completed' &&
        sr.output
      )
      if (stepRun?.output) {
        return stepRun.output as Record<string, unknown>
      }
    }
  }
  return null
})

// Get suggested fields from previous step output
const suggestedFields = computed<Array<{ name: string; value: unknown; type: string }>>(() => {
  const output = previousStepOutput.value
  if (!output || typeof output !== 'object') return []

  return Object.entries(output).map(([name, value]) => ({
    name,
    value,
    type: Array.isArray(value) ? 'array' : typeof value
  }))
})

// Insert field into JSON input
function insertSuggestedField(fieldName: string, value: unknown) {
  try {
    const current = JSON.parse(customInputJson.value)
    current[fieldName] = value
    customInputJson.value = JSON.stringify(current, null, 2)
    stepInputValues.value = current
    toast.success(t('execution.fieldInserted', { field: fieldName }))
  } catch {
    // If JSON is invalid, start fresh
    const newObj: Record<string, unknown> = {}
    newObj[fieldName] = value
    customInputJson.value = JSON.stringify(newObj, null, 2)
    stepInputValues.value = newObj
  }
}

// Find the start step
// If a Start step is selected, use that one; otherwise fall back to the first Start step
const startStep = computed(() => {
  if (props.step?.type === 'start') {
    return props.step
  }
  return props.steps.find(s => s.type === 'start')
})

// Find the first executable step (after start)
const firstExecutableStep = computed(() => {
  if (!startStep.value) return null
  const edge = props.edges.find(e => e.source_step_id === startStep.value!.id)
  if (!edge) return null
  return props.steps.find(s => s.id === edge.target_step_id) || null
})

// Get the block definition for the first executable step (for workflow execution)
const firstStepBlock = computed(() => {
  if (!firstExecutableStep.value) return null
  return props.blocks.find(b => b.slug === firstExecutableStep.value!.type) || null
})

// Check if the selected step is a start step
const isStartStep = computed(() => props.step?.type === 'start')

// Get the block definition for the selected step (for step execution)
// If start step is selected, use the first executable step's block
const selectedStepBlock = computed(() => {
  if (!props.step) return null
  // For start step, use the first executable step's block definition
  if (isStartStep.value) {
    return firstStepBlock.value
  }
  return props.blocks.find(b => b.slug === props.step!.type) || null
})

// Get the effective step for display (used in descriptions)
const effectiveStep = computed(() => {
  if (isStartStep.value) {
    return firstExecutableStep.value
  }
  return props.step
})

// Convert input_schema to ConfigSchema format for workflow execution
// Derived from Start step's config.input_schema (user-defined workflow input schema)
const workflowInputSchema = computed<ConfigSchema | null>(() => {
  // Use Start step's config.input_schema (user-defined workflow inputs)
  if (!startStep.value?.config) return null

  const config = startStep.value.config as Record<string, unknown>
  const schema = config.input_schema as Record<string, unknown> | undefined
  if (!schema || schema.type !== 'object') return null

  const properties = schema.properties as Record<string, unknown> | undefined
  if (!properties || Object.keys(properties).length === 0) return null

  return {
    type: 'object',
    properties: properties,
    required: (schema.required as string[]) || [],
  } as ConfigSchema
})

// Convert config_schema to ConfigSchema format for step execution
// For Start step, use the workflow input schema (from Start step's config.input_schema)
const stepInputSchema = computed<ConfigSchema | null>(() => {
  // For Start step, use workflowInputSchema (which comes from Start step's config.input_schema)
  if (isStartStep.value) {
    return workflowInputSchema.value
  }
  const schema = selectedStepBlock.value?.config_schema as Record<string, unknown> | undefined
  if (!schema || schema.type !== 'object') return null
  const properties = schema.properties as Record<string, unknown> | undefined
  if (!properties || Object.keys(properties).length === 0) return null
  return {
    type: 'object',
    properties: properties || {},
    required: (schema.required as string[]) || [],
  } as ConfigSchema
})

// Check if workflow has input fields
const hasWorkflowInputFields = computed(() => {
  if (!workflowInputSchema.value?.properties) return false
  return Object.keys(workflowInputSchema.value.properties).length > 0
})

// Check if step has input fields
// For Start step, check workflowInputSchema instead
const hasStepInputFields = computed(() => {
  if (isStartStep.value) {
    return hasWorkflowInputFields.value
  }
  if (!stepInputSchema.value?.properties) return false
  return Object.keys(stepInputSchema.value.properties).length > 0
})

// Real-time validation on JSON input change
watch(customInputJson, () => {
  if (useJsonMode.value || !hasStepInputFields.value) {
    try {
      const parsed = JSON.parse(customInputJson.value)
      const schema = selectedStepBlock.value?.config_schema as ConfigSchema | undefined
      const result = validateConfig(schema, parsed)
      schemaValidationErrors.value = result.errors.map(e => ({ field: e.field, message: e.message }))
      inputError.value = null
    } catch {
      schemaValidationErrors.value = []
      // Only show parse error if they have typed something
      if (customInputJson.value.trim() !== '' && customInputJson.value.trim() !== '{}') {
        inputError.value = t('execution.errors.invalidJson')
      }
    }
  }
}, { immediate: true })

// Schema preview information for JSON input fallback
interface SchemaField {
  name: string
  type: string
  description: string
  required: boolean
}

// Get schema fields for preview
function getSchemaFields(schema: Record<string, unknown> | undefined | null): SchemaField[] {
  if (!schema) return []
  const properties = schema.properties as Record<string, Record<string, unknown>> | undefined
  if (!properties) return []

  const required = (schema.required as string[]) || []

  return Object.entries(properties).map(([name, prop]) => ({
    name,
    type: String(prop.type || 'any'),
    description: String(prop.description || ''),
    required: required.includes(name),
  }))
}

// Get workflow schema fields for preview
const workflowSchemaFields = computed(() => {
  // Use Start step's config.input_schema (user-defined workflow inputs)
  if (!startStep.value?.config) return []

  const config = startStep.value.config as Record<string, unknown>
  const schema = config.input_schema as Record<string, unknown> | undefined
  return getSchemaFields(schema)
})

// Get step schema fields for preview
const stepSchemaFields = computed(() => {
  const schema = selectedStepBlock.value?.config_schema as Record<string, unknown> | undefined
  return getSchemaFields(schema)
})

// Generate example JSON from schema
function generateExampleJson(fields: SchemaField[]): string {
  if (fields.length === 0) return '{}'

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

// Type badge color
function getTypeBadgeClass(type: string): string {
  switch (type) {
    case 'string': return 'type-string'
    case 'number':
    case 'integer': return 'type-number'
    case 'boolean': return 'type-boolean'
    case 'array': return 'type-array'
    case 'object': return 'type-object'
    default: return 'type-any'
  }
}

// Template variables detection and preview using composable
const stepConfigRef = computed(() => props.step?.config as Record<string, unknown> | undefined)
const {
  variables: templateVariables,
  formatVariable: formatTemplateVar,
  createPreview: createTemplatePreview,
} = useTemplateVariables(stepConfigRef)

// Template preview with resolved values
const templatePreview = computed(() => {
  const input = useJsonMode.value || !hasStepInputFields.value
    ? (() => { try { return JSON.parse(customInputJson.value) } catch { return {} } })()
    : stepInputValues.value
  return createTemplatePreview(input)
})

// Workflow test runs (shared for both step-selected and no-step modes)
const testRuns = ref<Run[]>([])
const loadingTestRuns = ref(false)

// Parse and validate custom input
function parseCustomInput(): object | null {
  inputError.value = null

  try {
    const parsed = JSON.parse(customInputJson.value)
    if (typeof parsed !== 'object' || parsed === null) {
      inputError.value = t('execution.errors.invalidJsonObject')
      return null
    }
    return parsed
  } catch {
    inputError.value = t('execution.errors.invalidJson')
    return null
  }
}

// Get input for workflow execution
function getWorkflowInput(): object | null {
  if (hasWorkflowInputFields.value) {
    // Use DynamicConfigForm values
    if (!workflowFormValid.value) {
      inputError.value = t('execution.errors.invalidForm')
      return null
    }
    inputError.value = null
    return workflowInputValues.value
  } else {
    // Fall back to JSON input
    return parseCustomInput()
  }
}

// Get input for step execution
function getStepInput(): object | null {
  let input: object | null = null

  if (useJsonMode.value || !hasStepInputFields.value) {
    // Use JSON input
    input = parseCustomInput()
  } else {
    // Use DynamicConfigForm values
    if (!stepFormValid.value) {
      inputError.value = t('execution.errors.invalidForm')
      return null
    }
    inputError.value = null
    input = stepInputValues.value
  }

  // Save input to storage for reuse
  if (input && props.step?.id) {
    storedInput.save(props.step.id, input as Record<string, unknown>)
  }

  return input
}

// Execute workflow (test mode)
async function executeWorkflow() {
  const input = getWorkflowInput()
  if (input === null) return

  if (!startStep.value) {
    toast.error(t('execution.errors.noStartStep'))
    return
  }

  executing.value = true

  try {
    const response = await runsApi.create(props.workflowId, {
      triggered_by: 'test',
      input: Object.keys(input).length > 0 ? input : {},
      start_step_id: startStep.value.id,
    })

    toast.success(t('execution.workflowStarted'))

    // Fetch full run details and emit event
    const detailedRun = await runsApi.get(response.data.id)
    emit('run:created', detailedRun.data)

    // Reload test runs after execution
    await loadTestRuns()
  } catch (e) {
    toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    executing.value = false
  }
}

// Execute this step only
async function executeThisStepOnly() {
  if (!props.step) {
    toast.error(t('execution.errors.noStepSelected'))
    return
  }

  const input = getStepInput()
  if (input === null) return

  executing.value = true

  try {
    let runId: string

    if (props.latestRun) {
      // Use existing run for re-execution
      await runsApi.executeSingleStep(
        props.latestRun.id,
        props.step.id,
        Object.keys(input).length > 0 ? input : undefined
      )
      runId = props.latestRun.id
      toast.success(t('execution.stepExecuted'))
    } else {
      // Use inline test API (creates new test run)
      const response = await runsApi.testStepInline(
        props.workflowId,
        props.step.id,
        Object.keys(input).length > 0 ? input : undefined
      )
      runId = response.data.run_id
      toast.success(t('execution.stepQueued'))

      // Start polling for result
      startPolling(response.data.run_id, props.step.id)
    }

    // Fetch full run details and emit event to transition to Run detail panel
    const detailedRun = await runsApi.get(runId)
    emit('run:created', detailedRun.data)

    await loadTestRuns()
  } catch (e) {
    toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    executing.value = false
  }
}

// Execute from this step (run workflow starting from this step)
async function executeFromThisStep() {
  if (!props.step) {
    toast.error(t('execution.errors.noStepSelected'))
    return
  }

  const input = getStepInput()
  if (input === null) return

  executing.value = true

  try {
    const response = await runsApi.create(props.workflowId, {
      triggered_by: 'test',
      input: Object.keys(input).length > 0 ? input : {},
      start_step_id: props.step.id,
    })

    toast.success(t('execution.workflowStarted'))

    // Fetch full run details and emit event
    const detailedRun = await runsApi.get(response.data.id)
    emit('run:created', detailedRun.data)

    // Reload test runs after execution
    await loadTestRuns()
  } catch (e) {
    toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    executing.value = false
  }
}

// Poll for inline test result using composable
const { pollingId: pollingRunId, start: startPollingInternal } = usePolling<Run>({
  interval: 1000,
  maxAttempts: 60,
  onTimeout: () => toast.warning(t('execution.pollingTimeout')),
})

async function startPolling(runId: string, stepId: string) {
  startPollingInternal(
    runId,
    async () => {
      const response = await runsApi.get(runId)
      return response.data
    },
    (run) => {
      // Always emit the updated run to keep Run Detail Panel in sync
      emit('run:created', run)

      // Check if step has completed
      const stepRun = run.step_runs?.find((sr: { step_id: string }) => sr.step_id === stepId)
      if (stepRun) {
        if (stepRun.status === 'completed') {
          toast.success(t('execution.stepTestCompleted'))
          loadTestRuns()
          return true // Stop polling
        } else if (stepRun.status === 'failed') {
          toast.error(t('execution.stepTestFailed'))
          loadTestRuns()
          return true // Stop polling
        }
      }
      return false // Continue polling
    }
  )
}

// Load workflow test runs (all test triggered runs)
async function loadTestRuns() {
  loadingTestRuns.value = true
  try {
    // Get all runs without limit (or large limit)
    const response = await runsApi.list(props.workflowId, { limit: 1000 })
    const allRuns = response.data || []

    // Filter to test triggered runs only and fetch detailed data
    const testModeRuns = allRuns.filter(run => run.triggered_by === 'test')

    // Fetch detailed run data with step_runs for each run
    const detailedRuns: Run[] = []
    for (const run of testModeRuns) {
      try {
        const detailedResponse = await runsApi.get(run.id)
        detailedRuns.push(detailedResponse.data)
      } catch {
        // If detail fetch fails, use the basic run info
        detailedRuns.push(run)
      }
    }

    testRuns.value = detailedRuns
  } catch (e) {
    console.error('Failed to load test runs:', e)
  } finally {
    loadingTestRuns.value = false
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
      <template v-if="step && !isStartStep">
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
                @click="usePreviousOutput"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="1 4 1 10 7 10"/>
                  <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"/>
                </svg>
              </button>
              <!-- Clear Button -->
              <button
                class="btn-icon"
                :title="t('execution.clearInput')"
                @click="clearInput"
              >
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
                @click="toggleInputMode"
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
              v-model="stepInputValues"
              :schema="stepInputSchema"
              :disabled="executing"
              @validation-change="(valid) => stepFormValid = valid"
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
              v-model="customInputJson"
              class="json-input"
              rows="4"
              :placeholder="stepSchemaFields.length > 0 ? generateExampleJson(stepSchemaFields) : t('execution.inputPlaceholder')"
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
                @click="insertSuggestedField(field.name, field.value)"
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
            @click="executeThisStepOnly"
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
            @click="executeFromThisStep"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
            </svg>
            {{ t('execution.executeFromThisStep') }}
          </button>
        </div>

      </template>

      <!-- No Step Selected OR Start Step: Show workflow execution controls and history -->
      <template v-else>
        <!-- Workflow Execution Section -->
        <div class="workflow-execution-section">
          <!-- Input Section (only shown when workflow has input fields) -->
          <div v-if="hasWorkflowInputFields" class="input-section">
            <div class="input-header">
              <label class="input-label">{{ t('execution.customInput') }}</label>
              <div class="input-actions">
                <!-- Clear Button -->
                <button
                  class="btn-icon"
                  :title="t('execution.clearInput')"
                  @click="clearInput"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M3 6h18"/>
                    <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
                  </svg>
                </button>
                <!-- Toggle Form/JSON Mode -->
                <button
                  v-if="hasWorkflowInputFields"
                  class="btn-icon"
                  :class="{ active: useWorkflowJsonMode }"
                  :title="useWorkflowJsonMode ? t('execution.switchToForm') : t('execution.switchToJson')"
                  @click="toggleWorkflowInputMode"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="16 18 22 12 16 6"/>
                    <polyline points="8 6 2 12 8 18"/>
                  </svg>
                </button>
              </div>
            </div>

            <!-- Dynamic Form (when input_schema is available and not in JSON mode) -->
            <template v-if="hasWorkflowInputFields && !useWorkflowJsonMode">
              <p class="input-description">
                {{ t('execution.inputDescription', { stepName: firstExecutableStep?.name }) }}
              </p>
              <DynamicConfigForm
                v-model="workflowInputValues"
                :schema="workflowInputSchema"
                :disabled="executing"
                @validation-change="(valid) => workflowFormValid = valid"
              />
            </template>

            <!-- JSON Textarea (when no schema, or JSON mode is active) -->
            <template v-else>
              <!-- Schema Preview (if we have schema info) -->
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
                v-model="customInputJson"
                class="json-input"
                rows="4"
                :placeholder="workflowSchemaFields.length > 0 ? generateExampleJson(workflowSchemaFields) : t('execution.inputPlaceholder')"
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
            @click="executeWorkflow"
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

/* Execution Content */
.execution-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

/* Section Title */
.section-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Workflow Execution Section */
.workflow-execution-section {
  padding-bottom: 1rem;
  border-bottom: 1px solid var(--color-border);
}

/* Input Section */
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

/* Info Banner */
.info-banner {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
  font-size: 0.75rem;
  color: #1e40af;
}

.info-banner svg {
  flex-shrink: 0;
  margin-top: 1px;
}

.info-banner-subtle {
  background: #f8fafc;
  border-color: #e2e8f0;
  color: #475569;
}

/* Schema Preview */
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

/* Type Badges */
.type-badge {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.type-string {
  background: #dcfce7;
  color: #16a34a;
}

.type-number {
  background: #dbeafe;
  color: #2563eb;
}

.type-boolean {
  background: #fef3c7;
  color: #d97706;
}

.type-array {
  background: #f3e8ff;
  color: #9333ea;
}

.type-object {
  background: #fce7f3;
  color: #db2777;
}

.type-any {
  background: #f3f4f6;
  color: #6b7280;
}

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

/* Validation Errors */
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

.validation-error svg {
  flex-shrink: 0;
}

/* Suggested Fields */
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

.chip-type.type-string {
  background: #dcfce7;
  color: #16a34a;
}

.chip-type.type-number {
  background: #dbeafe;
  color: #2563eb;
}

.chip-type.type-boolean {
  background: #fef3c7;
  color: #d97706;
}

.chip-type.type-array {
  background: #f3e8ff;
  color: #9333ea;
}

.chip-type.type-object {
  background: #fce7f3;
  color: #db2777;
}

/* Template Preview */
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

.template-arrow {
  color: var(--color-text-secondary);
}

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

/* Execution Buttons */
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

.full-width {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.info-text {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0;
  padding: 0.5rem;
  background: #f0f9ff;
  border-radius: 4px;
  border: 1px solid #bae6fd;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Step History */
.step-history {
  margin-top: 0.5rem;
}

.history-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 0.5rem 0;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.5rem;
  background: var(--color-background);
  border-radius: 4px;
  font-size: 0.6875rem;
  cursor: pointer;
  transition: background 0.15s;
}

.history-item:hover {
  background: var(--color-surface);
}

/* Status Badges */
.status-badge {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.status-badge.completed {
  background: #dcfce7;
  color: #16a34a;
}

.status-badge.failed {
  background: #fee2e2;
  color: #dc2626;
}

.status-badge.running {
  background: #dbeafe;
  color: #2563eb;
}

.status-badge.pending {
  background: #f3f4f6;
  color: #6b7280;
}

/* Action Buttons */
.action-buttons {
  display: flex;
  justify-content: flex-end;
}

.action-buttons .btn {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
}

.text-right {
  text-align: right;
}

.text-secondary {
  color: var(--color-text-secondary);
}

.text-sm {
  font-size: 0.6875rem;
}

/* Scrollbar */
.execution-content::-webkit-scrollbar,
.table-container::-webkit-scrollbar {
  width: 6px;
}

.execution-content::-webkit-scrollbar-track,
.table-container::-webkit-scrollbar-track {
  background: transparent;
}

.execution-content::-webkit-scrollbar-thumb,
.table-container::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}
</style>
