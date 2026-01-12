import type { Schedule, CreateScheduleRequest, UpdateScheduleRequest, ApiResponse, PaginatedResponse } from '~/types/api'

export function useSchedules() {
  const config = useRuntimeConfig()
  const baseUrl = config.public.apiBase || 'http://localhost:8080'

  const schedules = ref<Schedule[]>([])
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

  const list = async (workflowId?: string): Promise<Schedule[]> => {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (workflowId) {
        params.append('workflow_id', workflowId)
      }
      const queryString = params.toString()
      const url = `${baseUrl}/schedules${queryString ? `?${queryString}` : ''}`

      const response = await fetch(url, {
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to fetch schedules: ${response.statusText}`)
      }
      const result: PaginatedResponse<Schedule> = await response.json()
      schedules.value = result.data || []
      return result.data || []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const get = async (id: string): Promise<Schedule> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/schedules/${id}`, {
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to fetch schedule: ${response.statusText}`)
      }
      const result: ApiResponse<Schedule> = await response.json()
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const create = async (data: CreateScheduleRequest): Promise<Schedule> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/schedules`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        throw new Error(`Failed to create schedule: ${response.statusText}`)
      }
      const result: ApiResponse<Schedule> = await response.json()
      schedules.value.push(result.data)
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const update = async (id: string, data: UpdateScheduleRequest): Promise<Schedule> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/schedules/${id}`, {
        method: 'PUT',
        headers: getHeaders(),
        body: JSON.stringify(data),
      })
      if (!response.ok) {
        throw new Error(`Failed to update schedule: ${response.statusText}`)
      }
      const result: ApiResponse<Schedule> = await response.json()
      const index = schedules.value.findIndex(s => s.id === id)
      if (index !== -1) {
        schedules.value[index] = result.data
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
      const response = await fetch(`${baseUrl}/schedules/${id}`, {
        method: 'DELETE',
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to delete schedule: ${response.statusText}`)
      }
      schedules.value = schedules.value.filter(s => s.id !== id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const pause = async (id: string): Promise<Schedule> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/schedules/${id}/pause`, {
        method: 'POST',
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to pause schedule: ${response.statusText}`)
      }
      const result: ApiResponse<Schedule> = await response.json()
      const index = schedules.value.findIndex(s => s.id === id)
      if (index !== -1) {
        schedules.value[index] = result.data
      }
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const resume = async (id: string): Promise<Schedule> => {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${baseUrl}/schedules/${id}/resume`, {
        method: 'POST',
        headers: getHeaders(),
      })
      if (!response.ok) {
        throw new Error(`Failed to resume schedule: ${response.statusText}`)
      }
      const result: ApiResponse<Schedule> = await response.json()
      const index = schedules.value.findIndex(s => s.id === id)
      if (index !== -1) {
        schedules.value[index] = result.data
      }
      return result.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    schedules,
    loading,
    error,
    list,
    get,
    create,
    update,
    remove,
    pause,
    resume,
  }
}
