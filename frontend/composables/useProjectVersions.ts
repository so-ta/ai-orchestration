// Project Versions API composable
import type { ProjectVersion, ApiResponse } from '~/types/api'

export function useProjectVersions() {
  const api = useApi()

  // List all versions of a project
  async function list(projectId: string) {
    return api.get<ApiResponse<ProjectVersion[]>>(`/workflows/${projectId}/versions`)
  }

  // Get a specific version of a project
  async function get(projectId: string, version: number) {
    return api.get<ApiResponse<ProjectVersion>>(`/workflows/${projectId}/versions/${version}`)
  }

  return {
    list,
    get,
  }
}
