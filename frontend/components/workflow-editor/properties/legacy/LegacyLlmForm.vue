<script setup lang="ts">
const { t } = useI18n()

interface StepConfig {
  provider?: string
  model?: string
  system_prompt?: string
  prompt?: string
  temperature?: number
  max_tokens?: number
  [key: string]: unknown
}

const props = defineProps<{
  modelValue: StepConfig
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: StepConfig): void
}>()

const modelsByProvider: Record<string, { id: string; name: string }[]> = {
  openai: [
    { id: 'gpt-4', name: 'GPT-4' },
    { id: 'gpt-4-turbo', name: 'GPT-4 Turbo' },
    { id: 'gpt-3.5-turbo', name: 'GPT-3.5 Turbo' },
  ],
  anthropic: [
    { id: 'claude-3-opus', name: 'Claude 3 Opus' },
    { id: 'claude-3-sonnet', name: 'Claude 3 Sonnet' },
    { id: 'claude-3-haiku', name: 'Claude 3 Haiku' },
  ],
  mock: [
    { id: 'mock', name: 'Mock Model' },
  ],
}

const availableModels = computed(() => {
  const provider = props.modelValue.provider || 'mock'
  return modelsByProvider[provider] || modelsByProvider.mock
})

function updateField<K extends keyof StepConfig>(field: K, value: StepConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}

watch(() => props.modelValue.provider, (newProvider) => {
  if (newProvider && modelsByProvider[newProvider]) {
    updateField('model', modelsByProvider[newProvider][0]?.id || '')
  }
})
</script>

<template>
  <div class="form-section">
    <h4 class="section-title">{{ t('stepConfig.llm.title') }}</h4>

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
        <select
          :value="modelValue.model"
          class="form-input"
          :disabled="disabled"
          @change="updateField('model', ($event.target as HTMLSelectElement).value)"
        >
          <option v-for="model in availableModels" :key="model.id" :value="model.id">
            {{ model.name }}
          </option>
        </select>
      </div>
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.llm.systemPrompt') }}</label>
      <textarea
        :value="modelValue.system_prompt"
        class="form-input form-textarea"
        rows="3"
        :placeholder="t('stepConfig.llm.systemPromptPlaceholder')"
        :disabled="disabled"
        @input="updateField('system_prompt', ($event.target as HTMLTextAreaElement).value)"
      />
    </div>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.llm.userPrompt') }}</label>
      <textarea
        :value="modelValue.prompt"
        class="form-input form-textarea code-input"
        rows="4"
        :placeholder="t('stepConfig.llm.userPromptPlaceholder')"
        :disabled="disabled"
        @input="updateField('prompt', ($event.target as HTMLTextAreaElement).value)"
      />
      <p class="form-hint">{{ t('stepConfig.llm.userPromptHint') }}</p>
    </div>

    <div class="form-row">
      <div class="form-group">
        <label class="form-label">{{ t('stepConfig.llm.temperature') }}</label>
        <input
          :value="modelValue.temperature"
          type="number"
          class="form-input"
          min="0"
          max="2"
          step="0.1"
          placeholder="0.7"
          :disabled="disabled"
          @input="updateField('temperature', parseFloat(($event.target as HTMLInputElement).value))"
        >
      </div>
      <div class="form-group">
        <label class="form-label">{{ t('stepConfig.llm.maxTokens') }}</label>
        <input
          :value="modelValue.max_tokens"
          type="number"
          class="form-input"
          min="1"
          max="128000"
          placeholder="4096"
          :disabled="disabled"
          @input="updateField('max_tokens', parseInt(($event.target as HTMLInputElement).value))"
        >
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

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}
</style>
