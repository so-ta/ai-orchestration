/**
 * useCopilotActions.ts
 *
 * Composable for handling Copilot workflow actions.
 * Provides handlers for trigger configuration, credential linking, testing, and publishing.
 */
import type {
  TriggerType,
  ScheduleTriggerConfig,
  WebhookTriggerConfig,
  WorkflowTestResult,
  InlineActionResult,
  FormResult,
  SelectResult,
  OAuthResult,
  TestResult,
  ConfirmResult,
} from '~/components/workflow-editor/copilot/types'

export interface UseCopilotActionsOptions {
  workflowId: string
  onProgress?: () => void
  onTestStart?: () => void
  onTestComplete?: (result: WorkflowTestResult) => void
  onPublishComplete?: () => void
}

export function useCopilotActions(options: UseCopilotActionsOptions) {
  const api = useApi()
  const toast = useToast()
  const { t } = useI18n()

  // State
  const isConfiguring = ref(false)
  const isTesting = ref(false)
  const isPublishing = ref(false)
  const testResult = ref<WorkflowTestResult | null>(null)

  // ============================================================================
  // Trigger Configuration
  // ============================================================================

  /**
   * Configure trigger based on type selection
   */
  async function configureTrigger(triggerType: TriggerType): Promise<boolean> {
    isConfiguring.value = true

    try {
      // This will be called after the user selects a trigger type
      // The actual configuration form will be shown via inline action
      await api.post(`/workflows/${options.workflowId}/copilot/configure-trigger`, {
        type: triggerType,
      })

      options.onProgress?.()
      return true
    } catch (e) {
      console.error('Failed to configure trigger:', e)
      toast.error(t('copilot.errors.triggerConfigFailed'))
      return false
    } finally {
      isConfiguring.value = false
    }
  }

  /**
   * Apply schedule trigger configuration
   */
  async function applyScheduleTrigger(config: Omit<ScheduleTriggerConfig, 'type'>): Promise<boolean> {
    isConfiguring.value = true

    try {
      await api.post(`/workflows/${options.workflowId}/copilot/configure-trigger`, {
        type: 'schedule',
        config: {
          time: config.time,
          timezone: config.timezone,
          cron: config.cron,
          days: config.days,
        },
      })

      toast.success(t('copilot.trigger.configured'))
      options.onProgress?.()
      return true
    } catch (e) {
      console.error('Failed to apply schedule trigger:', e)
      toast.error(t('copilot.errors.triggerConfigFailed'))
      return false
    } finally {
      isConfiguring.value = false
    }
  }

  /**
   * Apply webhook trigger configuration
   */
  async function applyWebhookTrigger(config?: Omit<WebhookTriggerConfig, 'type'>): Promise<boolean> {
    isConfiguring.value = true

    try {
      const response = await api.post<{ webhookUrl: string }>(`/workflows/${options.workflowId}/copilot/configure-trigger`, {
        type: 'webhook',
        config: config || {},
      })

      toast.success(t('copilot.trigger.webhookCreated', { url: response.webhookUrl }))
      options.onProgress?.()
      return true
    } catch (e) {
      console.error('Failed to apply webhook trigger:', e)
      toast.error(t('copilot.errors.triggerConfigFailed'))
      return false
    } finally {
      isConfiguring.value = false
    }
  }

  // ============================================================================
  // Credential Linking
  // ============================================================================

  /**
   * Link a credential to a step
   */
  async function linkCredential(stepId: string, credentialId: string): Promise<boolean> {
    try {
      await api.post(`/workflows/${options.workflowId}/copilot/link-credential`, {
        step_id: stepId,
        credential_id: credentialId,
      })

      toast.success(t('copilot.credential.linked'))
      options.onProgress?.()
      return true
    } catch (e) {
      console.error('Failed to link credential:', e)
      toast.error(t('copilot.errors.credentialLinkFailed'))
      return false
    }
  }

  // ============================================================================
  // Workflow Testing
  // ============================================================================

  /**
   * Run workflow test
   */
  async function runTest(): Promise<WorkflowTestResult | null> {
    isTesting.value = true
    testResult.value = null
    options.onTestStart?.()

    try {
      // Start test run
      const startResponse = await api.post<{ runId: string }>(`/workflows/${options.workflowId}/copilot/test`, {})
      const runId = startResponse.runId

      // Create initial result structure
      const result: WorkflowTestResult = {
        runId,
        status: 'running',
        steps: [],
        startedAt: new Date().toISOString(),
      }
      testResult.value = result

      // Poll for completion
      const pollResult = await pollTestStatus(runId)
      testResult.value = pollResult
      options.onTestComplete?.(pollResult)

      if (pollResult.status === 'success') {
        toast.success(t('copilot.test.passed'))
        options.onProgress?.()
      } else {
        toast.error(t('copilot.test.failedMessage'))
      }

      return pollResult
    } catch (e) {
      console.error('Failed to run test:', e)
      toast.error(t('copilot.errors.testFailed'))
      return null
    } finally {
      isTesting.value = false
    }
  }

  /**
   * Poll test status until complete
   */
  async function pollTestStatus(runId: string): Promise<WorkflowTestResult> {
    const maxAttempts = 60
    const interval = 2000

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
      const response = await api.get<{
        status: 'running' | 'completed' | 'failed'
        steps: Array<{
          stepId: string
          stepName: string
          status: 'pending' | 'running' | 'success' | 'error'
          durationMs?: number
          error?: string
        }>
        totalDurationMs?: number
        completedAt?: string
      }>(`/runs/${runId}/status`)

      // Update test result state
      const result: WorkflowTestResult = {
        runId,
        status: response.status === 'completed' ? 'success' :
                response.status === 'failed' ? 'failed' : 'running',
        steps: response.steps.map(s => ({
          stepId: s.stepId,
          stepName: s.stepName,
          status: s.status,
          durationMs: s.durationMs,
          error: s.error,
        })),
        totalDurationMs: response.totalDurationMs,
        startedAt: testResult.value?.startedAt || new Date().toISOString(),
        completedAt: response.completedAt,
      }

      testResult.value = result

      if (response.status !== 'running') {
        return result
      }

      await new Promise(resolve => setTimeout(resolve, interval))
    }

    throw new Error('Test polling timeout')
  }

  // ============================================================================
  // Workflow Publishing
  // ============================================================================

  /**
   * Publish workflow
   */
  async function publishWorkflow(): Promise<boolean> {
    isPublishing.value = true

    try {
      await api.post(`/workflows/${options.workflowId}/publish`, {})

      toast.success(t('copilot.publish.success'))
      options.onPublishComplete?.()
      options.onProgress?.()
      return true
    } catch (e) {
      console.error('Failed to publish workflow:', e)
      toast.error(t('copilot.errors.publishFailed'))
      return false
    } finally {
      isPublishing.value = false
    }
  }

  // ============================================================================
  // Unified Action Handler
  // ============================================================================

  /**
   * Handle inline action result
   */
  async function handleActionResult(actionId: string, result: InlineActionResult): Promise<boolean> {
    switch (result.type) {
      case 'select': {
        const selectResult = result as SelectResult
        if (actionId === 'configure-trigger') {
          const triggerType = selectResult.selectedIds[0] as TriggerType
          return await configureTrigger(triggerType)
        }
        break
      }

      case 'form': {
        const formResult = result as FormResult
        if (actionId.startsWith('schedule-')) {
          return await applyScheduleTrigger({
            time: String(formResult.values.time || '09:00'),
            timezone: String(formResult.values.timezone || 'Asia/Tokyo'),
            cron: formResult.values.cron ? String(formResult.values.cron) : undefined,
          })
        }
        break
      }

      case 'oauth': {
        const oauthResult = result as OAuthResult
        // Extract step ID from action ID (format: oauth-{service}-{stepId})
        const parts = actionId.split('-')
        if (parts.length >= 3) {
          const stepId = parts.slice(2).join('-')
          return await linkCredential(stepId, oauthResult.credentialId)
        }
        break
      }

      case 'test': {
        const testRes = result as TestResult
        if (!testRes.skipped) {
          const testResult = await runTest()
          return testResult?.status === 'success'
        }
        // Skipped - just continue
        options.onProgress?.()
        return true
      }

      case 'confirm': {
        const confirmResult = result as ConfirmResult
        if (actionId === 'publish' && confirmResult.confirmed) {
          return await publishWorkflow()
        }
        break
      }
    }

    return false
  }

  return {
    // State
    isConfiguring: readonly(isConfiguring),
    isTesting: readonly(isTesting),
    isPublishing: readonly(isPublishing),
    testResult: readonly(testResult),

    // Actions
    configureTrigger,
    applyScheduleTrigger,
    applyWebhookTrigger,
    linkCredential,
    runTest,
    publishWorkflow,
    handleActionResult,
  }
}
