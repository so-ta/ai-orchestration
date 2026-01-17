<script setup lang="ts">
/**
 * RunHistoryList.vue
 * 実行履歴リストコンポーネント（Run一覧のみ表示）
 */
import type { Run, TriggerType } from '~/types/api'
import {
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  AlertCircle,
} from 'lucide-vue-next'

defineProps<{
  runs: Run[]
  selectedRunId?: string | null
  loading?: boolean
  error?: string | null
}>()

defineEmits<{
  (e: 'run:select', run: Run): void
  (e: 'retry'): void
}>()

const { t } = useI18n()

// Format status
function formatStatus(status: string): string {
  return t(`runs.status.${status}`)
}

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

// Format trigger type
function formatTrigger(triggeredBy: TriggerType): string {
  const triggerLabels: Record<TriggerType, string> = {
    manual: t('execution.trigger.manual'),
    test: t('execution.trigger.test'),
    schedule: t('execution.trigger.schedule'),
    webhook: t('execution.trigger.webhook'),
    internal: t('execution.trigger.internal'),
  }
  return triggerLabels[triggeredBy] || triggeredBy
}

// Format date
function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return t('common.justNow')
  if (diff < 3600000) return t('common.minutesAgo', { count: Math.floor(diff / 60000) })
  if (diff < 86400000) return t('common.hoursAgo', { count: Math.floor(diff / 3600000) })
  return date.toLocaleString()
}

// Format duration
function formatDuration(ms?: number): string {
  if (ms === undefined || ms === null) return '-'
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${(ms / 60000).toFixed(1)}m`
}

// Calculate run duration
function calculateRunDuration(run: Run): string {
  if (!run.started_at) return '-'
  const start = new Date(run.started_at).getTime()
  const end = run.completed_at ? new Date(run.completed_at).getTime() : Date.now()
  return formatDuration(end - start)
}
</script>

<template>
  <div class="run-history-list">
    <!-- Loading State -->
    <div v-if="loading && runs.length === 0" class="loading-state">
      <Loader2 :size="24" class="spinning" />
      <span>{{ t('runs.loading') }}</span>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="error-state">
      <AlertCircle :size="20" />
      <span>{{ error }}</span>
      <button class="retry-btn" @click="$emit('retry')">{{ t('common.retry') }}</button>
    </div>

    <!-- Empty State -->
    <div v-else-if="runs.length === 0" class="empty-state">
      <Clock :size="32" />
      <p>{{ t('execution.noRuns') }}</p>
    </div>

    <!-- Runs Container -->
    <div v-else class="runs-container">
      <div
        v-for="run in runs"
        :key="run.id"
        class="run-row"
        :class="{ selected: run.id === selectedRunId }"
        @click="$emit('run:select', run)"
      >
        <span class="run-number">#{{ run.run_number }}</span>

        <span :class="['status-badge', run.status]">
          <component :is="getStatusIcon(run.status)" :size="12" :class="{ spinning: run.status === 'running' }" />
          {{ formatStatus(run.status) }}
        </span>

        <span :class="['trigger-tag', run.triggered_by]">
          {{ formatTrigger(run.triggered_by) }}
        </span>

        <span class="run-duration">{{ calculateRunDuration(run) }}</span>

        <span class="run-date">{{ formatDate(run.created_at) }}</span>

        <span class="step-count">
          {{ run.step_runs?.length || 0 }} {{ t('execution.steps') }}
        </span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.run-history-list {
  height: 100%;
  overflow-y: auto;
  padding: 0.5rem;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  gap: 0.75rem;
  color: #64748b;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Error State */
.error-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 1rem;
  color: #dc2626;
  font-size: 0.8125rem;
}

.retry-btn {
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  border: 1px solid currentColor;
  background: transparent;
  border-radius: 4px;
  cursor: pointer;
}

.retry-btn:hover {
  background: rgba(220, 38, 38, 0.1);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  gap: 0.5rem;
  color: #94a3b8;
}

.empty-state p {
  margin: 0;
  font-size: 0.875rem;
}

/* Runs Container */
.runs-container {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

/* Run Row */
.run-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.75rem;
  background: white;
  border: 1px solid rgba(0, 0, 0, 0.06);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
}

.run-row:hover {
  border-color: rgba(0, 0, 0, 0.1);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.run-row.selected {
  border-color: #3b82f6;
  background: rgba(59, 130, 246, 0.04);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

.run-number {
  font-size: 0.8125rem;
  font-weight: 600;
  color: #1e293b;
  min-width: 40px;
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

/* Trigger Tag */
.trigger-tag {
  font-size: 0.625rem;
  font-weight: 600;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
  text-transform: uppercase;
}

.trigger-tag.test {
  background: #fef3c7;
  color: #d97706;
}

.trigger-tag.manual {
  background: #dcfce7;
  color: #16a34a;
}

.trigger-tag.webhook {
  background: #dbeafe;
  color: #2563eb;
}

.trigger-tag.schedule {
  background: #f3e8ff;
  color: #9333ea;
}

.trigger-tag.internal {
  background: #f1f5f9;
  color: #64748b;
}

.run-duration {
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  color: #64748b;
}

.run-date {
  font-size: 0.6875rem;
  color: #94a3b8;
  margin-left: auto;
}

.step-count {
  font-size: 0.6875rem;
  color: #94a3b8;
  margin-left: 0.5rem;
}
</style>
