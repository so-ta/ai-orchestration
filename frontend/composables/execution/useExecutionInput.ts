import type { ConfigSchema } from '~/components/workflow-editor/config/types/config-schema'
import { validateConfig } from '~/components/workflow-editor/config/composables/useValidation'
import { useStoredInput } from '~/composables/useStoredInput'

interface ExecutionInputOptions {
  workflowId: string
  stepId: Ref<string | undefined>
  hasStepInputFields: Ref<boolean>
  hasWorkflowInputFields: Ref<boolean>
  selectedStepBlock: Ref<{ config_schema?: unknown } | null>
}

/**
 * Composable for execution input state management
 */
export function useExecutionInput(options: ExecutionInputOptions) {
  const { workflowId, stepId, hasStepInputFields, hasWorkflowInputFields, selectedStepBlock } = options

  const { t } = useI18n()

  // Input mode toggle (form vs json)
  const useJsonMode = ref(false)
  const useWorkflowJsonMode = ref(false)

  // Custom input state
  const customInputJson = ref('{}')
  const inputError = ref<string | null>(null)
  const schemaValidationErrors = ref<Array<{ field: string; message: string }>>([])

  // Input values for DynamicConfigForm
  const workflowInputValues = ref<Record<string, unknown>>({})
  const stepInputValues = ref<Record<string, unknown>>({})
  const workflowFormValid = ref(true)
  const stepFormValid = ref(true)

  // Stored input composable for persisting step inputs
  const storedInput = useStoredInput({ keyPrefix: `aio:input:${workflowId}` })

  // Load saved input when step changes
  watch(stepId, (newStepId) => {
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

  /**
   * Parse and validate custom input JSON
   */
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

  /**
   * Get input for workflow execution
   */
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

  /**
   * Get input for step execution
   */
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
    if (input && stepId.value) {
      storedInput.save(stepId.value, input as Record<string, unknown>)
    }

    return input
  }

  /**
   * Toggle between form and JSON mode (for step execution)
   */
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

  /**
   * Toggle between form and JSON mode (for workflow execution)
   */
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

  /**
   * Use previous output as input
   */
  function usePreviousOutput(output: Record<string, unknown>) {
    stepInputValues.value = output
    customInputJson.value = JSON.stringify(output, null, 2)
  }

  /**
   * Clear input
   */
  function clearInput() {
    stepInputValues.value = {}
    workflowInputValues.value = {}
    customInputJson.value = '{}'
    inputError.value = null
    if (stepId.value) {
      storedInput.clear(stepId.value)
    }
  }

  /**
   * Insert field into JSON input
   */
  function insertSuggestedField(fieldName: string, value: unknown) {
    try {
      const current = JSON.parse(customInputJson.value)
      current[fieldName] = value
      customInputJson.value = JSON.stringify(current, null, 2)
      stepInputValues.value = current
    } catch {
      // If JSON is invalid, start fresh
      const newObj: Record<string, unknown> = {}
      newObj[fieldName] = value
      customInputJson.value = JSON.stringify(newObj, null, 2)
      stepInputValues.value = newObj
    }
  }

  return {
    // Mode states
    useJsonMode,
    useWorkflowJsonMode,

    // Input states
    customInputJson,
    inputError,
    schemaValidationErrors,
    workflowInputValues,
    stepInputValues,
    workflowFormValid,
    stepFormValid,

    // Methods
    parseCustomInput,
    getWorkflowInput,
    getStepInput,
    toggleInputMode,
    toggleWorkflowInputMode,
    usePreviousOutput,
    clearInput,
    insertSuggestedField,
  }
}
