import type { Webhook, CreateWebhookRequest, UpdateWebhookRequest, ApiResponse, PaginatedResponse } from '~/types/api'

export function useWebhooks() {
  const config = useRuntimeConfig()
  const baseUrl = config.public.apiBase || 'http://localhost:8080'

  const webhooks = ref<Webhook[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const getHeaders = () => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    // Add tenant ID header for development (client-side only)
    if (import.meta.client) {
      const tenantId = localStorage.getItem('tenant_id') || '00000000-0000-0000-0000-000000000001'
      headers['X-Tenant-ID'] = tenantId

      // Add auth token if available
      const token = localStorage.getItem('auth_token')
      if (token) {
        headers['Authorization'] = `Bearer ${token}`
      }
    } else {
      headers['X-Tenant-ID'] = '00000000-0000-0000-0000-000000000001'
    }

    return headers
  }

  const list = async (workflowId?: string): Promise<Webhook[]> => {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (workflowId) {
        params.append('workflow_id', workflowId)
      }
      const queryString = params.toString()
      const url = `${baseUrl}/webhooks${queryString ? `?${queryString}` : ''}`

      const response = await fetch(url, {
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to fetch webhooks: ${response.statusText}`)
      }
      const result: PaginatedResponse<Webhook> = await response.json()
      webhooks.value = result.data || []
      return result.data || []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const get = async (id: string): Promise<Webhook> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/webhooks/${id}`, {
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to fetch webhook: ${response.statusText}`)
      }
      const result: ApiResponse<Webhook> = await response.json()
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const create = async (data: CreateWebhookRequest): Promise<Webhook> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/webhooks`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        throw new Error(`Failed to create webhook: ${response.statusText}`)
      }
      const result: ApiResponse<Webhook> = await response.json()
      webhooks.value.push(result.data)
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const update = async (id: string, data: UpdateWebhookRequest): Promise<Webhook> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/webhooks/${id}`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        throw new Error(`Failed to update webhook: ${response.statusText}`)
      }
      const result: ApiResponse<Webhook> = await response.json()
      const index = webhooks.value.findIndex(w => w.id === id)
      if (index !== -1) {
        webhooks.value[index] = result.data
      }
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const remove = async (id: string): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/webhooks/${id}`, {
        method: 'DELETE',
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to delete webhook: ${response.statusText}`)
      }
      webhooks.value = webhooks.value.filter(w => w.id !== id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const enable = async (id: string): Promise<Webhook> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/webhooks/${id}/enable`, {
        method: 'POST',
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to enable webhook: ${response.statusText}`)
      }
      const result: ApiResponse<Webhook> = await response.json()
      const index = webhooks.value.findIndex(w => w.id === id)
      if (index !== -1) {
        webhooks.value[index] = result.data
      }
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const disable = async (id: string): Promise<Webhook> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/webhooks/${id}/disable`, {
        method: 'POST',
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to disable webhook: ${response.statusText}`)
      }
      const result: ApiResponse<Webhook> = await response.json()
      const index = webhooks.value.findIndex(w => w.id === id)
      if (index !== -1) {
        webhooks.value[index] = result.data
      }
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const regenerateSecret = async (id: string): Promise<Webhook> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/webhooks/${id}/regenerate-secret`, {
        method: 'POST',
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to regenerate webhook secret: ${response.statusText}`)
      }
      const result: ApiResponse<Webhook> = await response.json()
      const index = webhooks.value.findIndex(w => w.id === id)
      if (index !== -1) {
        webhooks.value[index] = result.data
      }
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Get the webhook URL for triggering
  const getWebhookUrl = (webhook: Webhook): string => {
    return `${baseUrl}/webhooks/${webhook.id}/trigger`
  }

  return {
    webhooks,
    loading,
    error,
    list,
    get,
    create,
    update,
    remove,
    enable,
    disable,
    regenerateSecret,
    getWebhookUrl,
  }
}
