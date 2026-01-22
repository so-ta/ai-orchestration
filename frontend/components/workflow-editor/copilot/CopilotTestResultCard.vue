<script setup lang="ts">
/**
 * CopilotTestResultCard.vue
 *
 * Test execution result card displayed within chat messages.
 * Shows step-by-step execution status with timing information.
 */
import type { WorkflowTestResult, TestStepStatus } from './types'

const { t } = useI18n()

const props = defineProps<{
  result: WorkflowTestResult
}>()

const emit = defineEmits<{
  'view-details': [runId: string]
  'retest': []
}>()

// Format duration
function formatDuration(ms: number | undefined): string {
  if (ms === undefined) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

// Status icon
function getStatusIcon(status: TestStepStatus): string {
  switch (status) {
    case 'success':
      return '✅'
    case 'error':
      return '❌'
    case 'running':
      return '🔄'
    default:
      return '⏳'
  }
}

// Status class
function getStatusClass(status: TestStepStatus): string {
  switch (status) {
    case 'success':
      return 'status-success'
    case 'error':
      return 'status-error'
    case 'running':
      return 'status-running'
    default:
      return 'status-pending'
  }
}

// Overall status display
const overallStatusIcon = computed(() => {
  switch (props.result.status) {
    case 'success':
      return '✅'
    case 'failed':
      return '❌'
    default:
      return '🔄'
  }
})

const overallStatusLabel = computed(() => {
  switch (props.result.status) {
    case 'success':
      return t('copilot.test.success')
    case 'failed':
      return t('copilot.test.failed')
    default:
      return t('copilot.test.running')
  }
})

// Computed: completed steps count
const completedSteps = computed(() => {
  return props.result.steps.filter(s => s.status === 'success' || s.status === 'error').length
})

// Handler functions
function handleViewDetails() {
  emit('view-details', props.result.runId)
}

function handleRetest() {
  emit('retest')
}
</script>

<template>
  <div
    class="test-result-card"
    :class="{
      'status-success': result.status === 'success',
      'status-failed': result.status === 'failed',
      'status-running': result.status === 'running'
    }"
  >
    <!-- Header -->
    <div class="result-header">
      <div class="header-left">
        <span class="status-icon">{{ overallStatusIcon }}</span>
        <span class="result-title">{{ t('copilot.test.title') }}</span>
        <span class="status-label">{{ overallStatusLabel }}</span>
      </div>
      <div v-if="result.totalDurationMs !== undefined" class="total-duration">
        {{ formatDuration(result.totalDurationMs) }}
      </div>
    </div>

    <!-- Step results -->
    <div class="step-results">
      <div
        v-for="step in result.steps"
        :key="step.stepId"
        class="step-result"
        :class="getStatusClass(step.status)"
      >
        <span class="step-icon">{{ getStatusIcon(step.status) }}</span>
        <span class="step-name">{{ step.stepName }}</span>
        <span v-if="step.status === 'success'" class="step-status ok">OK</span>
        <span v-else-if="step.status === 'error'" class="step-status error">ERROR</span>
        <span v-else-if="step.status === 'running'" class="step-status running">
          <span class="spinner" />
        </span>
        <span class="step-duration">{{ formatDuration(step.durationMs) }}</span>
      </div>
    </div>

    <!-- Error message (if any) -->
    <div v-if="result.status === 'failed'" class="error-section">
      <div
        v-for="step in result.steps.filter(s => s.error)"
        :key="step.stepId"
        class="error-item"
      >
        <span class="error-step">{{ step.stepName }}:</span>
        <span class="error-message">{{ step.error }}</span>
      </div>
    </div>

    <!-- Footer -->
    <div class="result-footer">
      <div class="footer-summary">
        {{ t('copilot.test.summary', { completed: completedSteps, total: result.steps.length }) }}
      </div>
      <div class="footer-actions">
        <button class="action-btn secondary" @click="handleViewDetails">
          {{ t('copilot.test.viewDetails') }}
        </button>
        <button class="action-btn primary" @click="handleRetest">
          {{ t('copilot.test.retest') }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.test-result-card {
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  overflow: hidden;
  margin-top: 0.5rem;
}

.test-result-card.status-success {
  border-color: var(--color-success);
}

.test-result-card.status-failed {
  border-color: var(--color-error);
}

.test-result-card.status-running {
  border-color: var(--color-warning);
}

/* Header */
.result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  background: var(--color-background);
  border-bottom: 1px solid var(--color-border);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-icon {
  font-size: 1rem;
}

.result-title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-text);
}

.status-label {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.125rem 0.5rem;
  border-radius: 4px;
}

.status-success .status-label {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
}

.status-failed .status-label {
  background: rgba(239, 68, 68, 0.15);
  color: #dc2626;
}

.status-running .status-label {
  background: rgba(234, 179, 8, 0.15);
  color: #ca8a04;
}

.total-duration {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-secondary);
}

/* Step results */
.step-results {
  display: flex;
  flex-direction: column;
  padding: 0.5rem 0;
}

.step-result {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 1rem;
  font-size: 0.8125rem;
}

.step-result:hover {
  background: var(--color-background);
}

.step-icon {
  width: 1.25rem;
  text-align: center;
  flex-shrink: 0;
}

.step-name {
  flex: 1;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.step-status {
  font-size: 0.6875rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
  text-transform: uppercase;
}

.step-status.ok {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
}

.step-status.error {
  background: rgba(239, 68, 68, 0.15);
  color: #dc2626;
}

.step-status.running {
  display: flex;
  align-items: center;
  background: rgba(234, 179, 8, 0.15);
  color: #ca8a04;
}

.spinner {
  width: 10px;
  height: 10px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.step-duration {
  width: 3.5rem;
  text-align: right;
  font-size: 0.75rem;
  color: var(--color-text-tertiary);
  font-family: ui-monospace, monospace;
}

/* Error section */
.error-section {
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.05);
  border-top: 1px solid var(--color-border);
}

.error-item {
  display: flex;
  gap: 0.5rem;
  font-size: 0.75rem;
  line-height: 1.4;
}

.error-step {
  color: var(--color-error);
  font-weight: 500;
  flex-shrink: 0;
}

.error-message {
  color: var(--color-text-secondary);
  word-break: break-word;
}

/* Footer */
.result-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-top: 1px solid var(--color-border);
  background: var(--color-background);
}

.footer-summary {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.footer-actions {
  display: flex;
  gap: 0.5rem;
}

.action-btn {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.action-btn.primary {
  background: var(--color-primary);
  color: white;
  border: none;
}

.action-btn.primary:hover {
  opacity: 0.9;
}

.action-btn.secondary {
  background: transparent;
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
}

.action-btn.secondary:hover {
  color: var(--color-text);
  border-color: var(--color-text-secondary);
}
</style>
