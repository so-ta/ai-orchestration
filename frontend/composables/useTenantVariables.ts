// Tenant (organization) variables composable
interface VariablesResponse {
  variables: Record<string, unknown>
}

export function useTenantVariables() {
  const api = useApi()
  const variables = ref<Record<string, unknown>>({})
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchVariables(): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const response = await api.get<VariablesResponse>('/tenant/variables')
      variables.value = response.variables || {}
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch tenant variables'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateVariables(newVariables: Record<string, unknown>): Promise<void> {
    loading.value = true
    error.value = null
    try {
      const response = await api.put<VariablesResponse>('/tenant/variables', {
        variables: newVariables,
      })
      variables.value = response.variables || {}
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update tenant variables'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    variables: readonly(variables),
    loading: readonly(loading),
    error: readonly(error),
    fetchVariables,
    updateVariables,
  }
}
