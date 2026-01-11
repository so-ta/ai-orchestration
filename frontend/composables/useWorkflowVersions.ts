// Workflow Versions API composable
import type { WorkflowVersion, ApiResponse } from '~/types/api'

export function useWorkflowVersions() {
  const api = useApi()

  // List all versions of a workflow
  async function list(workflowId: string) {
    return api.get<ApiResponse<WorkflowVersion[]>>(`/workflows/${workflowId}/versions`)
  }

  // Get a specific version of a workflow
  async function get(workflowId: string, version: number) {
    return api.get<ApiResponse<WorkflowVersion>>(`/workflows/${workflowId}/versions/${version}`)
  }

  return {
    list,
    get,
  }
}
