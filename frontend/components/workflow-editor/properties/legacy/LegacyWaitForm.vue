<script setup lang="ts">
const { t } = useI18n()

interface StepConfig {
  duration_ms?: number
  until?: string
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
    <h4 class="section-title">{{ t('stepConfig.wait.title') }}</h4>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.wait.duration') }}</label>
      <input
        :value="modelValue.duration_ms"
        type="number"
        class="form-input"
        min="0"
        max="3600000"
        placeholder="5000"
        :disabled="disabled"
        @input="updateField('duration_ms', parseInt(($event.target as HTMLInputElement).value))"
      >
      <p class="form-hint">{{ t('stepConfig.wait.durationHint') }}</p>
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.wait.until') }}</label>
      <input
        :value="modelValue.until"
        type="datetime-local"
        class="form-input"
        :disabled="disabled"
        @input="updateField('until', ($event.target as HTMLInputElement).value)"
      >
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

.form-hint {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
}
</style>
