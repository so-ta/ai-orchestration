<script setup lang="ts">
const { t } = useI18n()

interface StepConfig {
  provider?: string
  model?: string
  prompt?: string
  routes_json?: string
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
    <h4 class="section-title">{{ t('stepConfig.router.title') }}</h4>

    <div class="form-row">
      <div class="form-group">
        <label class="form-label">{{ t('stepConfig.llm.provider') }}</label>
        <select
          :value="modelValue.provider"
          class="form-input"
          :disabled="disabled"
          @change="updateField('provider', ($event.target as HTMLSelectElement).value)"
        >
          <option value="mock">{{ t('stepConfig.tool.adapters.mock') }}</option>
          <option value="openai">{{ t('stepConfig.tool.adapters.openai') }}</option>
          <option value="anthropic">{{ t('stepConfig.tool.adapters.anthropic') }}</option>
        </select>
      </div>
      <div class="form-group">
        <label class="form-label">{{ t('stepConfig.llm.model') }}</label>
        <input
          :value="modelValue.model"
          type="text"
          class="form-input"
          placeholder="gpt-4o-mini"
          :disabled="disabled"
          @input="updateField('model', ($event.target as HTMLInputElement).value)"
        >
      </div>
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.router.classificationPrompt') }}</label>
      <textarea
        :value="modelValue.prompt"
        class="form-input form-textarea"
        rows="3"
        :placeholder="t('stepConfig.router.classificationPromptPlaceholder')"
        :disabled="disabled"
        @input="updateField('prompt', ($event.target as HTMLTextAreaElement).value)"
      />
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.router.routes') }}</label>
      <textarea
        :value="modelValue.routes_json"
        class="form-input form-textarea code-input"
        rows="4"
        :placeholder="t('stepConfig.router.routesPlaceholder')"
        :disabled="disabled"
        @input="updateField('routes_json', ($event.target as HTMLTextAreaElement).value)"
      />
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

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}
</style>
