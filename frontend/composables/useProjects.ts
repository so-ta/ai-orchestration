// Project API composable
import type { Project, Step, Edge, ProjectVersion, ApiResponse, PaginatedResponse } from '~/types/api'

export function useProjects() {
  const api = useApi()

  // List projects
  async function list(params?: { status?: string; page?: number; limit?: number }) {
    const query = new URLSearchParams()
    if (params?.status) query.set('status', params.status)
    if (params?.page) query.set('page', params.page.toString())
    if (params?.limit) query.set('limit', params.limit.toString())

    const queryString = query.toString()
    const endpoint = `/projects${queryString ? `?${queryString}` : ''}`

    return api.get<PaginatedResponse<Project>>(endpoint)
  }

  // Get project by ID
  async function get(id: string) {
    return api.get<ApiResponse<Project>>(`/projects/${id}`)
  }

  // Create project
  async function create(data: { name: string; description?: string; input_schema?: object }) {
    return api.post<ApiResponse<Project>>('/projects', data)
  }

  // Update project
  async function update(id: string, data: { name?: string; description?: string; input_schema?: object }) {
    return api.put<ApiResponse<Project>>(`/projects/${id}`, data)
  }

  // Delete project
  async function remove(id: string) {
    return api.delete(`/projects/${id}`)
  }

  // Save project (creates a new version)
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
      source_step_id?: string | null
      target_step_id?: string | null
      source_block_group_id?: string | null
      target_block_group_id?: string | null
      source_port?: string
      target_port?: string
      condition?: string
    }>
  }) {
    return api.post<ApiResponse<Project>>(`/projects/${id}/save`, data)
  }

  // Save project as draft (no version created)
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
      source_step_id?: string | null
      target_step_id?: string | null
      source_block_group_id?: string | null
      target_block_group_id?: string | null
      source_port?: string
      target_port?: string
      condition?: string
    }>
  }) {
    return api.post<ApiResponse<Project>>(`/projects/${id}/draft`, data)
  }

  // Discard draft
  async function discardDraft(id: string) {
    return api.delete<ApiResponse<Project>>(`/projects/${id}/draft`)
  }

  // Restore version
  async function restoreVersion(id: string, version: number) {
    return api.post<ApiResponse<Project>>(`/projects/${id}/restore`, { version })
  }

  // Steps
  async function listSteps(projectId: string) {
    return api.get<ApiResponse<Step[]>>(`/projects/${projectId}/steps`)
  }

  async function createStep(projectId: string, data: {
    name: string
    type: string
    config?: object
    position?: { x: number; y: number }
  }) {
    return api.post<ApiResponse<Step>>(`/projects/${projectId}/steps`, data)
  }

  async function updateStep(projectId: string, stepId: string, data: {
    name?: string
    type?: string
    config?: object
    position?: { x: number; y: number }
  }) {
    return api.put<ApiResponse<Step>>(`/projects/${projectId}/steps/${stepId}`, data)
  }

  async function deleteStep(projectId: string, stepId: string) {
    return api.delete(`/projects/${projectId}/steps/${stepId}`)
  }

  // Edges
  async function listEdges(projectId: string) {
    return api.get<ApiResponse<Edge[]>>(`/projects/${projectId}/edges`)
  }

  async function createEdge(projectId: string, data: {
    source_step_id?: string
    target_step_id?: string
    source_block_group_id?: string
    target_block_group_id?: string
    source_port?: string
    target_port?: string
    condition?: string
  }) {
    return api.post<ApiResponse<Edge>>(`/projects/${projectId}/edges`, data)
  }

  async function deleteEdge(projectId: string, edgeId: string) {
    return api.delete(`/projects/${projectId}/edges/${edgeId}`)
  }

  // Versions
  async function listVersions(projectId: string) {
    return api.get<ApiResponse<ProjectVersion[]>>(`/projects/${projectId}/versions`)
  }

  async function getVersion(projectId: string, version: number) {
    return api.get<ApiResponse<ProjectVersion>>(`/projects/${projectId}/versions/${version}`)
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
