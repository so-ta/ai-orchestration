import type { Step, Run, StepRun } from '~/types/api'
import { usePolling } from '~/composables/usePolling'

/**
 * Result of a step test execution
 */
export interface StepTestResult {
  stepRun: StepRun
  run: Run
  executedAt: string
  duration: number
}

/**
 * Pinned output data structure
 */
export interface PinnedOutput {
  data: unknown
  pinnedAt: string
  stepId: string
  stepName: string
}

interface UseStepTestOptions {
  workflowId: string
  /** Callback when test runs list should be reloaded */
  onTestRunsChanged?: () => void
}

/**
 * Composable for step testing with inline result display
 * Unlike useWorkflowExecution, this does NOT navigate to Run detail panel
 */
export function useStepTest(options: UseStepTestOptions) {
  const { workflowId, onTestRunsChanged } = options

  const { t } = useI18n()
  const runsApi = useRuns()
  const toast = useToast()

  // Execution state
  const executing = ref(false)
  const currentResult = ref<StepTestResult | null>(null)
  const error = ref<string | null>(null)

  // Pinned data state
  const pinnedOutput = ref<PinnedOutput | null>(null)
  const currentStepId = ref<string | null>(null)

  // Polling for test result
  const { pollingId, start: startPollingInternal, stop: stopPolling } = usePolling<Run>({
    interval: 1000,
    maxAttempts: 60,
    onTimeout: () => {
      error.value = t('execution.pollingTimeout')
      executing.value = false
    },
  })

  /**
   * Get localStorage key for pinned output
   */
  function getPinKey(stepId: string): string {
    return `aio:pin:${workflowId}:${stepId}`
  }

  /**
   * Load pinned output from localStorage
   */
  function loadPinnedOutput(stepId: string): void {
    currentStepId.value = stepId
    const key = getPinKey(stepId)
    try {
      const stored = localStorage.getItem(key)
      if (stored) {
        pinnedOutput.value = JSON.parse(stored) as PinnedOutput
      } else {
        pinnedOutput.value = null
      }
    } catch {
      pinnedOutput.value = null
    }
  }

  /**
   * Start polling for step execution result
   */
  function startPolling(runId: string, stepId: string) {
    startPollingInternal(
      runId,
      async () => {
        const response = await runsApi.get(runId)
        return response.data
      },
      (run) => {
        const stepRun = run.step_runs?.find((sr: StepRun) => sr.step_id === stepId)
        if (stepRun) {
          if (stepRun.status === 'completed' || stepRun.status === 'failed') {
            // Store result locally (don't navigate to Run detail)
            currentResult.value = {
              stepRun,
              run,
              executedAt: new Date().toISOString(),
              duration: stepRun.duration_ms || 0,
            }
            executing.value = false

            if (stepRun.status === 'completed') {
              toast.success(t('execution.stepTestCompleted'))
            } else {
              toast.error(t('execution.stepTestFailed'))
              error.value = stepRun.error || null
            }

            // Notify parent to refresh test runs list
            onTestRunsChanged?.()

            return true // Stop polling
          }
        }
        return false // Continue polling
      }
    )
  }

  /**
   * Execute this step only (inline test)
   */
  async function executeStepOnly(step: Step, input: object): Promise<void> {
    // Clear previous state
    error.value = null
    executing.value = true
    // Don't clear currentResult - keep showing previous result while executing

    try {
      // Use inline test API (creates new test run)
      const response = await runsApi.testStepInline(
        workflowId,
        step.id,
        Object.keys(input).length > 0 ? input : undefined
      )

      // Start polling for result (result stored in currentResult, not navigated)
      startPolling(response.data.run_id, step.id)
    } catch (e) {
      toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
      error.value = e instanceof Error ? e.message : 'Unknown error'
      executing.value = false
    }
  }

  /**
   * Execute from this step (workflow execution starting from this step)
   */
  async function executeFromStep(step: Step, input: object): Promise<void> {
    error.value = null
    executing.value = true

    try {
      const response = await runsApi.create(workflowId, {
        triggered_by: 'test',
        input: Object.keys(input).length > 0 ? input : {},
        start_step_id: step.id,
      })

      // Start polling for this step's result
      startPolling(response.data.id, step.id)

      onTestRunsChanged?.()
    } catch (e) {
      toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
      error.value = e instanceof Error ? e.message : 'Unknown error'
      executing.value = false
    }
  }

  /**
   * Clear current result
   */
  function clearResult(): void {
    currentResult.value = null
    error.value = null
  }

  /**
   * Pin output data to localStorage
   */
  function pinOutput(output: unknown, stepId: string, stepName: string): void {
    const pinned: PinnedOutput = {
      data: output,
      pinnedAt: new Date().toISOString(),
      stepId,
      stepName,
    }
    const key = getPinKey(stepId)
    localStorage.setItem(key, JSON.stringify(pinned))
    pinnedOutput.value = pinned
    toast.success(t('test.pinData.pinned'))
  }

  /**
   * Unpin output data
   */
  function unpinOutput(stepId: string): void {
    const key = getPinKey(stepId)
    localStorage.removeItem(key)
    pinnedOutput.value = null
    toast.success(t('test.pinData.unpinned'))
  }

  /**
   * Edit pinned output data
   */
  function editPinnedOutput(newData: unknown, stepId: string): void {
    if (!pinnedOutput.value) return
    const pinned: PinnedOutput = {
      ...pinnedOutput.value,
      data: newData,
      pinnedAt: new Date().toISOString(),
    }
    const key = getPinKey(stepId)
    localStorage.setItem(key, JSON.stringify(pinned))
    pinnedOutput.value = pinned
  }

  /**
   * Cancel ongoing polling
   */
  function cancelExecution(): void {
    stopPolling()
    executing.value = false
  }

  return {
    // State (using computed to preserve reactivity without deep readonly issues)
    executing: computed(() => executing.value),
    currentResult: computed(() => currentResult.value),
    error: computed(() => error.value),
    pollingId,

    // Actions
    executeStepOnly,
    executeFromStep,
    clearResult,
    cancelExecution,
    loadPinnedOutput,

    // Pinned data
    pinnedOutput: computed(() => pinnedOutput.value),
    pinOutput,
    unpinOutput,
    editPinnedOutput,
  }
}
