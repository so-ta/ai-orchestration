<script setup lang="ts">
import type { Step, Run, StepRun } from '~/types/api'

const { t } = useI18n()
const runs = useRuns()
const toast = useToast()

const props = defineProps<{
  step: Step | null
  workflowId: string
  latestRun: Run | null
}>()

const emit = defineEmits<{
  (e: 'execute', data: { stepId: string; input: object; mode: 'test' | 'production' }): void
  (e: 'execute-workflow', mode: 'test' | 'production', input: object): void
}>()

// Execution state
const executing = ref(false)

// Custom input state (always shown, no toggle)
const customInputJson = ref('{}')
const inputError = ref<string | null>(null)

// Step execution history
const stepHistory = ref<StepRun[]>([])
const loadingHistory = ref(false)

// Modal state for viewing step run details
const selectedStepRun = ref<StepRun | null>(null)
const showStepRunModal = ref(false)

// Open modal with step run details
function openStepRunModal(stepRun: StepRun) {
  selectedStepRun.value = stepRun
  showStepRunModal.value = true
}

// Close modal
function closeStepRunModal() {
  showStepRunModal.value = false
  selectedStepRun.value = null
}

// Parse and validate custom input
function parseCustomInput(): object | null {
  inputError.value = null

  try {
    const parsed = JSON.parse(customInputJson.value)
    if (typeof parsed !== 'object' || parsed === null) {
      inputError.value = t('execution.errors.invalidJsonObject')
      return null
    }
    return parsed
  } catch (e) {
    inputError.value = t('execution.errors.invalidJson')
    return null
  }
}

// Execute this step only
async function executeThisStepOnly() {
  if (!props.step) {
    toast.error(t('execution.errors.noStepSelected'))
    return
  }

  const input = parseCustomInput()
  if (input === null) return

  executing.value = true

  try {
    if (props.latestRun) {
      // Use existing run for re-execution
      await runs.executeSingleStep(
        props.latestRun.id,
        props.step.id,
        Object.keys(input).length > 0 ? input : undefined
      )

      toast.success(t('execution.stepExecuted'))
      await loadStepHistory()
    } else {
      // Use inline test API (creates new test run)
      const response = await runs.testStepInline(
        props.workflowId,
        props.step.id,
        Object.keys(input).length > 0 ? input : undefined
      )

      toast.success(t('execution.stepQueued'))

      // Start polling for result
      startPolling(response.data.run_id, props.step.id)
    }
  } catch (e) {
    toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    executing.value = false
  }
}

// Execute from this step (run workflow starting from this step)
async function executeFromThisStep() {
  if (!props.step) {
    toast.error(t('execution.errors.noStepSelected'))
    return
  }

  const input = parseCustomInput()
  if (input === null) return

  executing.value = true

  try {
    await runs.create(props.workflowId, {
      mode: 'test',
      input: Object.keys(input).length > 0 ? input : {},
      start_step_id: props.step.id
    })

    emit('execute-workflow', 'test', input)
    toast.success(t('execution.workflowStarted'))
  } catch (e) {
    toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
  } finally {
    executing.value = false
  }
}

// Poll for inline test result
const pollingInterval = ref<ReturnType<typeof setInterval> | null>(null)
const pollingRunId = ref<string | null>(null)

async function startPolling(runId: string, stepId: string) {
  pollingRunId.value = runId
  let attempts = 0
  const maxAttempts = 60 // 60 seconds timeout

  pollingInterval.value = setInterval(async () => {
    attempts++
    if (attempts > maxAttempts) {
      stopPolling()
      toast.warning(t('execution.pollingTimeout'))
      return
    }

    try {
      const response = await runs.get(runId)
      const run = response.data

      // Check if step has completed
      const stepRun = run.step_runs?.find((sr: { step_id: string }) => sr.step_id === stepId)
      if (stepRun) {
        if (stepRun.status === 'completed') {
          stopPolling()
          toast.success(t('execution.stepTestCompleted'))
          await loadStepHistory()
        } else if (stepRun.status === 'failed') {
          stopPolling()
          toast.error(t('execution.stepTestFailed'))
          await loadStepHistory()
        }
      }
    } catch (e) {
      console.error('Polling error:', e)
    }
  }, 1000)
}

function stopPolling() {
  if (pollingInterval.value) {
    clearInterval(pollingInterval.value)
    pollingInterval.value = null
  }
  pollingRunId.value = null
}

// Cleanup on unmount
onUnmounted(() => {
  stopPolling()
})

// Load step execution history
async function loadStepHistory() {
  if (!props.step || !props.latestRun) return

  loadingHistory.value = true
  try {
    const response = await runs.getStepHistory(props.latestRun.id, props.step.id)
    stepHistory.value = response.data
  } catch (e) {
    console.error('Failed to load step history:', e)
  } finally {
    loadingHistory.value = false
  }
}

// Format step status
function formatStatus(status: string): string {
  return t(`execution.status.${status}`) || status
}

// Format duration
function formatDuration(ms?: number): string {
  if (!ms) return '-'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

// Watch for step changes
watch(() => props.step, () => {
  if (props.step && props.latestRun) {
    loadStepHistory()
  }
}, { immediate: true })

// Watch for latest run changes
watch(() => props.latestRun, () => {
  if (props.step && props.latestRun) {
    loadStepHistory()
  }
})
</script>

<template>
  <div class="execution-tab">
    <div class="execution-content">
      <!-- Custom Input (always visible) -->
      <div class="input-section">
        <label class="input-label">{{ t('execution.customInput') }}</label>
        <textarea
          v-model="customInputJson"
          class="json-input"
          rows="4"
          :placeholder="t('execution.inputPlaceholder')"
        ></textarea>
        <p v-if="inputError" class="input-error">{{ inputError }}</p>
      </div>

      <!-- Execution Buttons -->
      <div class="execution-buttons">
        <button
          class="btn btn-primary"
          :disabled="executing || !!pollingRunId"
          @click="executeThisStepOnly"
        >
          <svg v-if="!pollingRunId && !executing" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="5 3 19 12 5 21 5 3"></polygon>
          </svg>
          <svg v-else class="spinning" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 12a9 9 0 1 1-6.219-8.56"></path>
          </svg>
          {{ pollingRunId ? t('execution.waitingForResult') : (executing ? t('execution.executing') : t('execution.executeThisStepOnly')) }}
        </button>
        <button
          class="btn btn-outline"
          :disabled="executing || !!pollingRunId"
          @click="executeFromThisStep"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
          </svg>
          {{ t('execution.executeFromThisStep') }}
        </button>
      </div>

      <p v-if="!latestRun" class="info-text">
        {{ t('execution.inlineTestInfo') }}
      </p>

      <!-- Step History -->
      <div v-if="stepHistory.length > 0" class="step-history">
        <h4 class="history-title">{{ t('execution.history') }}</h4>
        <div class="history-list">
          <div
            v-for="run in stepHistory.slice(0, 5)"
            :key="run.id"
            class="history-item"
            @click="openStepRunModal(run)"
          >
            <span :class="['status-badge', run.status]">{{ formatStatus(run.status) }}</span>
            <span class="history-duration">{{ formatDuration(run.duration_ms) }}</span>
            <span class="history-attempt">{{ t('execution.attempt', { n: run.attempt }) }}</span>
            <svg class="history-arrow" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </div>
        </div>
      </div>
    </div>

    <!-- Step Run Modal -->
    <StepRunModal
      :step-run="selectedStepRun"
      :show="showStepRunModal"
      @close="closeStepRunModal"
    />
  </div>
</template>

<style scoped>
.execution-tab {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* Execution Content */
.execution-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.5rem 0;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

/* Input Section */
.input-section {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.input-label {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text);
}

.json-input {
  width: 100%;
  padding: 0.5rem;
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  resize: vertical;
  min-height: 60px;
}

.json-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.input-error {
  font-size: 0.6875rem;
  color: var(--color-error);
  margin: 0;
}

/* Execution Buttons */
.execution-buttons {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.execution-buttons .btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
}

.info-text {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0;
  padding: 0.5rem;
  background: #f0f9ff;
  border-radius: 4px;
  border: 1px solid #bae6fd;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Step History */
.step-history {
  margin-top: 0.5rem;
}

.history-title {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin: 0 0 0.5rem 0;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.5rem;
  background: var(--color-background);
  border-radius: 4px;
  font-size: 0.6875rem;
  cursor: pointer;
  transition: background 0.15s;
}

.history-item:hover {
  background: var(--color-surface);
}

.status-badge {
  font-size: 0.5625rem;
  font-weight: 600;
  text-transform: uppercase;
  padding: 0.125rem 0.375rem;
  border-radius: 3px;
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
  background: #f3f4f6;
  color: #6b7280;
}

.history-duration {
  color: var(--color-text-secondary);
}

.history-attempt {
  color: var(--color-text-secondary);
  margin-left: auto;
}

.history-arrow {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

/* Scrollbar */
.execution-content::-webkit-scrollbar {
  width: 6px;
}

.execution-content::-webkit-scrollbar-track {
  background: transparent;
}

.execution-content::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}
</style>
