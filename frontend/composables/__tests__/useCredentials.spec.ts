import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref } from 'vue'
import type { Credential, PaginatedResponse, CredentialMetadata } from '~/types/api'

// Mock useApi composable
const mockApi = {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
}

vi.mock('~/composables/useApi', () => ({
  useApi: () => mockApi,
}))

// Mock state refs
const mockItems = ref<Credential[]>([])
const mockLoading = ref(false)
const mockError = ref<Error | null>(null)
const mockPagination = ref({ page: 1, limit: 20, total: 0 })

vi.mock('~/composables/useAsyncState', () => ({
  useListState: () => ({
    items: mockItems,
    loading: mockLoading,
    error: mockError,
    pagination: mockPagination,
    execute: async <T>(fn: () => Promise<T>, _errMsg: string) => fn(),
    setItems: (items: Credential[]) => {
      mockItems.value = items
    },
    addItem: (item: Credential) => {
      mockItems.value = [...mockItems.value, item]
    },
    updateItem: (predicate: (c: Credential) => boolean, item: Credential) => {
      const index = mockItems.value.findIndex(predicate)
      if (index !== -1) {
        mockItems.value[index] = item
      }
    },
    removeItem: (predicate: (c: Credential) => boolean) => {
      mockItems.value = mockItems.value.filter(c => !predicate(c))
    },
    setPagination: (p: { page: number; limit: number; total: number }) => {
      mockPagination.value = p
    },
  }),
}))

describe('useCredentials', () => {
  const mockMetadata: CredentialMetadata = {
    provider: 'openai',
    environment: 'production',
  }

  const mockCredential: Credential = {
    id: 'cred-1',
    tenant_id: 'tenant-1',
    name: 'Test API Key',
    credential_type: 'api_key',
    status: 'active',
    metadata: mockMetadata,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }

  const mockPaginatedResponse: PaginatedResponse<Credential> = {
    data: [mockCredential],
    meta: { page: 1, limit: 20, total: 1 },
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockItems.value = []
    mockLoading.value = false
    mockError.value = null
    mockPagination.value = { page: 1, limit: 20, total: 0 }
  })

  describe('fetchCredentials', () => {
    it('should fetch credentials successfully', async () => {
      mockApi.get.mockResolvedValueOnce(mockPaginatedResponse)

      const { useCredentials } = await import('../useCredentials')
      const { fetchCredentials, credentials } = useCredentials()

      await fetchCredentials()

      expect(mockApi.get).toHaveBeenCalledWith('/credentials')
      expect(credentials.value).toEqual([mockCredential])
    })

    it('should fetch credentials with filters', async () => {
      mockApi.get.mockResolvedValueOnce(mockPaginatedResponse)

      const { useCredentials } = await import('../useCredentials')
      const { fetchCredentials } = useCredentials()

      await fetchCredentials({
        credential_type: 'api_key',
        status: 'active',
        page: 2,
        limit: 10,
      })

      expect(mockApi.get).toHaveBeenCalledWith(
        '/credentials?credential_type=api_key&status=active&page=2&limit=10'
      )
    })
  })

  describe('getCredential', () => {
    it('should fetch a single credential by id', async () => {
      mockApi.get.mockResolvedValueOnce(mockCredential)

      const { useCredentials } = await import('../useCredentials')
      const { getCredential } = useCredentials()

      const result = await getCredential('cred-1')

      expect(mockApi.get).toHaveBeenCalledWith('/credentials/cred-1')
      expect(result).toEqual(mockCredential)
    })
  })

  describe('createCredential', () => {
    it('should create a credential', async () => {
      mockApi.post.mockResolvedValueOnce(mockCredential)

      const { useCredentials } = await import('../useCredentials')
      const { createCredential, credentials } = useCredentials()

      const result = await createCredential({
        name: 'Test API Key',
        credential_type: 'api_key',
        data: { api_key: 'test-key' },
      })

      expect(mockApi.post).toHaveBeenCalledWith('/credentials', {
        name: 'Test API Key',
        credential_type: 'api_key',
        data: { api_key: 'test-key' },
      })
      expect(result).toEqual(mockCredential)
      expect(credentials.value.some(c => c.id === mockCredential.id)).toBe(true)
    })
  })

  describe('updateCredential', () => {
    it('should update a credential', async () => {
      const updatedCredential = { ...mockCredential, name: 'Updated API Key' }
      mockApi.put.mockResolvedValueOnce(updatedCredential)
      mockItems.value = [mockCredential]

      const { useCredentials } = await import('../useCredentials')
      const { updateCredential, credentials } = useCredentials()

      const result = await updateCredential('cred-1', { name: 'Updated API Key' })

      expect(mockApi.put).toHaveBeenCalledWith('/credentials/cred-1', { name: 'Updated API Key' })
      expect(result).toEqual(updatedCredential)
      expect(credentials.value[0].name).toBe('Updated API Key')
    })
  })

  describe('deleteCredential', () => {
    it('should delete a credential', async () => {
      mockApi.delete.mockResolvedValueOnce(undefined)
      mockItems.value = [mockCredential]

      const { useCredentials } = await import('../useCredentials')
      const { deleteCredential, credentials } = useCredentials()

      await deleteCredential('cred-1')

      expect(mockApi.delete).toHaveBeenCalledWith('/credentials/cred-1')
      expect(credentials.value).not.toContain(mockCredential)
    })
  })

  describe('revokeCredential', () => {
    it('should revoke a credential', async () => {
      const revokedCredential = { ...mockCredential, status: 'revoked' as const }
      mockApi.post.mockResolvedValueOnce(revokedCredential)
      mockItems.value = [mockCredential]

      const { useCredentials } = await import('../useCredentials')
      const { revokeCredential, credentials } = useCredentials()

      const result = await revokeCredential('cred-1')

      expect(mockApi.post).toHaveBeenCalledWith('/credentials/cred-1/revoke')
      expect(result.status).toBe('revoked')
      expect(credentials.value[0].status).toBe('revoked')
    })
  })

  describe('activateCredential', () => {
    it('should activate a credential', async () => {
      const revokedCredential = { ...mockCredential, status: 'revoked' as const }
      const activatedCredential = { ...mockCredential, status: 'active' as const }
      mockApi.post.mockResolvedValueOnce(activatedCredential)
      mockItems.value = [revokedCredential]

      const { useCredentials } = await import('../useCredentials')
      const { activateCredential, credentials } = useCredentials()

      const result = await activateCredential('cred-1')

      expect(mockApi.post).toHaveBeenCalledWith('/credentials/cred-1/activate')
      expect(result.status).toBe('active')
      expect(credentials.value[0].status).toBe('active')
    })
  })
})
