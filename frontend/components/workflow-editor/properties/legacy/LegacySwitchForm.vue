<script setup lang="ts">
import { useSwitchCases, type SwitchCase } from '../../composables/useSwitchCases'

const { t } = useI18n()

interface StepConfig {
  expression?: string
  cases?: SwitchCase[]
  [key: string]: unknown
}

const props = defineProps<{
  modelValue: StepConfig
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: StepConfig): void
}>()

const localConfig = ref<StepConfig>({ ...props.modelValue })

watch(() => props.modelValue, (newVal) => {
  localConfig.value = { ...newVal }
}, { deep: true })

watch(localConfig, (newVal) => {
  emit('update:modelValue', newVal)
}, { deep: true })

const { switchCases, addSwitchCase, removeSwitchCase, updateSwitchCase, getCaseDisplayName } = useSwitchCases(localConfig)

function updateExpression(value: string) {
  localConfig.value = { ...localConfig.value, expression: value }
}
</script>

<template>
  <div class="form-section">
    <h4 class="section-title">{{ t('stepConfig.switch.title') }}</h4>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.switch.expression') }}</label>
      <input
        :value="modelValue.expression"
        type="text"
        class="form-input code-input"
        placeholder="$.status"
        :disabled="disabled"
        @input="updateExpression(($event.target as HTMLInputElement).value)"
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
              :disabled="disabled"
              @input="updateSwitchCase(index, 'name', ($event.target as HTMLInputElement).value)"
            >
            <button
              v-if="!disabled"
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
            :disabled="disabled"
            @input="updateSwitchCase(index, 'expression', ($event.target as HTMLInputElement).value)"
          >
          <label class="form-checkbox case-default-checkbox">
            <input
              type="checkbox"
              :checked="switchCase.is_default"
              :disabled="disabled"
              @change="updateSwitchCase(index, 'is_default', ($event.target as HTMLInputElement).checked)"
            >
            <span>{{ t('stepConfig.switch.isDefault') }}</span>
          </label>
        </div>

        <button
          v-if="!disabled"
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
</template>

<style scoped>
.form-section {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
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

.code-input {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
}

.form-hint {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}

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

.branch-label {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}

.branch-case {
  background: #e0e7ff;
  color: #4f46e5;
}

.branch-default {
  background: #fef3c7;
  color: #92400e;
}

.branch-type {
  font-size: 0.625rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
}

.branch-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
}
</style>
