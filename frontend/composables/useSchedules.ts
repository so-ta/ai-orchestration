import type { Schedule, CreateScheduleRequest, UpdateScheduleRequest, ApiResponse, PaginatedResponse } from '~/types/api'

export function useSchedules() {
  const api = useApi()

  const schedules = ref<Schedule[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const list = async (workflowId?: string): Promise<Schedule[]> => {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (workflowId) {
        params.append('workflow_id', workflowId)
      }
      const queryString = params.toString()
      const endpoint = `/schedules${queryString ? `?${queryString}` : ''}`

      const result = await api.get<PaginatedResponse<Schedule>>(endpoint)
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
      const result = await api.get<ApiResponse<Schedule>>(`/schedules/${id}`)
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
      const result = await api.post<ApiResponse<Schedule>>('/schedules', data)
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
      const result = await api.put<ApiResponse<Schedule>>(`/schedules/${id}`, data)
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
      await api.delete(`/schedules/${id}`)
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
      const result = await api.post<ApiResponse<Schedule>>(`/schedules/${id}/pause`)
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
      const result = await api.post<ApiResponse<Schedule>>(`/schedules/${id}/resume`)
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
