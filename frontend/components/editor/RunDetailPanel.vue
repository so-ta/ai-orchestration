<script setup lang="ts">
/**
 * RunDetailPanel.vue
 * Run詳細パネル（ステップ一覧 + ステップ詳細）
 */
import type { Run, StepRun } from '~/types/api'
import {
  ArrowLeft,
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  AlertCircle,
  Copy,
  ChevronRight,
} from 'lucide-vue-next'

const props = defineProps<{
  run: Run
}>()

defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()

// Currently selected step run for detail view
const selectedStepRun = ref<StepRun | null>(null)

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

// Select a step run
function selectStepRun(stepRun: StepRun) {
  selectedStepRun.value = stepRun
}

// Go back to step list
function backToStepList() {
  selectedStepRun.value = null
}

// Calculate run duration
function calculateRunDuration(): string {
  if (!props.run.started_at) return '-'
  const start = new Date(props.run.started_at).getTime()
  const end = props.run.completed_at ? new Date(props.run.completed_at).getTime() : Date.now()
  return formatDuration(end - start)
}

// Watch for run changes - reset selected step
watch(() => props.run.id, () => {
  selectedStepRun.value = null
})
</script>

<template>
  <div class="run-detail-panel">
    <!-- Step Detail View -->
    <template v-if="selectedStepRun">
      <!-- Step Detail Header -->
      <div class="panel-section-header">
        <button class="back-btn" @click="backToStepList">
          <ArrowLeft :size="16" />
        </button>
        <span class="step-name">{{ selectedStepRun.step_name }}</span>
        <span :class="['status-badge', 'small', selectedStepRun.status]">
          <component :is="getStatusIcon(selectedStepRun.status)" :size="10" />
          {{ formatStatus(selectedStepRun.status) }}
        </span>
      </div>

      <!-- Step Meta -->
      <div class="step-meta">
        <div class="meta-item">
          <span class="meta-label">{{ t('execution.duration') }}</span>
          <span class="meta-value">{{ formatDuration(selectedStepRun.duration_ms) }}</span>
        </div>
        <div v-if="selectedStepRun.started_at" class="meta-item">
          <span class="meta-label">{{ t('execution.startedAt') }}</span>
          <span class="meta-value">{{ formatDate(selectedStepRun.started_at) }}</span>
        </div>
      </div>

      <!-- Error -->
      <div v-if="selectedStepRun.error" class="step-section error-section">
        <div class="section-header">
          <span class="section-title error">{{ t('execution.error') }}</span>
        </div>
        <pre class="section-content error">{{ selectedStepRun.error }}</pre>
      </div>

      <!-- Input -->
      <div class="step-section">
        <div class="section-header">
          <span class="section-title">{{ t('execution.input') }}</span>
          <button
            class="copy-btn"
            :title="t('common.copy')"
            @click="copyToClipboard(formatJson(selectedStepRun.input))"
          >
            <Copy :size="12" />
          </button>
        </div>
        <pre class="section-content">{{ formatJson(selectedStepRun.input) }}</pre>
      </div>

      <!-- Output -->
      <div class="step-section">
        <div class="section-header">
          <span class="section-title">{{ t('execution.output') }}</span>
          <button
            class="copy-btn"
            :title="t('common.copy')"
            @click="copyToClipboard(formatJson(selectedStepRun.output))"
          >
            <Copy :size="12" />
          </button>
        </div>
        <pre class="section-content">{{ formatJson(selectedStepRun.output) }}</pre>
      </div>
    </template>

    <!-- Step List View -->
    <template v-else>
      <!-- Run Header -->
      <div class="run-header">
        <div class="run-title">
          <span class="run-number">#{{ run.run_number }}</span>
          <span :class="['status-badge', run.status]">
            <component :is="getStatusIcon(run.status)" :size="12" :class="{ spinning: run.status === 'running' }" />
            {{ formatStatus(run.status) }}
          </span>
        </div>
        <div class="run-meta">
          <span class="meta-item">{{ calculateRunDuration() }}</span>
          <span class="meta-item">{{ formatDate(run.created_at) }}</span>
        </div>
      </div>

      <!-- Step List -->
      <div class="step-list">
        <div class="step-list-header">
          <span class="step-list-title">{{ t('execution.steps') }}</span>
          <span class="step-count">{{ run.step_runs?.length || 0 }}</span>
        </div>

        <div v-if="!run.step_runs || run.step_runs.length === 0" class="empty-steps">
          {{ t('execution.noSteps') }}
        </div>

        <div
          v-for="stepRun in run.step_runs"
          v-else
          :key="stepRun.id"
          class="step-row"
          @click="selectStepRun(stepRun)"
        >
          <span :class="['step-status', stepRun.status]">
            <component :is="getStatusIcon(stepRun.status)" :size="14" :class="{ spinning: stepRun.status === 'running' }" />
          </span>
          <span class="step-name">{{ stepRun.step_name }}</span>
          <span class="step-duration">{{ formatDuration(stepRun.duration_ms) }}</span>
          <ChevronRight :size="14" class="step-arrow" />
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.run-detail-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* Run Header */
.run-header {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(248, 250, 252, 0.5);
}

.run-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.375rem;
}

.run-number {
  font-size: 0.9375rem;
  font-weight: 600;
  color: #1e293b;
}

.run-meta {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.run-meta .meta-item {
  font-size: 0.75rem;
  color: #64748b;
}

/* Status Badge */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.6875rem;
  font-weight: 500;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
}

.status-badge.small {
  font-size: 0.625rem;
  padding: 0.125rem 0.375rem;
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

/* Step List */
.step-list {
  flex: 1;
  overflow-y: auto;
}

.step-list-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.step-list-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: #64748b;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.step-count {
  font-size: 0.6875rem;
  color: #94a3b8;
  background: rgba(0, 0, 0, 0.04);
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
}

.empty-steps {
  padding: 2rem 1rem;
  text-align: center;
  color: #94a3b8;
  font-size: 0.8125rem;
}

.step-row {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.625rem 1rem;
  cursor: pointer;
  transition: background 0.15s;
  border-bottom: 1px solid rgba(0, 0, 0, 0.03);
}

.step-row:hover {
  background: rgba(0, 0, 0, 0.02);
}

.step-row:last-child {
  border-bottom: none;
}

.step-status {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.step-status.completed {
  color: #16a34a;
}

.step-status.failed {
  color: #dc2626;
}

.step-status.running {
  color: #2563eb;
}

.step-status.pending {
  color: #d97706;
}

.step-status.cancelled {
  color: #64748b;
}

.step-row .step-name {
  flex: 1;
  font-size: 0.8125rem;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.step-duration {
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: #64748b;
}

.step-arrow {
  color: #cbd5e1;
  flex-shrink: 0;
}

/* Step Detail View */
.panel-section-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(248, 250, 252, 0.5);
}

.back-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  color: #64748b;
  transition: all 0.15s;
}

.back-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #1e293b;
}

.panel-section-header .step-name {
  flex: 1;
  font-size: 0.875rem;
  font-weight: 600;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
