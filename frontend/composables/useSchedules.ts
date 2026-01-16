import type { Schedule, CreateScheduleRequest, UpdateScheduleRequest, ApiResponse, PaginatedResponse } from '~/types/api'
import { useListState } from './useAsyncState'

export function useSchedules() {
  const api = useApi()
  const {
    items: schedules,
    loading,
    error,
    execute,
    setItems,
    addItem,
    updateItem,
    removeItem,
  } = useListState<Schedule>()

  async function list(workflowId?: string): Promise<Schedule[]> {
    return execute(async () => {
      const params = new URLSearchParams()
      if (workflowId) {
        params.append('workflow_id', workflowId)
      }
      const queryString = params.toString()
      const endpoint = `/schedules${queryString ? `?${queryString}` : ''}`

      const result = await api.get<PaginatedResponse<Schedule>>(endpoint)
      const items = result.data || []
      setItems(items)
      return items
    }, 'Failed to fetch schedules')
  }

  async function get(id: string): Promise<Schedule> {
    return execute(async () => {
      const result = await api.get<ApiResponse<Schedule>>(`/schedules/${id}`)
      return result.data
    }, 'Failed to fetch schedule')
  }

  async function create(data: CreateScheduleRequest): Promise<Schedule> {
    return execute(async () => {
      const result = await api.post<ApiResponse<Schedule>>('/schedules', data)
      addItem(result.data)
      return result.data
    }, 'Failed to create schedule')
  }

  async function update(id: string, data: UpdateScheduleRequest): Promise<Schedule> {
    return execute(async () => {
      const result = await api.put<ApiResponse<Schedule>>(`/schedules/${id}`, data)
      updateItem(s => s.id === id, result.data)
      return result.data
    }, 'Failed to update schedule')
  }

  async function remove(id: string): Promise<void> {
    return execute(async () => {
      await api.delete(`/schedules/${id}`)
      removeItem(s => s.id === id)
    }, 'Failed to delete schedule')
  }

  async function pause(id: string): Promise<Schedule> {
    return execute(async () => {
      const result = await api.post<ApiResponse<Schedule>>(`/schedules/${id}/pause`)
      updateItem(s => s.id === id, result.data)
      return result.data
    }, 'Failed to pause schedule')
  }

  async function resume(id: string): Promise<Schedule> {
    return execute(async () => {
      const result = await api.post<ApiResponse<Schedule>>(`/schedules/${id}/resume`)
      updateItem(s => s.id === id, result.data)
      return result.data
    }, 'Failed to resume schedule')
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
