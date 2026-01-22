<script setup lang="ts">
import type { Step, BlockDefinition } from '~/types/api'
import type { ConfigSchema } from '../config/types/config-schema'
import type { StepTestResult, PinnedOutput } from '~/composables/test'
import DynamicConfigForm from '../config/DynamicConfigForm.vue'
import TestResultPanel from '../test/TestResultPanel.vue'
import DataPinSection from '../test/DataPinSection.vue'

const { t } = useI18n()
const toast = useToast()

const props = defineProps<{
  step: Step | null
  workflowId: string
  isActive?: boolean
  steps?: Step[]
  edges?: Array<{ id: string; source_step_id?: string | null; target_step_id?: string | null }>
  blockDefinitions?: BlockDefinition[]
  // Step test state
  executing: boolean
  currentResult: StepTestResult | null
  error: string | null
  pinnedOutput: PinnedOutput | null
}>()

const emit = defineEmits<{
  'execute-step-only': [step: Step, input: object]
  'execute-from-step': [step: Step, input: object]
  'pin-output': [output: unknown, stepId: string, stepName: string]
  'unpin-output': [stepId: string]
  'edit-pinned': [data: unknown, stepId: string]
}>()

// Input mode and values
const useJsonMode = ref(false)
const customInputJson = ref('{}')
const inputValues = ref<Record<string, unknown>>({})
const formValid = ref(true)
const inputError = ref<string | null>(null)

// Get block definition for current step
const currentBlockDef = computed<BlockDefinition | null>(() => {
  if (!props.step || !props.blockDefinitions) return null
  return props.blockDefinitions.find(b => b.slug === props.step?.type) || null
})

// Check if step has input schema
const hasInputSchema = computed(() => {
  if (!currentBlockDef.value?.config_schema) return false
  const schema = currentBlockDef.value.config_schema as Record<string, unknown>
  return schema && typeof schema === 'object' && Object.keys(schema).length > 0
})

const inputSchema = computed<ConfigSchema | null>(() => {
  if (!hasInputSchema.value) return null
  return currentBlockDef.value?.config_schema as ConfigSchema
})

// Previous step for suggested input (reserved for future use)
const _previousStep = computed<Step | null>(() => {
  if (!props.step || !props.steps || !props.edges) return null
  const incomingEdge = props.edges.find(e => e.target_step_id === props.step?.id)
  if (!incomingEdge?.source_step_id) return null
  return props.steps.find(s => s.id === incomingEdge.source_step_id) || null
})

// Get input for execution
function getInput(): object | null {
  inputError.value = null

  if (useJsonMode.value) {
    try {
      const parsed = JSON.parse(customInputJson.value)
      if (typeof parsed !== 'object' || Array.isArray(parsed)) {
        inputError.value = t('execution.errors.invalidJsonObject')
        return null
      }
      return parsed
    } catch {
      inputError.value = t('execution.errors.invalidJson')
      return null
    }
  }

  if (!formValid.value) {
    inputError.value = t('execution.errors.invalidForm')
    return null
  }

  return inputValues.value
}

// Handle execute this step only
function handleExecuteStepOnly() {
  if (!props.step) return
  const input = getInput()
  if (input === null) return
  emit('execute-step-only', props.step, input)
}

// Handle execute from this step
function handleExecuteFromStep() {
  if (!props.step) return
  const input = getInput()
  if (input === null) return
  emit('execute-from-step', props.step, input)
}

// Handle pin output
function handlePinOutput(output: unknown) {
  if (!props.step) return
  emit('pin-output', output, props.step.id, props.step.name)
}

// Handle unpin
function handleUnpin() {
  if (!props.step) return
  emit('unpin-output', props.step.id)
}

// Handle edit pinned
function handleEditPinned(data: unknown) {
  if (!props.step) return
  emit('edit-pinned', data, props.step.id)
}

// Use pinned output as input
function usePinnedAsInput() {
  if (!props.pinnedOutput?.data) return
  customInputJson.value = JSON.stringify(props.pinnedOutput.data, null, 2)
  useJsonMode.value = true
  toast.success(t('test.pinnedUsed'))
}

// Toggle input mode
function toggleInputMode() {
  if (!useJsonMode.value) {
    // Switching to JSON mode
    customInputJson.value = JSON.stringify(inputValues.value, null, 2)
  } else {
    // Switching to form mode
    try {
      inputValues.value = JSON.parse(customInputJson.value)
    } catch {
      // Keep current form values if JSON is invalid
    }
  }
  useJsonMode.value = !useJsonMode.value
}

// Clear input
function clearInput() {
  customInputJson.value = '{}'
  inputValues.value = {}
}

// Check if step is a trigger block (don't show test for triggers)
const triggerBlockTypes = ['start', 'manual_trigger', 'schedule_trigger', 'webhook_trigger']
const isTriggerBlock = computed(() => {
  return props.step?.type ? triggerBlockTypes.includes(props.step.type) : false
})
</script>

<template>
  <div class="test-tab">
    <template v-if="step && !isTriggerBlock">
      <!-- Input Section -->
      <div class="test-section">
        <div class="section-header">
          <span class="section-title">{{ t('test.input') }}</span>
          <div class="section-actions">
            <!-- Use pinned output button -->
            <button
              v-if="pinnedOutput"
              class="btn-icon"
              :title="t('test.usePinned')"
              @click="usePinnedAsInput"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="m12 17 2 5 2-5"/>
                <path d="m6 12 1.09 4.36a2 2 0 0 0 1.94 1.64h5.94a2 2 0 0 0 1.94-1.64L18 12"/>
                <path d="M6 8a6 6 0 1 1 12 0c0 5-6 6-6 12h0c0-6-6-7-6-12Z"/>
              </svg>
            </button>
            <!-- Clear button -->
            <button class="btn-icon" :title="t('execution.clearInput')" @click="clearInput">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 6h18"/>
                <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
              </svg>
            </button>
            <!-- Toggle JSON/Form -->
            <button
              v-if="hasInputSchema"
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

        <!-- Form mode -->
        <div v-if="hasInputSchema && !useJsonMode" class="input-form">
          <DynamicConfigForm
            v-model="inputValues"
            :schema="inputSchema"
            :disabled="executing"
            @validation-change="formValid = $event"
          />
        </div>

        <!-- JSON mode -->
        <div v-else class="input-json">
          <textarea
            v-model="customInputJson"
            class="json-textarea"
            rows="4"
            :placeholder="t('execution.inputPlaceholder')"
            :disabled="executing"
          />
        </div>

        <!-- Input error -->
        <p v-if="inputError" class="input-error">{{ inputError }}</p>
      </div>

      <!-- Execute Buttons -->
      <div class="execute-section">
        <button
          class="btn btn-primary btn-full"
          :disabled="executing"
          @click="handleExecuteStepOnly"
        >
          <svg
            v-if="!executing"
            xmlns="http://www.w3.org/2000/svg"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <polygon points="5 3 19 12 5 21 5 3"/>
          </svg>
          <svg
            v-else
            class="spin"
            xmlns="http://www.w3.org/2000/svg"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          {{ executing ? t('test.executing') : t('test.executeThisStepOnly') }}
        </button>

        <button
          class="btn btn-outline btn-full"
          :disabled="executing"
          @click="handleExecuteFromStep"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
          </svg>
          {{ t('test.executeFromThisStep') }}
        </button>
      </div>

      <!-- Result Section -->
      <div class="test-section">
        <div class="section-header">
          <span class="section-title">{{ t('test.result') }}</span>
        </div>
        <TestResultPanel
          :result="currentResult"
          :executing="executing"
          :error="error"
          :step-name="step?.name"
          @pin-output="handlePinOutput"
        />
      </div>

      <!-- Pinned Data Section -->
      <DataPinSection
        v-if="pinnedOutput"
        :pinned-output="pinnedOutput"
        @unpin="handleUnpin"
        @edit="handleEditPinned"
      />
    </template>

    <!-- Trigger block message -->
    <div v-else-if="step && isTriggerBlock" class="trigger-message">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="12" cy="12" r="10"/>
        <line x1="12" y1="16" x2="12" y2="12"/>
        <line x1="12" y1="8" x2="12.01" y2="8"/>
      </svg>
      <p>{{ t('test.triggerBlockNote') }}</p>
    </div>

    <!-- No step selected -->
    <div v-else class="no-step">
      <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
        <line x1="9" y1="9" x2="15" y2="15"/>
        <line x1="15" y1="9" x2="9" y2="15"/>
      </svg>
      <p>{{ t('test.selectStep') }}</p>
    </div>
  </div>
</template>

<style scoped>
.test-tab {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 0.5rem 0;
  height: 100%;
  overflow-y: auto;
}

.test-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.section-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.section-actions {
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

.input-form {
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 0.75rem;
  background: white;
}

.input-json {
  display: flex;
  flex-direction: column;
}

.json-textarea {
  width: 100%;
  padding: 0.5rem;
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  resize: vertical;
  min-height: 80px;
}

.json-textarea:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.json-textarea:disabled {
  background: var(--color-surface);
  cursor: not-allowed;
}

.input-error {
  font-size: 0.6875rem;
  color: var(--color-error);
  margin: 0;
}

.execute-section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  font-size: 0.8125rem;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-full {
  width: 100%;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  border: none;
}

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-outline {
  background: white;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn-outline:hover:not(:disabled) {
  background: var(--color-surface);
  border-color: var(--color-text-secondary);
}

.btn-outline:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.trigger-message,
.no-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 2rem;
  text-align: center;
  color: var(--color-text-secondary);
}

.trigger-message svg,
.no-step svg {
  opacity: 0.5;
}

.trigger-message p,
.no-step p {
  font-size: 0.8125rem;
  margin: 0;
  max-width: 200px;
}

.test-tab::-webkit-scrollbar {
  width: 6px;
}

.test-tab::-webkit-scrollbar-track {
  background: transparent;
}

.test-tab::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}
</style>
