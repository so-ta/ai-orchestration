<script setup lang="ts">
import type { Step, Run, StepRun, BlockDefinition } from '~/types/api'
import type { ConfigSchema } from './config/types/config-schema'
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

// Schema validation function
function validateAgainstSchema(data: Record<string, unknown>, schema: Record<string, unknown> | null | undefined): Array<{ field: string; message: string }> {
  const errors: Array<{ field: string; message: string }> = []
  if (!schema) return errors

  const properties = schema.properties as Record<string, Record<string, unknown>> | undefined
  const required = (schema.required as string[]) || []

  if (!properties) return errors

  // Check required fields
  for (const fieldName of required) {
    if (data[fieldName] === undefined || data[fieldName] === null || data[fieldName] === '') {
      errors.push({
        field: fieldName,
        message: t('execution.validation.required', { field: fieldName })
      })
    }
  }

  // Check types
  for (const [fieldName, fieldValue] of Object.entries(data)) {
    const fieldSchema = properties[fieldName]
    if (!fieldSchema) continue

    const expectedType = fieldSchema.type as string
    if (!expectedType || expectedType === 'any') continue

    const actualType = Array.isArray(fieldValue) ? 'array' : typeof fieldValue
    let valid = false

    switch (expectedType) {
      case 'string':
        valid = actualType === 'string'
        break
      case 'number':
      case 'integer':
        valid = actualType === 'number'
        break
      case 'boolean':
        valid = actualType === 'boolean'
        break
      case 'array':
        valid = actualType === 'array'
        break
      case 'object':
        valid = actualType === 'object' && fieldValue !== null && !Array.isArray(fieldValue)
        break
      default:
        valid = true
    }

    if (!valid && fieldValue !== undefined && fieldValue !== null && fieldValue !== '') {
      errors.push({
        field: fieldName,
        message: t('execution.validation.typeMismatch', { field: fieldName, expected: expectedType, actual: actualType })
      })
    }
  }

  return errors
}

// Input values for DynamicConfigForm
const workflowInputValues = ref<Record<string, unknown>>({})
const stepInputValues = ref<Record<string, unknown>>({})
const workflowFormValid = ref(true)
const stepFormValid = ref(true)

// Storage key for remembering inputs
function getStorageKey(stepId: string): string {
  return `aio:input:${props.workflowId}:${stepId}`
}

// Save input to localStorage
function saveInputToStorage(stepId: string, input: Record<string, unknown>) {
  try {
    localStorage.setItem(getStorageKey(stepId), JSON.stringify(input))
  } catch {
    // Ignore storage errors
  }
}

// Load input from localStorage
function loadInputFromStorage(stepId: string): Record<string, unknown> | null {
  try {
    const stored = localStorage.getItem(getStorageKey(stepId))
    if (stored) {
      return JSON.parse(stored)
    }
  } catch {
    // Ignore storage errors
  }
  return null
}

// Clear stored input
function clearStoredInput(stepId: string) {
  try {
    localStorage.removeItem(getStorageKey(stepId))
  } catch {
    // Ignore storage errors
  }
}

// Load saved input when step changes
watch(() => props.step?.id, (newStepId) => {
  if (newStepId) {
    const saved = loadInputFromStorage(newStepId)
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
    clearStoredInput(props.step.id)
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
const startStep = computed(() => props.steps.find(s => s.type === 'start'))

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

// Convert input_schema to ConfigSchema format for step execution
// For Start step, use the workflow input schema (from Start step's config.input_schema)
const stepInputSchema = computed<ConfigSchema | null>(() => {
  // For Start step, use workflowInputSchema (which comes from Start step's config.input_schema)
  if (isStartStep.value) {
    return workflowInputSchema.value
  }
  const schema = selectedStepBlock.value?.input_schema as Record<string, unknown> | undefined
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
      const schema = selectedStepBlock.value?.input_schema as Record<string, unknown> | undefined
      schemaValidationErrors.value = validateAgainstSchema(parsed, schema)
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
  const schema = selectedStepBlock.value?.input_schema as Record<string, unknown> | undefined
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

// Template variables detection and preview
const templateVariables = computed<string[]>(() => {
  if (!props.step?.config) return []
  const config = props.step.config as Record<string, unknown>
  const variables = new Set<string>()

  // Find all double-brace variable patterns in config values
  const findVariables = (obj: unknown) => {
    if (typeof obj === 'string') {
      const matches = obj.match(/\{\{([^}]+)\}\}/g)
      if (matches) {
        for (const match of matches) {
          const varName = match.replace(/\{\{|\}\}/g, '').trim()
          variables.add(varName)
        }
      }
    } else if (Array.isArray(obj)) {
      for (const item of obj) findVariables(item)
    } else if (obj && typeof obj === 'object') {
      for (const value of Object.values(obj)) findVariables(value)
    }
  }

  findVariables(config)
  return Array.from(variables)
})

// Resolve template variable value from input
function resolveTemplateVariable(varPath: string, input: Record<string, unknown>): string {
  // Handle paths like "message" or "data.content"
  const parts = varPath.split('.')
  let value: unknown = input
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)

  for (const part of parts) {
    if (value && typeof value === 'object') {
      value = (value as Record<string, unknown>)[part]
    } else {
      return openBrace + varPath + closeBrace
    }
  }

  if (value === undefined || value === null) {
    return openBrace + varPath + closeBrace
  }

  if (typeof value === 'object') {
    return JSON.stringify(value)
  }

  return String(value)
}

// Format template variable for display (avoids i18n parsing issues)
function formatTemplateVar(variable: string): string {
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)
  return openBrace + variable + closeBrace
}

// Template preview with resolved values
const templatePreview = computed<Array<{ variable: string; resolved: string; isResolved: boolean }>>(() => {
  const input = useJsonMode.value || !hasStepInputFields.value
    ? (() => { try { return JSON.parse(customInputJson.value) } catch { return {} } })()
    : stepInputValues.value
  const unresolvedPrefix = String.fromCharCode(123, 123)

  return templateVariables.value.map(variable => {
    const resolved = resolveTemplateVariable(variable, input)
    return {
      variable,
      resolved,
      isResolved: !resolved.startsWith(unresolvedPrefix)
    }
  })
})

// Workflow test runs (shared for both step-selected and no-step modes)
const testRuns = ref<Run[]>([])
const loadingTestRuns = ref(false)

// Modal state for viewing step run details
const selectedStepRun = ref<StepRun | null>(null)
const showStepRunModal = ref(false)

// Open modal with step run details
function openStepRunModal(stepRun: StepRun) {
  selectedStepRun.value = stepRun
  showStepRunModal.value = true
}

// Close modal
function closeStepRunModal() {
  showStepRunModal.value = false
  selectedStepRun.value = null
}

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
  } catch (e) {
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
    saveInputToStorage(props.step.id, input as Record<string, unknown>)
  }

  return input
}

// Execute workflow (test mode)
async function executeWorkflow() {
  const input = getWorkflowInput()
  if (input === null) return

  executing.value = true

  try {
    await runsApi.create(props.workflowId, {
      triggered_by: 'test',
      input: Object.keys(input).length > 0 ? input : {},
    })

    toast.success(t('execution.workflowStarted'))
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
    if (props.latestRun) {
      // Use existing run for re-execution
      await runsApi.executeSingleStep(
        props.latestRun.id,
        props.step.id,
        Object.keys(input).length > 0 ? input : undefined
      )

      toast.success(t('execution.stepExecuted'))
      await loadTestRuns()
    } else {
      // Use inline test API (creates new test run)
      const response = await runsApi.testStepInline(
        props.workflowId,
        props.step.id,
        Object.keys(input).length > 0 ? input : undefined
      )

      toast.success(t('execution.stepQueued'))

      // Start polling for result
      startPolling(response.data.run_id, props.step.id)
    }
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
    await runsApi.create(props.workflowId, {
      triggered_by: 'test',
      input: Object.keys(input).length > 0 ? input : {},
    })

    toast.success(t('execution.workflowStarted'))
    // Reload test runs after execution
    await loadTestRuns()
  } catch (e) {
    toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    executing.value = false
  }
}

// Poll for inline test result
const pollingInterval = ref<ReturnType<typeof setInterval> | null>(null)
const pollingRunId = ref<string | null>(null)

async function startPolling(runId: string, stepId: string) {
  pollingRunId.value = runId
  let attempts = 0
  const maxAttempts = 60 // 60 seconds timeout

  pollingInterval.value = setInterval(async () => {
    attempts++
    if (attempts > maxAttempts) {
      stopPolling()
      toast.warning(t('execution.pollingTimeout'))
      return
    }

    try {
      const response = await runsApi.get(runId)
      const run = response.data

      // Check if step has completed
      const stepRun = run.step_runs?.find((sr: { step_id: string }) => sr.step_id === stepId)
      if (stepRun) {
        if (stepRun.status === 'completed') {
          stopPolling()
          toast.success(t('execution.stepTestCompleted'))
          await loadTestRuns()
        } else if (stepRun.status === 'failed') {
          stopPolling()
          toast.error(t('execution.stepTestFailed'))
          await loadTestRuns()
        }
      }
    } catch (e) {
      console.error('Polling error:', e)
    }
  }, 1000)
}

function stopPolling() {
  if (pollingInterval.value) {
    clearInterval(pollingInterval.value)
    pollingInterval.value = null
  }
  pollingRunId.value = null
}

// Cleanup on unmount
onUnmounted(() => {
  stopPolling()
})

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

// Flattened step runs from all test runs
interface StepRunWithRunInfo extends StepRun {
  run_id: string
  workflow_version: number
  run_number: number
  run_status: string
}

const allTestStepRuns = computed<StepRunWithRunInfo[]>(() => {
  const stepRuns: StepRunWithRunInfo[] = []
  for (const run of testRuns.value) {
    if (run.step_runs) {
      for (const stepRun of run.step_runs) {
        stepRuns.push({
          ...stepRun,
          run_id: run.id,
          workflow_version: run.workflow_version,
          run_number: run.run_number,
          run_status: run.status,
        })
      }
    }
  }
  // Sort by created_at descending (newest first)
  return stepRuns.sort((a, b) => {
    const dateA = new Date(a.completed_at || a.started_at || a.created_at).getTime()
    const dateB = new Date(b.completed_at || b.started_at || b.created_at).getTime()
    return dateB - dateA
  })
})


// Format step status
function formatStatus(status: string): string {
  return t(`execution.status.${status}`) || status
}

// Format duration
function formatDuration(ms?: number): string {
  if (!ms) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

// Format date
function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return t('execution.justNow')
  if (diff < 3600000) return t('execution.minutesAgo', { n: Math.floor(diff / 60000) })
  if (diff < 86400000) return t('execution.hoursAgo', { n: Math.floor(diff / 3600000) })
  return date.toLocaleString()
}

// Calculate step duration
function calculateStepDuration(stepRun: StepRunWithRunInfo): string {
  if (!stepRun.started_at) return '-'
  const start = new Date(stepRun.started_at).getTime()
  const end = stepRun.completed_at ? new Date(stepRun.completed_at).getTime() : Date.now()
  const ms = end - start

  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
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
        <!-- Input Section -->
        <div class="input-section">
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
                  <polyline points="1 4 1 10 7 10"></polyline>
                  <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"></path>
                </svg>
              </button>
              <!-- Clear Button -->
              <button
                class="btn-icon"
                :title="t('execution.clearInput')"
                @click="clearInput"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 6h18"></path>
                  <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"></path>
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
                  <polyline points="16 18 22 12 16 6"></polyline>
                  <polyline points="8 6 2 12 8 18"></polyline>
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
                  <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
                  <polyline points="14 2 14 8 20 8"></polyline>
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
            ></textarea>

            <p v-if="stepSchemaFields.length > 0" class="json-hint">
              {{ t('execution.jsonHint') }}
            </p>
          </template>

          <!-- Schema Validation Errors -->
          <div v-if="schemaValidationErrors.length > 0" class="validation-errors">
            <div v-for="error in schemaValidationErrors" :key="error.field" class="validation-error">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="15" y1="9" x2="9" y2="15"></line>
                <line x1="9" y1="9" x2="15" y2="15"></line>
              </svg>
              <span>{{ error.message }}</span>
            </div>
          </div>

          <p v-if="inputError" class="input-error">{{ inputError }}</p>

          <!-- Suggested Fields from Previous Step -->
          <div v-if="suggestedFields.length > 0 && (useJsonMode || !hasStepInputFields)" class="suggested-fields">
            <div class="suggested-header">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
              </svg>
              <span>{{ t('execution.suggestedFields') }} ({{ previousStep?.name }})</span>
            </div>
            <div class="suggested-chips">
              <button
                v-for="field in suggestedFields"
                :key="field.name"
                class="suggested-chip"
                @click="insertSuggestedField(field.name, field.value)"
                :title="JSON.stringify(field.value, null, 2)"
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
                <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
                <path d="M12 18v-6"></path>
                <path d="M8 15h8"></path>
              </svg>
              <span>{{ t('execution.templatePreview') }}</span>
            </div>
            <div class="template-items">
              <div v-for="item in templatePreview" :key="item.variable" class="template-item">
                <code class="template-variable" v-text="formatTemplateVar(item.variable)"></code>
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
              <polygon points="5 3 19 12 5 21 5 3"></polygon>
            </svg>
            <svg v-else class="spinning" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
            </svg>
            {{ pollingRunId ? t('execution.waitingForResult') : (executing ? t('execution.executing') : t('execution.executeThisStepOnly')) }}
          </button>
          <button
            class="btn btn-outline"
            :disabled="executing || !!pollingRunId"
            @click="executeFromThisStep"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
            </svg>
            {{ t('execution.executeFromThisStep') }}
          </button>
        </div>

        <!-- Step Execution History (from all test runs) -->
        <div class="test-run-history">
          <div class="history-header">
            <h4 class="section-title">{{ t('execution.history') }}</h4>
            <button class="btn btn-outline btn-sm" @click="loadTestRuns" :disabled="loadingTestRuns">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="23 4 23 10 17 10"></polyline>
                <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
              </svg>
            </button>
          </div>

          <!-- Loading State -->
          <div v-if="loadingTestRuns" class="loading-state">
            <div class="loading-spinner"></div>
            <p>{{ t('runs.loading') }}</p>
          </div>

          <!-- Empty State -->
          <div v-else-if="allTestStepRuns.length === 0" class="empty-state">
            <div class="empty-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
              </svg>
            </div>
            <p class="empty-title">{{ t('execution.noTestRuns') }}</p>
            <p class="empty-desc">{{ t('execution.noTestRunsDesc') }}</p>
          </div>

          <!-- Step Runs Table -->
          <div v-else class="table-container">
            <table class="history-table">
              <thead>
                <tr>
                  <th>#</th>
                  <th>{{ t('runs.table.status') }}</th>
                  <th>{{ t('runs.table.step') }}</th>
                  <th>{{ t('runs.table.duration') }}</th>
                  <th>{{ t('runs.table.created') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="stepRun in allTestStepRuns" :key="stepRun.id" @click="openStepRunModal(stepRun)" class="clickable-row">
                  <td>
                    <span class="attempt-badge">#{{ stepRun.run_number }}</span>
                  </td>
                  <td>
                    <span :class="['status-badge', stepRun.status]">
                      {{ formatStatus(stepRun.status) }}
                    </span>
                  </td>
                  <td>
                    <span class="step-name">{{ stepRun.step_name }}</span>
                  </td>
                  <td>
                    <span class="duration">{{ calculateStepDuration(stepRun) }}</span>
                  </td>
                  <td class="text-secondary text-sm">
                    {{ formatDate(stepRun.created_at) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>

      <!-- No Step Selected OR Start Step: Show workflow execution controls and history -->
      <template v-else>
        <!-- Workflow Execution Section -->
        <div class="workflow-execution-section">
          <!-- Input Section -->
          <div class="input-section">
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
                    <path d="M3 6h18"></path>
                    <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"></path>
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
                    <polyline points="16 18 22 12 16 6"></polyline>
                    <polyline points="8 6 2 12 8 18"></polyline>
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
                    <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
                    <polyline points="14 2 14 8 20 8"></polyline>
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
              ></textarea>

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
              <polygon points="5 3 19 12 5 21 5 3"></polygon>
            </svg>
            <svg v-else class="spinning" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
            </svg>
            {{ executing ? t('execution.executing') : t('execution.executeWorkflow') }}
          </button>
        </div>

        <!-- Test Run History -->
        <div class="test-run-history">
          <div class="history-header">
            <h4 class="section-title">{{ t('execution.testRunHistory') }}</h4>
            <button class="btn btn-outline btn-sm" @click="loadTestRuns" :disabled="loadingTestRuns">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="23 4 23 10 17 10"></polyline>
                <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
              </svg>
              {{ t('workflows.refresh') }}
            </button>
          </div>

          <!-- Loading State -->
          <div v-if="loadingTestRuns" class="loading-state">
            <div class="loading-spinner"></div>
            <p>{{ t('runs.loading') }}</p>
          </div>

          <!-- Empty State -->
          <div v-else-if="allTestStepRuns.length === 0" class="empty-state">
            <div class="empty-icon">
              <svg xmlns="http://www.w3.org/2000/svg" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
              </svg>
            </div>
            <p class="empty-title">{{ t('execution.noTestRuns') }}</p>
            <p class="empty-desc">{{ t('execution.noTestRunsDesc') }}</p>
          </div>

          <!-- Test Runs Table -->
          <div v-else class="table-container">
            <table class="history-table">
              <thead>
                <tr>
                  <th>#</th>
                  <th>{{ t('runs.table.status') }}</th>
                  <th>{{ t('runs.table.step') }}</th>
                  <th>{{ t('runs.table.duration') }}</th>
                  <th>{{ t('runs.table.created') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="stepRun in allTestStepRuns" :key="stepRun.id" @click="openStepRunModal(stepRun)" class="clickable-row">
                  <td>
                    <span class="attempt-badge">#{{ stepRun.run_number }}</span>
                  </td>
                  <td>
                    <span :class="['status-badge', stepRun.status]">
                      {{ formatStatus(stepRun.status) }}
                    </span>
                  </td>
                  <td>
                    <span class="step-name">{{ stepRun.step_name }}</span>
                  </td>
                  <td>
                    <span class="duration">{{ calculateStepDuration(stepRun) }}</span>
                  </td>
                  <td class="text-secondary text-sm">
                    {{ formatDate(stepRun.created_at) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>

    <!-- Step Run Modal -->
    <StepRunModal
      :step-run="selectedStepRun"
      :show="showStepRunModal"
      @close="closeStepRunModal"
    />
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

/* Attempt Badge */
.attempt-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.25rem;
  height: 1.25rem;
  padding: 0 0.25rem;
  font-size: 0.625rem;
  font-weight: 600;
  background: #e0e7ff;
  color: #4f46e5;
  border-radius: 4px;
}

.history-duration {
  color: var(--color-text-secondary);
}

.history-attempt {
  color: var(--color-text-secondary);
  margin-left: auto;
}

.history-arrow {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

/* Test Run History */
.test-run-history {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.history-header .section-title {
  margin: 0;
}

.history-header .btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2rem 1rem;
  color: var(--color-text-secondary);
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 0.5rem;
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2rem 1rem;
  text-align: center;
}

.empty-icon {
  color: var(--color-text-secondary);
  opacity: 0.4;
  margin-bottom: 0.75rem;
}

.empty-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.empty-desc {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin-top: 0.375rem;
}

/* Table */
.table-container {
  flex: 1;
  overflow-x: auto;
  overflow-y: auto;
}

.history-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.75rem;
}

.history-table th {
  font-size: 0.625rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.5rem;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
  position: sticky;
  top: 0;
}

.history-table td {
  padding: 0.5rem;
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}

.clickable-row {
  cursor: pointer;
  transition: background 0.15s;
}

.clickable-row:hover {
  background: var(--color-surface);
}

.step-name {
  font-weight: 500;
  color: var(--color-text);
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}

.duration {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.6875rem;
  color: var(--color-text);
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
