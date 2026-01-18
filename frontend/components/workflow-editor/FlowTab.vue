<script setup lang="ts">
import type { Step, BlockDefinition } from '~/types/api'
import ErrorHandlingForm, { type ErrorHandlingConfig } from './shared/ErrorHandlingForm.vue'
import ScriptEditor, { type ScriptConfig } from './shared/ScriptEditor.vue'

const { t } = useI18n()

const props = defineProps<{
  step: Step | null
  blockDefinitions?: BlockDefinition[]
  readonlyMode?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:flow-config', config: {
    prescript?: ScriptConfig
    postscript?: ScriptConfig
    error_handling?: ErrorHandlingConfig
  }): void
}>()

// Get current block definition
const currentBlockDef = computed(() => {
  if (!props.step || !props.blockDefinitions) return null
  return props.blockDefinitions.find(b => b.slug === props.step?.type) || null
})

// Format schema type for display
const formatSchemaType = (schema: { type?: string } | null | undefined): string => {
  if (!schema || !schema.type) return 'any'
  return schema.type
}

// Check if step has input ports
const hasInputPorts = computed(() => {
  return currentBlockDef.value?.input_ports && currentBlockDef.value.input_ports.length > 0
})

// Check if step has output ports
const hasOutputPorts = computed(() => {
  return currentBlockDef.value?.output_ports && currentBlockDef.value.output_ports.length > 0
})

// Local state for flow configuration
const prescriptConfig = ref<ScriptConfig>({
  enabled: false,
  code: ''
})

const postscriptConfig = ref<ScriptConfig>({
  enabled: false,
  code: ''
})

const errorHandlingConfig = ref<ErrorHandlingConfig>({
  enabled: true,
  retry: {
    max_retries: 3,
    interval_seconds: 1,
    backoff_strategy: 'fixed'
  },
  timeout_seconds: undefined,
  on_error: 'fail',
  fallback_value: undefined,
  enable_error_port: false
})

// Load config from step when step changes
watch(() => props.step, (newStep) => {
  if (newStep) {
    const config = newStep.config as Record<string, unknown>

    // Load prescript
    if (config.prescript && typeof config.prescript === 'object') {
      const ps = config.prescript as ScriptConfig
      prescriptConfig.value = {
        enabled: ps.enabled ?? false,
        code: ps.code ?? ''
      }
    } else {
      prescriptConfig.value = { enabled: false, code: '' }
    }

    // Load postscript
    if (config.postscript && typeof config.postscript === 'object') {
      const ps = config.postscript as ScriptConfig
      postscriptConfig.value = {
        enabled: ps.enabled ?? false,
        code: ps.code ?? ''
      }
    } else {
      postscriptConfig.value = { enabled: false, code: '' }
    }

    // Load error handling
    if (config.error_handling && typeof config.error_handling === 'object') {
      const eh = config.error_handling as ErrorHandlingConfig
      errorHandlingConfig.value = {
        enabled: eh.enabled ?? true,
        retry: eh.retry ?? { max_retries: 3, interval_seconds: 1, backoff_strategy: 'fixed' },
        timeout_seconds: eh.timeout_seconds,
        on_error: eh.on_error ?? 'fail',
        fallback_value: eh.fallback_value,
        enable_error_port: eh.enable_error_port ?? false
      }
    } else {
      errorHandlingConfig.value = {
        enabled: true,
        retry: { max_retries: 3, interval_seconds: 1, backoff_strategy: 'fixed' },
        timeout_seconds: undefined,
        on_error: 'fail',
        fallback_value: undefined,
        enable_error_port: false
      }
    }
  }
}, { immediate: true })

// Emit changes when config changes
const emitChanges = () => {
  emit('update:flow-config', {
    prescript: prescriptConfig.value.enabled ? prescriptConfig.value : undefined,
    postscript: postscriptConfig.value.enabled ? postscriptConfig.value : undefined,
    error_handling: errorHandlingConfig.value
  })
}

// Watch for changes and emit
watch(prescriptConfig, emitChanges, { deep: true })
watch(postscriptConfig, emitChanges, { deep: true })
watch(errorHandlingConfig, emitChanges, { deep: true })
</script>

<template>
  <div class="flow-tab">
    <!-- Empty State (no step selected) -->
    <div v-if="!step" class="flow-empty">
      <div class="empty-icon">
        <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5L17 3z"/>
        </svg>
      </div>
      <p class="empty-title">{{ t('flow.noStepSelected') }}</p>
      <p class="empty-desc">{{ t('flow.selectStepHint') }}</p>
    </div>

    <!-- Step Flow Configuration -->
    <div v-else class="flow-content">
      <!-- Input Ports Section -->
      <div class="flow-section">
        <h4 class="section-title">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
          {{ t('flow.inputPorts') }}
        </h4>
        <div v-if="hasInputPorts" class="ports-list">
          <div v-for="port in currentBlockDef?.input_ports" :key="port.name" class="port-item">
            <div class="port-header">
              <span class="port-name">{{ port.label || port.name }}</span>
              <code class="port-type">{{ formatSchemaType(port.schema) }}</code>
              <span v-if="port.required" class="port-required">{{ t('flow.portRequired') }}</span>
            </div>
            <p v-if="port.description" class="port-description">{{ port.description }}</p>
          </div>
        </div>
        <div v-else class="no-ports">
          <span>{{ t('flow.noInputPorts') }}</span>
        </div>
      </div>

      <!-- Output Ports Section -->
      <div class="flow-section">
        <h4 class="section-title">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
          {{ t('flow.outputPorts') }}
        </h4>
        <div v-if="hasOutputPorts" class="ports-list">
          <div v-for="port in currentBlockDef?.output_ports" :key="port.name" class="port-item">
            <div class="port-header">
              <span :class="['port-name', { 'port-default': port.is_default }]">{{ port.label || port.name }}</span>
              <code class="port-type">{{ formatSchemaType(port.schema) }}</code>
              <span v-if="port.is_default" class="port-default-badge">default</span>
            </div>
            <p v-if="port.description" class="port-description">{{ port.description }}</p>
          </div>
        </div>
        <div v-else class="no-ports">
          <span>{{ t('flow.noOutputPorts') }}</span>
        </div>
      </div>

      <!-- Scripts Section -->
      <div class="flow-section">
        <h4 class="section-title">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="16 18 22 12 16 6"/>
            <polyline points="8 6 2 12 8 18"/>
          </svg>
          {{ t('flow.scripts.title') }}
        </h4>

        <div class="scripts-container">
          <!-- Prescript -->
          <ScriptEditor
            v-model="prescriptConfig"
            :title="t('flow.scripts.prescript.title')"
            :description="t('flow.scripts.prescript.description')"
            :placeholder="t('flow.scripts.prescript.codePlaceholder')"
            :disabled="readonlyMode"
          />

          <!-- Postscript -->
          <ScriptEditor
            v-model="postscriptConfig"
            :title="t('flow.scripts.postscript.title')"
            :description="t('flow.scripts.postscript.description')"
            :placeholder="t('flow.scripts.postscript.codePlaceholder')"
            :disabled="readonlyMode"
          />
        </div>
      </div>

      <!-- Error Handling Section -->
      <div class="flow-section">
        <h4 class="section-title">
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
            <line x1="12" y1="9" x2="12" y2="13"/>
            <line x1="12" y1="17" x2="12.01" y2="17"/>
          </svg>
          {{ t('flow.errorHandling.title') }}
        </h4>

        <ErrorHandlingForm
          v-model="errorHandlingConfig"
          :disabled="readonlyMode"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.flow-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow-y: auto;
  padding: 1rem;
}

/* Empty State */
.flow-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  text-align: center;
  height: 100%;
}

.empty-icon {
  color: var(--color-primary);
  opacity: 0.6;
  margin-bottom: 1rem;
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

/* Content - Matching PropertiesPanel form-section style */
.flow-content {
  display: flex;
  flex-direction: column;
}

/* Section - Matching PropertiesPanel form-section style */
.flow-section {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.flow-section:last-child {
  margin-bottom: 0;
  padding-bottom: 0;
  border-bottom: none;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.section-title svg {
  opacity: 0.7;
}

/* Ports */
.ports-list {
  display: flex;
  flex-direction: column;
}

.port-item {
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--color-border-light, rgba(0, 0, 0, 0.05));
}

.port-item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.port-item:first-child {
  padding-top: 0;
}

.port-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.port-name {
  font-weight: 500;
  font-size: 0.8125rem;
  color: var(--color-text);
}

.port-name.port-default {
  color: var(--color-primary);
}

.port-type {
  font-size: 0.6875rem;
  background: var(--color-surface);
  color: var(--color-text-secondary);
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
}

.port-required {
  font-size: 0.6875rem;
  color: var(--color-warning);
  font-weight: 500;
}

.port-default-badge {
  font-size: 0.625rem;
  background: var(--color-primary);
  color: white;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  font-weight: 500;
}

.port-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0.375rem 0 0 0;
  line-height: 1.4;
}

.no-ports {
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  font-style: italic;
}

/* Scripts Container */
.scripts-container {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
</style>
