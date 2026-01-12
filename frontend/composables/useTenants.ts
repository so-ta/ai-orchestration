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

  const tenants = ref<Tenant[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const pagination = ref<{ page: number; limit: number; total: number }>({
    page: 1,
    limit: 20,
    total: 0,
  })

  async function fetchTenants(filters: TenantFilters = {}) {
    loading.value = true
    error.value = null

    try {
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
      tenants.value = response.data
      pagination.value = {
        page: response.meta.page,
        limit: response.meta.limit,
        total: response.meta.total,
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch tenants'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function getTenant(id: string): Promise<Tenant> {
    loading.value = true
    error.value = null

    try {
      const response = await api.get<Tenant>(`/admin/tenants/${id}`)
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch tenant'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createTenant(data: CreateTenantRequest): Promise<Tenant> {
    loading.value = true
    error.value = null

    try {
      const response = await api.post<Tenant>('/admin/tenants', data)
      // Add to local list
      tenants.value = [response, ...tenants.value]
      pagination.value.total += 1
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create tenant'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateTenant(id: string, data: UpdateTenantRequest): Promise<Tenant> {
    loading.value = true
    error.value = null

    try {
      const response = await api.put<Tenant>(`/admin/tenants/${id}`, data)
      // Update local list
      const index = tenants.value.findIndex(t => t.id === id)
      if (index !== -1) {
        tenants.value[index] = response
      }
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update tenant'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteTenant(id: string): Promise<void> {
    loading.value = true
    error.value = null

    try {
      await api.delete(`/admin/tenants/${id}`)
      // Remove from local list
      tenants.value = tenants.value.filter(t => t.id !== id)
      pagination.value.total -= 1
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete tenant'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function suspendTenant(id: string, reason: string): Promise<Tenant> {
    loading.value = true
    error.value = null

    try {
      const data: SuspendTenantRequest = { reason }
      const response = await api.post<Tenant>(`/admin/tenants/${id}/suspend`, data)
      // Update local list
      const index = tenants.value.findIndex(t => t.id === id)
      if (index !== -1) {
        tenants.value[index] = response
      }
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to suspend tenant'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function activateTenant(id: string): Promise<Tenant> {
    loading.value = true
    error.value = null

    try {
      const response = await api.post<Tenant>(`/admin/tenants/${id}/activate`)
      // Update local list
      const index = tenants.value.findIndex(t => t.id === id)
      if (index !== -1) {
        tenants.value[index] = response
      }
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to activate tenant'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function getTenantStats(id: string): Promise<TenantStats> {
    loading.value = true
    error.value = null

    try {
      const response = await api.get<TenantStats>(`/admin/tenants/${id}/stats`)
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch tenant stats'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function getOverviewStats(): Promise<TenantOverviewStats> {
    loading.value = true
    error.value = null

    try {
      const response = await api.get<TenantOverviewStats>('/admin/tenants/stats/overview')
      return response
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch overview stats'
      throw err
    } finally {
      loading.value = false
    }
  }

  // Helper functions for status and plan display
  function getStatusColor(status: TenantStatus): string {
    switch (status) {
      case 'active':
        return 'success'
      case 'suspended':
        return 'error'
      case 'pending':
        return 'warning'
      case 'inactive':
        return 'secondary'
      default:
        return 'secondary'
    }
  }

  function getPlanColor(plan: TenantPlan): string {
    switch (plan) {
      case 'enterprise':
        return 'primary'
      case 'professional':
        return 'info'
      case 'starter':
        return 'success'
      case 'free':
        return 'secondary'
      default:
        return 'secondary'
    }
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
