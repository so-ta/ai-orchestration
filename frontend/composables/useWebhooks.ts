import type { Webhook, CreateWebhookRequest, UpdateWebhookRequest, ApiResponse, PaginatedResponse } from '~/types/api'

export function useWebhooks() {
  const api = useApi()
  const config = useRuntimeConfig()
  const baseUrl = config.public.apiBase || 'http://localhost:8080'

  const webhooks = ref<Webhook[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const list = async (workflowId?: string): Promise<Webhook[]> => {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (workflowId) {
        params.append('workflow_id', workflowId)
      }
      const queryString = params.toString()
      const endpoint = `/webhooks${queryString ? `?${queryString}` : ''}`

      const result = await api.get<PaginatedResponse<Webhook>>(endpoint)
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
      const result = await api.get<ApiResponse<Webhook>>(`/webhooks/${id}`)
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
      const result = await api.post<ApiResponse<Webhook>>('/webhooks', data)
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
      const result = await api.put<ApiResponse<Webhook>>(`/webhooks/${id}`, data)
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
      await api.delete(`/webhooks/${id}`)
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
      const result = await api.post<ApiResponse<Webhook>>(`/webhooks/${id}/enable`)
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
      const result = await api.post<ApiResponse<Webhook>>(`/webhooks/${id}/disable`)
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
      const result = await api.post<ApiResponse<Webhook>>(`/webhooks/${id}/regenerate-secret`)
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
