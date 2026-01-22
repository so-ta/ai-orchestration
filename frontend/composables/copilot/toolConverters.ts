import type { DraftChange } from '~/composables/useCopilotDraft'
import type { StepType } from '~/types/api'
import type { CopilotChatExtension, InlineAction, WorkflowProgressStatus, WorkflowTestResult, SelectOption } from '~/components/workflow-editor/copilot/types'

export interface ProposalChange {
  type: 'step:create' | 'step:update' | 'step:delete' | 'edge:create' | 'edge:delete'
  temp_id?: string
  step_id?: string
  edge_id?: string
  name?: string
  step_type?: string
  config?: Record<string, unknown>
  position?: { x: number; y: number }
  patch?: Record<string, unknown>
  source_id?: string
  target_id?: string
  source_port?: string
}

/**
 * Convert tool call to draft change (can return array for create_workflow_structure)
 */
export function toolCallToDraftChange(tool: string, args: Record<string, unknown>): DraftChange | DraftChange[] | null {
  switch (tool) {
    case 'create_step':
      return {
        type: 'step:create',
        tempId: `temp-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
        stepType: (args.step_type || args.type) as StepType,
        name: (args.name || args.step_name || 'New Step') as string,
        config: (args.config || {}) as object,
        position: {
          x: (args.position_x ?? args.x ?? 200) as number,
          y: (args.position_y ?? args.y ?? 200) as number,
        },
      }
    case 'update_step':
      return {
        type: 'step:update',
        stepId: (args.step_id || args.id) as string,
        patch: args as Partial<import('~/types/api').Step>,
      }
    case 'delete_step':
      return {
        type: 'step:delete',
        stepId: (args.step_id || args.id) as string,
      }
    case 'create_edge':
      return {
        type: 'edge:create',
        sourceId: (args.source_step_id || args.source_id || args.from) as string,
        targetId: (args.target_step_id || args.target_id || args.to) as string,
        sourcePort: args.source_port as string | undefined,
      }
    case 'delete_edge':
      return {
        type: 'edge:delete',
        edgeId: (args.edge_id || args.id) as string,
      }
    case 'create_workflow_structure': {
      const changes: DraftChange[] = []
      const steps = (args.steps || []) as Array<{
        temp_id?: string
        name?: string
        type?: string
        config?: object
        position?: { x?: number; y?: number }
      }>
      const connections = (args.connections || []) as Array<{
        from?: string
        to?: string
        from_port?: string
      }>

      for (const step of steps) {
        changes.push({
          type: 'step:create',
          tempId: step.temp_id || `temp-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
          stepType: step.type as StepType,
          name: step.name || 'New Step',
          config: step.config || {},
          position: {
            x: step.position?.x ?? 200,
            y: step.position?.y ?? 200,
          },
        })
      }

      for (const conn of connections) {
        changes.push({
          type: 'edge:create',
          sourceId: conn.from || '',
          targetId: conn.to || '',
          sourcePort: conn.from_port,
        })
      }

      return changes.length > 0 ? changes : null
    }
    default:
      return null
  }
}

/**
 * Convert DraftChange to ProposalChange format
 */
export function draftChangeToProposalChange(change: DraftChange): ProposalChange {
  switch (change.type) {
    case 'step:create':
      return {
        type: 'step:create',
        temp_id: change.tempId,
        name: change.name,
        step_type: change.stepType,
        config: change.config as Record<string, unknown>,
        position: change.position,
      }
    case 'step:update':
      return {
        type: 'step:update',
        step_id: change.stepId,
        patch: change.patch as Record<string, unknown>,
      }
    case 'step:delete':
      return {
        type: 'step:delete',
        step_id: change.stepId,
      }
    case 'edge:create':
      return {
        type: 'edge:create',
        source_id: change.sourceId,
        target_id: change.targetId,
        source_port: change.sourcePort,
      }
    case 'edge:delete':
      return {
        type: 'edge:delete',
        edge_id: change.edgeId,
      }
  }
}

// ============================================================================
// E2E Workflow Tool Converters
// ============================================================================

/**
 * Check if a tool is an E2E workflow tool
 */
export function isE2EWorkflowTool(toolName: string): boolean {
  return [
    'get_workflow_status',
    'configure_trigger',
    'list_required_credentials',
    'link_credential',
    'validate_workflow',
    'test_workflow',
    'publish_workflow',
  ].includes(toolName)
}

/**
 * Convert E2E workflow tool result to chat extension
 */
export function toolResultToChatExtension(
  tool: string,
  _args: Record<string, unknown>,
  result: Record<string, unknown>
): CopilotChatExtension | null {
  switch (tool) {
    case 'get_workflow_status': {
      // Convert result to WorkflowProgressStatus
      if (result.progress) {
        return {
          progressCard: result.progress as WorkflowProgressStatus,
        }
      }
      break
    }

    case 'configure_trigger': {
      // After trigger is configured, may want to show next action
      if (result.next_action) {
        return {
          inlineAction: result.next_action as InlineAction,
        }
      }
      break
    }

    case 'list_required_credentials': {
      // Show OAuth actions for required credentials
      const credentials = result.credentials as Array<{
        id: string
        service: string
        serviceName: string
        serviceIcon?: string
        stepId: string
        isConfigured: boolean
      }>

      if (credentials && credentials.length > 0) {
        const unconfigured = credentials.filter(c => !c.isConfigured)
        if (unconfigured.length > 0) {
          const first = unconfigured[0]
          return {
            inlineAction: {
              id: `oauth-${first.service}-${first.stepId}`,
              type: 'oauth',
              service: first.service,
              serviceName: first.serviceName,
              serviceIcon: first.serviceIcon,
            },
          }
        }
      }
      break
    }

    case 'test_workflow': {
      // Show test result card
      if (result.result) {
        return {
          testResult: result.result as WorkflowTestResult,
        }
      }
      break
    }

    case 'validate_workflow': {
      // Show validation result as inline action (confirm to publish or fix issues)
      if (result.isValid) {
        return {
          inlineAction: {
            id: 'publish',
            type: 'confirm',
            title: 'ワークフローを公開しますか？',
            confirmLabel: '公開する',
            cancelLabel: 'まだ変更する',
          },
        }
      }
      break
    }
  }

  return null
}

/**
 * Create trigger selection action
 */
export function createTriggerSelectAction(): InlineAction {
  return {
    id: 'configure-trigger',
    type: 'select',
    title: 'トリガータイプを選択してください',
    options: [
      { id: 'schedule', label: 'スケジュール', icon: '⏰', description: '毎日決まった時間に実行', recommended: true },
      { id: 'webhook', label: 'Webhook', icon: '🔗', description: '外部からのHTTPリクエストで実行' },
      { id: 'manual', label: '手動実行', icon: '👤', description: '手動でワークフローを開始' },
    ] as SelectOption[],
  }
}

/**
 * Create schedule form action
 */
export function createScheduleFormAction(): InlineAction {
  return {
    id: 'schedule-config',
    type: 'form',
    title: 'スケジュールを設定',
    fields: [
      { id: 'time', type: 'time', label: '実行時刻', defaultValue: '09:00', required: true },
      { id: 'timezone', type: 'timezone', label: 'タイムゾーン', defaultValue: 'Asia/Tokyo', required: true },
    ],
    submitLabel: '設定を適用',
  }
}

/**
 * Create test confirm action
 */
export function createTestConfirmAction(): InlineAction {
  return {
    id: 'run-test',
    type: 'test',
    title: 'テスト実行しますか？',
    testLabel: 'テストする',
    skipLabel: 'スキップ',
  }
}

/**
 * Create publish confirm action
 */
export function createPublishConfirmAction(): InlineAction {
  return {
    id: 'publish',
    type: 'confirm',
    title: 'このワークフローを公開しますか？',
    confirmLabel: '公開する',
    cancelLabel: 'まだ変更する',
  }
}
