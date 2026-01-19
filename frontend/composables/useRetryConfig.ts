// Retry Config management composable
import type { RetryConfig } from '~/types/api'

export interface UpdateRetryConfigInput {
  max_retries: number
  delay_ms: number
  max_delay_ms?: number
  exponential_backoff?: boolean
  retry_on_errors?: string[]
}

export function useRetryConfig() {
  const api = useApi()

  // Get retry config for a step
  async function getRetryConfig(projectId: string, stepId: string): Promise<RetryConfig> {
    return api.get<RetryConfig>(`/api/v1/workflows/${projectId}/steps/${stepId}/retry-config`)
  }

  // Update retry config for a step
  async function updateRetryConfig(
    projectId: string,
    stepId: string,
    input: UpdateRetryConfigInput
  ): Promise<void> {
    await api.put(`/api/v1/workflows/${projectId}/steps/${stepId}/retry-config`, input)
  }

  // Delete retry config for a step (reset to defaults)
  async function deleteRetryConfig(projectId: string, stepId: string): Promise<void> {
    await api.delete(`/api/v1/workflows/${projectId}/steps/${stepId}/retry-config`)
  }

  // Helper: calculate delay for a specific retry attempt
  function calculateDelay(config: RetryConfig, attempt: number): number {
    if (config.exponential_backoff) {
      const delay = config.delay_ms * Math.pow(2, attempt - 1)
      return Math.min(delay, config.max_delay_ms || 30000)
    }
    return config.delay_ms
  }

  return {
    getRetryConfig,
    updateRetryConfig,
    deleteRetryConfig,
    calculateDelay,
  }
}
