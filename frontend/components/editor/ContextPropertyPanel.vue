<script setup lang="ts">
import type { Step, StepType, BlockDefinition } from '~/types/api'
import type { ConfigSchema, UIConfig } from '~/components/workflow-editor/config/types/config-schema'
import DynamicConfigForm from '~/components/workflow-editor/config/DynamicConfigForm.vue'
import FlowTab from '~/components/workflow-editor/FlowTab.vue'

const { t } = useI18n()
const blocks = useBlocks()
const { confirm } = useConfirm()

const props = defineProps<{
  step: Step
  position: { x: number; y: number } | null
  workflowId: string
  saving?: boolean
}>()

const emit = defineEmits<{
  save: [data: { name: string; type: StepType; config: Record<string, unknown> }]
  delete: []
  close: []
  'update:name': [name: string]
}>()

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

// Form state
const formName = ref('')
const formConfig = ref<Record<string, unknown>>({})
const showAdvanced = ref(false)

// Initialize form when step changes
watch(() => props.step, (newStep) => {
  if (newStep) {
    formName.value = newStep.name
    formConfig.value = { ...(newStep.config as Record<string, unknown>) }
  }
}, { immediate: true, deep: true })

// Emit name changes
watch(formName, (newName) => {
  if (props.step && newName !== props.step.name) {
    emit('update:name', newName)
  }
})

// Block definition
const currentBlockDef = ref<BlockDefinition | null>(null)
const loadingBlockDef = ref(false)

watch(() => props.step?.type, async (newType) => {
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
  }
}, { immediate: true })

// Config schema
const hasConfigSchema = computed(() => {
  const schema = currentBlockDef.value?.config_schema
  return schema && typeof schema === 'object' && Object.keys(schema).length > 0
})

const configSchema = computed<ConfigSchema | null>(() => {
  if (!hasConfigSchema.value) return null
  return currentBlockDef.value?.config_schema as ConfigSchema
})

const uiConfig = computed<UIConfig | undefined>(() => {
  return currentBlockDef.value?.ui_config as UIConfig | undefined
})

// Step color
const stepColor = computed(() => stepTypeColors[props.step?.type || 'tool'] || '#6b7280')

// Is start block (cannot delete)
const isStartBlock = computed(() => props.step?.type === 'start')

// Flow config
const flowConfig = ref<Record<string, unknown>>({})

function handleFlowConfigUpdate(config: Record<string, unknown>) {
  flowConfig.value = config
}

// Panel position style
const panelStyle = computed(() => {
  if (!props.position) return {}
  return {
    left: `${props.position.x}px`,
    top: `${props.position.y}px`,
  }
})

// Save handler
function handleSave() {
  const mergedConfig = {
    ...formConfig.value,
    ...flowConfig.value
  }

  emit('save', {
    name: formName.value,
    type: props.step.type,
    config: mergedConfig
  })
}

// Delete handler
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

// Close on Escape
function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="panel">
      <div
        v-if="step"
        class="context-panel"
        :style="panelStyle"
      >
        <!-- Header -->
        <div class="panel-header">
          <div class="step-indicator" :style="{ background: stepColor }" />
          <input
            v-model="formName"
            class="step-name-input"
            :placeholder="t('editor.stepName')"
          >
          <button class="close-btn" :title="t('common.close')" @click="emit('close')">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Content -->
        <div class="panel-content">
          <!-- Loading -->
          <div v-if="loadingBlockDef" class="loading-state">
            <span class="loading-spinner" />
          </div>

          <!-- Dynamic Config Form -->
          <template v-else>
            <DynamicConfigForm
              v-if="hasConfigSchema && configSchema"
              v-model="formConfig"
              :schema="configSchema"
              :ui-config="uiConfig"
            />

            <!-- Fallback for blocks without schema -->
            <div v-else class="no-config-message">
              {{ t('editor.noConfigOptions') }}
            </div>
          </template>

          <!-- Advanced Settings (Collapsible) -->
          <details
            v-if="!isStartBlock"
            class="advanced-section"
            :open="showAdvanced"
            @toggle="showAdvanced = ($event.target as HTMLDetailsElement).open"
          >
            <summary class="advanced-summary">
              <svg
                :class="['chevron', { expanded: showAdvanced }]"
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <path d="m9 18 6-6-6-6" />
              </svg>
              {{ t('editor.advancedSettings') }}
            </summary>
            <div class="advanced-content">
              <FlowTab
                :step="step"
                :workflow-id="workflowId"
                @update="handleFlowConfigUpdate"
              />
            </div>
          </details>
        </div>

        <!-- Footer -->
        <div class="panel-footer">
          <button
            v-if="!isStartBlock"
            class="btn-delete"
            @click="handleDelete"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" />
            </svg>
            {{ t('common.delete') }}
          </button>
          <div v-else />

          <button
            class="btn-save"
            :disabled="saving"
            @click="handleSave"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" />
              <polyline points="17 21 17 13 7 13 7 21" />
              <polyline points="7 3 7 8 15 8" />
            </svg>
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.context-panel {
  position: fixed;
  z-index: 150;

  width: 360px;
  max-height: 75vh;

  background: rgba(255, 255, 255, 0.98);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 14px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.15);

  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Header */
.panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(249, 250, 251, 0.5);
}

.step-indicator {
  width: 4px;
  height: 28px;
  border-radius: 2px;
  flex-shrink: 0;
}

.step-name-input {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 15px;
  font-weight: 600;
  color: #111827;
  outline: none;
}

.step-name-input:focus {
  background: rgba(0, 0, 0, 0.03);
  border-radius: 6px;
  padding: 6px 10px;
  margin: -6px -10px;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  color: #9ca3af;
  transition: all 0.15s;
}

.close-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #6b7280;
}

/* Content */
.panel-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.no-config-message {
  padding: 24px;
  text-align: center;
  color: #9ca3af;
  font-size: 13px;
}

/* Advanced Section */
.advanced-section {
  margin-top: 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  padding-top: 12px;
}

.advanced-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: #6b7280;
  user-select: none;
  list-style: none;
}

.advanced-summary::-webkit-details-marker {
  display: none;
}

.advanced-summary:hover {
  color: #374151;
}

.chevron {
  transition: transform 0.15s;
}

.chevron.expanded {
  transform: rotate(90deg);
}

.advanced-content {
  margin-top: 12px;
}

/* Footer */
.panel-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(249, 250, 251, 0.5);
}

.btn-delete {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: transparent;
  border: 1px solid #fecaca;
  border-radius: 8px;
  color: #dc2626;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.15s;
}

.btn-delete:hover {
  background: #fef2f2;
  border-color: #fca5a5;
}

.btn-save {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #3b82f6;
  border: none;
  border-radius: 8px;
  color: white;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.15s;
}

.btn-save:hover {
  background: #2563eb;
}

.btn-save:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Scrollbar */
.panel-content::-webkit-scrollbar {
  width: 6px;
}

.panel-content::-webkit-scrollbar-track {
  background: transparent;
}

.panel-content::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.1);
  border-radius: 3px;
}

.panel-content::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.2);
}

/* Panel Transition */
.panel-enter-active,
.panel-leave-active {
  transition: all 0.2s ease;
}

.panel-enter-from,
.panel-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-10px);
}
</style>
