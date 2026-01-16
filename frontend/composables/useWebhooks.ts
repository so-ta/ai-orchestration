/**
 * @deprecated Webhooks are now configured in Start block trigger_config.
 * This composable is kept for backward compatibility but should not be used for new implementations.
 * Use the Start block with trigger_type: 'webhook' instead.
 */
import type { Webhook, CreateWebhookRequest, UpdateWebhookRequest, ApiResponse, PaginatedResponse } from '~/types/api'
import { useListState } from './useAsyncState'

export function useWebhooks() {
  const api = useApi()
  const config = useRuntimeConfig()
  const baseUrl = config.public.apiBase || 'http://localhost:8080'

  const {
    items: webhooks,
    loading,
    error,
    execute,
    setItems,
    addItem,
    updateItem,
    removeItem,
  } = useListState<Webhook>()

  async function list(projectId?: string): Promise<Webhook[]> {
    return execute(async () => {
      const params = new URLSearchParams()
      if (projectId) {
        params.append('project_id', projectId)
      }
      const queryString = params.toString()
      const endpoint = `/webhooks${queryString ? `?${queryString}` : ''}`

      const result = await api.get<PaginatedResponse<Webhook>>(endpoint)
      const items = result.data || []
      setItems(items)
      return items
    }, 'Failed to fetch webhooks')
  }

  async function get(id: string): Promise<Webhook> {
    return execute(async () => {
      const result = await api.get<ApiResponse<Webhook>>(`/webhooks/${id}`)
      return result.data
    }, 'Failed to fetch webhook')
  }

  async function create(data: CreateWebhookRequest): Promise<Webhook> {
    return execute(async () => {
      const result = await api.post<ApiResponse<Webhook>>('/webhooks', data)
      addItem(result.data)
      return result.data
    }, 'Failed to create webhook')
  }

  async function update(id: string, data: UpdateWebhookRequest): Promise<Webhook> {
    return execute(async () => {
      const result = await api.put<ApiResponse<Webhook>>(`/webhooks/${id}`, data)
      updateItem(w => w.id === id, result.data)
      return result.data
    }, 'Failed to update webhook')
  }

  async function remove(id: string): Promise<void> {
    return execute(async () => {
      await api.delete(`/webhooks/${id}`)
      removeItem(w => w.id === id)
    }, 'Failed to delete webhook')
  }

  async function enable(id: string): Promise<Webhook> {
    return execute(async () => {
      const result = await api.post<ApiResponse<Webhook>>(`/webhooks/${id}/enable`)
      updateItem(w => w.id === id, result.data)
      return result.data
    }, 'Failed to enable webhook')
  }

  async function disable(id: string): Promise<Webhook> {
    return execute(async () => {
      const result = await api.post<ApiResponse<Webhook>>(`/webhooks/${id}/disable`)
      updateItem(w => w.id === id, result.data)
      return result.data
    }, 'Failed to disable webhook')
  }

  async function regenerateSecret(id: string): Promise<Webhook> {
    return execute(async () => {
      const result = await api.post<ApiResponse<Webhook>>(`/webhooks/${id}/regenerate-secret`)
      updateItem(w => w.id === id, result.data)
      return result.data
    }, 'Failed to regenerate webhook secret')
  }

  // Get the webhook URL for triggering
  function getWebhookUrl(webhook: Webhook): string {
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
