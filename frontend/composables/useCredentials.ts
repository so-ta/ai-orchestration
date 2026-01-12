import type {
  Credential,
  CreateCredentialRequest,
  UpdateCredentialRequest,
  PaginatedResponse,
  CredentialType,
  CredentialStatus,
} from '~/types/api'

interface CredentialFilters {
  credential_type?: CredentialType
  status?: CredentialStatus
  page?: number
  limit?: number
}

export function useCredentials() {
  const api = useApi()

  const credentials = ref<Credential[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const pagination = ref<{ page: number; limit: number; total: number }>({
    page: 1,
    limit: 20,
    total: 0,
  })

  async function fetchCredentials(filters: CredentialFilters = {}) {
    loading.value = true
    error.value = null

    try {
      const params = new URLSearchParams()
      if (filters.credential_type) params.set('credential_type', filters.credential_type)
      if (filters.status) params.set('status', filters.status)
      if (filters.page) params.set('page', String(filters.page))
      if (filters.limit) params.set('limit', String(filters.limit))

      const queryString = params.toString()
      const endpoint = `/credentials${queryString ? `?${queryString}` : ''}`

      const response = await api.get<PaginatedResponse<Credential>>(endpoint)
      credentials.value = response.data
      pagination.value = {
        page: response.meta.page,
        limit: response.meta.limit,
        total: response.meta.total,
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch credentials'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function getCredential(id: string): Promise<Credential> {
    loading.value = true
    error.value = null

    try {
      const response = await api.get<Credential>(`/credentials/${id}`)
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch credential'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createCredential(data: CreateCredentialRequest): Promise<Credential> {
    loading.value = true
    error.value = null

    try {
      const response = await api.post<Credential>('/credentials', data)
      // Add to local list
      credentials.value = [response, ...credentials.value]
      pagination.value.total += 1
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create credential'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateCredential(id: string, data: UpdateCredentialRequest): Promise<Credential> {
    loading.value = true
    error.value = null

    try {
      const response = await api.put<Credential>(`/credentials/${id}`, data)
      // Update local list
      const index = credentials.value.findIndex(c => c.id === id)
      if (index !== -1) {
        credentials.value[index] = response
      }
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update credential'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteCredential(id: string): Promise<void> {
    loading.value = true
    error.value = null

    try {
      await api.delete(`/credentials/${id}`)
      // Remove from local list
      credentials.value = credentials.value.filter(c => c.id !== id)
      pagination.value.total -= 1
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete credential'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function revokeCredential(id: string): Promise<Credential> {
    loading.value = true
    error.value = null

    try {
      const response = await api.post<Credential>(`/credentials/${id}/revoke`)
      // Update local list
      const index = credentials.value.findIndex(c => c.id === id)
      if (index !== -1) {
        credentials.value[index] = response
      }
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to revoke credential'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function activateCredential(id: string): Promise<Credential> {
    loading.value = true
    error.value = null

    try {
      const response = await api.post<Credential>(`/credentials/${id}/activate`)
      // Update local list
      const index = credentials.value.findIndex(c => c.id === id)
      if (index !== -1) {
        credentials.value[index] = response
      }
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to activate credential'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    credentials,
    loading,
    error,
    pagination,
    fetchCredentials,
    getCredential,
    createCredential,
    updateCredential,
    deleteCredential,
    revokeCredential,
    activateCredential,
  }
}
