// Steps API composable
import type { Step, ApiResponse } from '~/types/api'

interface TriggerStatusResponse {
  enabled: boolean
}

export function useSteps() {
  const api = useApi()

  // Enable trigger for a Start block
  async function enableTrigger(projectId: string, stepId: string) {
    return api.post<ApiResponse<Step>>(`/workflows/${projectId}/steps/${stepId}/trigger/enable`)
  }

  // Disable trigger for a Start block
  async function disableTrigger(projectId: string, stepId: string) {
    return api.post<ApiResponse<Step>>(`/workflows/${projectId}/steps/${stepId}/trigger/disable`)
  }

  // Get trigger status for a Start block
  async function getTriggerStatus(projectId: string, stepId: string) {
    return api.get<ApiResponse<TriggerStatusResponse>>(`/workflows/${projectId}/steps/${stepId}/trigger/status`)
  }

  // Toggle trigger enabled state
  async function toggleTrigger(projectId: string, stepId: string, enabled: boolean) {
    if (enabled) {
      return enableTrigger(projectId, stepId)
    } else {
      return disableTrigger(projectId, stepId)
    }
  }

  return {
    enableTrigger,
    disableTrigger,
    getTriggerStatus,
    toggleTrigger,
  }
}
