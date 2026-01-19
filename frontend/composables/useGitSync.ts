// Git Sync management composable
import type { ProjectGitSync, GitSyncDirection, GitSyncOperation } from '~/types/api'

export interface CreateGitSyncInput {
  project_id: string
  repository_url: string
  branch?: string
  file_path?: string
  sync_direction?: GitSyncDirection
  auto_sync?: boolean
  credentials_id?: string
}

export interface UpdateGitSyncInput {
  repository_url?: string
  branch?: string
  file_path?: string
  sync_direction?: GitSyncDirection
  auto_sync?: boolean
  credentials_id?: string
}

export function useGitSync() {
  const api = useApi()

  // Create git sync configuration for a project
  async function createGitSync(input: CreateGitSyncInput): Promise<ProjectGitSync> {
    return api.post<ProjectGitSync>('/api/v1/git-sync', input)
  }

  // Get git sync by ID
  async function getGitSync(id: string): Promise<ProjectGitSync> {
    return api.get<ProjectGitSync>(`/api/v1/git-sync/${id}`)
  }

  // Get git sync by project ID
  async function getGitSyncByProject(projectId: string): Promise<ProjectGitSync> {
    return api.get<ProjectGitSync>(`/api/v1/workflows/${projectId}/git-sync`)
  }

  // List all git sync configurations for the tenant
  async function listGitSyncs(): Promise<ProjectGitSync[]> {
    return api.get<ProjectGitSync[]>('/api/v1/git-sync')
  }

  // Update git sync configuration
  async function updateGitSync(id: string, input: UpdateGitSyncInput): Promise<ProjectGitSync> {
    return api.put<ProjectGitSync>(`/api/v1/git-sync/${id}`, input)
  }

  // Delete git sync configuration
  async function deleteGitSync(id: string): Promise<void> {
    return api.delete<void>(`/api/v1/git-sync/${id}`)
  }

  // Trigger a sync operation
  async function triggerSync(id: string, operation: 'push' | 'pull'): Promise<GitSyncOperation> {
    return api.post<GitSyncOperation>(`/api/v1/git-sync/${id}/sync`, { operation })
  }

  // Create git sync directly from project view
  async function createProjectGitSync(
    projectId: string,
    input: Omit<CreateGitSyncInput, 'project_id'>
  ): Promise<ProjectGitSync> {
    return api.post<ProjectGitSync>(`/api/v1/workflows/${projectId}/git-sync`, input)
  }

  // Update git sync directly from project view
  async function updateProjectGitSync(
    projectId: string,
    input: UpdateGitSyncInput
  ): Promise<ProjectGitSync> {
    return api.put<ProjectGitSync>(`/api/v1/workflows/${projectId}/git-sync`, input)
  }

  // Delete git sync directly from project view
  async function deleteProjectGitSync(projectId: string): Promise<void> {
    return api.delete<void>(`/api/v1/workflows/${projectId}/git-sync`)
  }

  // Trigger sync directly from project view
  async function triggerProjectSync(projectId: string, operation: 'push' | 'pull'): Promise<GitSyncOperation> {
    return api.post<GitSyncOperation>(`/api/v1/workflows/${projectId}/git-sync/sync`, { operation })
  }

  return {
    createGitSync,
    getGitSync,
    getGitSyncByProject,
    listGitSyncs,
    updateGitSync,
    deleteGitSync,
    triggerSync,
    createProjectGitSync,
    updateProjectGitSync,
    deleteProjectGitSync,
    triggerProjectSync,
  }
}
