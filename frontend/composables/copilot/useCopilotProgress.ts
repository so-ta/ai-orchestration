/**
 * useCopilotProgress.ts
 *
 * Composable for managing workflow progress state in Copilot.
 * Tracks the current phase, checklist items, and completion status.
 */
import type {
  WorkflowPhase,
  WorkflowProgressStatus,
  ChecklistItem,
  ChecklistItemStatus,
  RequiredCredential,
  TriggerConfig,
  WorkflowCopilotStatusResponse,
} from '~/components/workflow-editor/copilot/types'

export interface UseCopilotProgressOptions {
  workflowId: string
  onPhaseChange?: (phase: WorkflowPhase) => void
}

export function useCopilotProgress(options: UseCopilotProgressOptions) {
  const api = useApi()
  const { t } = useI18n()

  // State
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const currentPhase = ref<WorkflowPhase>('creation')
  const progress = ref<WorkflowProgressStatus | null>(null)
  const requiredCredentials = ref<RequiredCredential[]>([])
  const triggerConfig = ref<{ type: string; isConfigured: boolean; config?: TriggerConfig } | null>(null)
  const isPublished = ref(false)
  const canPublish = ref(false)

  // Build checklist items based on workflow state
  function buildChecklistItems(response: WorkflowCopilotStatusResponse): ChecklistItem[] {
    const items: ChecklistItem[] = []

    // 1. Workflow creation
    items.push({
      id: 'creation',
      phase: 'creation',
      label: t('copilot.progress.items.creation'),
      status: response.currentPhase === 'creation' ? 'in_progress' : 'completed',
    })

    // 2. Trigger configuration
    const triggerStatus: ChecklistItemStatus =
      response.trigger?.isConfigured ? 'completed' :
      response.currentPhase === 'configuration' ? 'in_progress' : 'pending'

    const triggerDescription = response.trigger?.isConfigured
      ? getTriggerDescription(response.trigger.config)
      : undefined

    items.push({
      id: 'configuration',
      phase: 'configuration',
      label: t('copilot.progress.items.trigger'),
      description: triggerDescription,
      status: triggerStatus,
      action: triggerStatus !== 'completed' ? {
        id: 'configure-trigger',
        type: 'select',
        title: t('copilot.progress.configure'),
        options: [
          { id: 'schedule', label: t('copilot.trigger.schedule'), icon: '⏰', description: t('copilot.trigger.scheduleDesc'), recommended: true },
          { id: 'webhook', label: t('copilot.trigger.webhook'), icon: '🔗', description: t('copilot.trigger.webhookDesc') },
          { id: 'manual', label: t('copilot.trigger.manual'), icon: '👤', description: t('copilot.trigger.manualDesc') },
        ],
      } : undefined,
    })

    // 3. Credential setup
    if (response.requiredCredentials.length > 0) {
      const configuredCount = response.requiredCredentials.filter(c => c.isConfigured).length
      const totalCount = response.requiredCredentials.length
      const allConfigured = configuredCount === totalCount

      const credentialStatus: ChecklistItemStatus =
        allConfigured ? 'completed' :
        response.currentPhase === 'setup' ? 'in_progress' : 'pending'

      const children: ChecklistItem[] = response.requiredCredentials.map(cred => ({
        id: `cred-${cred.id}`,
        phase: 'setup',
        label: `${cred.serviceName}${cred.isConfigured ? ' 連携済み' : ''}`,
        status: cred.isConfigured ? 'completed' : 'pending',
        action: !cred.isConfigured ? {
          id: `oauth-${cred.service}`,
          type: 'oauth',
          service: cred.service,
          serviceName: cred.serviceName,
          serviceIcon: cred.serviceIcon,
        } : undefined,
      }))

      items.push({
        id: 'setup',
        phase: 'setup',
        label: t('copilot.progress.items.credentials'),
        description: `${configuredCount}/${totalCount} ${t('copilot.progress.completed')}`,
        status: credentialStatus,
        children,
      })
    }

    // 4. Validation (test)
    const validationStatus: ChecklistItemStatus =
      response.validation?.isValid ? 'completed' :
      response.currentPhase === 'validation' ? 'in_progress' : 'pending'

    items.push({
      id: 'validation',
      phase: 'validation',
      label: t('copilot.progress.items.test'),
      status: validationStatus,
      action: validationStatus !== 'completed' ? {
        id: 'run-test',
        type: 'test',
        title: t('copilot.progress.runTest'),
      } : undefined,
    })

    // 5. Deploy (publish)
    const deployStatus: ChecklistItemStatus =
      response.isPublished ? 'completed' :
      response.currentPhase === 'deploy' ? 'in_progress' : 'pending'

    items.push({
      id: 'deploy',
      phase: 'deploy',
      label: t('copilot.progress.items.publish'),
      status: deployStatus,
      action: deployStatus !== 'completed' && response.canPublish ? {
        id: 'publish',
        type: 'confirm',
        title: t('copilot.progress.publish'),
        confirmLabel: t('copilot.progress.publishNow'),
        cancelLabel: t('copilot.progress.later'),
      } : undefined,
    })

    return items
  }

  // Get trigger description
  function getTriggerDescription(config?: TriggerConfig): string | undefined {
    if (!config) return undefined

    switch (config.type) {
      case 'schedule':
        return `${t('copilot.trigger.schedule')}: ${config.time} (${config.timezone})`
      case 'webhook':
        return t('copilot.trigger.webhook')
      case 'manual':
        return t('copilot.trigger.manual')
      case 'slack_event':
        return `Slack ${config.eventType}`
      default:
        return undefined
    }
  }

  // Fetch progress status from API
  async function fetchProgress(): Promise<void> {
    isLoading.value = true
    error.value = null

    try {
      const response = await api.get<WorkflowCopilotStatusResponse>(
        `/workflows/${options.workflowId}/copilot/status`
      )

      currentPhase.value = response.currentPhase
      requiredCredentials.value = response.requiredCredentials
      triggerConfig.value = response.trigger || null
      isPublished.value = response.isPublished
      canPublish.value = response.canPublish

      const items = buildChecklistItems(response)
      const completedCount = items.filter(i => i.status === 'completed').length

      progress.value = {
        currentPhase: response.currentPhase,
        items,
        completedCount,
        totalCount: items.length,
        isComplete: completedCount === items.length,
      }

      options.onPhaseChange?.(response.currentPhase)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch progress'
      console.error('Failed to fetch copilot progress:', e)
    } finally {
      isLoading.value = false
    }
  }

  // Update item status locally (optimistic update)
  function updateItemStatus(itemId: string, status: ChecklistItemStatus): void {
    if (!progress.value) return

    const item = progress.value.items.find(i => i.id === itemId)
    if (item) {
      item.status = status
      progress.value.completedCount = progress.value.items.filter(i => i.status === 'completed').length
      progress.value.isComplete = progress.value.completedCount === progress.value.totalCount
    }
  }

  // Update child item status
  function updateChildItemStatus(parentId: string, childId: string, status: ChecklistItemStatus): void {
    if (!progress.value) return

    const parent = progress.value.items.find(i => i.id === parentId)
    if (parent?.children) {
      const child = parent.children.find(c => c.id === childId)
      if (child) {
        child.status = status

        // Check if all children are completed
        const allChildrenCompleted = parent.children.every(c => c.status === 'completed')
        if (allChildrenCompleted) {
          parent.status = 'completed'
          progress.value.completedCount = progress.value.items.filter(i => i.status === 'completed').length
          progress.value.isComplete = progress.value.completedCount === progress.value.totalCount
        }
      }
    }
  }

  // Manually set a mock progress for development/demo
  function setMockProgress(mockProgress: WorkflowProgressStatus): void {
    progress.value = mockProgress
    currentPhase.value = mockProgress.currentPhase
  }

  // Create initial progress (when starting fresh)
  function createInitialProgress(): WorkflowProgressStatus {
    const items: ChecklistItem[] = [
      { id: 'creation', phase: 'creation', label: t('copilot.progress.items.creation'), status: 'in_progress' },
      { id: 'configuration', phase: 'configuration', label: t('copilot.progress.items.trigger'), status: 'pending' },
      { id: 'validation', phase: 'validation', label: t('copilot.progress.items.test'), status: 'pending' },
      { id: 'deploy', phase: 'deploy', label: t('copilot.progress.items.publish'), status: 'pending' },
    ]

    return {
      currentPhase: 'creation',
      items,
      completedCount: 0,
      totalCount: items.length,
      isComplete: false,
    }
  }

  return {
    // State
    isLoading: readonly(isLoading),
    error: readonly(error),
    currentPhase: readonly(currentPhase),
    progress: readonly(progress),
    requiredCredentials: readonly(requiredCredentials),
    triggerConfig: readonly(triggerConfig),
    isPublished: readonly(isPublished),
    canPublish: readonly(canPublish),

    // Actions
    fetchProgress,
    updateItemStatus,
    updateChildItemStatus,
    setMockProgress,
    createInitialProgress,
  }
}
