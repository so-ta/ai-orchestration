// Run API composable
import type { Run, StepRun, ApiResponse, PaginatedResponse, TriggerType } from '~/types/api'

// Response types for step re-execution
interface ExecuteSingleStepResponse {
  data: StepRun
}

interface ResumeFromStepResponse {
  data: {
    run_id: string
    from_step_id: string
    steps_to_execute: string[]
  }
}

interface StepHistoryResponse {
  data: StepRun[]
}

// Response type for inline step testing
interface TestStepInlineResponse {
  data: {
    run_id: string
    step_id: string
    step_name: string
    is_queued: boolean
  }
}

export function useRuns() {
  const api = useApi()

  // List runs for a project
  async function list(projectId: string, params?: { page?: number; limit?: number }) {
    const query = new URLSearchParams()
    if (params?.page) query.set('page', params.page.toString())
    if (params?.limit) query.set('limit', params.limit.toString())

    const queryString = query.toString()
    const endpoint = `/projects/${projectId}/runs${queryString ? `?${queryString}` : ''}`

    return api.get<PaginatedResponse<Run>>(endpoint)
  }

  // Get run by ID
  async function get(runId: string) {
    return api.get<ApiResponse<Run>>(`/runs/${runId}`)
  }

  // Create run (execute project)
  // version: 0 or omitted means latest version
  // triggered_by: trigger type (manual, test, etc.)
  async function create(projectId: string, data: { input?: object; triggered_by?: TriggerType; version?: number }) {
    return api.post<ApiResponse<Run>>(`/projects/${projectId}/runs`, data)
  }

  // Cancel run
  async function cancel(runId: string) {
    return api.post<ApiResponse<Run>>(`/runs/${runId}/cancel`)
  }

  // Execute a single step (re-execute)
  async function executeSingleStep(runId: string, stepId: string, input?: object) {
    return api.post<ExecuteSingleStepResponse>(`/runs/${runId}/steps/${stepId}/execute`, { input })
  }

  // Resume execution from a step (re-execute from here)
  async function resumeFromStep(runId: string, fromStepId: string, inputOverride?: object) {
    return api.post<ResumeFromStepResponse>(`/runs/${runId}/resume`, {
      from_step_id: fromStepId,
      input_override: inputOverride
    })
  }

  // Get execution history for a step
  async function getStepHistory(runId: string, stepId: string) {
    return api.get<StepHistoryResponse>(`/runs/${runId}/steps/${stepId}/history`)
  }

  // Test a single step inline (without requiring an existing run)
  // Creates a new test run and executes only the specified step
  async function testStepInline(projectId: string, stepId: string, input?: object) {
    return api.post<TestStepInlineResponse>(`/projects/${projectId}/steps/${stepId}/test`, { input })
  }

  return {
    list,
    get,
    create,
    cancel,
    executeSingleStep,
    resumeFromStep,
    getStepHistory,
    testStepInline,
  }
}
