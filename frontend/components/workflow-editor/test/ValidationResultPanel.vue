<script setup lang="ts">
/**
 * ValidationResultPanel.vue
 * ワークフロー検証結果を表示するパネルコンポーネント
 */

const { t } = useI18n()

interface ValidationCheck {
  id: string
  label: string
  status: 'passed' | 'warning' | 'error'
  message?: string
}

interface ValidationResult {
  checks: ValidationCheck[]
  can_publish: boolean
  error_count: number
  warning_count: number
}

defineProps<{
  result: ValidationResult | null
  loading?: boolean
}>()

const emit = defineEmits<{
  (e: 'retry'): void
  (e: 'jump-to-step', stepId: string): void
}>()

// Get status icon
function getStatusIcon(status: string) {
  switch (status) {
    case 'passed':
      return '✓'
    case 'warning':
      return '⚠'
    case 'error':
      return '✗'
    default:
      return '?'
  }
}

// Get status class
function getStatusClass(status: string) {
  switch (status) {
    case 'passed':
      return 'status-passed'
    case 'warning':
      return 'status-warning'
    case 'error':
      return 'status-error'
    default:
      return ''
  }
}
</script>

<template>
  <div class="validation-result-panel">
    <!-- Loading State -->
    <div v-if="loading" class="validation-loading">
      <div class="loading-spinner"></div>
      <span>{{ t('validation.validating') }}</span>
    </div>

    <!-- Results -->
    <div v-else-if="result" class="validation-results">
      <!-- Summary -->
      <div class="validation-summary" :class="{ 'has-errors': result.error_count > 0 }">
        <div v-if="result.can_publish" class="summary-status success">
          <span class="summary-icon">✓</span>
          <span>{{ t('validation.readyToPublish') }}</span>
        </div>
        <div v-else class="summary-status error">
          <span class="summary-icon">✗</span>
          <span>{{ t('validation.notReadyToPublish') }}</span>
        </div>
        <div class="summary-counts">
          <span v-if="result.error_count > 0" class="count error">
            {{ result.error_count }} {{ t('validation.errors') }}
          </span>
          <span v-if="result.warning_count > 0" class="count warning">
            {{ result.warning_count }} {{ t('validation.warnings') }}
          </span>
        </div>
      </div>

      <!-- Check List -->
      <div class="validation-checks">
        <div
          v-for="check in result.checks"
          :key="check.id"
          class="validation-check"
          :class="getStatusClass(check.status)"
        >
          <span class="check-icon">{{ getStatusIcon(check.status) }}</span>
          <div class="check-content">
            <span class="check-label">{{ check.label }}</span>
            <span v-if="check.message" class="check-message">{{ check.message }}</span>
          </div>
        </div>
      </div>

      <!-- Retry Button -->
      <button class="retry-button" @click="emit('retry')">
        <span>↻</span>
        {{ t('validation.revalidate') }}
      </button>
    </div>

    <!-- No Results -->
    <div v-else class="validation-empty">
      <p>{{ t('validation.clickToValidate') }}</p>
    </div>
  </div>
</template>

<style scoped>
.validation-result-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* Loading */
.validation-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 2rem;
  color: var(--color-text-secondary);
}

.loading-spinner {
  width: 20px;
  height: 20px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Summary */
.validation-summary {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
  background: var(--color-surface);
  border-radius: 8px;
  border: 1px solid var(--color-border);
}

.validation-summary.has-errors {
  border-color: var(--color-error, #ef4444);
  background: rgba(239, 68, 68, 0.05);
}

.summary-status {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 500;
}

.summary-status.success {
  color: var(--color-success, #10b981);
}

.summary-status.error {
  color: var(--color-error, #ef4444);
}

.summary-icon {
  font-size: 1.25rem;
}

.summary-counts {
  display: flex;
  gap: 0.75rem;
  font-size: 0.8125rem;
}

.count {
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}

.count.error {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-error, #ef4444);
}

.count.warning {
  background: rgba(245, 158, 11, 0.1);
  color: var(--color-warning, #f59e0b);
}

/* Check List */
.validation-checks {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.validation-check {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
}

.validation-check.status-passed {
  border-left: 3px solid var(--color-success, #10b981);
}

.validation-check.status-warning {
  border-left: 3px solid var(--color-warning, #f59e0b);
}

.validation-check.status-error {
  border-left: 3px solid var(--color-error, #ef4444);
}

.check-icon {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
}

.status-passed .check-icon {
  color: var(--color-success, #10b981);
}

.status-warning .check-icon {
  color: var(--color-warning, #f59e0b);
}

.status-error .check-icon {
  color: var(--color-error, #ef4444);
}

.check-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.check-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
}

.check-message {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

/* Retry Button */
.retry-button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.retry-button:hover {
  color: var(--color-text);
  border-color: var(--color-border-hover);
}

/* Empty State */
.validation-empty {
  padding: 2rem;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}
</style>
