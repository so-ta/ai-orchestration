// API client composable with Keycloak authentication
export function useApi() {
  const config = useRuntimeConfig()
  const baseURL = config.public.apiBase
  const { getToken, isAuthenticated, isDevMode, devRole } = useAuth()

  async function request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${baseURL}${endpoint}`

    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...options.headers,
    }

    // Add auth token if authenticated
    if (isAuthenticated.value) {
      const token = await getToken()
      if (token) {
        ;(headers as Record<string, string>)['Authorization'] = `Bearer ${token}`
      }
    } else {
      // Development fallback: use default tenant ID
      ;(headers as Record<string, string>)['X-Tenant-ID'] = '00000000-0000-0000-0000-000000000001'
    }

    // In dev mode, send the dev role header
    if (isDevMode.value) {
      ;(headers as Record<string, string>)['X-Dev-Role'] = devRole.value
    }

    const response = await fetch(url, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: { message: 'Request failed' } }))
      throw new Error(error.error?.message || 'Request failed')
    }

    // Handle empty responses (e.g., 204 No Content or DELETE requests)
    const contentLength = response.headers.get('content-length')
    if (response.status === 204 || contentLength === '0') {
      return undefined as T
    }

    // Try to parse JSON, return undefined if empty
    const text = await response.text()
    if (!text) {
      return undefined as T
    }

    return JSON.parse(text)
  }

  return {
    get: <T>(endpoint: string) => request<T>(endpoint, { method: 'GET' }),

    post: <T>(endpoint: string, data?: unknown) =>
      request<T>(endpoint, {
        method: 'POST',
        body: data ? JSON.stringify(data) : undefined,
      }),

    put: <T>(endpoint: string, data?: unknown) =>
      request<T>(endpoint, {
        method: 'PUT',
        body: data ? JSON.stringify(data) : undefined,
      }),

    delete: <T>(endpoint: string) => request<T>(endpoint, { method: 'DELETE' }),
  }
}
