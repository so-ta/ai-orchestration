// Workflow API composable
import type { Workflow, Step, Edge, WorkflowVersion, ApiResponse, PaginatedResponse } from '~/types/api'

export function useWorkflows() {
  const api = useApi()

  // List workflows
  async function list(params?: { status?: string; page?: number; limit?: number }) {
    const query = new URLSearchParams()
    if (params?.status) query.set('status', params.status)
    if (params?.page) query.set('page', params.page.toString())
    if (params?.limit) query.set('limit', params.limit.toString())

    const queryString = query.toString()
    const endpoint = `/workflows${queryString ? `?${queryString}` : ''}`

    return api.get<PaginatedResponse<Workflow>>(endpoint)
  }

  // Get workflow by ID
  async function get(id: string) {
    return api.get<ApiResponse<Workflow>>(`/workflows/${id}`)
  }

  // Create workflow
  async function create(data: { name: string; description?: string; input_schema?: object }) {
    return api.post<ApiResponse<Workflow>>('/workflows', data)
  }

  // Update workflow
  async function update(id: string, data: { name?: string; description?: string; input_schema?: object }) {
    return api.put<ApiResponse<Workflow>>(`/workflows/${id}`, data)
  }

  // Delete workflow
  async function remove(id: string) {
    return api.delete(`/workflows/${id}`)
  }

  // Save workflow (creates a new version)
  async function save(id: string, data: {
    name: string
    description?: string
    input_schema?: object
    steps: Array<{
      id: string
      name: string
      type: string
      config: object
      position_x: number
      position_y: number
    }>
    edges: Array<{
      id: string
      source_step_id: string
      target_step_id: string
      source_port?: string
      target_port?: string
      condition?: string
    }>
  }) {
    return api.post<ApiResponse<Workflow>>(`/workflows/${id}/save`, data)
  }

  // Save workflow as draft (no version created)
  async function saveDraft(id: string, data: {
    name: string
    description?: string
    input_schema?: object
    steps: Array<{
      id: string
      name: string
      type: string
      config: object
      position_x: number
      position_y: number
    }>
    edges: Array<{
      id: string
      source_step_id: string
      target_step_id: string
      source_port?: string
      target_port?: string
      condition?: string
    }>
  }) {
    return api.post<ApiResponse<Workflow>>(`/workflows/${id}/draft`, data)
  }

  // Discard draft
  async function discardDraft(id: string) {
    return api.delete<ApiResponse<Workflow>>(`/workflows/${id}/draft`)
  }

  // Restore version
  async function restoreVersion(id: string, version: number) {
    return api.post<ApiResponse<Workflow>>(`/workflows/${id}/restore`, { version })
  }

  // Publish workflow (deprecated - kept for backward compatibility)
  async function publish(id: string) {
    return api.post<ApiResponse<Workflow>>(`/workflows/${id}/publish`)
  }

  // Steps
  async function listSteps(workflowId: string) {
    return api.get<ApiResponse<Step[]>>(`/workflows/${workflowId}/steps`)
  }

  async function createStep(workflowId: string, data: {
    name: string
    type: string
    config?: object
    position?: { x: number; y: number }
  }) {
    return api.post<ApiResponse<Step>>(`/workflows/${workflowId}/steps`, data)
  }

  async function updateStep(workflowId: string, stepId: string, data: {
    name?: string
    type?: string
    config?: object
    position?: { x: number; y: number }
  }) {
    return api.put<ApiResponse<Step>>(`/workflows/${workflowId}/steps/${stepId}`, data)
  }

  async function deleteStep(workflowId: string, stepId: string) {
    return api.delete(`/workflows/${workflowId}/steps/${stepId}`)
  }

  // Edges
  async function listEdges(workflowId: string) {
    return api.get<ApiResponse<Edge[]>>(`/workflows/${workflowId}/edges`)
  }

  async function createEdge(workflowId: string, data: {
    source_step_id: string
    target_step_id: string
    source_port?: string
    target_port?: string
    condition?: string
  }) {
    return api.post<ApiResponse<Edge>>(`/workflows/${workflowId}/edges`, data)
  }

  async function deleteEdge(workflowId: string, edgeId: string) {
    return api.delete(`/workflows/${workflowId}/edges/${edgeId}`)
  }

  // Versions
  async function listVersions(workflowId: string) {
    return api.get<ApiResponse<WorkflowVersion[]>>(`/workflows/${workflowId}/versions`)
  }

  async function getVersion(workflowId: string, version: number) {
    return api.get<ApiResponse<WorkflowVersion>>(`/workflows/${workflowId}/versions/${version}`)
  }

  return {
    list,
    get,
    create,
    update,
    remove,
    save,
    saveDraft,
    discardDraft,
    restoreVersion,
    publish, // Deprecated
    listSteps,
    createStep,
    updateStep,
    deleteStep,
    listEdges,
    createEdge,
    deleteEdge,
    listVersions,
    getVersion,
  }
}
