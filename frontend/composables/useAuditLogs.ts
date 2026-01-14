import type { AuditLog, AuditAction, PaginatedResponse } from '~/types/api'

export interface AuditLogFilter {
  resource_type?: string
  resource_id?: string
  action?: AuditAction
  user_id?: string
  from_date?: string
  to_date?: string
  page?: number
  limit?: number
}

export function useAuditLogs() {
  const api = useApi()

  const auditLogs = ref<AuditLog[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const list = async (filter?: AuditLogFilter): Promise<{ data: AuditLog[]; total: number }> => {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      if (filter) {
        if (filter.resource_type) params.append('resource_type', filter.resource_type)
        if (filter.resource_id) params.append('resource_id', filter.resource_id)
        if (filter.action) params.append('action', filter.action)
        if (filter.user_id) params.append('user_id', filter.user_id)
        if (filter.from_date) params.append('from_date', filter.from_date)
        if (filter.to_date) params.append('to_date', filter.to_date)
        if (filter.page) params.append('page', String(filter.page))
        if (filter.limit) params.append('limit', String(filter.limit))
      }
      const queryString = params.toString()
      const endpoint = `/audit-logs${queryString ? `?${queryString}` : ''}`

      const result = await api.get<PaginatedResponse<AuditLog>>(endpoint)
      auditLogs.value = result.data || []
      total.value = result.meta?.total || 0
      return { data: result.data || [], total: result.meta?.total || 0 }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  // Format action for display
  const formatAction = (action: AuditAction): string => {
    const actionMap: Record<AuditAction, string> = {
      create: 'Created',
      update: 'Updated',
      delete: 'Deleted',
      publish: 'Published',
      execute: 'Executed',
      cancel: 'Cancelled',
      approve: 'Approved',
      reject: 'Rejected',
    }
    return actionMap[action] || action
  }

  // Format resource type for display
  const formatResourceType = (type: string): string => {
    const typeMap: Record<string, string> = {
      workflow: 'Workflow',
      step: 'Step',
      edge: 'Edge',
      run: 'Run',
      schedule: 'Schedule',
      webhook: 'Webhook',
      credential: 'Credential',
      block: 'Block',
      tenant: 'Tenant',
    }
    return typeMap[type] || type
  }

  // Get action badge color
  const getActionColor = (action: AuditAction): string => {
    const colorMap: Record<AuditAction, string> = {
      create: '#22c55e',
      update: '#3b82f6',
      delete: '#ef4444',
      publish: '#8b5cf6',
      execute: '#f59e0b',
      cancel: '#6b7280',
      approve: '#22c55e',
      reject: '#ef4444',
    }
    return colorMap[action] || '#6b7280'
  }

  return {
    auditLogs,
    total,
    loading,
    error,
    list,
    formatAction,
    formatResourceType,
    getActionColor,
  }
}
