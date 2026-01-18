<script setup lang="ts">
const { t } = useI18n()

interface StepConfig {
  message?: string
  level?: string
  data?: string
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
    <h4 class="section-title">{{ t('stepConfig.log.title') }}</h4>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.log.message') }}</label>
      <textarea
        :value="modelValue.message"
        class="form-input form-textarea"
        rows="3"
        :placeholder="t('stepConfig.log.messagePlaceholder')"
        :disabled="disabled"
        @input="updateField('message', ($event.target as HTMLTextAreaElement).value)"
      />
      <p class="form-hint">{{ t('stepConfig.log.messageHint') }}</p>
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.log.level') }}</label>
      <select
        :value="modelValue.level"
        class="form-input"
        :disabled="disabled"
        @change="updateField('level', ($event.target as HTMLSelectElement).value)"
      >
        <option value="debug">{{ t('stepConfig.log.levels.debug') }}</option>
        <option value="info">{{ t('stepConfig.log.levels.info') }}</option>
        <option value="warn">{{ t('stepConfig.log.levels.warn') }}</option>
        <option value="error">{{ t('stepConfig.log.levels.error') }}</option>
      </select>
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.log.data') }}</label>
      <input
        :value="modelValue.data"
        type="text"
        class="form-input code-input"
        :placeholder="t('stepConfig.log.dataPlaceholder')"
        :disabled="disabled"
        @input="updateField('data', ($event.target as HTMLInputElement).value)"
      >
      <p class="form-hint">{{ t('stepConfig.log.dataHint') }}</p>
    </div>

    <div class="info-box log-info-box">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="4 17 10 11 4 5"/>
        <line x1="12" y1="19" x2="20" y2="19"/>
      </svg>
      <span>{{ t('stepConfig.log.viewNote') }}</span>
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

.code-input {
  font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
  font-size: 0.75rem;
}

.form-hint {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.25rem;
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
