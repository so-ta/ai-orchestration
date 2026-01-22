<script setup lang="ts">
import type { Step, StepType, Run } from '~/types/api'
import type { GenerateWorkflowResponse } from '~/composables/useCopilot'
import type { ConfigSchema, UIConfig } from './config/types/config-schema'
import DynamicConfigForm from './config/DynamicConfigForm.vue'
import FlowTab from './FlowTab.vue'
import TriggerConfigPanel from './TriggerConfigPanel.vue'
import TestTab from './properties/TestTab.vue'
// Sub-components
import PropertiesEmptyState from './properties/PropertiesEmptyState.vue'
import TemplatePreviewSection from './properties/TemplatePreviewSection.vue'
import AvailableVariablesSection from './properties/AvailableVariablesSection.vue'
import IOPortsDisplay from './properties/IOPortsDisplay.vue'
import OutputSchemaPreview from './properties/OutputSchemaPreview.vue'
import CredentialBindingsSection from '../credentials/CredentialBindingsSection.vue'
import { useStepTest, type StepTestResult } from '~/composables/test'
// Composables
import { useAvailableVariables } from './composables/useAvailableVariables'
import { AVAILABLE_VARIABLES_KEY, ACTIVE_FIELD_INSERTER_KEY } from './config/variable-picker/useVariableInsertion'
import {
  usePropertyForm,
  useBlockDefinition,
  useTemplateVariablesExtractor,
  useFieldInserter,
  type StepConfig,
} from './properties/composables/usePropertyForm'

type StartTriggerType = 'manual' | 'webhook' | 'schedule' | 'slack' | 'email'

const { t } = useI18n()
const { confirm } = useConfirm()

const props = defineProps<{
  step: Step | null
  workflowId: string
  readonlyMode?: boolean
  latestRun?: Run | null
  steps?: Step[]
  edges?: Array<{ id: string; source_step_id?: string | null; target_step_id?: string | null }>
  blockDefinitions?: import('~/types/api').BlockDefinition[]
}>()

const emit = defineEmits<{
  'save': [data: { name: string; type: StepType; config: StepConfig; credential_bindings?: Record<string, string> }]
  'delete': []
  'open-settings': []
  'apply-workflow': [workflow: GenerateWorkflowResponse]
  'execute': [data: { stepId: string; input: object; triggered_by: 'test' | 'manual' }]
  'execute-workflow': [triggered_by: 'test' | 'manual', input: object]
  'update:name': [name: string]
  'update:trigger': [data: { trigger_type: StartTriggerType; trigger_config: object }]
  'update:credential-bindings': [bindings: Record<string, string>]
  'run:created': [run: Run]
  'test-result': [result: StepTestResult]
}>()

// Active tab state
const activeTab = ref<'config' | 'test' | 'flow'>('config')

// Step test composable
const {
  executing: testExecuting,
  currentResult: testResult,
  error: testError,
  pinnedOutput,
  executeStepOnly,
  executeFromStep,
  clearResult,
  loadPinnedOutput,
  pinOutput,
  unpinOutput,
  editPinnedOutput,
} = useStepTest({
  workflowId: props.workflowId,
  onTestRunsChanged: () => {
    // Optionally notify parent to refresh test runs
  },
})

// Load pinned output when step changes
watch(() => props.step?.id, (stepId) => {
  if (stepId) {
    loadPinnedOutput(stepId)
    clearResult()
  }
}, { immediate: true })

// Emit test result to parent for floating panel display
watch(testResult, (result) => {
  if (result) {
    emit('test-result', result)
  }
})

// Trigger block types
const TRIGGER_BLOCK_TYPES = ['start', 'manual_trigger', 'schedule_trigger', 'webhook_trigger']

// Check if step is a trigger block (start or any trigger variant)
const isTriggerBlock = computed(() => {
  const stepType = props.step?.type
  return stepType ? TRIGGER_BLOCK_TYPES.includes(stepType) : false
})

const isStartNode = computed(() => {
  const stepType = props.step?.type
  return stepType ? TRIGGER_BLOCK_TYPES.includes(stepType) : false
})

// Step type colors
const stepTypeColors: Record<string, string> = {
  start: '#10b981',
  manual_trigger: '#10b981',
  schedule_trigger: '#22c55e',
  webhook_trigger: '#3b82f6',
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

// Refs for composables
const stepRef = computed(() => props.step)
const readonlyRef = computed(() => props.readonlyMode ?? false)

// Use property form composable
const {
  formName,
  formType,
  formConfig,
  handleFlowConfigUpdate,
  handleCredentialBindingsUpdate,
} = usePropertyForm({
  step: stepRef,
  readonlyMode: readonlyRef,
  onSave: (data) => emit('save', data),
  onUpdateName: (name) => emit('update:name', name),
  onUpdateCredentialBindings: (bindings) => emit('update:credential-bindings', bindings),
})

// Use block definition composable
const { currentBlockDef } = useBlockDefinition({
  stepType: formType,
})

// Use template variables extractor
const { templateVariables } = useTemplateVariablesExtractor(formConfig)

// Use field inserter composable
const {
  activeFieldId,
  register: registerFieldInserter,
  unregister: unregisterFieldInserter,
  setActive: setActiveFieldInserter,
  insertVariableIntoActiveField,
} = useFieldInserter()

// Config schema computed
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

// Fixed trigger type from block definition (for inherited trigger blocks)
const fixedTriggerType = computed<StartTriggerType | undefined>(() => {
  if (!currentBlockDef.value) return undefined
  const defaults = currentBlockDef.value.resolved_config_defaults || (currentBlockDef.value as unknown as { config_defaults?: Record<string, unknown> }).config_defaults
  if (defaults && typeof defaults === 'object' && 'trigger_type' in defaults) {
    return (defaults as { trigger_type: string }).trigger_type as StartTriggerType
  }
  return undefined
})

// Template block detection
const isTemplateBlock = computed(() => {
  const templateTypes = ['llm', 'discord', 'slack', 'email_sendgrid', 'log', 'function']
  return templateTypes.includes(formType.value)
})

const showTemplatePreview = computed(() => {
  return templateVariables.value.length > 0 && isTemplateBlock.value
})

// Available variables from previous steps
const stepsRef = computed(() => props.steps)
const edgesRef = computed(() => props.edges)
const { previousSteps, availableInputVariables, hasAvailableVariables } = useAvailableVariables(stepRef, stepsRef, edgesRef)

// Provide available variables to child widgets
provide(AVAILABLE_VARIABLES_KEY, availableInputVariables)

// Provide field inserter to child widgets
provide(ACTIVE_FIELD_INSERTER_KEY, {
  register: registerFieldInserter,
  unregister: unregisterFieldInserter,
  setActive: setActiveFieldInserter,
  activeId: activeFieldId
})

// Show I/O ports
const showIOPorts = computed(() => {
  return currentBlockDef.value && (currentBlockDef.value.output_ports?.length ?? 0) > 1
})

// Handlers
function handleTriggerUpdate(data: { trigger_type: StartTriggerType; trigger_config: object }) {
  emit('update:trigger', data)
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

function handleOpenSettings() {
  emit('open-settings')
}
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
      <button v-if="!isTriggerBlock" class="tab-button" :class="{ active: activeTab === 'flow' }" @click="activeTab = 'flow'">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M5 3l-1 9"/><path d="M19 3l1 9"/><polyline points="8 14 12 18 16 14"/>
        </svg>
        {{ t('editor.tabs.flow') }}
      </button>
      <button class="tab-button" :class="{ active: activeTab === 'test' }" @click="activeTab = 'test'">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="5 3 19 12 5 21 5 3"/>
        </svg>
        {{ t('editor.tabs.test') }}
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
        <div v-if="hasConfigSchema && !['join', 'note', ...TRIGGER_BLOCK_TYPES].includes(formType)" class="form-section">
          <h4 class="section-title">{{ t('widgets.propertiesPanel.configSectionTitle', { name: currentBlockDef?.name || formType }) }}</h4>
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
        <div v-if="isTriggerBlock" class="form-section trigger-section">
          <TriggerConfigPanel
            :trigger-type="(step?.trigger_type as StartTriggerType) || 'manual'"
            :trigger-config="step?.trigger_config as object || {}"
            :step-id="step?.id"
            :workflow-id="workflowId"
            :readonly="readonlyMode"
            :fixed-trigger-type="fixedTriggerType"
            class="integrated-trigger-panel"
            @update:trigger="handleTriggerUpdate"
          />
        </div>

        <!-- Available Variables Section -->
        <AvailableVariablesSection
          v-if="hasAvailableVariables && isTemplateBlock"
          :variables="availableInputVariables"
          @insert="(v) => insertVariableIntoActiveField(v.path)"
        />

        <!-- Output Schema Preview -->
        <OutputSchemaPreview
          v-if="previousSteps.length > 0 && isTemplateBlock"
          :previous-steps="previousSteps"
          @insert="insertVariableIntoActiveField"
        />

        <!-- Template Preview Section -->
        <TemplatePreviewSection v-if="showTemplatePreview" :variables="templateVariables" />

        <!-- I/O Ports Display -->
        <IOPortsDisplay
          v-if="showIOPorts"
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

    <!-- Test Tab Content -->
    <div v-if="activeTab === 'test'" class="properties-body execution-container">
      <TestTab
        :step="step"
        :workflow-id="workflowId"
        :is-active="activeTab === 'test'"
        :steps="steps || []"
        :edges="edges || []"
        :block-definitions="blockDefinitions || []"
        :executing="testExecuting"
        :current-result="testResult"
        :error="testError"
        :pinned-output="pinnedOutput"
        @execute-step-only="(step, input) => executeStepOnly(step, input)"
        @execute-from-step="(step, input) => executeFromStep(step, input)"
        @pin-output="(output, stepId, stepName) => pinOutput(output, stepId, stepName)"
        @unpin-output="(stepId) => unpinOutput(stepId)"
        @edit-pinned="(data, stepId) => editPinnedOutput(data, stepId)"
      />
    </div>

    <!-- Footer (delete button only - changes are auto-saved) -->
    <div v-if="step && !readonlyMode && !isStartNode" class="properties-footer">
      <button class="btn btn-danger-outline" @click="handleDelete">
        {{ t('common.delete') }}
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
