import type {
  Tenant,
  CreateTenantRequest,
  UpdateTenantRequest,
  SuspendTenantRequest,
  TenantOverviewStats,
  TenantStats,
  TenantStatus,
  TenantPlan,
  PaginatedResponse,
} from '~/types/api'
import { useListState } from './useAsyncState'

interface TenantFilters {
  status?: TenantStatus
  plan?: TenantPlan
  search?: string
  page?: number
  limit?: number
  include_deleted?: boolean
}

export function useTenants() {
  const api = useApi()

  const {
    items: tenants,
    loading,
    error,
    pagination,
    execute,
    setItems,
    addItem,
    updateItem,
    removeItem,
    setPagination,
  } = useListState<Tenant>()

  async function fetchTenants(filters: TenantFilters = {}) {
    return execute(async () => {
      const params = new URLSearchParams()
      if (filters.status) params.set('status', filters.status)
      if (filters.plan) params.set('plan', filters.plan)
      if (filters.search) params.set('search', filters.search)
      if (filters.page) params.set('page', String(filters.page))
      if (filters.limit) params.set('limit', String(filters.limit))
      if (filters.include_deleted) params.set('include_deleted', 'true')

      const queryString = params.toString()
      const endpoint = `/admin/tenants${queryString ? `?${queryString}` : ''}`

      const response = await api.get<PaginatedResponse<Tenant>>(endpoint)
      setItems(response.data)
      setPagination({
        page: response.meta.page,
        limit: response.meta.limit,
        total: response.meta.total,
      })
    }, 'Failed to fetch tenants')
  }

  async function getTenant(id: string): Promise<Tenant> {
    return execute(async () => {
      const response = await api.get<Tenant>(`/admin/tenants/${id}`)
      return response
    }, 'Failed to fetch tenant')
  }

  async function createTenant(data: CreateTenantRequest): Promise<Tenant> {
    return execute(async () => {
      const response = await api.post<Tenant>('/admin/tenants', data)
      addItem(response)
      return response
    }, 'Failed to create tenant')
  }

  async function updateTenant(id: string, data: UpdateTenantRequest): Promise<Tenant> {
    return execute(async () => {
      const response = await api.put<Tenant>(`/admin/tenants/${id}`, data)
      updateItem(t => t.id === id, response)
      return response
    }, 'Failed to update tenant')
  }

  async function deleteTenant(id: string): Promise<void> {
    return execute(async () => {
      await api.delete(`/admin/tenants/${id}`)
      removeItem(t => t.id === id)
    }, 'Failed to delete tenant')
  }

  async function suspendTenant(id: string, reason: string): Promise<Tenant> {
    return execute(async () => {
      const data: SuspendTenantRequest = { reason }
      const response = await api.post<Tenant>(`/admin/tenants/${id}/suspend`, data)
      updateItem(t => t.id === id, response)
      return response
    }, 'Failed to suspend tenant')
  }

  async function activateTenant(id: string): Promise<Tenant> {
    return execute(async () => {
      const response = await api.post<Tenant>(`/admin/tenants/${id}/activate`)
      updateItem(t => t.id === id, response)
      return response
    }, 'Failed to activate tenant')
  }

  async function getTenantStats(id: string): Promise<TenantStats> {
    return execute(async () => {
      const response = await api.get<TenantStats>(`/admin/tenants/${id}/stats`)
      return response
    }, 'Failed to fetch tenant stats')
  }

  async function getOverviewStats(): Promise<TenantOverviewStats> {
    return execute(async () => {
      const response = await api.get<TenantOverviewStats>('/admin/tenants/stats/overview')
      return response
    }, 'Failed to fetch overview stats')
  }

  // Helper functions for status and plan display
  function getStatusColor(status: TenantStatus): string {
    const colorMap: Record<TenantStatus, string> = {
      active: 'success',
      suspended: 'error',
      pending: 'warning',
      inactive: 'secondary',
    }
    return colorMap[status] || 'secondary'
  }

  function getPlanColor(plan: TenantPlan): string {
    const colorMap: Record<TenantPlan, string> = {
      enterprise: 'primary',
      professional: 'info',
      starter: 'success',
      free: 'secondary',
    }
    return colorMap[plan] || 'secondary'
  }

  return {
    tenants,
    loading,
    error,
    pagination,
    fetchTenants,
    getTenant,
    createTenant,
    updateTenant,
    deleteTenant,
    suspendTenant,
    activateTenant,
    getTenantStats,
    getOverviewStats,
    getStatusColor,
    getPlanColor,
  }
}
