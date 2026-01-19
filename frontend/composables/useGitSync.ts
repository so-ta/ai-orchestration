// Git Sync management composable
import type { ProjectGitSync, GitSyncDirection } from '~/types/api'

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

  // Reactive state
  const gitSync = ref<ProjectGitSync | null>(null)
  const loading = ref(false)
  const syncing = ref(false)
  const error = ref<string | null>(null)

  // Fetch git sync by project ID
  async function fetchGitSync(projectId: string): Promise<ProjectGitSync | null> {
    loading.value = true
    error.value = null
    try {
      const result = await api.get<ProjectGitSync>(`/api/v1/workflows/${projectId}/git-sync`)
      gitSync.value = result
      return result
    } catch (e: unknown) {
      // 404 is expected when no git sync is configured
      if (e && typeof e === 'object' && 'status' in e && e.status === 404) {
        gitSync.value = null
        return null
      }
      error.value = e instanceof Error ? e.message : 'Failed to fetch git sync'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Configure git sync for a project (create)
  async function configureGitSync(
    projectId: string,
    input: Omit<CreateGitSyncInput, 'project_id'>
  ): Promise<ProjectGitSync> {
    loading.value = true
    error.value = null
    try {
      const result = await api.post<ProjectGitSync>(`/api/v1/workflows/${projectId}/git-sync`, input)
      gitSync.value = result
      return result
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to configure git sync'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Update git sync configuration
  async function updateGitSync(projectId: string, input: UpdateGitSyncInput): Promise<ProjectGitSync> {
    loading.value = true
    error.value = null
    try {
      const result = await api.put<ProjectGitSync>(`/api/v1/workflows/${projectId}/git-sync`, input)
      gitSync.value = result
      return result
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to update git sync'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Delete git sync configuration
  async function deleteGitSync(projectId: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await api.delete<void>(`/api/v1/workflows/${projectId}/git-sync`)
      gitSync.value = null
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Failed to delete git sync'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Trigger a sync operation
  async function triggerSync(projectId: string, operation: 'push' | 'pull' = 'push'): Promise<void> {
    syncing.value = true
    error.value = null
    try {
      await api.post<void>(`/api/v1/workflows/${projectId}/git-sync/sync`, { operation })
    } catch (e: unknown) {
      error.value = e instanceof Error ? e.message : 'Sync failed'
      throw e
    } finally {
      syncing.value = false
    }
  }

  // List all git sync configurations for the tenant
  async function listGitSyncs(): Promise<ProjectGitSync[]> {
    return api.get<ProjectGitSync[]>('/api/v1/git-sync')
  }

  return {
    // Reactive state
    gitSync,
    loading,
    syncing,
    error,
    // Methods
    fetchGitSync,
    configureGitSync,
    updateGitSync,
    deleteGitSync,
    triggerSync,
    listGitSyncs,
  }
}
