<script setup lang="ts">
const { t } = useI18n()

interface StepConfig {
  adapter_id?: string
  url?: string
  method?: string
  [key: string]: unknown
}

const props = defineProps<{
  modelValue: StepConfig
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: StepConfig): void
}>()

const adapters = computed(() => [
  { id: 'mock', name: t('stepConfig.tool.adapters.mock'), description: t('stepConfig.tool.adapters.mockDesc') },
  { id: 'openai', name: t('stepConfig.tool.adapters.openai'), description: t('stepConfig.tool.adapters.openaiDesc') },
  { id: 'anthropic', name: t('stepConfig.tool.adapters.anthropic'), description: t('stepConfig.tool.adapters.anthropicDesc') },
  { id: 'http', name: t('stepConfig.tool.adapters.http'), description: t('stepConfig.tool.adapters.httpDesc') },
])

function updateField<K extends keyof StepConfig>(field: K, value: StepConfig[K]) {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}
</script>

<template>
  <div class="form-section">
    <h4 class="section-title">{{ t('stepConfig.tool.title') }}</h4>

    <div class="form-group">
      <label class="form-label">{{ t('stepConfig.tool.adapter') }}</label>
      <div class="adapter-grid">
        <label
          v-for="adapter in adapters"
          :key="adapter.id"
          :class="['adapter-option', { selected: modelValue.adapter_id === adapter.id }]"
        >
          <input
            type="radio"
            :value="adapter.id"
            :checked="modelValue.adapter_id === adapter.id"
            :disabled="disabled"
            @change="updateField('adapter_id', adapter.id)"
          >
          <div class="adapter-info">
            <div class="adapter-name">{{ adapter.name }}</div>
            <div class="adapter-desc">{{ adapter.description }}</div>
          </div>
        </label>
      </div>
    </div>

    <div v-if="modelValue.adapter_id === 'http'" class="form-group">
      <label class="form-label">{{ t('stepConfig.tool.httpEndpoint') }}</label>
      <input
        :value="modelValue.url"
        type="url"
        class="form-input"
        :placeholder="t('stepConfig.tool.httpEndpointPlaceholder')"
        :disabled="disabled"
        @input="updateField('url', ($event.target as HTMLInputElement).value)"
      >
    </div>

    <div v-if="modelValue.adapter_id === 'http'" class="form-group">
      <label class="form-label">{{ t('stepConfig.tool.httpMethod') }}</label>
      <select
        :value="modelValue.method"
        class="form-input"
        :disabled="disabled"
        @change="updateField('method', ($event.target as HTMLSelectElement).value)"
      >
        <option value="GET">GET</option>
        <option value="POST">POST</option>
        <option value="PUT">PUT</option>
        <option value="DELETE">DELETE</option>
      </select>
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

.adapter-grid {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.adapter-option {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.625rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.adapter-option:hover {
  border-color: var(--color-primary);
}

.adapter-option.selected {
  border-color: var(--color-primary);
  background: rgba(59, 130, 246, 0.05);
}

.adapter-option input {
  margin-top: 0.125rem;
}

.adapter-info {
  flex: 1;
}

.adapter-name {
  font-size: 0.8125rem;
  font-weight: 500;
}

.adapter-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin-top: 0.125rem;
}
</style>
