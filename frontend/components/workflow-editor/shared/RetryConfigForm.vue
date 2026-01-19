<script setup lang="ts">
import type { RetryConfig } from '~/types/api'

const { t } = useI18n()

const props = defineProps<{
  modelValue: RetryConfig
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: RetryConfig): void
}>()

const localConfig = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const maxRetries = computed({
  get: () => localConfig.value.max_retries,
  set: (value) => {
    localConfig.value = { ...localConfig.value, max_retries: value }
  }
})

const delayMs = computed({
  get: () => localConfig.value.delay_ms,
  set: (value) => {
    localConfig.value = { ...localConfig.value, delay_ms: value }
  }
})

const exponentialBackoff = computed({
  get: () => localConfig.value.exponential_backoff,
  set: (value) => {
    localConfig.value = { ...localConfig.value, exponential_backoff: value }
  }
})

const maxDelayMs = computed({
  get: () => localConfig.value.max_delay_ms,
  set: (value) => {
    localConfig.value = { ...localConfig.value, max_delay_ms: value }
  }
})

const retryOnErrors = computed({
  get: () => (localConfig.value.retry_on_errors || []).join(', '),
  set: (value: string) => {
    const errors = value.split(',').map(s => s.trim()).filter(s => s)
    localConfig.value = { ...localConfig.value, retry_on_errors: errors.length > 0 ? errors : undefined }
  }
})

// Preview of backoff delays
const backoffPreview = computed(() => {
  if (maxRetries.value === 0) return []
  const delays: number[] = []
  let delay = delayMs.value
  for (let i = 1; i <= Math.min(maxRetries.value, 5); i++) {
    delays.push(Math.min(delay, maxDelayMs.value))
    if (exponentialBackoff.value) {
      delay = delay * 2
    }
  }
  return delays
})

const formatDelay = (ms: number): string => {
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(1)}s`
  }
  return `${ms}ms`
}
</script>

<template>
  <div class="retry-config-form">
    <!-- Max Retries -->
    <div class="form-group">
      <label class="form-label">{{ t('retryConfig.maxRetries') }}</label>
      <input
        v-model.number="maxRetries"
        type="number"
        min="0"
        max="10"
        class="form-input"
        :disabled="disabled"
      >
      <p class="form-hint">{{ t('retryConfig.maxRetriesHint') }}</p>
    </div>

    <!-- Initial Delay -->
    <div class="form-group">
      <label class="form-label">{{ t('retryConfig.delayMs') }}</label>
      <input
        v-model.number="delayMs"
        type="number"
        min="100"
        max="60000"
        step="100"
        class="form-input"
        :disabled="disabled"
      >
      <p class="form-hint">{{ t('retryConfig.delayMsHint') }}</p>
    </div>

    <!-- Exponential Backoff -->
    <div class="form-group checkbox-group">
      <label class="checkbox-label">
        <input
          v-model="exponentialBackoff"
          type="checkbox"
          class="form-checkbox"
          :disabled="disabled"
        >
        <span>{{ t('retryConfig.exponentialBackoff') }}</span>
      </label>
      <p class="form-hint">{{ t('retryConfig.exponentialBackoffHint') }}</p>
    </div>

    <!-- Max Delay (visible when exponential backoff is enabled) -->
    <div v-if="exponentialBackoff" class="form-group">
      <label class="form-label">{{ t('retryConfig.maxDelayMs') }}</label>
      <input
        v-model.number="maxDelayMs"
        type="number"
        min="1000"
        max="300000"
        step="1000"
        class="form-input"
        :disabled="disabled"
      >
      <p class="form-hint">{{ t('retryConfig.maxDelayMsHint') }}</p>
    </div>

    <!-- Retry On Errors -->
    <div class="form-group">
      <label class="form-label">{{ t('retryConfig.retryOnErrors') }}</label>
      <input
        v-model="retryOnErrors"
        type="text"
        class="form-input"
        :placeholder="t('retryConfig.retryOnErrorsPlaceholder')"
        :disabled="disabled"
      >
      <p class="form-hint">{{ t('retryConfig.retryOnErrorsHint') }}</p>
    </div>

    <!-- Backoff Preview -->
    <div v-if="maxRetries > 0 && backoffPreview.length > 0" class="backoff-preview">
      <label class="form-label">{{ t('retryConfig.backoffPreview') }}</label>
      <div class="preview-timeline">
        <div v-for="(delay, index) in backoffPreview" :key="index" class="preview-item">
          <span class="retry-number">{{ t('retryConfig.retryNumber', { n: index + 1 }) }}</span>
          <span class="delay-value">{{ formatDelay(delay) }}</span>
        </div>
        <div v-if="maxRetries > 5" class="preview-more">
          {{ t('retryConfig.moreRetries', { n: maxRetries - 5 }) }}
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.retry-config-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
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
  opacity: 0.7;
}

.form-hint {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0;
  line-height: 1.4;
}

.checkbox-group {
  flex-direction: row;
  flex-wrap: wrap;
  align-items: flex-start;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8125rem;
  color: var(--color-text);
  cursor: pointer;
}

.form-checkbox {
  width: 1rem;
  height: 1rem;
  cursor: pointer;
}

.form-checkbox:disabled {
  cursor: not-allowed;
}

.checkbox-group .form-hint {
  width: 100%;
  margin-top: 0.25rem;
}

.backoff-preview {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  padding: 0.75rem;
}

.backoff-preview .form-label {
  margin-bottom: 0.5rem;
}

.preview-timeline {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.preview-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0.375rem 0.625rem;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  min-width: 60px;
}

.retry-number {
  font-size: 0.625rem;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.delay-value {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-primary);
}

.preview-more {
  display: flex;
  align-items: center;
  padding: 0.375rem 0.625rem;
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  font-style: italic;
}
</style>
