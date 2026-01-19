import type {
  OAuth2Provider,
  OAuth2ProviderWithStatus,
  OAuth2App,
  OAuth2Connection,
  CreateOAuth2AppRequest,
  StartAuthorizationRequest,
  StartAuthorizationResponse,
  CredentialShare,
  ShareWithUserRequest,
  ShareWithProjectRequest,
  UpdateShareRequest,
} from '~/types/oauth2'
import { useListState } from './useAsyncState'

/**
 * OAuth2 Composable
 * 外部サービス認証機能のAPI操作を提供
 */
export function useOAuth2() {
  const api = useApi()

  // Providers state
  const {
    items: providers,
    loading: providersLoading,
    error: providersError,
    execute: executeProviders,
    setItems: setProviders,
  } = useListState<OAuth2ProviderWithStatus>()

  // Apps state
  const {
    items: apps,
    loading: appsLoading,
    error: appsError,
    execute: executeApps,
    setItems: setApps,
    addItem: addApp,
    removeItem: removeApp,
  } = useListState<OAuth2App>()

  // ============================================================================
  // Providers
  // ============================================================================

  /**
   * Fetch all OAuth2 providers
   */
  async function fetchProviders(): Promise<OAuth2ProviderWithStatus[]> {
    return executeProviders(async () => {
      const response = await api.get<OAuth2ProviderWithStatus[]>('/oauth2/providers')
      setProviders(response)
      return response
    }, 'Failed to fetch OAuth2 providers')
  }

  /**
   * Get a single provider by slug
   */
  async function getProvider(slug: string): Promise<OAuth2Provider> {
    return api.get<OAuth2Provider>(`/oauth2/providers/${slug}`)
  }

  // ============================================================================
  // Apps (Tenant's OAuth2 Client Configuration)
  // ============================================================================

  /**
   * Fetch all OAuth2 apps for current tenant
   */
  async function fetchApps(): Promise<OAuth2App[]> {
    return executeApps(async () => {
      const response = await api.get<OAuth2App[]>('/oauth2/apps')
      setApps(response)
      return response
    }, 'Failed to fetch OAuth2 apps')
  }

  /**
   * Create a new OAuth2 app
   */
  async function createApp(data: CreateOAuth2AppRequest): Promise<OAuth2App> {
    return executeApps(async () => {
      const response = await api.post<OAuth2App>('/oauth2/apps', data)
      addApp(response)
      return response
    }, 'Failed to create OAuth2 app')
  }

  /**
   * Get a single OAuth2 app
   */
  async function getApp(appId: string): Promise<OAuth2App> {
    return api.get<OAuth2App>(`/oauth2/apps/${appId}`)
  }

  /**
   * Delete an OAuth2 app
   */
  async function deleteApp(appId: string): Promise<void> {
    return executeApps(async () => {
      await api.delete(`/oauth2/apps/${appId}`)
      removeApp(a => a.id === appId)
    }, 'Failed to delete OAuth2 app')
  }

  // ============================================================================
  // Authorization Flow
  // ============================================================================

  /**
   * Start OAuth2 authorization flow
   * Returns authorization URL to redirect user to
   */
  async function startAuthorization(data: StartAuthorizationRequest): Promise<StartAuthorizationResponse> {
    return api.post<StartAuthorizationResponse>('/oauth2/authorize/start', data)
  }

  // ============================================================================
  // Connections
  // ============================================================================

  /**
   * Get OAuth2 connection by ID
   */
  async function getConnection(connectionId: string): Promise<OAuth2Connection> {
    return api.get<OAuth2Connection>(`/oauth2/connections/${connectionId}`)
  }

  /**
   * Get OAuth2 connection by credential ID
   */
  async function getConnectionByCredential(credentialId: string): Promise<OAuth2Connection> {
    return api.get<OAuth2Connection>(`/oauth2/connections/by-credential/${credentialId}`)
  }

  /**
   * Refresh OAuth2 connection tokens
   */
  async function refreshConnection(connectionId: string): Promise<OAuth2Connection> {
    return api.post<OAuth2Connection>(`/oauth2/connections/${connectionId}/refresh`)
  }

  /**
   * Revoke OAuth2 connection (invalidate tokens)
   */
  async function revokeConnection(connectionId: string): Promise<void> {
    return api.post(`/oauth2/connections/${connectionId}/revoke`)
  }

  /**
   * Delete OAuth2 connection
   */
  async function deleteConnection(connectionId: string): Promise<void> {
    return api.delete(`/oauth2/connections/${connectionId}`)
  }

  return {
    // Providers
    providers,
    providersLoading,
    providersError,
    fetchProviders,
    getProvider,

    // Apps
    apps,
    appsLoading,
    appsError,
    fetchApps,
    createApp,
    getApp,
    deleteApp,

    // Authorization
    startAuthorization,

    // Connections
    getConnection,
    getConnectionByCredential,
    refreshConnection,
    revokeConnection,
    deleteConnection,
  }
}

/**
 * Credential Shares Composable
 * 認証情報の共有機能のAPI操作を提供
 */
export function useCredentialShares(credentialId: Ref<string | undefined>) {
  const api = useApi()

  const {
    items: shares,
    loading,
    error,
    execute,
    setItems,
    addItem,
    updateItem,
    removeItem,
  } = useListState<CredentialShare>()

  /**
   * Fetch shares for a credential
   */
  async function fetchShares(): Promise<CredentialShare[]> {
    const id = credentialId.value
    if (!id) return []

    return execute(async () => {
      const response = await api.get<CredentialShare[]>(`/credentials/${id}/shares`)
      setItems(response)
      return response
    }, 'Failed to fetch credential shares')
  }

  /**
   * Share credential with a user
   */
  async function shareWithUser(data: ShareWithUserRequest): Promise<CredentialShare> {
    const id = credentialId.value
    if (!id) throw new Error('Credential ID is required')

    return execute(async () => {
      const response = await api.post<CredentialShare>(`/credentials/${id}/shares/user`, data)
      addItem(response)
      return response
    }, 'Failed to share credential with user')
  }

  /**
   * Share credential with a project
   */
  async function shareWithProject(data: ShareWithProjectRequest): Promise<CredentialShare> {
    const id = credentialId.value
    if (!id) throw new Error('Credential ID is required')

    return execute(async () => {
      const response = await api.post<CredentialShare>(`/credentials/${id}/shares/project`, data)
      addItem(response)
      return response
    }, 'Failed to share credential with project')
  }

  /**
   * Update share permissions
   */
  async function updateShare(shareId: string, data: UpdateShareRequest): Promise<CredentialShare> {
    const id = credentialId.value
    if (!id) throw new Error('Credential ID is required')

    return execute(async () => {
      const response = await api.put<CredentialShare>(`/credentials/${id}/shares/${shareId}`, data)
      updateItem(s => s.id === shareId, response)
      return response
    }, 'Failed to update share')
  }

  /**
   * Revoke (delete) share
   */
  async function revokeShare(shareId: string): Promise<void> {
    const id = credentialId.value
    if (!id) throw new Error('Credential ID is required')

    return execute(async () => {
      await api.delete(`/credentials/${id}/shares/${shareId}`)
      removeItem(s => s.id === shareId)
    }, 'Failed to revoke share')
  }

  // Auto-fetch when credential ID changes
  watch(credentialId, (newId) => {
    if (newId) {
      fetchShares()
    } else {
      setItems([])
    }
  }, { immediate: true })

  return {
    shares,
    loading,
    error,
    fetchShares,
    shareWithUser,
    shareWithProject,
    updateShare,
    revokeShare,
  }
}
