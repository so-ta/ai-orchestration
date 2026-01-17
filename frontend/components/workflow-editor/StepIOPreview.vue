<script setup lang="ts">
/**
 * StepIOPreview.vue
 * ステップ入出力プレビューコンポーネント
 * PropertiesPanelに表示する
 */
import type { StepRun } from '~/types/api'
import {
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  AlertCircle,
  Copy,
  Maximize2,
} from 'lucide-vue-next'

const props = defineProps<{
  stepRun: StepRun | null
}>()

defineEmits<{
  (e: 'expand'): void
}>()

const { t } = useI18n()
const toast = useToast()

// Get status icon component
function getStatusIcon(status: string) {
  switch (status) {
    case 'completed':
      return CheckCircle2
    case 'failed':
      return XCircle
    case 'running':
      return Loader2
    case 'pending':
      return Clock
    case 'cancelled':
      return AlertCircle
    default:
      return Clock
  }
}

// Format JSON
function formatJson(data: unknown): string {
  if (data === undefined || data === null) return '-'
  try {
    return JSON.stringify(data, null, 2)
  } catch {
    return String(data)
  }
}

// Format duration
function formatDuration(ms?: number): string {
  if (ms === undefined || ms === null) return '-'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

// Format date
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString()
}

// Copy to clipboard
async function copyToClipboard(data: unknown, label: string) {
  try {
    const text = typeof data === 'string' ? data : JSON.stringify(data, null, 2)
    await navigator.clipboard.writeText(text)
    toast.success(t('common.copied', { item: label }))
  } catch {
    toast.error(t('common.copyFailed'))
  }
}

function copyInput() {
  if (props.stepRun?.input) {
    copyToClipboard(props.stepRun.input, 'Input')
  }
}

function copyOutput() {
  if (props.stepRun?.output) {
    copyToClipboard(props.stepRun.output, 'Output')
  }
}
</script>

<template>
  <div v-if="stepRun" class="step-io-preview">
    <!-- Header -->
    <div class="preview-header">
      <span :class="['status-icon', stepRun.status]">
        <component :is="getStatusIcon(stepRun.status)" :size="14" :class="{ spinning: stepRun.status === 'running' }" />
      </span>
      <span class="step-name">{{ stepRun.step_name }}</span>
      <span :class="['status-badge', stepRun.status]">
        {{ t(`runs.status.${stepRun.status}`) }}
      </span>
    </div>

    <!-- Meta -->
    <div class="preview-meta">
      <span class="duration">{{ formatDuration(stepRun.duration_ms) }}</span>
      <span class="separator">|</span>
      <span class="timestamp">{{ formatDate(stepRun.completed_at || stepRun.started_at) }}</span>
    </div>

    <!-- Error Display -->
    <div v-if="stepRun.error" class="preview-error">
      <div class="error-label">Error</div>
      <pre class="error-content">{{ stepRun.error }}</pre>
    </div>

    <!-- Input Preview -->
    <div class="preview-section">
      <div class="section-header">
        <span class="section-title">{{ t('execution.input') }}</span>
        <button class="copy-btn" :title="t('common.copy')" @click="copyInput">
          <Copy :size="12" />
        </button>
      </div>
      <pre class="section-content">{{ formatJson(stepRun.input) }}</pre>
    </div>

    <!-- Output Preview -->
    <div class="preview-section">
      <div class="section-header">
        <span class="section-title">{{ t('execution.output') }}</span>
        <button class="copy-btn" :title="t('common.copy')" @click="copyOutput">
          <Copy :size="12" />
        </button>
      </div>
      <pre class="section-content">{{ formatJson(stepRun.output) }}</pre>
    </div>

    <!-- Expand Button -->
    <button class="expand-btn" @click="$emit('expand')">
      <Maximize2 :size="14" />
      {{ t('execution.showDetails') }}
    </button>
  </div>

  <!-- Empty State -->
  <div v-else class="step-io-empty">
    <Clock :size="24" />
    <p>{{ t('execution.selectStepRun') }}</p>
  </div>
</template>

<style scoped>
.step-io-preview {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 0.875rem;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid rgba(0, 0, 0, 0.06);
}

/* Header */
.preview-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-icon.completed {
  color: #16a34a;
}

.status-icon.failed {
  color: #dc2626;
}

.status-icon.running {
  color: #2563eb;
}

.status-icon.pending {
  color: #d97706;
}

.status-icon.cancelled {
  color: #64748b;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.step-name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #1e293b;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-badge {
  font-size: 0.625rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  text-transform: uppercase;
}

.status-badge.completed {
  background: #dcfce7;
  color: #16a34a;
}

.status-badge.failed {
  background: #fee2e2;
  color: #dc2626;
}

.status-badge.running {
  background: #dbeafe;
  color: #2563eb;
}

.status-badge.pending {
  background: #fef3c7;
  color: #d97706;
}

.status-badge.cancelled {
  background: #f1f5f9;
  color: #64748b;
}

/* Meta */
.preview-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.6875rem;
  color: #64748b;
}

.duration {
  font-family: 'SF Mono', Monaco, monospace;
}

.separator {
  color: #cbd5e1;
}

/* Error */
.preview-error {
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 6px;
  padding: 0.5rem;
}

.error-label {
  font-size: 0.625rem;
  font-weight: 600;
  color: #dc2626;
  text-transform: uppercase;
  margin-bottom: 0.25rem;
}

.error-content {
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: #b91c1c;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 60px;
  overflow-y: auto;
}

/* Section */
.preview-section {
  display: flex;
  flex-direction: column;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.25rem;
}

.section-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
}

.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  border: none;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
  color: #94a3b8;
  transition: all 0.15s;
}

.copy-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #64748b;
}

.section-content {
  font-size: 0.6875rem;
  font-family: 'SF Mono', Monaco, monospace;
  background: white;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 4px;
  padding: 0.5rem;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 80px;
  overflow-y: auto;
  color: #475569;
}

/* Expand Button */
.expand-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.375rem;
  width: 100%;
  padding: 0.5rem;
  border: 1px solid rgba(0, 0, 0, 0.08);
  background: white;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.75rem;
  font-weight: 500;
  color: #64748b;
  transition: all 0.15s;
}

.expand-btn:hover {
  background: #f8fafc;
  color: #3b82f6;
  border-color: rgba(59, 130, 246, 0.3);
}

/* Empty State */
.step-io-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px dashed rgba(0, 0, 0, 0.1);
  color: #94a3b8;
  text-align: center;
}

.step-io-empty p {
  margin: 0.5rem 0 0;
  font-size: 0.75rem;
}
</style>
