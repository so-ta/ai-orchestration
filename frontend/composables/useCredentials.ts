import type {
  Credential,
  CreateCredentialRequest,
  UpdateCredentialRequest,
  PaginatedResponse,
  CredentialType,
  CredentialStatus,
} from '~/types/api'
import { useListState } from './useAsyncState'

interface CredentialFilters {
  credential_type?: CredentialType
  status?: CredentialStatus
  page?: number
  limit?: number
}

export function useCredentials() {
  const api = useApi()

  const {
    items: credentials,
    loading,
    error,
    pagination,
    execute,
    setItems,
    addItem,
    updateItem,
    removeItem,
    setPagination,
  } = useListState<Credential>()

  async function fetchCredentials(filters: CredentialFilters = {}) {
    return execute(async () => {
      const params = new URLSearchParams()
      if (filters.credential_type) params.set('credential_type', filters.credential_type)
      if (filters.status) params.set('status', filters.status)
      if (filters.page) params.set('page', String(filters.page))
      if (filters.limit) params.set('limit', String(filters.limit))

      const queryString = params.toString()
      const endpoint = `/credentials${queryString ? `?${queryString}` : ''}`

      const response = await api.get<PaginatedResponse<Credential>>(endpoint)
      setItems(response.data)
      setPagination({
        page: response.meta.page,
        limit: response.meta.limit,
        total: response.meta.total,
      })
    }, 'Failed to fetch credentials')
  }

  async function getCredential(id: string): Promise<Credential> {
    return execute(async () => {
      const response = await api.get<Credential>(`/credentials/${id}`)
      return response
    }, 'Failed to fetch credential')
  }

  async function createCredential(data: CreateCredentialRequest): Promise<Credential> {
    return execute(async () => {
      const response = await api.post<Credential>('/credentials', data)
      addItem(response)
      return response
    }, 'Failed to create credential')
  }

  async function updateCredential(id: string, data: UpdateCredentialRequest): Promise<Credential> {
    return execute(async () => {
      const response = await api.put<Credential>(`/credentials/${id}`, data)
      updateItem(c => c.id === id, response)
      return response
    }, 'Failed to update credential')
  }

  async function deleteCredential(id: string): Promise<void> {
    return execute(async () => {
      await api.delete(`/credentials/${id}`)
      removeItem(c => c.id === id)
    }, 'Failed to delete credential')
  }

  async function revokeCredential(id: string): Promise<Credential> {
    return execute(async () => {
      const response = await api.post<Credential>(`/credentials/${id}/revoke`)
      updateItem(c => c.id === id, response)
      return response
    }, 'Failed to revoke credential')
  }

  async function activateCredential(id: string): Promise<Credential> {
    return execute(async () => {
      const response = await api.post<Credential>(`/credentials/${id}/activate`)
      updateItem(c => c.id === id, response)
      return response
    }, 'Failed to activate credential')
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
