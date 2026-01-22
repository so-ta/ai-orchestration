/**
 * Composable for tool display helpers (Claude Code-like verbose logging)
 */
export function useToolDisplay() {
  const { t } = useI18n()

  const toolKeyMap: Record<string, string> = {
    create_step: 'createStep',
    update_step: 'updateStep',
    delete_step: 'deleteStep',
    create_edge: 'createEdge',
    delete_edge: 'deleteEdge',
    create_workflow_structure: 'createWorkflowStructure',
    get_workflow: 'getWorkflow',
    list_steps: 'listSteps',
    get_step: 'getStep',
    list_blocks: 'listBlocks',
    get_block: 'getBlock',
  }

  /**
   * Convert tool name to human-readable label using i18n
   */
  function getToolLabel(tool: string): string {
    const key = toolKeyMap[tool]
    return key ? t(`copilot.tools.${key}`) : tool
  }

  /**
   * Get a human-readable description of tool arguments
   */
  function getToolDescription(tool: string, args?: Record<string, unknown>): string {
    if (!args) return ''

    switch (tool) {
      case 'create_step': {
        const name = args.name || args.step_name || ''
        const type = args.step_type || args.type || ''
        return name ? `「${name}」(${type})` : type ? `(${type})` : ''
      }
      case 'update_step': {
        const stepId = args.step_id || args.id || ''
        const name = args.name || ''
        const fields = Object.keys(args).filter(k => !['step_id', 'id'].includes(k))
        const fieldStr = fields.length > 0 ? `[${fields.join(', ')}]` : ''
        return name ? `「${name}」${fieldStr}` : stepId ? `ID: ${String(stepId).slice(0, 8)}...${fieldStr}` : fieldStr
      }
      case 'delete_step': {
        const stepId = args.step_id || args.id || ''
        return stepId ? `ID: ${String(stepId).slice(0, 8)}...` : ''
      }
      case 'create_edge': {
        const source = args.source_step_id || args.source_id || args.from || ''
        const target = args.target_step_id || args.target_id || args.to || ''
        if (source && target) {
          const sourceShort = String(source).slice(0, 8)
          const targetShort = String(target).slice(0, 8)
          return `${sourceShort}... → ${targetShort}...`
        }
        return ''
      }
      case 'delete_edge': {
        const edgeId = args.edge_id || args.id || ''
        return edgeId ? `ID: ${String(edgeId).slice(0, 8)}...` : ''
      }
      case 'get_workflow':
        return ''
      case 'list_steps':
        return ''
      case 'get_step': {
        const stepId = args.step_id || args.id || ''
        return stepId ? `ID: ${String(stepId).slice(0, 8)}...` : ''
      }
      case 'list_blocks': {
        const category = args.category || ''
        return category ? t('copilot.toolArgs.category', { category }) : ''
      }
      case 'get_block': {
        const slug = args.slug || ''
        return slug ? `「${slug}」` : ''
      }
      case 'create_workflow_structure': {
        const steps = Array.isArray(args.steps) ? args.steps : []
        const connections = Array.isArray(args.connections) ? args.connections : []
        return t('copilot.toolArgs.stepsAndConnections', { steps: steps.length, connections: connections.length })
      }
      default: {
        const keys = Object.keys(args).slice(0, 2)
        if (keys.length === 0) return ''
        return keys.map(k => `${k}: ${JSON.stringify(args[k]).slice(0, 20)}`).join(', ')
      }
    }
  }

  /**
   * Get a summary of tool result
   */
  function getToolResultSummary(tool: string, result?: unknown, isError?: boolean): string {
    if (isError) {
      if (typeof result === 'object' && result !== null) {
        const err = result as Record<string, unknown>
        return err.error ? String(err.error) : JSON.stringify(result).slice(0, 50)
      }
      return String(result).slice(0, 50)
    }

    if (!result) return ''

    switch (tool) {
      case 'create_step': {
        const r = result as Record<string, unknown>
        const id = r.id || r.step_id || ''
        return id ? t('copilot.toolResults.createCompleteWithId', { id: String(id).slice(0, 8) }) : t('copilot.toolResults.createComplete')
      }
      case 'update_step':
        return t('copilot.toolResults.updateComplete')
      case 'delete_step':
        return t('copilot.toolResults.deleteComplete')
      case 'create_edge':
        return t('copilot.toolResults.connectComplete')
      case 'delete_edge':
        return t('copilot.toolResults.deleteComplete')
      case 'get_workflow': {
        const r = result as Record<string, unknown>
        const name = r.name || ''
        const stepCount = Array.isArray(r.steps) ? r.steps.length : 0
        return name ? t('copilot.toolResults.workflowInfo', { name, stepCount }) : ''
      }
      case 'list_steps': {
        const r = result as Record<string, unknown>
        const steps = Array.isArray(r.steps) ? r.steps : Array.isArray(r) ? r : []
        return t('copilot.toolResults.stepsCount', { count: steps.length })
      }
      case 'get_step': {
        const r = result as Record<string, unknown>
        const name = r.name || ''
        const type = r.type || ''
        return name ? `「${name}」(${type})` : type ? `(${type})` : ''
      }
      case 'list_blocks': {
        const r = result as Record<string, unknown>
        const blocks = Array.isArray(r.blocks) ? r.blocks : Array.isArray(r) ? r : []
        return t('copilot.toolResults.blocksCount', { count: blocks.length })
      }
      case 'get_block': {
        const r = result as Record<string, unknown>
        const name = r.name || ''
        return name ? `「${name}」` : ''
      }
      case 'create_workflow_structure': {
        const r = result as Record<string, unknown>
        const createdSteps = Array.isArray(r.created_steps) ? r.created_steps.length : 0
        const createdEdges = Array.isArray(r.created_edges) ? r.created_edges.length : 0
        return t('copilot.toolResults.structureCreated', { steps: createdSteps, edges: createdEdges })
      }
      default:
        return ''
    }
  }

  return {
    getToolLabel,
    getToolDescription,
    getToolResultSummary,
  }
}
