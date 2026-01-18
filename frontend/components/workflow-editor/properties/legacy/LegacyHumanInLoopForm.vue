<script setup lang="ts">
const { t } = useI18n()

interface StepConfig {
  instructions?: string
  timeout_hours?: number
  approval_url?: boolean
  [key: string]: unknown
}

const props = defineProps<{
  modelValue: StepConfig
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: StepConfig): void
}>()

function updateField<K extends keyof StepConfig>(field: K, value: StepConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}
</script>

<template>
  <div class="form-section">
    <h4 class="section-title">{{ t('stepConfig.humanInLoop.title') }}</h4>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.humanInLoop.instructions') }}</label>
      <textarea
        :value="modelValue.instructions"
        class="form-input form-textarea"
        rows="3"
        :placeholder="t('stepConfig.humanInLoop.instructionsPlaceholder')"
        :disabled="disabled"
        @input="updateField('instructions', ($event.target as HTMLTextAreaElement).value)"
      />
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.humanInLoop.timeoutHours') }}</label>
      <input
        :value="modelValue.timeout_hours"
        type="number"
        class="form-input"
        min="1"
        max="168"
        placeholder="24"
        :disabled="disabled"
        @input="updateField('timeout_hours', parseInt(($event.target as HTMLInputElement).value))"
      >
    </div>

    <div class="form-group">
      <label class="form-checkbox">
        <input
          type="checkbox"
          :checked="modelValue.approval_url"
          :disabled="disabled"
          @change="updateField('approval_url', ($event.target as HTMLInputElement).checked)"
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
</style>
