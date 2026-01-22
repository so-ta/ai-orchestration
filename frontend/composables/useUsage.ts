// Usage tracking composable
import { extractErrorMessage } from './useAsyncState'

export interface UsageSummary {
  period: string
  total_cost_usd: number
  total_requests: number
  total_input_tokens: number
  total_output_tokens: number
  by_provider: Record<string, { cost_usd: number; requests: number }>
  by_model: Record<string, { provider: string; cost_usd: number; requests: number; input_tokens: number; output_tokens: number }>
  budget?: {
    monthly_limit_usd?: number
    daily_limit_usd?: number
    consumed_percent: number
    alert_triggered: boolean
  }
}

export interface DailyUsage {
  date: string
  total_cost_usd: number
  total_requests: number
  total_tokens: number
}

export interface ProjectUsage {
  project_id: string
  project_name: string
  total_cost_usd: number
  total_requests: number
  total_tokens: number
}

export interface ModelUsage {
  provider: string
  cost_usd: number
  requests: number
  input_tokens: number
  output_tokens: number
}

export interface UsageRecord {
  id: string
  tenant_id: string
  workflow_id?: string
  run_id?: string
  step_run_id?: string
  provider: string
  model: string
  operation: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  input_cost_usd: number
  output_cost_usd: number
  total_cost_usd: number
  latency_ms?: number
  success: boolean
  error_message?: string
  created_at: string
}

export interface Budget {
  id: string
  tenant_id: string
  workflow_id?: string
  budget_type: 'daily' | 'monthly'
  budget_amount_usd: number
  alert_threshold: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface TokenPricing {
  Provider: string
  Model: string
  InputPer1K: number
  OutputPer1K: number
}

export function useUsage() {
  const api = useApi()

  const summary = ref<UsageSummary | null>(null)
  const dailyData = ref<DailyUsage[]>([])
  const projectData = ref<ProjectUsage[]>([])
  const modelData = ref<Record<string, ModelUsage>>({})
  const budgets = ref<Budget[]>([])
  const pricing = ref<TokenPricing[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Fetch usage summary
  async function fetchSummary(period = 'month') {
    loading.value = true
    error.value = null
    try {
      summary.value = await api.get<UsageSummary>(`/usage/summary?period=${period}`)
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to fetch usage summary')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch daily usage data
  async function fetchDaily(start?: string, end?: string) {
    loading.value = true
    error.value = null
    try {
      let url = '/usage/daily'
      const params = new URLSearchParams()
      if (start) params.append('start', start)
      if (end) params.append('end', end)
      if (params.toString()) url += `?${params.toString()}`

      const response = await api.get<{ daily: DailyUsage[]; start: string; end: string }>(url)
      dailyData.value = response.daily || []
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to fetch daily usage')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch usage by project
  async function fetchByProject(period = 'month') {
    loading.value = true
    error.value = null
    try {
      const response = await api.get<{ projects: ProjectUsage[] }>(`/usage/by-project?period=${period}`)
      projectData.value = response.projects || []
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to fetch project usage')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch usage by model
  async function fetchByModel(period = 'month') {
    loading.value = true
    error.value = null
    try {
      const response = await api.get<{ models: Record<string, ModelUsage> }>(`/usage/by-model?period=${period}`)
      modelData.value = response.models || {}
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to fetch model usage')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch usage for a specific run
  async function fetchByRun(runId: string): Promise<{ records: UsageRecord[]; total_cost_usd: number; total_input_tokens: number; total_output_tokens: number }> {
    loading.value = true
    error.value = null
    try {
      return await api.get(`/runs/${runId}/usage`)
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to fetch run usage')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch budgets
  async function fetchBudgets() {
    loading.value = true
    error.value = null
    try {
      const response = await api.get<{ budgets: Budget[] }>('/usage/budgets')
      budgets.value = response.budgets || []
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to fetch budgets')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Create a budget
  async function createBudget(data: {
    workflow_id?: string
    budget_type: 'daily' | 'monthly'
    budget_amount_usd: number
    alert_threshold?: number
  }): Promise<Budget> {
    loading.value = true
    error.value = null
    try {
      const budget = await api.post<Budget>('/usage/budgets', data)
      await fetchBudgets()
      return budget
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to create budget')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Update a budget
  async function updateBudget(id: string, data: {
    budget_amount_usd?: number
    alert_threshold?: number
    enabled?: boolean
  }): Promise<Budget> {
    loading.value = true
    error.value = null
    try {
      const budget = await api.put<Budget>(`/usage/budgets/${id}`, data)
      await fetchBudgets()
      return budget
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to update budget')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Delete a budget
  async function deleteBudget(id: string) {
    loading.value = true
    error.value = null
    try {
      await api.delete(`/usage/budgets/${id}`)
      await fetchBudgets()
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to delete budget')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch pricing information
  async function fetchPricing() {
    loading.value = true
    error.value = null
    try {
      const response = await api.get<{ pricing: TokenPricing[] }>('/usage/pricing')
      pricing.value = response.pricing || []
    } catch (e) {
      error.value = extractErrorMessage(e, 'Failed to fetch pricing')
      throw e
    } finally {
      loading.value = false
    }
  }

  // Fetch all data for dashboard
  async function fetchDashboardData(period = 'month') {
    await Promise.all([
      fetchSummary(period),
      fetchDaily(),
      fetchByProject(period),
      fetchByModel(period),
      fetchBudgets(),
    ])
  }

  // Computed helpers
  const budgetStatus = computed(() => {
    if (!summary.value?.budget) return null
    return {
      limit: summary.value.budget.monthly_limit_usd ?? summary.value.budget.daily_limit_usd,
      consumed: summary.value.budget.consumed_percent,
      alert: summary.value.budget.alert_triggered,
    }
  })

  const topModels = computed(() => {
    return Object.entries(modelData.value)
      .map(([model, data]) => ({ model, ...data }))
      .sort((a, b) => b.cost_usd - a.cost_usd)
      .slice(0, 5)
  })

  const topProjects = computed(() => {
    return [...projectData.value]
      .sort((a, b) => b.total_cost_usd - a.total_cost_usd)
      .slice(0, 5)
  })

  // Note: State refs not wrapped in readonly() for direct mutation support
  return {
    // State
    summary,
    dailyData,
    projectData,
    modelData,
    budgets,
    pricing,
    loading,
    error,

    // Actions
    fetchSummary,
    fetchDaily,
    fetchByProject,
    fetchByModel,
    fetchByRun,
    fetchBudgets,
    createBudget,
    updateBudget,
    deleteBudget,
    fetchPricing,
    fetchDashboardData,

    // Computed
    budgetStatus,
    topModels,
    topProjects,
  }
}
