<script setup lang="ts">
import type { StepRun } from '~/types/api'

const { t } = useI18n()

const props = defineProps<{
  stepRun: StepRun | null
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

// Output tab state
type OutputTab = 'view' | 'markdown' | 'json'
const activeOutputTab = ref<OutputTab>('json')

// Check if output has markdown
const hasMarkdown = computed(() => {
  if (!props.stepRun?.output) return false
  const output = props.stepRun.output as Record<string, unknown>
  return typeof output.markdown === 'string' && output.markdown.length > 0
})

const outputMarkdown = computed(() => {
  if (!hasMarkdown.value) return ''
  const output = props.stepRun?.output as Record<string, unknown>
  return output.markdown as string
})

// Reset tab when modal opens: View if markdown available, otherwise JSON
watch(() => props.show, (show) => {
  if (show) {
    activeOutputTab.value = hasMarkdown.value ? 'view' : 'json'
  }
})

// Format JSON for display
function formatJson(obj: object | null | undefined): string {
  if (!obj) return '{}'
  try {
    return JSON.stringify(obj, null, 2)
  } catch {
    return String(obj)
  }
}

// Format duration
function formatDuration(ms?: number): string {
  if (!ms) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

// Format timestamp
function formatTimestamp(timestamp?: string): string {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  return date.toLocaleString()
}

// Get status icon
function getStatusIcon(status: string): string {
  switch (status) {
    case 'completed': return '\u2713'
    case 'failed': return '\u2717'
    case 'running': return '\u25B6'
    case 'pending': return '\u25CB'
    default: return '\u25CB'
  }
}

// Get status badge class
function getStatusBadge(status: string): string {
  switch (status) {
    case 'completed': return 'badge-success'
    case 'failed': return 'badge-error'
    case 'running': return 'badge-info'
    case 'pending': return 'badge-warning'
    default: return 'badge-secondary'
  }
}

// Copy to clipboard
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch (e) {
    console.error('Failed to copy:', e)
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="show && stepRun" class="step-modal-overlay" @click.self="emit('close')">
      <div class="step-modal">
        <div class="step-modal-header">
          <div class="step-modal-title-area">
            <span :class="['step-modal-status-icon', `status-${stepRun.status}`]">
              {{ getStatusIcon(stepRun.status) }}
            </span>
            <h3 class="step-modal-title">{{ stepRun.step_name }}</h3>
            <span :class="['badge', getStatusBadge(stepRun.status)]">
              {{ stepRun.status }}
            </span>
          </div>
          <button class="step-modal-close" @click="emit('close')">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>

        <div class="step-modal-body">
          <div class="step-modal-meta">
            <div class="step-meta-item">
              <span class="meta-label">Step ID</span>
              <code class="meta-value">{{ stepRun.step_id.substring(0, 8) }}...</code>
            </div>
            <div class="step-meta-item">
              <span class="meta-label">{{ t('execution.duration') }}</span>
              <span class="meta-value">{{ formatDuration(stepRun.duration_ms) }}</span>
            </div>
            <div v-if="stepRun.attempt > 1" class="step-meta-item">
              <span class="meta-label">{{ t('execution.attempt', { n: stepRun.attempt }) }}</span>
              <span class="meta-value attempt">{{ stepRun.attempt }}</span>
            </div>
            <div class="step-meta-item">
              <span class="meta-label">{{ t('execution.startedAt') }}</span>
              <span class="meta-value">{{ formatTimestamp(stepRun.started_at) }}</span>
            </div>
            <div class="step-meta-item">
              <span class="meta-label">{{ t('execution.completedAt') }}</span>
              <span class="meta-value">{{ formatTimestamp(stepRun.completed_at) }}</span>
            </div>
          </div>

          <div v-if="stepRun.error" class="step-modal-error">
            <div class="error-header">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"></circle>
                <line x1="12" y1="8" x2="12" y2="12"></line>
                <line x1="12" y1="16" x2="12.01" y2="16"></line>
              </svg>
              {{ t('execution.error') }}
            </div>
            <pre class="error-content">{{ stepRun.error }}</pre>
          </div>

          <div class="step-modal-data-section">
            <div class="data-section">
              <div class="data-section-header">
                <h4 class="data-section-title">{{ t('execution.input') }}</h4>
                <button
                  v-if="stepRun.input"
                  class="btn btn-outline btn-xs"
                  @click="copyToClipboard(formatJson(stepRun.input))"
                >
                  {{ t('common.copy') }}
                </button>
              </div>
              <pre v-if="stepRun.input && Object.keys(stepRun.input).length > 0" class="data-section-content">{{ formatJson(stepRun.input) }}</pre>
              <div v-else class="data-section-empty">{{ t('execution.noInputData') }}</div>
            </div>

            <div class="data-section">
              <div class="data-section-header">
                <h4 class="data-section-title">{{ t('execution.output') }}</h4>
                <div class="data-section-actions">
                  <!-- Tab switcher: View/Markdown only when markdown available -->
                  <div class="output-tabs">
                    <button
                      v-if="hasMarkdown"
                      :class="['tab-btn', { active: activeOutputTab === 'view' }]"
                      @click="activeOutputTab = 'view'"
                    >
                      View
                    </button>
                    <button
                      v-if="hasMarkdown"
                      :class="['tab-btn', { active: activeOutputTab === 'markdown' }]"
                      @click="activeOutputTab = 'markdown'"
                    >
                      Markdown
                    </button>
                    <button
                      :class="['tab-btn', { active: activeOutputTab === 'json' }]"
                      @click="activeOutputTab = 'json'"
                    >
                      JSON
                    </button>
                  </div>
                  <button
                    v-if="stepRun.output"
                    class="btn btn-outline btn-xs"
                    @click="copyToClipboard(formatJson(stepRun.output))"
                  >
                    {{ t('common.copy') }}
                  </button>
                </div>
              </div>
              <!-- View (Rich rendered markdown) -->
              <div v-if="activeOutputTab === 'view' && hasMarkdown" class="markdown-section-content">
                <ExtendedMarkdownRenderer :content="outputMarkdown" />
              </div>
              <!-- Markdown (raw markdown text) -->
              <pre v-else-if="activeOutputTab === 'markdown' && hasMarkdown" class="data-section-content">{{ outputMarkdown }}</pre>
              <!-- JSON view -->
              <pre v-else-if="activeOutputTab === 'json' && stepRun.output && Object.keys(stepRun.output).length > 0" class="data-section-content">{{ formatJson(stepRun.output) }}</pre>
              <div v-else-if="!stepRun.output || Object.keys(stepRun.output).length === 0" class="data-section-empty">{{ t('execution.noOutputData') }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
/* Step Details Modal */
.step-modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
  animation: fadeIn 0.15s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.step-modal {
  background: white;
  border-radius: var(--radius-lg, 12px);
  max-width: 700px;
  width: 100%;
  max-height: 85vh;
  overflow: hidden;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  animation: slideUp 0.2s ease-out;
}

@keyframes slideUp {
  from { transform: translateY(10px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.step-modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}

.step-modal-title-area {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.step-modal-status-icon {
  font-size: 1rem;
}

.step-modal-status-icon.status-completed {
  color: #10b981;
}

.step-modal-status-icon.status-failed {
  color: var(--color-error);
}

.step-modal-status-icon.status-running {
  color: var(--color-primary);
}

.step-modal-status-icon.status-pending {
  color: var(--color-text-secondary);
}

.step-modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  margin: 0;
}

.step-modal-close {
  background: none;
  border: none;
  padding: 0.5rem;
  cursor: pointer;
  color: var(--color-text-secondary);
  border-radius: 6px;
  transition: all 0.15s;
}

.step-modal-close:hover {
  background: rgba(0, 0, 0, 0.05);
  color: var(--color-text);
}

.step-modal-body {
  padding: 1.5rem;
  overflow-y: auto;
  max-height: calc(85vh - 60px);
}

.step-modal-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  padding: 1rem;
  background: var(--color-surface);
  border-radius: var(--radius);
  margin-bottom: 1.5rem;
}

.step-meta-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 100px;
}

.meta-label {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
}

.meta-value {
  font-size: 0.875rem;
  font-weight: 500;
}

.meta-value.attempt {
  color: #d97706;
}

.step-modal-error {
  margin-bottom: 1.5rem;
  padding: 1rem;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: var(--radius);
}

.step-modal-error .error-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--color-error);
  margin-bottom: 0.75rem;
}

.step-modal-error .error-content {
  font-size: 0.75rem;
  color: #b91c1c;
  margin: 0;
  white-space: pre-wrap;
  font-family: 'SF Mono', Monaco, monospace;
}

.step-modal-data-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

@media (max-width: 640px) {
  .step-modal-data-section {
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }
}

.data-section {
  background: var(--color-surface);
  border-radius: var(--radius);
  overflow: hidden;
}

.data-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  background: rgba(0, 0, 0, 0.02);
  border-bottom: 1px solid var(--color-border);
}

.data-section-title {
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.data-section-content {
  margin: 0;
  padding: 1rem;
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  overflow-x: auto;
  max-height: 300px;
  overflow-y: auto;
  background: white;
  line-height: 1.5;
}

.data-section-empty {
  padding: 2rem;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.btn-xs {
  padding: 0.25rem 0.5rem;
  font-size: 0.625rem;
}

/* Data section actions */
.data-section-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Output tabs */
.output-tabs {
  display: flex;
  gap: 2px;
  background: rgba(0, 0, 0, 0.05);
  padding: 2px;
  border-radius: 6px;
}

.tab-btn {
  padding: 0.25rem 0.625rem;
  font-size: 0.6875rem;
  font-weight: 500;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.tab-btn:hover {
  color: var(--color-text);
  background: rgba(0, 0, 0, 0.05);
}

.tab-btn.active {
  background: white;
  color: var(--color-text);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* Markdown section content */
.markdown-section-content {
  padding: 1rem;
  background: white;
  max-height: 400px;
  overflow-y: auto;
}
</style>
