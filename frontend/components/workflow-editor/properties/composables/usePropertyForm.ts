/**
 * Composable for properties panel form state management
 * Handles form state, auto-save, and step configuration
 */
import { useDebounceFn } from '@vueuse/core'
import type { Step, StepType, BlockDefinition } from '~/types/api'

export interface StepConfig {
  provider?: string
  model?: string
  system_prompt?: string
  prompt?: string
  temperature?: number
  max_tokens?: number
  adapter_id?: string
  url?: string
  method?: string
  expression?: string
  cases?: Array<{ name: string; expression: string; is_default?: boolean }>
  loop_type?: string
  count?: number
  input_path?: string
  condition?: string
  max_iterations?: number
  duration_ms?: number
  until?: string
  code?: string
  timeout_ms?: number
  routes_json?: string
  instructions?: string
  timeout_hours?: number
  approval_url?: boolean
  parallel?: number
  workflow_id?: string
  message?: string
  level?: string
  data?: string
  output_schema?: object
  [key: string]: unknown
}

export interface FlowConfig {
  prescript?: { enabled: boolean; code: string }
  postscript?: { enabled: boolean; code: string }
  error_handling?: {
    enabled: boolean
    retry?: object
    timeout_seconds?: number
    on_error: string
    fallback_value?: unknown
    enable_error_port?: boolean
  }
}

export interface UsePropertyFormOptions {
  step: Ref<Step | null>
  readonlyMode: Ref<boolean>
  onSave: (data: {
    name: string
    type: StepType
    config: StepConfig
    credential_bindings?: Record<string, string>
  }) => void
  onUpdateName: (name: string) => void
  onUpdateCredentialBindings: (bindings: Record<string, string>) => void
}

export function usePropertyForm(options: UsePropertyFormOptions) {
  const { step, readonlyMode, onSave, onUpdateName, onUpdateCredentialBindings } = options

  // Form state
  const formName = ref('')
  const formType = ref<StepType>('tool')
  const formConfig = ref<StepConfig>({})
  const isInitializing = ref(true)
  const flowConfig = ref<FlowConfig>({})
  const localCredentialBindings = ref<Record<string, string>>({})

  // Watch for step changes and reset form
  watch(step, (newStep) => {
    isInitializing.value = true

    if (newStep) {
      formName.value = newStep.name
      formType.value = newStep.type
      formConfig.value = { ...(newStep.config as StepConfig) }
    } else {
      formName.value = ''
      formType.value = 'tool'
      formConfig.value = {}
    }

    nextTick(() => {
      isInitializing.value = false
    })
  }, { immediate: true, deep: true })

  // Emit name changes for reactive updates
  watch(formName, (newName) => {
    if (step.value && newName !== step.value.name) {
      onUpdateName(newName)
    }
  })

  // Initialize credential bindings from step
  watch(() => step.value?.credential_bindings, (bindings) => {
    localCredentialBindings.value = { ...bindings }
  }, { immediate: true, deep: true })

  // Save handler
  function handleSave() {
    if (readonlyMode.value || !step.value) return

    const mergedConfig = { ...formConfig.value, ...flowConfig.value }
    const saveData: {
      name: string
      type: StepType
      config: StepConfig
      credential_bindings?: Record<string, string>
    } = {
      name: formName.value,
      type: formType.value,
      config: mergedConfig
    }

    if (Object.keys(localCredentialBindings.value).length > 0) {
      saveData.credential_bindings = localCredentialBindings.value
    }

    onSave(saveData)
  }

  // Auto-save with debounce
  const debouncedSave = useDebounceFn(handleSave, 500)

  // Watch for config changes and trigger auto-save
  watch([formConfig, flowConfig, localCredentialBindings], () => {
    if (!isInitializing.value && !readonlyMode.value && step.value) {
      debouncedSave()
    }
  }, { deep: true })

  // Watch for name changes and trigger auto-save
  watch(formName, (newName, oldName) => {
    if (!isInitializing.value && !readonlyMode.value && step.value && newName !== oldName) {
      debouncedSave()
    }
  })

  // Flow config update handler
  function handleFlowConfigUpdate(config: FlowConfig) {
    flowConfig.value = config
  }

  // Credential bindings update handler
  function handleCredentialBindingsUpdate(bindings: Record<string, string>) {
    localCredentialBindings.value = bindings
    onUpdateCredentialBindings(bindings)
  }

  return {
    // State
    formName,
    formType,
    formConfig,
    flowConfig,
    localCredentialBindings,
    isInitializing,
    // Actions
    handleSave,
    handleFlowConfigUpdate,
    handleCredentialBindingsUpdate,
  }
}

/**
 * Composable for block definition loading
 */
export interface UseBlockDefinitionOptions {
  stepType: Ref<StepType>
}

export function useBlockDefinition(options: UseBlockDefinitionOptions) {
  const { stepType } = options
  const blocks = useBlocks()

  const currentBlockDef = ref<BlockDefinition | null>(null)
  const loadingBlockDef = ref(false)

  watch(stepType, async (newType) => {
    if (newType) {
      loadingBlockDef.value = true
      try {
        const response = await blocks.get(newType)
        currentBlockDef.value = response.data
      } catch {
        currentBlockDef.value = null
      } finally {
        loadingBlockDef.value = false
      }
    } else {
      currentBlockDef.value = null
    }
  }, { immediate: true })

  return {
    currentBlockDef,
    loadingBlockDef,
  }
}

/**
 * Composable for template variable extraction
 */
export function useTemplateVariablesExtractor(config: Ref<StepConfig>) {
  const templateVariables = computed<string[]>(() => {
    const variables: Set<string> = new Set()
    const templateRegex = /\{\{([^}]+)\}\}/g

    function extractFromValue(value: unknown) {
      if (typeof value === 'string') {
        let match
        while ((match = templateRegex.exec(value)) !== null) {
          variables.add(match[1].trim())
        }
      } else if (Array.isArray(value)) {
        value.forEach(extractFromValue)
      } else if (value && typeof value === 'object') {
        Object.values(value).forEach(extractFromValue)
      }
    }

    extractFromValue(config.value)
    return Array.from(variables)
  })

  return { templateVariables }
}

/**
 * Composable for field inserter management (click-to-insert variables)
 */
export interface FieldInserter {
  insert: (text: string) => void
  focus: () => void
}

export function useFieldInserter() {
  const fieldInserters = ref<Map<string, FieldInserter>>(new Map())
  const activeFieldId = ref<string | null>(null)

  function register(id: string, inserter: FieldInserter) {
    fieldInserters.value.set(id, inserter)
  }

  function unregister(id: string) {
    fieldInserters.value.delete(id)
    if (activeFieldId.value === id) {
      activeFieldId.value = null
    }
  }

  function setActive(id: string | null) {
    activeFieldId.value = id
  }

  function insertVariableIntoActiveField(variablePath: string) {
    const openBrace = String.fromCharCode(123, 123)
    const closeBrace = String.fromCharCode(125, 125)
    const template = openBrace + variablePath + closeBrace

    if (activeFieldId.value) {
      const inserter = fieldInserters.value.get(activeFieldId.value)
      if (inserter) {
        inserter.insert(template)
        return
      }
    }

    // If no active field, try to find the first text field
    const firstInserter = fieldInserters.value.values().next().value
    if (firstInserter) {
      firstInserter.focus()
      firstInserter.insert(template)
    }
  }

  return {
    fieldInserters,
    activeFieldId,
    register,
    unregister,
    setActive,
    insertVariableIntoActiveField,
  }
}
