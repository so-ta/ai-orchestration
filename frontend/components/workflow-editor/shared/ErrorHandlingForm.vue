<script setup lang="ts">
const { t } = useI18n()

export interface ErrorHandlingConfig {
  enabled: boolean
  retry?: {
    max_retries: number
    interval_seconds: number
    backoff_strategy: 'fixed' | 'exponential'
  }
  timeout_seconds?: number
  on_error: 'fail' | 'skip' | 'fallback'
  fallback_value?: unknown
}

const props = defineProps<{
  modelValue: ErrorHandlingConfig
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: ErrorHandlingConfig): void
}>()

const config = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// Initialize retry config if needed
const ensureRetryConfig = () => {
  if (!config.value.retry) {
    config.value = {
      ...config.value,
      retry: {
        max_retries: 3,
        interval_seconds: 1,
        backoff_strategy: 'fixed'
      }
    }
  }
}

// Update specific fields
const updateEnabled = (enabled: boolean) => {
  config.value = { ...config.value, enabled }
}

const updateRetryMaxRetries = (max_retries: number) => {
  ensureRetryConfig()
  config.value = {
    ...config.value,
    retry: { ...config.value.retry!, max_retries }
  }
}

const updateRetryInterval = (interval_seconds: number) => {
  ensureRetryConfig()
  config.value = {
    ...config.value,
    retry: { ...config.value.retry!, interval_seconds }
  }
}

const updateRetryBackoff = (backoff_strategy: 'fixed' | 'exponential') => {
  ensureRetryConfig()
  config.value = {
    ...config.value,
    retry: { ...config.value.retry!, backoff_strategy }
  }
}

const updateTimeout = (timeout_seconds: number | undefined) => {
  config.value = { ...config.value, timeout_seconds }
}

const updateOnError = (on_error: 'fail' | 'skip' | 'fallback') => {
  config.value = { ...config.value, on_error }
}

const updateFallbackValue = (value: string) => {
  try {
    const parsed = value ? JSON.parse(value) : undefined
    config.value = { ...config.value, fallback_value: parsed }
  } catch {
    // Invalid JSON, keep as string
    config.value = { ...config.value, fallback_value: value }
  }
}

const fallbackValueString = computed(() => {
  if (config.value.fallback_value === undefined) return ''
  return typeof config.value.fallback_value === 'string'
    ? config.value.fallback_value
    : JSON.stringify(config.value.fallback_value, null, 2)
})
</script>

<template>
  <div class="error-handling-form">
    <!-- Enable Toggle -->
    <div class="form-group">
      <label class="toggle-label">
        <input
          type="checkbox"
          :checked="config.enabled"
          :disabled="disabled"
          @change="updateEnabled(($event.target as HTMLInputElement).checked)"
        >
        <span class="toggle-text">{{ t('flow.errorHandling.enabled') }}</span>
      </label>
    </div>

    <div v-if="config.enabled" class="error-handling-options">
      <!-- Retry Settings -->
      <div class="subsection">
        <h5 class="subsection-title">{{ t('flow.errorHandling.retry.title') }}</h5>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">{{ t('flow.errorHandling.retry.maxRetries') }}</label>
            <input
              type="number"
              class="form-input"
              :value="config.retry?.max_retries ?? 3"
              :disabled="disabled"
              min="0"
              max="10"
              @input="updateRetryMaxRetries(Number(($event.target as HTMLInputElement).value))"
            >
          </div>
          <div class="form-group">
            <label class="form-label">{{ t('flow.errorHandling.retry.intervalSeconds') }}</label>
            <input
              type="number"
              class="form-input"
              :value="config.retry?.interval_seconds ?? 1"
              :disabled="disabled"
              min="0"
              max="300"
              @input="updateRetryInterval(Number(($event.target as HTMLInputElement).value))"
            >
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">{{ t('flow.errorHandling.retry.backoffStrategy') }}</label>
          <div class="radio-group">
            <label class="radio-label">
              <input
                type="radio"
                name="backoff"
                value="fixed"
                :checked="(config.retry?.backoff_strategy ?? 'fixed') === 'fixed'"
                :disabled="disabled"
                @change="updateRetryBackoff('fixed')"
              >
              <span>{{ t('flow.errorHandling.retry.fixed') }}</span>
            </label>
            <label class="radio-label">
              <input
                type="radio"
                name="backoff"
                value="exponential"
                :checked="config.retry?.backoff_strategy === 'exponential'"
                :disabled="disabled"
                @change="updateRetryBackoff('exponential')"
              >
              <span>{{ t('flow.errorHandling.retry.exponential') }}</span>
            </label>
          </div>
        </div>
      </div>

      <!-- Timeout Settings -->
      <div class="subsection">
        <h5 class="subsection-title">{{ t('flow.errorHandling.timeout.title') }}</h5>
        <div class="form-group">
          <label class="form-label">{{ t('flow.errorHandling.timeout.seconds') }}</label>
          <input
            type="number"
            class="form-input"
            :value="config.timeout_seconds ?? ''"
            :disabled="disabled"
            min="1"
            max="3600"
            placeholder="60"
            @input="updateTimeout(($event.target as HTMLInputElement).value ? Number(($event.target as HTMLInputElement).value) : undefined)"
          >
        </div>
      </div>

      <!-- On Error Action -->
      <div class="subsection">
        <h5 class="subsection-title">{{ t('flow.errorHandling.onError.title') }}</h5>
        <div class="action-options">
          <label :class="['action-option', { selected: config.on_error === 'fail' }]">
            <input
              type="radio"
              name="on_error"
              value="fail"
              :checked="config.on_error === 'fail'"
              :disabled="disabled"
              @change="updateOnError('fail')"
            >
            <div class="action-content">
              <span class="action-name">{{ t('flow.errorHandling.onError.fail') }}</span>
              <span class="action-desc">{{ t('flow.errorHandling.onError.failDesc') }}</span>
            </div>
          </label>
          <label :class="['action-option', { selected: config.on_error === 'skip' }]">
            <input
              type="radio"
              name="on_error"
              value="skip"
              :checked="config.on_error === 'skip'"
              :disabled="disabled"
              @change="updateOnError('skip')"
            >
            <div class="action-content">
              <span class="action-name">{{ t('flow.errorHandling.onError.skip') }}</span>
              <span class="action-desc">{{ t('flow.errorHandling.onError.skipDesc') }}</span>
            </div>
          </label>
          <label :class="['action-option', { selected: config.on_error === 'fallback' }]">
            <input
              type="radio"
              name="on_error"
              value="fallback"
              :checked="config.on_error === 'fallback'"
              :disabled="disabled"
              @change="updateOnError('fallback')"
            >
            <div class="action-content">
              <span class="action-name">{{ t('flow.errorHandling.onError.fallback') }}</span>
              <span class="action-desc">{{ t('flow.errorHandling.onError.fallbackDesc') }}</span>
            </div>
          </label>
        </div>

        <!-- Fallback Value (shown when fallback is selected) -->
        <div v-if="config.on_error === 'fallback'" class="form-group fallback-value-group">
          <label class="form-label">{{ t('flow.errorHandling.onError.fallbackValue') }}</label>
          <textarea
            class="form-input form-textarea code-input"
            :value="fallbackValueString"
            :disabled="disabled"
            rows="3"
            placeholder='{"default": "value"}'
            @input="updateFallbackValue(($event.target as HTMLTextAreaElement).value)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.error-handling-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.toggle-label input {
  width: 1rem;
  height: 1rem;
  cursor: pointer;
}

.toggle-text {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.error-handling-options {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding-left: 0.5rem;
  border-left: 2px solid var(--color-border);
}

.subsection {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.subsection-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.form-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.form-input {
  padding: 0.5rem 0.625rem;
  font-size: 0.8125rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-surface);
  color: var(--color-text);
  transition: border-color 0.2s, box-shadow 0.2s;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.form-textarea {
  resize: vertical;
  min-height: 60px;
}

.code-input {
  font-family: var(--font-mono);
  font-size: 0.75rem;
}

.radio-group {
  display: flex;
  gap: 1rem;
}

.radio-label {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  cursor: pointer;
  font-size: 0.8125rem;
  color: var(--color-text);
}

.radio-label input {
  cursor: pointer;
}

.action-options {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.action-option {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: border-color 0.2s, background-color 0.2s;
}

.action-option:hover {
  border-color: var(--color-border-hover);
  background: var(--color-surface-raised);
}

.action-option.selected {
  border-color: var(--color-primary);
  background: rgba(59, 130, 246, 0.05);
}

.action-option input {
  margin-top: 0.125rem;
  cursor: pointer;
}

.action-content {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.action-name {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.action-desc {
  font-size: 0.6875rem;
  color: var(--color-text-tertiary);
}

.fallback-value-group {
  margin-top: 0.5rem;
}
</style>
