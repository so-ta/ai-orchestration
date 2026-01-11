// API client composable with Keycloak authentication
export function useApi() {
  const config = useRuntimeConfig()
  const baseURL = config.public.apiBase
  const { getToken, isAuthenticated } = useAuth()

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

    const response = await fetch(url, {
      ...options,
      headers,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: { message: 'Request failed' } }))
      throw new Error(error.error?.message || 'Request failed')
    }

    return response.json()
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
