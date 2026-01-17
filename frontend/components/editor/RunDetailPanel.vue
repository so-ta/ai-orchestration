<script setup lang="ts">
/**
 * RunDetailPanel.vue
 * Run詳細パネル（ステップ一覧のみ表示、詳細は外部パネルに委譲）
 * - 実行中は3秒ごとに自動更新
 * - 実行中の視覚的フィードバック（プログレスバー）
 */
import type { Run, StepRun } from '~/types/api'
import {
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  AlertCircle,
  ChevronRight,
  RefreshCw,
} from 'lucide-vue-next'

const props = defineProps<{
  run: Run
  /** 現在選択されているステップのID */
  selectedStepRunId?: string | null
}>()

const emit = defineEmits<{
  (e: 'close' | 'refresh'): void
  (e: 'step:select', stepRun: StepRun): void
}>()

const { t } = useI18n()

// Auto-refresh state
const refreshing = ref(false)
let refreshInterval: ReturnType<typeof setInterval> | null = null

// Check if run is active (needs refresh)
const isActiveRun = computed(() => {
  return ['pending', 'running'].includes(props.run.status)
})

// Start auto-refresh when run is active
function startAutoRefresh() {
  if (refreshInterval) return
  refreshInterval = setInterval(() => {
    if (isActiveRun.value) {
      refreshing.value = true
      emit('refresh')
      // Reset refreshing state after a short delay
      setTimeout(() => {
        refreshing.value = false
      }, 500)
    } else {
      stopAutoRefresh()
    }
  }, 3000) // 3 seconds
}

// Stop auto-refresh
function stopAutoRefresh() {
  if (refreshInterval) {
    clearInterval(refreshInterval)
    refreshInterval = null
  }
}

// Watch for run changes to manage auto-refresh
watch(() => props.run.status, (newStatus) => {
  if (['pending', 'running'].includes(newStatus)) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
}, { immediate: true })

// Cleanup on unmount
onUnmounted(() => {
  stopAutoRefresh()
})

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

// Calculate run duration (updates in real-time for active runs)
const runDuration = ref('-')
let durationInterval: ReturnType<typeof setInterval> | null = null

function updateDuration() {
  if (!props.run.started_at) {
    runDuration.value = '-'
    return
  }
  const start = new Date(props.run.started_at).getTime()
  const end = props.run.completed_at ? new Date(props.run.completed_at).getTime() : Date.now()
  runDuration.value = formatDuration(end - start)
}

// Start duration timer for active runs
function startDurationTimer() {
  if (durationInterval) return
  updateDuration()
  durationInterval = setInterval(updateDuration, 1000)
}

function stopDurationTimer() {
  if (durationInterval) {
    clearInterval(durationInterval)
    durationInterval = null
  }
}

watch(() => props.run.status, (newStatus) => {
  if (['pending', 'running'].includes(newStatus)) {
    startDurationTimer()
  } else {
    stopDurationTimer()
    updateDuration() // Final update
  }
}, { immediate: true })

onUnmounted(() => {
  stopDurationTimer()
})

// Handle step click
function handleStepClick(stepRun: StepRun) {
  emit('step:select', stepRun)
}

// Manual refresh
function handleManualRefresh() {
  refreshing.value = true
  emit('refresh')
  setTimeout(() => {
    refreshing.value = false
  }, 500)
}
</script>

<template>
  <div class="run-detail-panel" :class="{ 'is-active': isActiveRun }">
    <!-- Running Progress Bar -->
    <div v-if="isActiveRun" class="running-progress">
      <div class="progress-bar" />
    </div>

    <!-- Run Header -->
    <div class="run-header">
      <div class="run-title">
        <span class="run-number">#{{ run.run_number }}</span>
        <span :class="['status-badge', run.status]">
          <component :is="getStatusIcon(run.status)" :size="12" :class="{ spinning: run.status === 'running' }" />
          {{ formatStatus(run.status) }}
        </span>
        <button
          v-if="isActiveRun"
          class="refresh-btn"
          :class="{ refreshing: refreshing }"
          :title="t('common.refresh')"
          @click="handleManualRefresh"
        >
          <RefreshCw :size="12" :class="{ spinning: refreshing }" />
        </button>
      </div>
      <div class="run-meta">
        <span class="meta-item duration" :class="{ 'is-live': isActiveRun }">
          {{ runDuration }}
        </span>
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
        :class="{ selected: stepRun.id === selectedStepRunId }"
        @click="handleStepClick(stepRun)"
      >
        <span :class="['step-status', stepRun.status]">
          <component :is="getStatusIcon(stepRun.status)" :size="14" :class="{ spinning: stepRun.status === 'running' }" />
        </span>
        <span class="step-name">{{ stepRun.step_name }}</span>
        <span class="step-duration">{{ formatDuration(stepRun.duration_ms) }}</span>
        <ChevronRight :size="14" class="step-arrow" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.run-detail-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  position: relative;
}

.run-detail-panel.is-active {
  border-top: 2px solid #3b82f6;
}

/* Running Progress Bar */
.running-progress {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: rgba(59, 130, 246, 0.2);
  overflow: hidden;
}

.progress-bar {
  width: 30%;
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
  animation: progress-slide 1.5s ease-in-out infinite;
}

@keyframes progress-slide {
  0% {
    transform: translateX(-100%);
  }
  100% {
    transform: translateX(400%);
  }
}

/* Run Header */
.run-header {
  padding: 0.75rem 1rem;
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  background: rgba(248, 250, 252, 0.5);
}

/* Refresh Button */
.refresh-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  background: rgba(59, 130, 246, 0.1);
  border-radius: 4px;
  cursor: pointer;
  color: #3b82f6;
  transition: all 0.15s;
  margin-left: auto;
}

.refresh-btn:hover {
  background: rgba(59, 130, 246, 0.2);
}

.refresh-btn.refreshing {
  background: rgba(59, 130, 246, 0.15);
}

/* Live Duration Indicator */
.meta-item.duration.is-live {
  color: #3b82f6;
  font-weight: 500;
  font-family: 'SF Mono', Monaco, monospace;
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
  transition: all 0.15s;
  border-bottom: 1px solid rgba(0, 0, 0, 0.03);
}

.step-row:hover {
  background: rgba(0, 0, 0, 0.02);
}

.step-row.selected {
  background: rgba(59, 130, 246, 0.06);
  border-left: 3px solid #3b82f6;
  padding-left: calc(1rem - 3px);
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
  transition: transform 0.15s;
}

.step-row:hover .step-arrow {
  transform: translateX(2px);
  color: #94a3b8;
}

.step-row.selected .step-arrow {
  color: #3b82f6;
}
</style>
