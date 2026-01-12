import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock useRuntimeConfig
const mockConfig = {
  public: {
    apiBase: 'http://localhost:8080/api/v1',
  },
}

vi.mock('#app', () => ({
  useRuntimeConfig: () => mockConfig,
}))

// Mock useAuth
const mockAuth = {
  getToken: vi.fn(),
  isAuthenticated: { value: false },
  isDevMode: { value: true },
  devRole: { value: 'admin' },
}

vi.mock('../useAuth', () => ({
  useAuth: () => mockAuth,
}))

// Mock global fetch
const mockFetch = vi.fn()
global.fetch = mockFetch

import { useApi } from '../useApi'

describe('useApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockAuth.isAuthenticated.value = false
    mockAuth.isDevMode.value = true
    mockAuth.devRole.value = 'admin'
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('GET requests', () => {
    it('should make a GET request to the correct URL', async () => {
      const responseData = { data: { id: '123', name: 'Test' } }
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-length': '100' }),
        text: () => Promise.resolve(JSON.stringify(responseData)),
      })

      const api = useApi()
      const result = await api.get('/workflows')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/workflows',
        expect.objectContaining({
          method: 'GET',
          headers: expect.objectContaining({
            'Content-Type': 'application/json',
          }),
        })
      )
      expect(result).toEqual(responseData)
    })

    it('should include X-Tenant-ID header when not authenticated', async () => {
      mockAuth.isAuthenticated.value = false
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-length': '100' }),
        text: () => Promise.resolve('{"data": {}}'),
      })

      const api = useApi()
      await api.get('/workflows')

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'X-Tenant-ID': '00000000-0000-0000-0000-000000000001',
          }),
        })
      )
    })

    it('should include X-Dev-Role header in dev mode', async () => {
      mockAuth.isDevMode.value = true
      mockAuth.devRole.value = 'user'
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-length': '100' }),
        text: () => Promise.resolve('{"data": {}}'),
      })

      const api = useApi()
      await api.get('/workflows')

      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'X-Dev-Role': 'user',
          }),
        })
      )
    })

    it('should include Authorization header when authenticated', async () => {
      mockAuth.isAuthenticated.value = true
      mockAuth.getToken.mockResolvedValueOnce('test-jwt-token')
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-length': '100' }),
        text: () => Promise.resolve('{"data": {}}'),
      })

      const api = useApi()
      await api.get('/workflows')

      expect(mockAuth.getToken).toHaveBeenCalled()
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            Authorization: 'Bearer test-jwt-token',
          }),
        })
      )
    })
  })

  describe('POST requests', () => {
    it('should make a POST request with JSON body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 201,
        headers: new Headers({ 'content-length': '100' }),
        text: () => Promise.resolve('{"data": {"id": "new-123"}}'),
      })

      const api = useApi()
      const payload = { name: 'New Workflow' }
      const result = await api.post('/workflows', payload)

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/workflows',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify(payload),
        })
      )
      expect(result).toEqual({ data: { id: 'new-123' } })
    })

    it('should make a POST request without body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-length': '100' }),
        text: () => Promise.resolve('{"data": {}}'),
      })

      const api = useApi()
      await api.post('/workflows/123/publish')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/workflows/123/publish',
        expect.objectContaining({
          method: 'POST',
          body: undefined,
        })
      )
    })
  })

  describe('PUT requests', () => {
    it('should make a PUT request with JSON body', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-length': '100' }),
        text: () => Promise.resolve('{"data": {"id": "123"}}'),
      })

      const api = useApi()
      const payload = { name: 'Updated Workflow' }
      await api.put('/workflows/123', payload)

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/workflows/123',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify(payload),
        })
      )
    })
  })

  describe('DELETE requests', () => {
    it('should make a DELETE request', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
        headers: new Headers({ 'content-length': '0' }),
        text: () => Promise.resolve(''),
      })

      const api = useApi()
      const result = await api.delete('/workflows/123')

      expect(mockFetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/workflows/123',
        expect.objectContaining({
          method: 'DELETE',
        })
      )
      expect(result).toBeUndefined()
    })
  })

  describe('Error handling', () => {
    it('should throw error on non-ok response', async () => {
      const errorResponse = { error: { message: 'Workflow not found' } }
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: () => Promise.resolve(errorResponse),
      })

      const api = useApi()

      await expect(api.get('/workflows/invalid')).rejects.toThrow('Workflow not found')
    })

    it('should handle error response with no message', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.resolve({}),
      })

      const api = useApi()

      await expect(api.get('/workflows')).rejects.toThrow('Request failed')
    })

    it('should handle JSON parse error in error response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('Invalid JSON')),
      })

      const api = useApi()

      await expect(api.get('/workflows')).rejects.toThrow('Request failed')
    })
  })

  describe('Empty response handling', () => {
    it('should handle 204 No Content response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 204,
        headers: new Headers(),
        text: () => Promise.resolve(''),
      })

      const api = useApi()
      const result = await api.delete('/workflows/123')

      expect(result).toBeUndefined()
    })

    it('should handle response with content-length: 0', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-length': '0' }),
        text: () => Promise.resolve(''),
      })

      const api = useApi()
      const result = await api.get('/health')

      expect(result).toBeUndefined()
    })

    it('should handle empty text response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: new Headers({ 'content-length': '10' }),
        text: () => Promise.resolve(''),
      })

      const api = useApi()
      const result = await api.get('/health')

      expect(result).toBeUndefined()
    })
  })
})
