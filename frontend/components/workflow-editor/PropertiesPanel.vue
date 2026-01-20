<script setup lang="ts">
import type { Step, StepType, BlockDefinition, Run } from '~/types/api'
import type { GenerateWorkflowResponse } from '~/composables/useCopilot'
import type { ConfigSchema, UIConfig } from './config/types/config-schema'
import DynamicConfigForm from './config/DynamicConfigForm.vue'
import FlowTab from './FlowTab.vue'
import TriggerConfigPanel from './TriggerConfigPanel.vue'
// Sub-components
import PropertiesEmptyState from './properties/PropertiesEmptyState.vue'
import TemplatePreviewSection from './properties/TemplatePreviewSection.vue'
import AvailableVariablesSection from './properties/AvailableVariablesSection.vue'
import IOPortsDisplay from './properties/IOPortsDisplay.vue'
import CredentialBindingsSection from '../credentials/CredentialBindingsSection.vue'
// Legacy Forms
import LegacyLlmForm from './properties/legacy/LegacyLlmForm.vue'
import LegacyToolForm from './properties/legacy/LegacyToolForm.vue'
import LegacyConditionForm from './properties/legacy/LegacyConditionForm.vue'
import LegacySwitchForm from './properties/legacy/LegacySwitchForm.vue'
import LegacyLoopForm from './properties/legacy/LegacyLoopForm.vue'
import LegacyWaitForm from './properties/legacy/LegacyWaitForm.vue'
import LegacyFunctionForm from './properties/legacy/LegacyFunctionForm.vue'
import LegacyRouterForm from './properties/legacy/LegacyRouterForm.vue'
import LegacyHumanInLoopForm from './properties/legacy/LegacyHumanInLoopForm.vue'
import LegacyMapForm from './properties/legacy/LegacyMapForm.vue'
import LegacySubflowForm from './properties/legacy/LegacySubflowForm.vue'
import LegacyStartForm from './properties/legacy/LegacyStartForm.vue'
import LegacyLogForm from './properties/legacy/LegacyLogForm.vue'
// Composables
import { useAvailableVariables } from './composables/useAvailableVariables'

type StartTriggerType = 'manual' | 'webhook' | 'schedule' | 'slack' | 'email'

const { t } = useI18n()
const blocks = useBlocks()
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
const activeTab = ref<'config' | 'flow' | 'copilot' | 'run'>('config')

// Check if step is a generic start block
const isGenericStartBlock = computed(() => props.step?.type === 'start')

const emit = defineEmits<{
  (e: 'save', data: { name: string; type: StepType; config: StepConfig; credential_bindings?: Record<string, string> }): void
  (e: 'delete' | 'open-settings'): void
  (e: 'apply-workflow', workflow: GenerateWorkflowResponse): void
  (e: 'execute', data: { stepId: string; input: object; triggered_by: 'test' | 'manual' }): void
  (e: 'execute-workflow', triggered_by: 'test' | 'manual', input: object): void
  (e: 'update:name', name: string): void
  (e: 'update:trigger', data: { trigger_type: StartTriggerType; trigger_config: object }): void
  (e: 'update:credential-bindings', bindings: Record<string, string>): void
  (e: 'run:created', run: Run): void
}>()

// Step config type
interface StepConfig {
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

// Emit name changes for reactive updates
watch(formName, (newName) => {
  if (props.step && newName !== props.step.name) {
    emit('update:name', newName)
  }
})

// Step type colors
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

const isStartNode = computed(() => props.step?.type === 'start')

// Flow config
const flowConfig = ref<{
  prescript?: { enabled: boolean; code: string }
  postscript?: { enabled: boolean; code: string }
  error_handling?: { enabled: boolean; retry?: object; timeout_seconds?: number; on_error: string; fallback_value?: unknown; enable_error_port?: boolean }
}>({})

function handleFlowConfigUpdate(config: typeof flowConfig.value) {
  flowConfig.value = config
}

function handleTriggerUpdate(data: { trigger_type: StartTriggerType; trigger_config: object }) {
  emit('update:trigger', data)
}

function handleSave() {
  const mergedConfig = { ...formConfig.value, ...flowConfig.value }
  const saveData: { name: string; type: StepType; config: StepConfig; credential_bindings?: Record<string, string> } = {
    name: formName.value,
    type: formType.value,
    config: mergedConfig
  }
  // Include credential_bindings if any are set
  if (Object.keys(localCredentialBindings.value).length > 0) {
    saveData.credential_bindings = localCredentialBindings.value
  }
  emit('save', saveData)
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

// Credential bindings state
const localCredentialBindings = ref<Record<string, string>>({})

// Initialize credential bindings from props
watch(() => props.step?.credential_bindings, (bindings) => {
  localCredentialBindings.value = { ...bindings }
}, { immediate: true, deep: true })

function handleCredentialBindingsUpdate(bindings: Record<string, string>) {
  localCredentialBindings.value = bindings
  emit('update:credential-bindings', bindings)
}

function handleOpenSettings() {
  emit('open-settings')
}

// Block definition for current step type
const currentBlockDef = ref<BlockDefinition | null>(null)
const loadingBlockDef = ref(false)

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

const hasConfigSchema = computed(() => {
  const schema = currentBlockDef.value?.config_schema
  return schema && typeof schema === 'object' && Object.keys(schema).length > 0
})

const configSchema = computed<ConfigSchema | null>(() => {
  if (!hasConfigSchema.value) return null
  return currentBlockDef.value?.config_schema as ConfigSchema
})

const uiConfig = computed<UIConfig | undefined>(() => {
  if (!currentBlockDef.value?.ui_config) return undefined
  return currentBlockDef.value.ui_config as UIConfig
})

// Template variables
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

const isTemplateBlock = computed(() => {
  const templateTypes = ['llm', 'discord', 'slack', 'email_sendgrid', 'log', 'function']
  return templateTypes.includes(formType.value)
})

const showTemplatePreview = computed(() => {
  return templateVariables.value.length > 0 && isTemplateBlock.value
})

// Available variables from previous steps
const stepRef = computed(() => props.step)
const stepsRef = computed(() => props.steps)
const edgesRef = computed(() => props.edges)
const { availableInputVariables, hasAvailableVariables } = useAvailableVariables(stepRef, stepsRef, edgesRef)

// Show I/O ports
const showIOPorts = computed(() => {
  return currentBlockDef.value && (
    (currentBlockDef.value.input_ports?.length ?? 0) > 1 ||
    (currentBlockDef.value.output_ports?.length ?? 0) > 1
  )
})
</script>

<template>
  <div class="properties-panel">
    <!-- Header -->
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

    <!-- Tab Bar -->
    <div class="properties-tabs">
      <button class="tab-button" :class="{ active: activeTab === 'config' }" @click="activeTab = 'config'">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
        </svg>
        {{ t('editor.tabs.config') }}
      </button>
      <button v-if="!isGenericStartBlock" class="tab-button" :class="{ active: activeTab === 'flow' }" @click="activeTab = 'flow'">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M5 3l-1 9"/><path d="M19 3l1 9"/><polyline points="8 14 12 18 16 14"/>
        </svg>
        {{ t('editor.tabs.flow') }}
      </button>
      <button class="tab-button" :class="{ active: activeTab === 'copilot' }" @click="activeTab = 'copilot'">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73A2 2 0 0 1 10 4a2 2 0 0 1 2-2z"/>
          <circle cx="8" cy="14" r="2"/><circle cx="16" cy="14" r="2"/>
        </svg>
        {{ t('editor.tabs.copilot') }}
      </button>
      <button class="tab-button" :class="{ active: activeTab === 'run' }" @click="activeTab = 'run'">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="5 3 19 12 5 21 5 3"/>
        </svg>
        {{ t('editor.tabs.run') }}
      </button>
    </div>

    <!-- Config Tab Content -->
    <div v-if="activeTab === 'config'" class="properties-body-wrapper">
      <PropertiesEmptyState v-if="!step" />

      <div v-else class="properties-body">
        <!-- Basic Information -->
        <div class="form-section">
          <h4 class="section-title">{{ t('stepConfig.basicInfo') }}</h4>
          <div class="form-group">
            <label class="form-label">{{ t('stepConfig.stepName') }}</label>
            <input v-model="formName" type="text" class="form-input" :placeholder="t('stepConfig.stepNamePlaceholder')" :disabled="readonlyMode">
          </div>
        </div>

        <!-- Dynamic Config Form -->
        <div v-if="hasConfigSchema && !['start', 'join', 'note'].includes(formType)" class="form-section">
          <h4 class="section-title">{{ currentBlockDef?.name || formType }} 設定</h4>
          <DynamicConfigForm v-model="formConfig" :schema="configSchema" :ui-config="uiConfig" :disabled="readonlyMode" />
        </div>

        <!-- Credential Bindings -->
        <CredentialBindingsSection
          v-if="currentBlockDef"
          :block-definition="currentBlockDef"
          :credential-bindings="step?.credential_bindings"
          :readonly="readonlyMode"
          @update:credential-bindings="handleCredentialBindingsUpdate"
          @open-settings="handleOpenSettings"
        />

        <!-- Trigger Configuration -->
        <div v-if="isGenericStartBlock" class="form-section trigger-section">
          <h4 class="section-title">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/>
            </svg>
            {{ t('editor.tabs.trigger') }}
          </h4>
          <TriggerConfigPanel
            :trigger-type="(step?.trigger_type as StartTriggerType) || 'manual'"
            :trigger-config="step?.trigger_config as object || {}"
            :step-id="step?.id"
            :readonly="readonlyMode"
            class="integrated-trigger-panel"
            @update:trigger="handleTriggerUpdate"
          />
        </div>

        <!-- Available Variables Section -->
        <AvailableVariablesSection v-if="hasAvailableVariables && isTemplateBlock" :variables="availableInputVariables" />

        <!-- Template Preview Section -->
        <TemplatePreviewSection v-if="showTemplatePreview" :variables="templateVariables" />

        <!-- Legacy Forms (when no config_schema) -->
        <LegacyLlmForm v-if="!hasConfigSchema && formType === 'llm'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyToolForm v-if="!hasConfigSchema && formType === 'tool'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyConditionForm v-if="!hasConfigSchema && formType === 'condition'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacySwitchForm v-if="!hasConfigSchema && formType === 'switch'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyLoopForm v-if="!hasConfigSchema && formType === 'loop'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyWaitForm v-if="!hasConfigSchema && formType === 'wait'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyFunctionForm v-if="!hasConfigSchema && formType === 'function'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyRouterForm v-if="!hasConfigSchema && formType === 'router'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyHumanInLoopForm v-if="!hasConfigSchema && formType === 'human_in_loop'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyMapForm v-if="!hasConfigSchema && formType === 'map'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacySubflowForm v-if="!hasConfigSchema && formType === 'subflow'" v-model="formConfig" :disabled="readonlyMode" />
        <LegacyStartForm v-if="!hasConfigSchema && formType === 'start'" :disabled="readonlyMode" />
        <LegacyLogForm v-if="!hasConfigSchema && formType === 'log'" v-model="formConfig" :disabled="readonlyMode" />

        <!-- I/O Ports Display -->
        <IOPortsDisplay
          v-if="showIOPorts"
          :input-ports="currentBlockDef?.input_ports"
          :output-ports="currentBlockDef?.output_ports"
          :step-type="formType"
        />

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
      <FlowTab :step="step" :block-definitions="blockDefinitions" :readonly-mode="readonlyMode" @update:flow-config="handleFlowConfigUpdate" />
    </div>

    <!-- Copilot Tab Content -->
    <div v-if="activeTab === 'copilot'" class="properties-body copilot-container">
      <CopilotTab :workflow-id="workflowId" />
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
        @run:created="(run) => emit('run:created', run)"
      />
    </div>

    <!-- Footer -->
    <div v-if="step" class="properties-footer">
      <button v-if="!readonlyMode && !isStartNode" class="btn btn-danger-outline" :disabled="saving" @click="handleDelete">
        {{ t('common.delete') }}
      </button>
      <div class="footer-spacer" />
      <button v-if="!readonlyMode" class="btn btn-primary" :disabled="saving" @click="handleSave">
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

.properties-body-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

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

.properties-body {
  flex: 1;
  overflow-y: auto;
  padding: 1rem;
}

.properties-body.copilot-container,
.properties-body.execution-container {
  padding: 0.75rem 1rem;
  overflow: hidden;
}

.properties-body.flow-container {
  padding: 1rem;
}

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

.integrated-trigger-panel {
  margin-top: 0.5rem;
}

.trigger-section .section-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.trigger-section .section-title svg {
  opacity: 0.7;
}

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

.btn-danger-outline {
  background: white;
  border: 1px solid #fecaca;
  color: var(--color-error);
}

.btn-danger-outline:hover {
  background: #fef2f2;
  border-color: var(--color-error);
}

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
</style>
