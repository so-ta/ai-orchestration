<script setup lang="ts">
import type { Step, StepType, BlockDefinition, Run } from '~/types/api'
import type { StepSuggestion, GenerateWorkflowResponse } from '~/composables/useCopilot'
import type { ConfigSchema, UIConfig } from './config/types/config-schema'
import DynamicConfigForm from './config/DynamicConfigForm.vue'
import FlowTab from './FlowTab.vue'
import TriggerConfigPanel from './TriggerConfigPanel.vue'

type StartTriggerType = 'manual' | 'webhook' | 'schedule' | 'slack' | 'email'

const { t } = useI18n()
const blocks = useBlocks()
const toast = useToast()
const { confirm } = useConfirm()

const props = defineProps<{
  step: Step | null
  workflowId: string
  readonlyMode?: boolean
  saving?: boolean
  latestRun?: Run | null
  steps?: Step[]
  edges?: Array<{ id: string; source_step_id?: string | null; target_step_id?: string | null }>
  blockDefinitions?: BlockDefinition[]
}>()

// Active tab state
const activeTab = ref<'config' | 'flow' | 'trigger' | 'copilot' | 'run'>('config')

// Check if step is a start block (for showing trigger tab)
const isStartBlock = computed(() => props.step?.type === 'start')

// Keep current tab when step changes (no automatic tab switching)

const emit = defineEmits<{
  (e: 'save', data: { name: string; type: StepType; config: StepConfig }): void
  (e: 'delete'): void
  (e: 'apply-workflow', workflow: GenerateWorkflowResponse): void
  (e: 'execute', data: { stepId: string; input: object; triggered_by: 'test' | 'manual' }): void
  (e: 'execute-workflow', triggered_by: 'test' | 'manual', input: object): void
  (e: 'update:name', name: string): void
  (e: 'update:trigger', data: { trigger_type: StartTriggerType; trigger_config: object }): void
}>()

// Step config type - dynamic form configuration with common known fields
// Using index signature for dynamic access while keeping type safety for known fields
interface StepConfig {
  // LLM config
  provider?: string
  model?: string
  system_prompt?: string
  prompt?: string
  temperature?: number
  max_tokens?: number
  // Tool config
  adapter_id?: string
  url?: string
  method?: string
  // Condition/Switch config
  expression?: string
  cases?: Array<{ name: string; expression: string; is_default?: boolean }>
  // Loop config
  loop_type?: string
  count?: number
  input_path?: string
  condition?: string
  max_iterations?: number
  // Wait config
  duration_ms?: number
  until?: string
  // Function config
  code?: string
  timeout_ms?: number
  // Router config
  routes_json?: string
  // Human in loop config
  instructions?: string
  timeout_hours?: number
  approval_url?: boolean
  // Map config
  parallel?: number
  // Subflow config
  workflow_id?: string
  // Log config
  message?: string
  level?: string
  data?: string
  // Output schema
  output_schema?: object
  // Dynamic access for other fields
  [key: string]: unknown
}

// Form state
const formName = ref('')
const formType = ref<StepType>('tool')
const formConfig = ref<StepConfig>({})

// Watch for step changes and reset form
watch(() => props.step, (newStep) => {
  if (newStep) {
    formName.value = newStep.name
    formType.value = newStep.type
    formConfig.value = { ...(newStep.config as StepConfig) }
  } else {
    formName.value = ''
    formType.value = 'tool'
    formConfig.value = {}
  }
}, { immediate: true, deep: true })

// Emit name changes for reactive updates in the flow editor
watch(formName, (newName) => {
  if (props.step && newName !== props.step.name) {
    emit('update:name', newName)
  }
})

// Step type descriptions (computed for i18n reactivity, reserved for future use)
const _stepTypeDescriptions = computed(() => ({
  start: t('editor.stepTypes.startDesc'),
  llm: t('editor.stepTypes.llmDesc'),
  tool: t('editor.stepTypes.toolDesc'),
  condition: t('editor.stepTypes.conditionDesc'),
  switch: t('editor.stepTypes.switchDesc'),
  map: t('editor.stepTypes.mapDesc'),
  join: t('editor.stepTypes.joinDesc'),
  subflow: t('editor.stepTypes.subflowDesc'),
  loop: t('editor.stepTypes.loopDesc'),
  wait: t('editor.stepTypes.waitDesc'),
  function: t('editor.stepTypes.functionDesc'),
  router: t('editor.stepTypes.routerDesc'),
  human_in_loop: t('editor.stepTypes.humanInLoopDesc'),
  filter: t('editor.stepTypes.filterDesc'),
  split: t('editor.stepTypes.splitDesc'),
  aggregate: t('editor.stepTypes.aggregateDesc'),
  error: t('editor.stepTypes.errorDesc'),
  note: t('editor.stepTypes.noteDesc'),
  log: t('editor.stepTypes.logDesc')
} as Record<StepType, string>))

// Step type colors (matching DagEditor)
const stepTypeColors: Record<StepType, string> = {
  start: '#10b981',
  llm: '#3b82f6',
  tool: '#22c55e',
  condition: '#f59e0b',
  switch: '#eab308',
  map: '#8b5cf6',
  subflow: '#ec4899',
  loop: '#14b8a6',
  wait: '#64748b',
  function: '#f97316',
  router: '#a855f7',
  human_in_loop: '#ef4444',
  filter: '#06b6d4',
  split: '#0ea5e9',
  aggregate: '#0284c7',
  error: '#dc2626',
  note: '#9ca3af',
  log: '#10b981'
}

// Check if step is a start node (cannot be deleted)
const isStartNode = computed(() => props.step?.type === 'start')

// Flow config from FlowTab (prescript, postscript, error_handling)
const flowConfig = ref<{
  prescript?: { enabled: boolean; code: string }
  postscript?: { enabled: boolean; code: string }
  error_handling?: { enabled: boolean; retry?: object; timeout_seconds?: number; on_error: string; fallback_value?: unknown; enable_error_port?: boolean }
}>({})

function handleFlowConfigUpdate(config: typeof flowConfig.value) {
  flowConfig.value = config
}

// Handle trigger config update from TriggerConfigPanel
function handleTriggerUpdate(data: { trigger_type: StartTriggerType; trigger_config: object }) {
  emit('update:trigger', data)
}

function handleSave() {
  // Merge flow config into the main config
  const mergedConfig = {
    ...formConfig.value,
    ...flowConfig.value
  }

  emit('save', {
    name: formName.value,
    type: formType.value,
    config: mergedConfig
  })
}

async function handleDelete() {
  const confirmed = await confirm({
    title: t('editor.deleteStepTitle'),
    message: t('editor.confirmDeleteStep'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    variant: 'danger',
  })
  if (confirmed) {
    emit('delete')
  }
}

// Handle suggestion from Copilot
function handleApplySuggestion(suggestion: StepSuggestion) {
  // Apply suggestion to current step
  formType.value = suggestion.type as StepType
  formName.value = suggestion.name
  formConfig.value = { ...(suggestion.config || {}) }
  activeTab.value = 'config'
  toast.success(t('copilot.suggestionApplied'))
}

// Handle generated workflow from Copilot
function handleApplyWorkflow(workflow: GenerateWorkflowResponse) {
  // Pass to parent to apply to canvas
  emit('apply-workflow', workflow)
}

// Available adapters for tool step (computed for i18n reactivity)
const adapters = computed(() => [
  { id: 'mock', name: t('stepConfig.tool.adapters.mock'), description: t('stepConfig.tool.adapters.mockDesc') },
  { id: 'openai', name: t('stepConfig.tool.adapters.openai'), description: t('stepConfig.tool.adapters.openaiDesc') },
  { id: 'anthropic', name: t('stepConfig.tool.adapters.anthropic'), description: t('stepConfig.tool.adapters.anthropicDesc') },
  { id: 'http', name: t('stepConfig.tool.adapters.http'), description: t('stepConfig.tool.adapters.httpDesc') },
])

// Available models by provider
const modelsByProvider: Record<string, { id: string; name: string }[]> = {
  openai: [
    { id: 'gpt-4', name: 'GPT-4' },
    { id: 'gpt-4-turbo', name: 'GPT-4 Turbo' },
    { id: 'gpt-3.5-turbo', name: 'GPT-3.5 Turbo' },
  ],
  anthropic: [
    { id: 'claude-3-opus', name: 'Claude 3 Opus' },
    { id: 'claude-3-sonnet', name: 'Claude 3 Sonnet' },
    { id: 'claude-3-haiku', name: 'Claude 3 Haiku' },
  ],
  mock: [
    { id: 'mock', name: 'Mock Model' },
  ],
}

// Get available models for current provider
const availableModels = computed(() => {
  const provider = formConfig.value.provider || 'mock'
  return modelsByProvider[provider] || modelsByProvider.mock
})

// Watch provider changes to reset model
watch(() => formConfig.value.provider, (newProvider) => {
  if (newProvider && modelsByProvider[newProvider]) {
    formConfig.value.model = modelsByProvider[newProvider][0]?.id || ''
  }
})

// Block definition for current step type
const currentBlockDef = ref<BlockDefinition | null>(null)
const loadingBlockDef = ref(false)

// Fetch block definition when step type changes
watch(() => formType.value, async (newType) => {
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

// Check if block has config_schema for dynamic form rendering
const hasConfigSchema = computed(() => {
  const schema = currentBlockDef.value?.config_schema
  return schema && typeof schema === 'object' && Object.keys(schema).length > 0
})

// Get config_schema as ConfigSchema type
const configSchema = computed<ConfigSchema | null>(() => {
  if (!hasConfigSchema.value) return null
  return currentBlockDef.value?.config_schema as ConfigSchema
})

// Get ui_config for additional UI customization
const uiConfig = computed<UIConfig | undefined>(() => {
  if (!currentBlockDef.value?.ui_config) return undefined
  return currentBlockDef.value.ui_config as UIConfig
})

// Helper to format schema type for display
function formatSchemaType(schema: object | undefined): string {
  if (!schema) return 'any'
  const s = schema as Record<string, unknown>
  if (s.type === 'array' && s.items) {
    const items = s.items as Record<string, unknown>
    return `${items.type || 'any'}[]`
  }
  return String(s.type || 'any')
}

// Switch cases management
const switchCases = computed({
  get: () => {
    const cases = formConfig.value.cases as Array<{ name: string; expression: string; is_default?: boolean }> || []
    return cases
  },
  set: (val) => {
    formConfig.value.cases = val
  }
})

function addSwitchCase() {
  const cases = [...(formConfig.value.cases as Array<{ name: string; expression: string; is_default?: boolean }> || [])]
  const newIndex = cases.length + 1
  cases.push({
    name: `case_${newIndex}`,
    expression: '',
    is_default: false
  })
  formConfig.value.cases = cases
}

function removeSwitchCase(index: number) {
  const cases = [...(formConfig.value.cases as Array<{ name: string; expression: string; is_default?: boolean }> || [])]
  cases.splice(index, 1)
  formConfig.value.cases = cases
}

function updateSwitchCase(index: number, field: 'name' | 'expression' | 'is_default', value: string | boolean) {
  const cases = [...(formConfig.value.cases as Array<{ name: string; expression: string; is_default?: boolean }> || [])]
  if (cases[index]) {
    if (field === 'is_default') {
      // Only one case can be default
      cases.forEach((c, i) => {
        c.is_default = i === index ? Boolean(value) : false
      })
    } else {
      (cases[index] as Record<string, string | boolean>)[field] = value
    }
    formConfig.value.cases = cases
  }
}

// Helper to get case display name (avoids template literal in template)
function getCaseDisplayName(caseName: string, index: number): string {
  return caseName || 'case_' + (index + 1)
}

// Expression helper insertions (avoids escaped quotes in template)
function insertExpression(expr: string) {
  formConfig.value.expression = (formConfig.value.expression || '') + expr
}
const expressionTemplates = {
  equals: '$.field == "value"',
  notEquals: '$.field != "value"',
  greaterThan: '$.field > 0',
  lessThan: '$.field < 0',
  exists: '$.field'
}

// =============================================================================
// Template Preview (Prompt Template Variables)
// =============================================================================

/**
 * Extract all template variables from config values
 * Detects double-brace variable patterns in string values
 */
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

  extractFromValue(formConfig.value)
  return Array.from(variables)
})

/**
 * Check if current block type typically uses templates
 * (LLM prompts, notifications, etc.)
 */
const isTemplateBlock = computed(() => {
  const templateTypes = ['llm', 'discord', 'slack', 'email_sendgrid', 'log', 'function']
  return templateTypes.includes(formType.value)
})

/**
 * Show template preview section
 */
const showTemplatePreview = computed(() => {
  return templateVariables.value.length > 0 && isTemplateBlock.value
})

/**
 * Format template variable for display (avoids i18n parsing issues with curly braces)
 */
function formatTemplateVariable(variable: string): string {
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)
  return openBrace + variable + closeBrace
}

// =============================================================================
// Output Schema Display
// =============================================================================

interface OutputSchemaProperty {
  type: string
  title?: string
  description?: string
}

interface ParsedOutputSchema {
  type: string
  properties: Record<string, OutputSchemaProperty>
  required?: string[]
}

// =============================================================================
// Available Input Variables (from previous steps)
// =============================================================================

interface AvailableVariable {
  path: string
  type: string
  title?: string
  description?: string
  source: string // step name
}

/**
 * Find previous steps connected to current step
 */
const previousSteps = computed(() => {
  if (!props.step || !props.edges || !props.steps) return []

  const incomingEdges = props.edges.filter(e => e.target_step_id === props.step?.id)
  const prevStepIds = incomingEdges.map(e => e.source_step_id)

  return props.steps.filter(s => prevStepIds.includes(s.id))
})

/**
 * Get available variables from previous steps' output schemas
 */
const availableInputVariables = computed<AvailableVariable[]>(() => {
  const variables: AvailableVariable[] = []

  for (const prevStep of previousSteps.value) {
    const config = prevStep.config as Record<string, unknown> | undefined
    if (!config) continue

    // Check for output_schema in the config
    const outputSchema = config.output_schema as ParsedOutputSchema | undefined
    if (!outputSchema || outputSchema.type !== 'object' || !outputSchema.properties) {
      // If no output_schema, add generic input reference
      variables.push({
        path: `$.steps.${prevStep.name}.output`,
        type: 'object',
        title: prevStep.name,
        source: prevStep.name
      })
      continue
    }

    // Add each field from the output schema
    for (const [fieldName, fieldDef] of Object.entries(outputSchema.properties)) {
      variables.push({
        path: `$.steps.${prevStep.name}.output.${fieldName}`,
        type: fieldDef.type || 'any',
        title: fieldDef.title || fieldName,
        description: fieldDef.description,
        source: prevStep.name
      })
    }
  }

  // Also add input reference
  variables.unshift({
    path: '$.input',
    type: 'object',
    title: 'ワークフロー入力',
    source: 'input'
  })

  return variables
})

/**
 * Check if there are available input variables
 */
const hasAvailableVariables = computed(() => availableInputVariables.value.length > 1)
</script>

<template>
  <div class="properties-panel">
    <!-- Header: shows step info if selected, otherwise generic header -->
    <div v-if="step" class="properties-header">
      <div class="header-color" :style="{ backgroundColor: stepTypeColors[step.type] }" />
      <div class="header-info">
        <h3 class="header-title">{{ readonlyMode ? t('editor.viewStep') : t('editor.editStep') }}</h3>
        <span class="header-type">{{ step.type }}</span>
      </div>
    </div>
    <div v-else class="properties-header properties-header-empty">
      <div class="header-info">
        <h3 class="header-title">{{ t('editor.propertiesPanel') }}</h3>
      </div>
    </div>

    <!-- Tab Bar (always visible) -->
    <div class="properties-tabs">
      <button
        class="tab-button"
        :class="{ active: activeTab === 'config' }"
        @click="activeTab = 'config'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/>
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
        </svg>
        {{ t('editor.tabs.config') }}
      </button>
      <button
        class="tab-button"
        :class="{ active: activeTab === 'flow' }"
        @click="activeTab = 'flow'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/>
          <path d="M5 3l-1 9"/>
          <path d="M19 3l1 9"/>
          <polyline points="8 14 12 18 16 14"/>
        </svg>
        {{ t('editor.tabs.flow') }}
      </button>
      <!-- Trigger Tab (only for Start blocks) -->
      <button
        v-if="isStartBlock"
        class="tab-button"
        :class="{ active: activeTab === 'trigger' }"
        @click="activeTab = 'trigger'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
        </svg>
        {{ t('editor.tabs.trigger') }}
      </button>
      <button
        class="tab-button"
        :class="{ active: activeTab === 'copilot' }"
        @click="activeTab = 'copilot'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73A2 2 0 0 1 10 4a2 2 0 0 1 2-2z"/>
          <circle cx="8" cy="14" r="2"/>
          <circle cx="16" cy="14" r="2"/>
        </svg>
        {{ t('editor.tabs.copilot') }}
      </button>
      <button
        class="tab-button"
        :class="{ active: activeTab === 'run' }"
        @click="activeTab = 'run'"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="5 3 19 12 5 21 5 3"/>
        </svg>
        {{ t('editor.tabs.run') }}
      </button>
    </div>


    <!-- Config Tab Content -->
    <div v-if="activeTab === 'config'" class="properties-body-wrapper">
      <!-- Empty State (no step selected) -->
      <div v-if="!step" class="properties-empty">
        <div class="empty-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M12 3l8 4.5v9L12 21l-8-4.5v-9L12 3z"/>
            <path d="M12 12l8-4.5"/>
            <path d="M12 12v9"/>
            <path d="M12 12L4 7.5"/>
          </svg>
        </div>
        <p class="empty-title">{{ t('editor.noStepSelected') }}</p>
        <p class="empty-desc">{{ t('editor.selectStepHint') }}</p>

        <!-- Quick Tips -->
        <div class="empty-tips">
          <div class="empty-tips-title">{{ t('editor.quickTips') }}</div>
          <ul class="empty-tips-list">
            <li>
              <span class="tip-icon">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M5 12h14"/>
                  <path d="M12 5v14"/>
                </svg>
              </span>
              {{ t('editor.tipDragBlock') }}
            </li>
            <li>
              <span class="tip-icon">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                  <polyline points="15 3 21 3 21 9"/>
                  <line x1="10" y1="14" x2="21" y2="3"/>
                </svg>
              </span>
              {{ t('editor.tipConnectNodes') }}
            </li>
            <li>
              <span class="tip-icon">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                </svg>
              </span>
              {{ t('editor.tipClickToSelect') }}
            </li>
          </ul>
        </div>

        <!-- Keyboard Shortcuts -->
        <div class="empty-shortcuts">
          <div class="empty-shortcuts-title">{{ t('editor.keyboardShortcuts') }}</div>
          <div class="shortcut-item">
            <kbd>Delete</kbd>
            <span>{{ t('editor.shortcutDelete') }}</span>
          </div>
          <div class="shortcut-item">
            <kbd>Esc</kbd>
            <span>{{ t('editor.shortcutDeselect') }}</span>
          </div>
          <div class="shortcut-item">
            <kbd>Ctrl</kbd> + <kbd>C</kbd>
            <span>{{ t('editor.shortcutCopy') }}</span>
          </div>
          <div class="shortcut-item">
            <kbd>Ctrl</kbd> + <kbd>V</kbd>
            <span>{{ t('editor.shortcutPaste') }}</span>
          </div>
        </div>
      </div>

      <!-- Step Properties (when step is selected) -->
      <div v-else class="properties-body">
        <!-- Basic Information -->
        <div class="form-section">
          <h4 class="section-title">{{ t('stepConfig.basicInfo') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.stepName') }}</label>
            <input
              v-model="formName"
              type="text"
              class="form-input"
              :placeholder="t('stepConfig.stepNamePlaceholder')"
              :disabled="readonlyMode"
            >
          </div>

        </div>

        <!-- Dynamic Config Form (when config_schema is defined) -->
        <div v-if="hasConfigSchema && !['start', 'join', 'note'].includes(formType)" class="form-section">
          <h4 class="section-title">{{ currentBlockDef?.name || formType }} 設定</h4>
          <DynamicConfigForm
            v-model="formConfig"
            :schema="configSchema"
            :ui-config="uiConfig"
            :disabled="readonlyMode"
          />
        </div>

        <!-- Available Input Variables Section -->
        <div v-if="hasAvailableVariables && isTemplateBlock" class="form-section available-variables-section">
          <h4 class="section-title">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            {{ t('stepConfig.availableVariables.title') }}
          </h4>
          <p class="section-description">
            {{ t('stepConfig.availableVariables.description') }}
          </p>
          <div class="available-variables-list">
            <div v-for="variable in availableInputVariables" :key="variable.path" class="available-variable-item">
              <div class="variable-header">
                <code class="variable-path">{{ formatTemplateVariable(variable.path) }}</code>
                <code class="variable-type">{{ variable.type }}</code>
              </div>
              <div class="variable-meta">
                <span class="variable-source">{{ variable.source }}</span>
                <span v-if="variable.title && variable.title !== variable.source" class="variable-title">{{ variable.title }}</span>
              </div>
              <div v-if="variable.description" class="variable-description">
                {{ variable.description }}
              </div>
            </div>
          </div>
        </div>

        <!-- Template Preview Section -->
        <div v-if="showTemplatePreview" class="form-section template-preview-section">
          <h4 class="section-title">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
              <polyline points="14 2 14 8 20 8"/>
              <path d="M12 18v-6"/>
              <path d="m9 15 3 3 3-3"/>
            </svg>
            {{ t('editor.templatePreview.title') }}
          </h4>
          <p class="template-preview-hint">{{ t('editor.templatePreview.hint') }}</p>
          <div class="template-variables-list">
            <div v-for="variable in templateVariables" :key="variable" class="template-variable-item">
              <code class="variable-name" v-text="formatTemplateVariable(variable)"/>
              <span class="variable-arrow">→</span>
              <span class="variable-placeholder">{{ t('editor.templatePreview.runtimeValue') }}</span>
            </div>
          </div>
          <div class="template-preview-note">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="16" x2="12" y2="12"/>
              <line x1="12" y1="8" x2="12.01" y2="8"/>
            </svg>
            <span>{{ t('editor.templatePreview.executionNote') }}</span>
          </div>
        </div>

        <!-- LLM Configuration (Legacy fallback when no config_schema) -->
        <div v-else-if="formType === 'llm'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.llm.title') }}</h4>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">{{ t('stepConfig.llm.provider') }}</label>
              <select
                v-model="formConfig.provider"
                class="form-input"
                :disabled="readonlyMode"
              >
                <option value="mock">{{ t('stepConfig.tool.adapters.mock') }}</option>
                <option value="openai">{{ t('stepConfig.tool.adapters.openai') }}</option>
                <option value="anthropic">{{ t('stepConfig.tool.adapters.anthropic') }}</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">{{ t('stepConfig.llm.model') }}</label>
              <select
                v-model="formConfig.model"
                class="form-input"
                :disabled="readonlyMode"
              >
                <option v-for="model in availableModels" :key="model.id" :value="model.id">
                  {{ model.name }}
                </option>
              </select>
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.llm.systemPrompt') }}</label>
            <textarea
              v-model="formConfig.system_prompt"
              class="form-input form-textarea"
              rows="3"
              :placeholder="t('stepConfig.llm.systemPromptPlaceholder')"
              :disabled="readonlyMode"
            />
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.llm.userPrompt') }}</label>
            <textarea
              v-model="formConfig.prompt"
              class="form-input form-textarea code-input"
              rows="4"
              :placeholder="t('stepConfig.llm.userPromptPlaceholder')"
              :disabled="readonlyMode"
            />
            <p class="form-hint">{{ t('stepConfig.llm.userPromptHint') }}</p>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">{{ t('stepConfig.llm.temperature') }}</label>
              <input
                v-model.number="formConfig.temperature"
                type="number"
                class="form-input"
                min="0"
                max="2"
                step="0.1"
                placeholder="0.7"
                :disabled="readonlyMode"
              >
            </div>
            <div class="form-group">
              <label class="form-label">{{ t('stepConfig.llm.maxTokens') }}</label>
              <input
                v-model.number="formConfig.max_tokens"
                type="number"
                class="form-input"
                min="1"
                max="128000"
                placeholder="4096"
                :disabled="readonlyMode"
              >
            </div>
          </div>
        </div>

        <!-- Tool Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'tool'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.tool.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.tool.adapter') }}</label>
            <div class="adapter-grid">
              <label
                v-for="adapter in adapters"
                :key="adapter.id"
                :class="['adapter-option', { selected: formConfig.adapter_id === adapter.id }]"
              >
                <input
                  v-model="formConfig.adapter_id"
                  type="radio"
                  :value="adapter.id"
                  :disabled="readonlyMode"
                >
                <div class="adapter-info">
                  <div class="adapter-name">{{ adapter.name }}</div>
                  <div class="adapter-desc">{{ adapter.description }}</div>
                </div>
              </label>
            </div>
          </div>

          <div v-if="formConfig.adapter_id === 'http'" class="form-group">
            <label class="form-label">{{ t('stepConfig.tool.httpEndpoint') }}</label>
            <input
              v-model="formConfig.url"
              type="url"
              class="form-input"
              :placeholder="t('stepConfig.tool.httpEndpointPlaceholder')"
              :disabled="readonlyMode"
            >
          </div>

          <div v-if="formConfig.adapter_id === 'http'" class="form-group">
            <label class="form-label">{{ t('stepConfig.tool.httpMethod') }}</label>
            <select
              v-model="formConfig.method"
              class="form-input"
              :disabled="readonlyMode"
            >
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="DELETE">DELETE</option>
            </select>
          </div>
        </div>

        <!-- Condition Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'condition'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.condition.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.condition.expression') }}</label>
            <textarea
              v-model="formConfig.expression"
              class="form-input form-textarea code-input"
              rows="2"
              :placeholder="t('stepConfig.condition.expressionPlaceholder')"
              :disabled="readonlyMode"
            />
            <p class="form-hint">{{ t('stepConfig.condition.expressionHint') }}</p>
          </div>

          <!-- Expression helpers -->
          <div class="expression-helpers">
            <span class="helper-label">{{ t('stepConfig.expressionHelpers.title') }}:</span>
            <div class="helper-chips">
              <button type="button" class="helper-chip" :disabled="readonlyMode" @click="insertExpression(expressionTemplates.equals)">
                ==
              </button>
              <button type="button" class="helper-chip" :disabled="readonlyMode" @click="insertExpression(expressionTemplates.notEquals)">
                !=
              </button>
              <button type="button" class="helper-chip" :disabled="readonlyMode" @click="insertExpression(expressionTemplates.greaterThan)">
                &gt;
              </button>
              <button type="button" class="helper-chip" :disabled="readonlyMode" @click="insertExpression(expressionTemplates.lessThan)">
                &lt;
              </button>
              <button type="button" class="helper-chip" :disabled="readonlyMode" @click="insertExpression(expressionTemplates.exists)">
                {{ t('stepConfig.expressionHelpers.exists') }}
              </button>
            </div>
          </div>

          <!-- Output ports display -->
          <div class="output-ports-preview">
            <div class="ports-header">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
              <span>{{ t('stepConfig.outputPorts') }}</span>
            </div>
            <div class="condition-preview">
              <div class="condition-branch">
                <span class="branch-label branch-true">{{ t('stepConfig.condition.trueBranch') }}</span>
                <span class="branch-type">boolean</span>
                <span class="branch-desc">{{ t('stepConfig.condition.trueBranchDesc') }}</span>
              </div>
              <div class="condition-branch">
                <span class="branch-label branch-false">{{ t('stepConfig.condition.falseBranch') }}</span>
                <span class="branch-type">boolean</span>
                <span class="branch-desc">{{ t('stepConfig.condition.falseBranchDesc') }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Switch Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'switch'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.switch.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.switch.expression') }}</label>
            <input
              v-model="formConfig.expression"
              type="text"
              class="form-input code-input"
              placeholder="$.status"
              :disabled="readonlyMode"
            >
            <p class="form-hint">{{ t('stepConfig.switch.expressionHint') }}</p>
          </div>

          <!-- Cases list -->
          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.switch.cases') }}</label>
            <div class="switch-cases-list">
              <div
                v-for="(switchCase, index) in switchCases"
                :key="index"
                class="switch-case-item"
              >
                <div class="switch-case-header">
                  <input
                    :value="switchCase.name"
                    type="text"
                    class="form-input case-name-input"
                    :placeholder="t('stepConfig.switch.caseNamePlaceholder')"
                    :disabled="readonlyMode"
                    @input="updateSwitchCase(index, 'name', ($event.target as HTMLInputElement).value)"
                  >
                  <button
                    v-if="!readonlyMode"
                    type="button"
                    class="btn-icon btn-remove-case"
                    :title="t('common.delete')"
                    @click="removeSwitchCase(index)"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="18" y1="6" x2="6" y2="18"/>
                      <line x1="6" y1="6" x2="18" y2="18"/>
                    </svg>
                  </button>
                </div>
                <input
                  :value="switchCase.expression"
                  type="text"
                  class="form-input code-input"
                  :placeholder="t('stepConfig.switch.caseExpressionPlaceholder')"
                  :disabled="readonlyMode"
                  @input="updateSwitchCase(index, 'expression', ($event.target as HTMLInputElement).value)"
                >
                <label class="form-checkbox case-default-checkbox">
                  <input
                    type="checkbox"
                    :checked="switchCase.is_default"
                    :disabled="readonlyMode"
                    @change="updateSwitchCase(index, 'is_default', ($event.target as HTMLInputElement).checked)"
                  >
                  <span>{{ t('stepConfig.switch.isDefault') }}</span>
                </label>
              </div>

              <button
                v-if="!readonlyMode"
                type="button"
                class="btn btn-add-case"
                @click="addSwitchCase"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19"/>
                  <line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
                {{ t('stepConfig.switch.addCase') }}
              </button>
            </div>
          </div>

          <!-- Output ports preview -->
          <div v-if="switchCases.length > 0" class="output-ports-preview">
            <div class="ports-header">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
              <span>{{ t('stepConfig.outputPorts') }}</span>
            </div>
            <div class="switch-ports-list">
              <div v-for="(switchCase, index) in switchCases" :key="index" class="switch-port-item">
                <span :class="['branch-label', switchCase.is_default ? 'branch-default' : 'branch-case']">
                  {{ getCaseDisplayName(switchCase.name, index) }}
                </span>
                <span class="branch-type">any</span>
                <span v-if="switchCase.is_default" class="branch-desc">({{ t('stepConfig.switch.defaultBranch') }})</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Loop Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'loop'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.loop.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.loop.loopType') }}</label>
            <select
              v-model="formConfig.loop_type"
              class="form-input"
              :disabled="readonlyMode"
            >
              <option value="for">{{ t('stepConfig.loop.for') }}</option>
              <option value="forEach">{{ t('stepConfig.loop.forEach') }}</option>
              <option value="while">{{ t('stepConfig.loop.while') }}</option>
              <option value="doWhile">{{ t('stepConfig.loop.doWhile') }}</option>
            </select>
          </div>

          <div v-if="formConfig.loop_type === 'for'" class="form-group">
            <label class="form-label">{{ t('stepConfig.loop.count') }}</label>
            <input
              v-model.number="formConfig.count"
              type="number"
              class="form-input"
              min="1"
              max="1000"
              placeholder="10"
              :disabled="readonlyMode"
            >
          </div>

          <div v-if="formConfig.loop_type === 'forEach'" class="form-group">
            <label class="form-label">{{ t('stepConfig.loop.inputPath') }}</label>
            <input
              v-model="formConfig.input_path"
              type="text"
              class="form-input code-input"
              :placeholder="t('stepConfig.loop.inputPathPlaceholder')"
              :disabled="readonlyMode"
            >
          </div>

          <div v-if="formConfig.loop_type === 'while' || formConfig.loop_type === 'doWhile'" class="form-group">
            <label class="form-label">{{ t('stepConfig.loop.continueCondition') }}</label>
            <input
              v-model="formConfig.condition"
              type="text"
              class="form-input code-input"
              :placeholder="t('stepConfig.loop.continueConditionPlaceholder')"
              :disabled="readonlyMode"
            >
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.loop.maxIterations') }}</label>
            <input
              v-model.number="formConfig.max_iterations"
              type="number"
              class="form-input"
              min="1"
              max="1000"
              placeholder="100"
              :disabled="readonlyMode"
            >
          </div>
        </div>

        <!-- Wait Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'wait'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.wait.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.wait.duration') }}</label>
            <input
              v-model.number="formConfig.duration_ms"
              type="number"
              class="form-input"
              min="0"
              max="3600000"
              placeholder="5000"
              :disabled="readonlyMode"
            >
            <p class="form-hint">{{ t('stepConfig.wait.durationHint') }}</p>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.wait.until') }}</label>
            <input
              v-model="formConfig.until"
              type="datetime-local"
              class="form-input"
              :disabled="readonlyMode"
            >
          </div>
        </div>

        <!-- Function Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'function'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.function.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.function.code') }}</label>
            <textarea
              v-model="formConfig.code"
              class="form-input form-textarea code-input"
              rows="8"
              :placeholder="t('stepConfig.function.codePlaceholder')"
              :disabled="readonlyMode"
            />
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.function.timeout') }}</label>
            <input
              v-model.number="formConfig.timeout_ms"
              type="number"
              class="form-input"
              min="100"
              max="30000"
              placeholder="5000"
              :disabled="readonlyMode"
            >
          </div>
        </div>

        <!-- Router Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'router'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.router.title') }}</h4>

          <div class="form-row">
            <div class="form-group">
              <label class="form-label">{{ t('stepConfig.llm.provider') }}</label>
              <select
                v-model="formConfig.provider"
                class="form-input"
                :disabled="readonlyMode"
              >
                <option value="mock">{{ t('stepConfig.tool.adapters.mock') }}</option>
                <option value="openai">{{ t('stepConfig.tool.adapters.openai') }}</option>
                <option value="anthropic">{{ t('stepConfig.tool.adapters.anthropic') }}</option>
              </select>
            </div>
            <div class="form-group">
              <label class="form-label">{{ t('stepConfig.llm.model') }}</label>
              <input
                v-model="formConfig.model"
                type="text"
                class="form-input"
                placeholder="gpt-4o-mini"
                :disabled="readonlyMode"
              >
            </div>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.router.classificationPrompt') }}</label>
            <textarea
              v-model="formConfig.prompt"
              class="form-input form-textarea"
              rows="3"
              :placeholder="t('stepConfig.router.classificationPromptPlaceholder')"
              :disabled="readonlyMode"
            />
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.router.routes') }}</label>
            <textarea
              v-model="formConfig.routes_json"
              class="form-input form-textarea code-input"
              rows="4"
              :placeholder="t('stepConfig.router.routesPlaceholder')"
              :disabled="readonlyMode"
            />
          </div>
        </div>

        <!-- Human in Loop Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'human_in_loop'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.humanInLoop.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.humanInLoop.instructions') }}</label>
            <textarea
              v-model="formConfig.instructions"
              class="form-input form-textarea"
              rows="3"
              :placeholder="t('stepConfig.humanInLoop.instructionsPlaceholder')"
              :disabled="readonlyMode"
            />
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.humanInLoop.timeoutHours') }}</label>
            <input
              v-model.number="formConfig.timeout_hours"
              type="number"
              class="form-input"
              min="1"
              max="168"
              placeholder="24"
              :disabled="readonlyMode"
            >
          </div>

          <div class="form-group">
            <label class="form-checkbox">
              <input
                v-model="formConfig.approval_url"
                type="checkbox"
                :disabled="readonlyMode"
              >
              <span>{{ t('stepConfig.humanInLoop.generateApprovalUrl') }}</span>
            </label>
          </div>

          <div class="info-box">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="16" x2="12" y2="12"/>
              <line x1="12" y1="8" x2="12.01" y2="8"/>
            </svg>
            <span>{{ t('stepConfig.humanInLoop.testModeNote') }}</span>
          </div>
        </div>

        <!-- Map Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'map'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.map.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.map.inputPath') }}</label>
            <input
              v-model="formConfig.input_path"
              type="text"
              class="form-input code-input"
              :placeholder="t('stepConfig.map.inputPathPlaceholder')"
              :disabled="readonlyMode"
            >
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.map.parallelism') }}</label>
            <input
              v-model.number="formConfig.parallel"
              type="number"
              class="form-input"
              min="1"
              max="100"
              placeholder="10"
              :disabled="readonlyMode"
            >
          </div>
        </div>

        <!-- Subflow Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'subflow'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.subflow.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.subflow.workflowId') }}</label>
            <input
              v-model="formConfig.workflow_id"
              type="text"
              class="form-input code-input"
              :placeholder="t('stepConfig.subflow.workflowIdPlaceholder')"
              :disabled="readonlyMode"
            >
          </div>
        </div>

        <!-- Start Configuration (Always shown - no config needed) -->
        <div v-if="!hasConfigSchema && formType === 'start'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.start.title') }}</h4>
          <div class="start-info-box">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="currentColor" class="start-icon">
              <polygon points="5 3 19 12 5 21 5 3" />
            </svg>
            <div class="start-info-content">
              <p class="start-info-title">{{ t('stepConfig.start.entryPoint') }}</p>
              <p class="start-info-desc">{{ t('stepConfig.start.description') }}</p>
            </div>
          </div>
          <div class="start-warning">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
              <line x1="12" y1="9" x2="12" y2="13"/>
              <line x1="12" y1="17" x2="12.01" y2="17"/>
            </svg>
            <span>{{ t('stepConfig.start.cannotDelete') }}</span>
          </div>
        </div>

        <!-- Log Configuration (Legacy fallback when no config_schema) -->
        <div v-if="!hasConfigSchema && formType === 'log'" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.log.title') }}</h4>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.log.message') }}</label>
            <textarea
              v-model="formConfig.message"
              class="form-input form-textarea"
              rows="3"
              :placeholder="t('stepConfig.log.messagePlaceholder')"
              :disabled="readonlyMode"
            />
            <p class="form-hint">{{ t('stepConfig.log.messageHint') }}</p>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.log.level') }}</label>
            <select
              v-model="formConfig.level"
              class="form-input"
              :disabled="readonlyMode"
            >
              <option value="debug">{{ t('stepConfig.log.levels.debug') }}</option>
              <option value="info">{{ t('stepConfig.log.levels.info') }}</option>
              <option value="warn">{{ t('stepConfig.log.levels.warn') }}</option>
              <option value="error">{{ t('stepConfig.log.levels.error') }}</option>
            </select>
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.log.data') }}</label>
            <input
              v-model="formConfig.data"
              type="text"
              class="form-input code-input"
              :placeholder="t('stepConfig.log.dataPlaceholder')"
              :disabled="readonlyMode"
            >
            <p class="form-hint">{{ t('stepConfig.log.dataHint') }}</p>
          </div>

          <div class="info-box log-info-box">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="4 17 10 11 4 5"/>
              <line x1="12" y1="19" x2="20" y2="19"/>
            </svg>
            <span>{{ t('stepConfig.log.viewNote') }}</span>
          </div>
        </div>

        <!-- I/O Ports Section (for blocks with typed ports) -->
        <div v-if="currentBlockDef && (currentBlockDef.input_ports?.length > 1 || currentBlockDef.output_ports?.length > 1)" class="form-section">
          <h4 class="section-title">{{ t('stepConfig.ioPorts.title') }}</h4>

          <!-- Input Ports -->
          <div v-if="currentBlockDef.input_ports?.length > 1" class="io-ports-group">
            <div class="ports-header">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="15 18 9 12 15 6"/>
              </svg>
              <span>{{ t('stepConfig.ioPorts.inputs') }}</span>
            </div>
            <div class="ports-list">
              <div v-for="port in currentBlockDef.input_ports" :key="port.name" class="port-item">
                <span class="port-name">{{ port.label }}</span>
                <code class="port-type">{{ formatSchemaType(port.schema) }}</code>
                <span v-if="port.required" class="port-required">*</span>
                <span v-if="port.description" class="port-desc">{{ port.description }}</span>
              </div>
            </div>
          </div>

          <!-- Output Ports (for blocks not covered by specific config sections) -->
          <div v-if="currentBlockDef.output_ports?.length > 1 && !['condition', 'switch', 'human_in_loop'].includes(formType)" class="io-ports-group">
            <div class="ports-header">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="9 18 15 12 9 6"/>
              </svg>
              <span>{{ t('stepConfig.ioPorts.outputs') }}</span>
            </div>
            <div class="ports-list">
              <div v-for="port in currentBlockDef.output_ports" :key="port.name" class="port-item">
                <span :class="['port-name', { 'port-default': port.is_default }]">{{ port.label }}</span>
                <code class="port-type">{{ formatSchemaType(port.schema) }}</code>
                <span v-if="port.is_default" class="port-default-badge">{{ t('stepConfig.ioPorts.default') }}</span>
                <span v-if="port.description" class="port-desc">{{ port.description }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Metadata -->
        <div class="form-section">
          <h4 class="section-title">{{ t('stepConfig.metadata') }}</h4>
          <div class="metadata-item">
            <span class="metadata-label">{{ t('stepConfig.stepId') }}</span>
            <code class="metadata-value">{{ step.id }}</code>
          </div>
        </div>
      </div>
    </div>

    <!-- Flow Tab Content -->
    <div v-if="activeTab === 'flow'" class="properties-body flow-container">
      <FlowTab
        :step="step"
        :block-definitions="blockDefinitions"
        :readonly-mode="readonlyMode"
        @update:flow-config="handleFlowConfigUpdate"
      />
    </div>

    <!-- Trigger Tab Content (only for Start blocks) -->
    <div v-if="activeTab === 'trigger' && isStartBlock" class="properties-body trigger-container">
      <TriggerConfigPanel
        :trigger-type="(step?.trigger_type as StartTriggerType) || 'manual'"
        :trigger-config="step?.trigger_config as object || {}"
        :step-id="step?.id"
        :readonly="readonlyMode"
        @update:trigger="handleTriggerUpdate"
      />
    </div>

    <!-- Copilot Tab Content (always available) -->
    <div v-if="activeTab === 'copilot'" class="properties-body copilot-container">
      <CopilotTab
        :step="step"
        :workflow-id="workflowId"
        @apply-suggestion="handleApplySuggestion"
        @apply-workflow="handleApplyWorkflow"
      />
    </div>

    <!-- Run Tab Content -->
    <div v-if="activeTab === 'run'" class="properties-body execution-container">
      <ExecutionTab
        :step="step"
        :workflow-id="workflowId"
        :latest-run="latestRun || null"
        :is-active="activeTab === 'run'"
        :steps="steps || []"
        :edges="edges || []"
        :blocks="blockDefinitions || []"
        @execute="(data) => emit('execute', data)"
        @execute-workflow="(mode, input) => emit('execute-workflow', mode, input)"
      />
    </div>

    <!-- Footer with actions (only when step selected) -->
    <div v-if="step" class="properties-footer">
      <button
        v-if="!readonlyMode && !isStartNode"
        class="btn btn-danger-outline"
        :disabled="saving"
        @click="handleDelete"
      >
        {{ t('common.delete') }}
      </button>
      <div class="footer-spacer" />
      <button
        v-if="!readonlyMode"
        class="btn btn-primary"
        :disabled="saving"
        @click="handleSave"
      >
        {{ saving ? t('common.saving') : t('common.save') }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.properties-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--color-surface);
}
/* Body Wrapper for Settings Tab */
.properties-body-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* Empty State */
.properties-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 1.5rem;
  text-align: center;
  overflow-y: auto;
  height: 100%;
}

.empty-icon {
  color: var(--color-primary);
  opacity: 0.6;
  margin-bottom: 1rem;
  margin-top: 1rem;
}

.empty-title {
  font-weight: 600;
  font-size: 1rem;
  color: var(--color-text);
  margin: 0;
}

.empty-desc {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  margin-top: 0.5rem;
  line-height: 1.5;
}

/* Quick Tips */
.empty-tips {
  width: 100%;
  margin-top: 1.5rem;
  padding: 1rem;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border-radius: 8px;
  text-align: left;
}

.empty-tips-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: #0369a1;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.75rem;
}

.empty-tips-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.empty-tips-list li {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  color: #0c4a6e;
  padding: 0.375rem 0;
}

.tip-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  background: white;
  border-radius: 4px;
  color: #0284c7;
  flex-shrink: 0;
}

/* Keyboard Shortcuts */
.empty-shortcuts {
  width: 100%;
  margin-top: 1rem;
  padding: 1rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  text-align: left;
}

.empty-shortcuts-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 0.75rem;
}

.shortcut-item {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  padding: 0.25rem 0;
}

.shortcut-item kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 1.5rem;
  padding: 0.125rem 0.375rem;
  font-family: inherit;
  font-size: 0.625rem;
  font-weight: 500;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.shortcut-item span {
  margin-left: auto;
  color: var(--color-text);
}

/* Header */
.properties-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.header-color {
  width: 4px;
  height: 32px;
  border-radius: 2px;
  flex-shrink: 0;
}

.header-info {
  flex: 1;
}

.header-title {
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.header-type {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Tab Bar */
.properties-tabs {
  display: flex;
  gap: 0;
  padding: 0 1rem;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.tab-button {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.625rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: all 0.15s;
  margin-bottom: -1px;
}

.tab-button:hover {
  color: var(--color-text);
}

.tab-button.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

.tab-button svg {
  opacity: 0.7;
}

.tab-button.active svg {
  opacity: 1;
}

/* Flow Container */
.flow-container {
  padding: 0;
}

/* Copilot Container */
.copilot-container {
  padding: 0.75rem 1rem;
}

/* Execution Container */
.execution-container {
  padding: 0.75rem 1rem;
}

/* Body */
.properties-body {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
}

/* Form Sections */
.form-section {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.form-section:last-child {
  margin-bottom: 0;
  padding-bottom: 0;
  border-bottom: none;
}

.section-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* Form Elements */
.form-group {
  margin-bottom: 0.875rem;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-label {
  display: block;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
  margin-bottom: 0.375rem;
}

.form-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  font-size: 0.8125rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: white;
  color: var(--color-text);
  transition: border-color 0.15s;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input:disabled {
  background: var(--color-surface);
  cursor: not-allowed;
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
}

.code-input {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
}

.form-hint {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

.form-hint code {
  background: var(--color-surface);
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  font-size: 0.625rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

/* Adapter Grid */
.adapter-grid {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.adapter-option {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.625rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.adapter-option:hover {
  border-color: var(--color-primary);
}

.adapter-option.selected {
  border-color: var(--color-primary);
  background: rgba(59, 130, 246, 0.05);
}

.adapter-option input {
  margin-top: 0.125rem;
}

.adapter-info {
  flex: 1;
}

.adapter-name {
  font-size: 0.8125rem;
  font-weight: 500;
}

.adapter-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.125rem;
}

/* Condition Preview */
.condition-preview {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: var(--color-background);
  border-radius: 6px;
}

.condition-branch {
  display: flex;
  align-items: center;
  gap: 0.625rem;
}

.branch-label {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}

.branch-true {
  background: #dcfce7;
  color: #16a34a;
}

.branch-false {
  background: #fee2e2;
  color: #dc2626;
}

.branch-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}

/* Checkbox */
.form-checkbox {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.8125rem;
}

.form-checkbox input {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

/* Info Box */
.info-box {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.75rem;
  background: rgba(59, 130, 246, 0.05);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 6px;
  margin-top: 0.75rem;
  font-size: 0.6875rem;
  color: #1e40af;
}

.info-box svg {
  flex-shrink: 0;
  margin-top: 0.125rem;
}

/* Start Node Info Box */
.start-info-box {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem;
  background: linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%);
  border: 1px solid #a7f3d0;
  border-radius: 8px;
}

.start-icon {
  color: #10b981;
  flex-shrink: 0;
}

.start-info-content {
  flex: 1;
}

.start-info-title {
  font-size: 0.875rem;
  font-weight: 600;
  color: #047857;
  margin: 0 0 0.25rem 0;
}

.start-info-desc {
  font-size: 0.75rem;
  color: #065f46;
  margin: 0;
  line-height: 1.4;
}

/* Start Warning */
.start-warning {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  background: #fef3c7;
  border: 1px solid #fcd34d;
  border-radius: 6px;
  margin-top: 0.75rem;
  font-size: 0.6875rem;
  color: #92400e;
}

.start-warning svg {
  flex-shrink: 0;
  color: #f59e0b;
}

/* Metadata */
.metadata-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.metadata-label {
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.metadata-value {
  font-size: 0.6875rem;
  background: var(--color-background);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

/* Footer */
.properties-footer {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  border-top: 1px solid var(--color-border);
  flex-shrink: 0;
}

.footer-spacer {
  flex: 1;
}

/* Button Variants */
.btn-danger-outline {
  background: white;
  border: 1px solid #fecaca;
  color: var(--color-error);
}

.btn-danger-outline:hover {
  background: #fef2f2;
  border-color: var(--color-error);
}

/* Scrollbar */
.properties-body::-webkit-scrollbar {
  width: 6px;
}

.properties-body::-webkit-scrollbar-track {
  background: transparent;
}

.properties-body::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}

.properties-body::-webkit-scrollbar-thumb:hover {
  background: var(--color-text-secondary);
}

/* Expression Helpers */
.expression-helpers {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-top: 0.5rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.helper-label {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}

.helper-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.helper-chip {
  padding: 0.25rem 0.5rem;
  font-size: 0.6875rem;
  font-family: 'SF Mono', Monaco, monospace;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.15s;
}

.helper-chip:hover:not(:disabled) {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

.helper-chip:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

/* Output Ports Preview */
.output-ports-preview {
  margin-top: 0.75rem;
}

.ports-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 0.5rem;
}

.branch-type {
  font-size: 0.625rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

/* Switch Cases */
.switch-cases-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.switch-case-item {
  padding: 0.75rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.switch-case-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.case-name-input {
  flex: 1;
  padding: 0.375rem 0.5rem;
  font-size: 0.75rem;
}

.btn-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  cursor: pointer;
  color: var(--color-text-secondary);
  transition: all 0.15s;
}

.btn-remove-case:hover {
  background: #fef2f2;
  border-color: #fecaca;
  color: var(--color-error);
}

.case-default-checkbox {
  margin-top: 0.5rem;
  font-size: 0.75rem;
}

.btn-add-case {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  width: 100%;
  padding: 0.5rem;
  font-size: 0.75rem;
  color: var(--color-primary);
  background: transparent;
  border: 1px dashed var(--color-primary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-add-case:hover {
  background: rgba(59, 130, 246, 0.05);
}

.switch-ports-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.switch-port-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.branch-case {
  background: #e0e7ff;
  color: #4f46e5;
}

.branch-default {
  background: #fef3c7;
  color: #92400e;
}

/* I/O Ports Section */
.io-ports-group {
  margin-bottom: 0.75rem;
}

.io-ports-group:last-child {
  margin-bottom: 0;
}

.ports-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  padding: 0.5rem;
  background: var(--color-background);
  border-radius: 6px;
}

.port-item {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.375rem;
  padding: 0.25rem 0;
}

.port-name {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text);
}

.port-name.port-default {
  color: var(--color-primary);
}

.port-type {
  font-size: 0.625rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.port-required {
  font-size: 0.75rem;
  color: var(--color-error);
  font-weight: 600;
}

.port-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  width: 100%;
}

.port-default-badge {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 0.125rem 0.375rem;
  background: #e0e7ff;
  color: #4f46e5;
  border-radius: 3px;
}

.section-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.75rem 0;
  line-height: 1.4;
}

/* Available Input Variables Section */
.available-variables-section {
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
  border: 1px solid #86efac;
  border-radius: 8px;
  padding: 0.875rem !important;
  margin-top: 0.5rem;
}

.available-variables-section .section-title {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  color: #166534;
  margin-bottom: 0.5rem;
}

.available-variables-section .section-title svg {
  color: #22c55e;
}

.available-variables-section .section-description {
  color: #15803d;
}

.available-variables-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  background: rgba(255, 255, 255, 0.7);
  border-radius: 6px;
  padding: 0.75rem;
  max-height: 200px;
  overflow-y: auto;
}

.available-variable-item {
  padding: 0.5rem;
  background: white;
  border: 1px solid #bbf7d0;
  border-radius: 4px;
}

.variable-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  flex-wrap: wrap;
}

.variable-path {
  font-size: 0.6875rem;
  font-family: 'SF Mono', Monaco, monospace;
  background: #dcfce7;
  color: #166534;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  border: 1px solid #86efac;
  word-break: break-all;
}

.variable-type {
  font-size: 0.5625rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: var(--color-text-secondary);
  background: var(--color-background);
  padding: 0.125rem 0.25rem;
  border-radius: 3px;
}

.variable-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.25rem;
  font-size: 0.625rem;
}

.variable-source {
  color: #15803d;
  font-weight: 500;
}

.variable-title {
  color: var(--color-text-secondary);
}

.variable-description {
  font-size: 0.5625rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
  line-height: 1.4;
}

/* Template Preview Section */
.template-preview-section {
  background: linear-gradient(135deg, #fefce8 0%, #fef9c3 100%);
  border: 1px solid #fde047;
  border-radius: 8px;
  padding: 0.875rem !important;
  margin-top: 0.5rem;
}

.template-preview-section .section-title {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  color: #854d0e;
  margin-bottom: 0.5rem;
}

.template-preview-section .section-title svg {
  color: #ca8a04;
}

.template-preview-hint {
  font-size: 0.6875rem;
  color: #a16207;
  margin: 0 0 0.75rem 0;
  line-height: 1.4;
}

.template-variables-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  background: rgba(255, 255, 255, 0.7);
  border-radius: 6px;
  padding: 0.75rem;
}

.template-variable-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
}

.variable-name {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.6875rem;
  background: #fef3c7;
  color: #92400e;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  border: 1px solid #fcd34d;
}

.variable-arrow {
  color: #d97706;
  font-weight: bold;
}

.variable-placeholder {
  font-size: 0.6875rem;
  color: #78716c;
  font-style: italic;
}

.template-preview-note {
  display: flex;
  align-items: flex-start;
  gap: 0.375rem;
  margin-top: 0.75rem;
  padding: 0.5rem 0.625rem;
  background: rgba(255, 255, 255, 0.5);
  border-radius: 4px;
  font-size: 0.625rem;
  color: #78716c;
}

.template-preview-note svg {
  flex-shrink: 0;
  margin-top: 0.125rem;
  color: #a3a3a3;
}
</style>
