// Run API composable
import type { Run, ApiResponse, PaginatedResponse } from '~/types/api'

export function useRuns() {
  const api = useApi()

  // List runs for a workflow
  async function list(workflowId: string, params?: { page?: number; limit?: number }) {
    const query = new URLSearchParams()
    if (params?.page) query.set('page', params.page.toString())
    if (params?.limit) query.set('limit', params.limit.toString())

    const queryString = query.toString()
    const endpoint = `/workflows/${workflowId}/runs${queryString ? `?${queryString}` : ''}`

    return api.get<PaginatedResponse<Run>>(endpoint)
  }

  // Get run by ID
  async function get(runId: string) {
    return api.get<ApiResponse<Run>>(`/runs/${runId}`)
  }

  // Create run (execute workflow)
  // version: 0 or omitted means latest version
  async function create(workflowId: string, data: { input?: object; mode?: 'test' | 'production'; version?: number }) {
    return api.post<ApiResponse<Run>>(`/workflows/${workflowId}/runs`, data)
  }

  // Cancel run
  async function cancel(runId: string) {
    return api.post<ApiResponse<Run>>(`/runs/${runId}/cancel`)
  }

  return {
    list,
    get,
    create,
    cancel,
  }
}
