import type { Step, Run } from '~/types/api'
import { usePolling } from '~/composables/usePolling'

interface WorkflowExecutionOptions {
  workflowId: string
  onRunCreated: (run: Run) => void
  onTestRunsLoaded: (runs: Run[]) => void
}

/**
 * Composable for workflow and step execution with polling
 */
export function useWorkflowExecution(options: WorkflowExecutionOptions) {
  const { workflowId, onRunCreated, onTestRunsLoaded } = options

  const { t } = useI18n()
  const runsApi = useRuns()
  const toast = useToast()

  const executing = ref(false)
  const loadingTestRuns = ref(false)
  const testRuns = ref<Run[]>([])

  // Poll for inline test result using composable
  const { pollingId: pollingRunId, start: startPollingInternal } = usePolling<Run>({
    interval: 1000,
    maxAttempts: 60,
    onTimeout: () => toast.warning(t('execution.pollingTimeout')),
  })

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
        // Always emit the updated run to keep Run Detail Panel in sync
        onRunCreated(run)

        // Check if step has completed
        const stepRun = run.step_runs?.find((sr: { step_id: string }) => sr.step_id === stepId)
        if (stepRun) {
          if (stepRun.status === 'completed' || stepRun.status === 'failed') {
            loadTestRuns()
            return true // Stop polling
          }
        }
        return false // Continue polling
      }
    )
  }

  /**
   * Load workflow test runs (all test triggered runs)
   */
  async function loadTestRuns() {
    loadingTestRuns.value = true
    try {
      // Get all runs without limit (or large limit)
      const response = await runsApi.list(workflowId, { limit: 1000 })
      const allRuns = response.data || []

      // Filter to test triggered runs only and fetch detailed data
      const testModeRuns = allRuns.filter(run => run.triggered_by === 'test')

      // Fetch detailed run data with step_runs for each run
      const detailedRuns: Run[] = []
      for (const run of testModeRuns) {
        try {
          const detailedResponse = await runsApi.get(run.id)
          detailedRuns.push(detailedResponse.data)
        } catch {
          // If detail fetch fails, use the basic run info
          detailedRuns.push(run)
        }
      }

      testRuns.value = detailedRuns
      onTestRunsLoaded(detailedRuns)
    } catch (e) {
      console.error('Failed to load test runs:', e)
    } finally {
      loadingTestRuns.value = false
    }
  }

  /**
   * Execute workflow (test mode)
   */
  async function executeWorkflow(input: object, startStepId: string) {
    executing.value = true

    try {
      const response = await runsApi.create(workflowId, {
        triggered_by: 'test',
        input: Object.keys(input).length > 0 ? input : {},
        start_step_id: startStepId,
      })

      // Fetch full run details and emit event
      const detailedRun = await runsApi.get(response.data.id)
      onRunCreated(detailedRun.data)

      // Reload test runs after execution
      await loadTestRuns()
      return true
    } catch (e) {
      toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
      return false
    } finally {
      executing.value = false
    }
  }

  /**
   * Execute this step only
   */
  async function executeThisStepOnly(step: Step, input: object, latestRun: Run | null) {
    executing.value = true

    try {
      let runId: string

      if (latestRun) {
        // Use existing run for re-execution
        await runsApi.executeSingleStep(
          latestRun.id,
          step.id,
          Object.keys(input).length > 0 ? input : undefined
        )
        runId = latestRun.id
      } else {
        // Use inline test API (creates new test run)
        const response = await runsApi.testStepInline(
          workflowId,
          step.id,
          Object.keys(input).length > 0 ? input : undefined
        )
        runId = response.data.run_id

        // Start polling for result
        startPolling(response.data.run_id, step.id)
      }

      // Fetch full run details and emit event to transition to Run detail panel
      const detailedRun = await runsApi.get(runId)
      onRunCreated(detailedRun.data)

      await loadTestRuns()
      return true
    } catch (e) {
      toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
      return false
    } finally {
      executing.value = false
    }
  }

  /**
   * Execute from this step (run workflow starting from this step)
   */
  async function executeFromThisStep(step: Step, input: object) {
    executing.value = true

    try {
      const response = await runsApi.create(workflowId, {
        triggered_by: 'test',
        input: Object.keys(input).length > 0 ? input : {},
        start_step_id: step.id,
      })

      // Fetch full run details and emit event
      const detailedRun = await runsApi.get(response.data.id)
      onRunCreated(detailedRun.data)

      // Reload test runs after execution
      await loadTestRuns()
      return true
    } catch (e) {
      toast.error(t('execution.errors.executionFailed'), e instanceof Error ? e.message : undefined)
      return false
    } finally {
      executing.value = false
    }
  }

  return {
    executing: readonly(executing),
    loadingTestRuns: readonly(loadingTestRuns),
    testRuns,
    pollingRunId,
    loadTestRuns,
    executeWorkflow,
    executeThisStepOnly,
    executeFromThisStep,
  }
}
