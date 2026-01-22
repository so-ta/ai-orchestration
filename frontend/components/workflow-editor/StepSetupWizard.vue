<script setup lang="ts">
/**
 * StepSetupWizard.vue
 * Copilot生成後に不完全な設定を持つステップを順番に設定するウィザード
 */
import type { Step, BlockDefinition } from '~/types/api'

const { t } = useI18n()

interface IncompleteStep {
  step: Step
  incompleteFields: string[]
  blockDefinition?: BlockDefinition
}

const props = defineProps<{
  modelValue: boolean
  steps: Step[]
  blockDefinitions: BlockDefinition[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'step:select': [step: Step]
  'complete': []
  'skip-all': []
}>()

// Current wizard step index
const currentIndex = ref(0)

// Find steps with incomplete config
const incompleteSteps = computed<IncompleteStep[]>(() => {
  const result: IncompleteStep[] = []

  for (const step of props.steps) {
    const blockDef = props.blockDefinitions.find(b => b.slug === step.type)
    if (!blockDef?.config_schema) continue

    // Parse config schema
    let schema: { required?: string[] }
    try {
      schema = typeof blockDef.config_schema === 'string'
        ? JSON.parse(blockDef.config_schema)
        : blockDef.config_schema
    } catch {
      continue
    }

    // Check required fields
    const config = step.config as Record<string, unknown> | undefined
    const incompleteFields: string[] = []

    for (const field of schema.required || []) {
      const value = config?.[field]
      if (value === undefined || value === null || value === '') {
        incompleteFields.push(field)
      }
    }

    if (incompleteFields.length > 0) {
      result.push({
        step,
        incompleteFields,
        blockDefinition: blockDef,
      })
    }
  }

  return result
})

// Current step being configured
const currentStep = computed(() => incompleteSteps.value[currentIndex.value])

// Progress percentage
const progressPercent = computed(() => {
  if (incompleteSteps.value.length === 0) return 100
  return Math.round((currentIndex.value / incompleteSteps.value.length) * 100)
})

// Navigate to previous step
function previousStep() {
  if (currentIndex.value > 0) {
    currentIndex.value--
  }
}

// Navigate to next step
function nextStep() {
  if (currentIndex.value < incompleteSteps.value.length - 1) {
    currentIndex.value++
  } else {
    // Complete wizard
    emit('complete')
    close()
  }
}

// Go to step and open properties panel
function configureStep() {
  if (currentStep.value) {
    emit('step:select', currentStep.value.step)
  }
}

// Skip all remaining steps
function skipAll() {
  emit('skip-all')
  close()
}

// Close wizard
function close() {
  emit('update:modelValue', false)
}

// Auto-close if no incomplete steps
watch(
  () => incompleteSteps.value.length,
  (count) => {
    if (count === 0 && props.modelValue) {
      emit('complete')
      close()
    }
  },
  { immediate: true }
)

// Reset index when opening
watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      currentIndex.value = 0
    }
  }
)
</script>

<template>
  <Teleport to="body">
    <div v-if="modelValue && incompleteSteps.length > 0" class="wizard-overlay">
      <div class="wizard-modal">
        <!-- Header -->
        <div class="wizard-header">
          <div class="wizard-title">
            <span class="wizard-icon">&#9881;</span>
            {{ t('setupWizard.title') }}
          </div>
          <button class="wizard-close" @click="close">
            &times;
          </button>
        </div>

        <!-- Progress -->
        <div class="wizard-progress">
          <div class="wizard-progress-bar">
            <div class="wizard-progress-fill" :style="{ width: `${progressPercent}%` }" />
          </div>
          <div class="wizard-progress-text">
            {{ currentIndex + 1 }} / {{ incompleteSteps.length }}
          </div>
        </div>

        <!-- Content -->
        <div class="wizard-content">
          <div v-if="currentStep" class="wizard-step-info">
            <div class="wizard-step-name">
              <span class="wizard-step-icon">{{ currentStep.blockDefinition?.icon || '&#128260;' }}</span>
              {{ currentStep.step.name }}
            </div>
            <div class="wizard-step-type">{{ currentStep.step.type }}</div>

            <div class="wizard-fields">
              <div class="wizard-fields-label">{{ t('setupWizard.incompleteFields') }}:</div>
              <ul class="wizard-fields-list">
                <li v-for="field in currentStep.incompleteFields" :key="field" class="wizard-field-item">
                  {{ field }}
                </li>
              </ul>
            </div>

            <button class="wizard-configure-btn" @click="configureStep">
              {{ t('setupWizard.configureNow') }}
            </button>
          </div>
        </div>

        <!-- Footer -->
        <div class="wizard-footer">
          <button class="wizard-btn wizard-btn-secondary" @click="skipAll">
            {{ t('setupWizard.skipAll') }}
          </button>
          <div class="wizard-nav-buttons">
            <button
              class="wizard-btn wizard-btn-secondary"
              :disabled="currentIndex === 0"
              @click="previousStep"
            >
              {{ t('setupWizard.previous') }}
            </button>
            <button class="wizard-btn wizard-btn-primary" @click="nextStep">
              {{ currentIndex === incompleteSteps.length - 1 ? t('setupWizard.finish') : t('setupWizard.next') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.wizard-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.wizard-modal {
  background: var(--color-surface, #fff);
  border-radius: 12px;
  width: 480px;
  max-width: 90vw;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
}

.wizard-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--color-border, #e5e7eb);
}

.wizard-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--color-text, #1f2937);
}

.wizard-icon {
  font-size: 1.25rem;
}

.wizard-close {
  background: none;
  border: none;
  font-size: 1.5rem;
  color: var(--color-text-secondary, #6b7280);
  cursor: pointer;
  padding: 0;
  line-height: 1;
}

.wizard-close:hover {
  color: var(--color-text, #1f2937);
}

.wizard-progress {
  padding: 1rem 1.5rem;
  display: flex;
  align-items: center;
  gap: 1rem;
  background: var(--color-background, #f9fafb);
}

.wizard-progress-bar {
  flex: 1;
  height: 8px;
  background: var(--color-border, #e5e7eb);
  border-radius: 4px;
  overflow: hidden;
}

.wizard-progress-fill {
  height: 100%;
  background: var(--color-primary, #3b82f6);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.wizard-progress-text {
  font-size: 0.875rem;
  color: var(--color-text-secondary, #6b7280);
  white-space: nowrap;
}

.wizard-content {
  padding: 1.5rem;
}

.wizard-step-info {
  text-align: center;
}

.wizard-step-name {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--color-text, #1f2937);
  margin-bottom: 0.25rem;
}

.wizard-step-icon {
  font-size: 1.5rem;
}

.wizard-step-type {
  font-size: 0.875rem;
  color: var(--color-text-secondary, #6b7280);
  margin-bottom: 1.5rem;
}

.wizard-fields {
  background: var(--color-background, #f9fafb);
  border-radius: 8px;
  padding: 1rem;
  margin-bottom: 1.5rem;
  text-align: left;
}

.wizard-fields-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary, #6b7280);
  margin-bottom: 0.5rem;
}

.wizard-fields-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.wizard-field-item {
  padding: 0.5rem 0.75rem;
  background: rgba(245, 158, 11, 0.1);
  border-left: 3px solid #f59e0b;
  border-radius: 0 4px 4px 0;
  margin-bottom: 0.5rem;
  font-size: 0.875rem;
  color: var(--color-text, #1f2937);
}

.wizard-field-item:last-child {
  margin-bottom: 0;
}

.wizard-configure-btn {
  background: var(--color-primary, #3b82f6);
  color: white;
  border: none;
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s;
}

.wizard-configure-btn:hover {
  background: var(--color-primary-dark, #2563eb);
}

.wizard-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--color-border, #e5e7eb);
}

.wizard-nav-buttons {
  display: flex;
  gap: 0.5rem;
}

.wizard-btn {
  padding: 0.625rem 1rem;
  border-radius: 6px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.wizard-btn-primary {
  background: var(--color-primary, #3b82f6);
  color: white;
  border: none;
}

.wizard-btn-primary:hover {
  background: var(--color-primary-dark, #2563eb);
}

.wizard-btn-secondary {
  background: transparent;
  color: var(--color-text-secondary, #6b7280);
  border: 1px solid var(--color-border, #e5e7eb);
}

.wizard-btn-secondary:hover {
  background: var(--color-background, #f9fafb);
  color: var(--color-text, #1f2937);
}

.wizard-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
