<script setup lang="ts">
const { t } = useI18n()

interface StepConfig {
  input_path?: string
  parallel?: number
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
    <h4 class="section-title">{{ t('stepConfig.map.title') }}</h4>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.map.inputPath') }}</label>
      <input
        :value="modelValue.input_path"
        type="text"
        class="form-input code-input"
        :placeholder="t('stepConfig.map.inputPathPlaceholder')"
        :disabled="disabled"
        @input="updateField('input_path', ($event.target as HTMLInputElement).value)"
      >
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.map.parallelism') }}</label>
      <input
        :value="modelValue.parallel"
        type="number"
        class="form-input"
        min="1"
        max="100"
        placeholder="10"
        :disabled="disabled"
        @input="updateField('parallel', parseInt(($event.target as HTMLInputElement).value))"
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

.code-input {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
}
</style>
