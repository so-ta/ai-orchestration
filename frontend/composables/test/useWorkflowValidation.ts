// Workflow validation composable
import type { ApiResponse } from '~/types/api'

export interface ValidationCheck {
  id: string
  label: string
  status: 'passed' | 'warning' | 'error'
  message?: string
}

export interface ValidationResult {
  checks: ValidationCheck[]
  can_publish: boolean
  error_count: number
  warning_count: number
}

export function useWorkflowValidation() {
  const api = useApi()

  const validating = ref(false)
  const validationResult = ref<ValidationResult | null>(null)
  const validationError = ref<string | null>(null)

  // Validate workflow
  async function validate(projectId: string): Promise<ValidationResult | null> {
    validating.value = true
    validationError.value = null

    try {
      const response = await api.post<ApiResponse<ValidationResult>>(`/workflows/${projectId}/validate`)
      validationResult.value = response.data
      return response.data
    } catch (error) {
      validationError.value = error instanceof Error ? error.message : 'Validation failed'
      return null
    } finally {
      validating.value = false
    }
  }

  // Check if all validations passed
  const allPassed = computed(() => {
    if (!validationResult.value) return false
    return validationResult.value.error_count === 0
  })

  // Check if there are any warnings
  const hasWarnings = computed(() => {
    if (!validationResult.value) return false
    return validationResult.value.warning_count > 0
  })

  // Get error checks only
  const errorChecks = computed(() => {
    if (!validationResult.value) return []
    return validationResult.value.checks.filter(c => c.status === 'error')
  })

  // Get warning checks only
  const warningChecks = computed(() => {
    if (!validationResult.value) return []
    return validationResult.value.checks.filter(c => c.status === 'warning')
  })

  // Get passed checks only
  const passedChecks = computed(() => {
    if (!validationResult.value) return []
    return validationResult.value.checks.filter(c => c.status === 'passed')
  })

  // Reset validation state
  function reset() {
    validationResult.value = null
    validationError.value = null
  }

  return {
    validating: readonly(validating),
    validationResult: readonly(validationResult),
    validationError: readonly(validationError),
    allPassed,
    hasWarnings,
    errorChecks,
    warningChecks,
    passedChecks,
    validate,
    reset,
  }
}
