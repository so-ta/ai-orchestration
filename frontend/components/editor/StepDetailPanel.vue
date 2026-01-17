<script setup lang="ts">
/**
 * StepDetailPanel.vue
 * ステップ詳細表示パネル（入出力データ、エラー情報）
 */
import type { StepRun } from '~/types/api'
import {
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  AlertCircle,
  Copy,
} from 'lucide-vue-next'

defineProps<{
  stepRun: StepRun
}>()

const { t } = useI18n()

// Get status icon
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

// Format status
function formatStatus(status: string): string {
  return t(`runs.status.${status}`)
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
  return new Date(dateStr).toLocaleString()
}

// Format JSON for display
function formatJson(data: unknown): string {
  if (data === undefined || data === null) return '-'
  try {
    return JSON.stringify(data, null, 2)
  } catch {
    return String(data)
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
  <div class="step-detail-panel">
    <!-- Step Header -->
    <div class="step-header">
      <span class="step-name">{{ stepRun.step_name }}</span>
      <span :class="['status-badge', stepRun.status]">
        <component :is="getStatusIcon(stepRun.status)" :size="10" :class="{ spinning: stepRun.status === 'running' }" />
        {{ formatStatus(stepRun.status) }}
      </span>
    </div>

    <!-- Step Meta -->
    <div class="step-meta">
      <div class="meta-item">
        <span class="meta-label">{{ t('execution.duration') }}</span>
        <span class="meta-value">{{ formatDuration(stepRun.duration_ms) }}</span>
      </div>
      <div v-if="stepRun.started_at" class="meta-item">
        <span class="meta-label">{{ t('execution.startedAt') }}</span>
        <span class="meta-value">{{ formatDate(stepRun.started_at) }}</span>
      </div>
    </div>

    <!-- Error -->
    <div v-if="stepRun.error" class="step-section error-section">
      <div class="section-header">
        <span class="section-title error">{{ t('execution.error') }}</span>
      </div>
      <pre class="section-content error">{{ stepRun.error }}</pre>
    </div>

    <!-- Input -->
    <div class="step-section">
      <div class="section-header">
        <span class="section-title">{{ t('execution.input') }}</span>
        <button
          class="copy-btn"
          :title="t('common.copy')"
          @click="copyToClipboard(formatJson(stepRun.input))"
        >
          <Copy :size="12" />
        </button>
      </div>
      <pre class="section-content">{{ formatJson(stepRun.input) }}</pre>
    </div>

    <!-- Output -->
    <div class="step-section">
      <div class="section-header">
        <span class="section-title">{{ t('execution.output') }}</span>
        <button
          class="copy-btn"
          :title="t('common.copy')"
          @click="copyToClipboard(formatJson(stepRun.output))"
        >
          <Copy :size="12" />
        </button>
      </div>
      <pre class="section-content">{{ formatJson(stepRun.output) }}</pre>
    </div>
  </div>
</template>

<style scoped>
.step-detail-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* Step Header */
.step-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(248, 250, 252, 0.5);
}

.step-name {
  flex: 1;
  font-size: 0.875rem;
  font-weight: 600;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Status Badge */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.625rem;
  font-weight: 500;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
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

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Step Meta */
.step-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.5rem;
  padding: 0.625rem 1rem;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
  background: rgba(248, 250, 252, 0.3);
}

.step-meta .meta-item {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.step-meta .meta-label {
  font-size: 0.625rem;
  font-weight: 500;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.step-meta .meta-value {
  font-size: 0.75rem;
  color: #475569;
  font-family: 'SF Mono', Monaco, monospace;
}

/* Step Section (Input/Output/Error) */
.step-section {
  display: flex;
  flex-direction: column;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.step-section:last-child {
  border-bottom: none;
  flex: 1;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.5rem 1rem;
  background: rgba(248, 250, 252, 0.5);
}

.section-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.section-title.error {
  color: #dc2626;
}

.copy-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
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
  margin: 0;
  padding: 0.75rem 1rem;
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  line-height: 1.5;
  color: #475569;
  background: rgba(248, 250, 252, 0.3);
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
}

.section-content.error {
  color: #dc2626;
  background: rgba(254, 226, 226, 0.3);
}

.error-section {
  flex-shrink: 0;
}
</style>
