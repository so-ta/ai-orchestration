import { MarkerType, type Edge as FlowEdge } from '@vue-flow/core'
import type { Edge, StepRun, Step, BlockDefinition, OutputPort, BlockGroup } from '~/types/api'
import { GROUP_NODE_PREFIX, getGroupUuidFromNodeId, getGroupOutputPorts } from '../utils/dagHelpers'

// Preview state for Copilot changes
export interface PreviewState {
  addedStepIds: Set<string>
  modifiedStepIds: Set<string>
  deletedStepIds: Set<string>
  addedEdgeIds: Set<string>
  deletedEdgeIds: Set<string>
}

interface UseFlowEdgesOptions {
  edges: Ref<Edge[]>
  steps: Ref<Step[]>
  stepRuns: Ref<StepRun[] | undefined>
  blockDefinitions: Ref<BlockDefinition[] | undefined>
  blockGroups: Ref<BlockGroup[] | undefined>
  selectedEdgeId: Ref<string | null>
  previewState: Ref<PreviewState | null | undefined>
}

export function useFlowEdges(options: UseFlowEdgesOptions) {
  const { edges, steps, stepRuns, blockDefinitions, blockGroups, selectedEdgeId, previewState } = options

  // Create a map of step ID to step run for quick lookup
  const stepRunMap = computed(() => {
    const map = new Map<string, StepRun>()
    if (stepRuns.value) {
      for (const run of stepRuns.value) {
        map.set(run.step_id, run)
      }
    }
    return map
  })

  // Create a map of block slug to output ports
  const outputPortsMap = computed(() => {
    const map = new Map<string, OutputPort[]>()
    if (blockDefinitions.value) {
      for (const block of blockDefinitions.value) {
        map.set(block.slug, block.output_ports || [])
      }
    }
    return map
  })

  /**
   * Get output ports for a step type (or step for dynamic ports like switch)
   */
  function getOutputPorts(stepType: string, step?: Step): OutputPort[] {
    const config = step?.config as Record<string, unknown> | undefined

    // Special handling for switch blocks - generate dynamic ports from cases
    if (stepType === 'switch' && config?.cases) {
      const casesConfig = config.cases
      const dynamicPorts: OutputPort[] = []

      if (Array.isArray(casesConfig)) {
        for (const caseItem of casesConfig as Array<{ name: string; expression?: string; is_default?: boolean }>) {
          if (caseItem.is_default) {
            dynamicPorts.push({
              name: 'default',
              label: 'Default',
              is_default: true,
            })
          } else {
            dynamicPorts.push({
              name: caseItem.name || `case_${dynamicPorts.length + 1}`,
              label: caseItem.name || `Case ${dynamicPorts.length + 1}`,
              is_default: false,
            })
          }
        }
      } else if (typeof casesConfig === 'object' && casesConfig !== null) {
        for (const caseName of Object.keys(casesConfig as Record<string, unknown>)) {
          dynamicPorts.push({
            name: caseName,
            label: caseName,
            is_default: false,
          })
        }
      }

      if (dynamicPorts.length === 0) {
        return [{ name: 'default', label: 'Default', is_default: true }]
      }

      return dynamicPorts
    }

    // Special handling for router blocks
    if (stepType === 'router' && config?.routes) {
      const routes = config.routes as Array<{ name: string; description?: string }>
      const dynamicPorts: OutputPort[] = [
        { name: 'default', label: 'Default', is_default: true }
      ]

      for (const route of routes) {
        dynamicPorts.push({
          name: route.name,
          label: route.name,
          is_default: false,
        })
      }

      return dynamicPorts
    }

    // Build output ports dynamically from block definition and step config
    let basePorts = outputPortsMap.value.get(stepType) || []
    if (basePorts.length === 0) {
      basePorts = [{ name: 'output', label: 'Output', is_default: true }]
    }

    // Check for custom_output_ports in config
    const customOutputPorts = config?.custom_output_ports as string[] | undefined
    if (customOutputPorts && customOutputPorts.length > 0) {
      basePorts = customOutputPorts.map((name, index) => ({
        name,
        label: name,
        is_default: index === 0,
      }))
    }

    // Check for enable_error_port in config
    const enableErrorPort = config?.enable_error_port as boolean | undefined
    if (enableErrorPort) {
      const hasErrorPort = basePorts.some(p => p.name === 'error')
      if (!hasErrorPort) {
        basePorts = [
          ...basePorts,
          { name: 'error', label: 'Error', is_default: false }
        ]
      }
    }

    return basePorts
  }

  /**
   * Get edge color based on source port
   */
  function getEdgeColor(sourcePort?: string): string {
    if (!sourcePort) return '#94a3b8'

    const portColors: Record<string, string> = {
      'true': '#22c55e',
      'false': '#ef4444',
      'approved': '#22c55e',
      'rejected': '#ef4444',
      'timeout': '#f59e0b',
      'matched': '#22c55e',
      'unmatched': '#94a3b8',
      'loop': '#3b82f6',
      'complete': '#22c55e',
      'item': '#3b82f6',
      'out': '#22c55e',
      'default': '#94a3b8',
      'output': '#94a3b8',
    }

    return portColors[sourcePort] || '#6366f1'
  }

  /**
   * Get edge label text
   */
  function getEdgeLabel(sourcePort?: string, condition?: string): string | undefined {
    if (condition) return condition
    if (!sourcePort || sourcePort === 'output') return undefined

    const portLabels: Record<string, string> = {
      'true': 'Yes',
      'false': 'No',
      'approved': 'Approved',
      'rejected': 'Rejected',
      'timeout': 'Timeout',
      'matched': 'Matched',
      'unmatched': 'Unmatched',
      'loop': 'Loop',
      'complete': 'Complete',
      'item': 'Item',
      'out': 'Output',
      'default': 'Default',
    }

    return portLabels[sourcePort] || sourcePort
  }

  /**
   * Get edge data flow status based on step runs
   */
  function getEdgeFlowStatus(sourceStepId: string, targetStepId: string): { animated: boolean; status: 'idle' | 'flowing' | 'completed' | 'error' } {
    if (!stepRuns.value || stepRuns.value.length === 0) {
      return { animated: false, status: 'idle' }
    }

    const sourceRun = stepRunMap.value.get(sourceStepId)
    const targetRun = stepRunMap.value.get(targetStepId)

    if (sourceRun?.status === 'completed' && targetRun?.status === 'running') {
      return { animated: true, status: 'flowing' }
    }

    if (sourceRun?.status === 'completed' && targetRun?.status === 'completed') {
      return { animated: false, status: 'completed' }
    }

    if (targetRun?.status === 'failed') {
      return { animated: false, status: 'error' }
    }

    return { animated: false, status: 'idle' }
  }

  /**
   * Get preview class for an edge based on Copilot preview state
   */
  function getEdgePreviewClass(edgeId: string, sourceId: string, targetId: string): string | undefined {
    if (!previewState.value) return undefined

    if (previewState.value.deletedEdgeIds?.has(edgeId)) return 'preview-edge-deleted'

    const compositeKey = `${sourceId}->${targetId}`
    if (previewState.value.addedEdgeIds?.has(compositeKey)) return 'preview-edge-added'

    return undefined
  }

  /**
   * Get existing outgoing edges from a step
   */
  function getOutgoingEdgesFromStep(stepId: string): Edge[] {
    return edges.value.filter(e => e.source_step_id === stepId)
  }

  /**
   * Get existing outgoing edges from a group
   */
  function getOutgoingEdgesFromGroup(groupId: string): Edge[] {
    return edges.value.filter(e => e.source_block_group_id === groupId)
  }

  /**
   * Get default source port for a node (step or group)
   */
  function getDefaultSourcePort(nodeId: string, step: Step | undefined): string | undefined {
    if (step) {
      const ports = getOutputPorts(step.type, step)
      return ports.length > 0 ? ports[0].name : undefined
    }
    if (nodeId?.startsWith(GROUP_NODE_PREFIX)) {
      const groupUuid = getGroupUuidFromNodeId(nodeId)
      const group = blockGroups.value?.find(g => g.id === groupUuid)
      if (group) {
        const groupPorts = getGroupOutputPorts(group.type)
        return groupPorts.length > 0 ? groupPorts[0].name : undefined
      }
    }
    return undefined
  }

  /**
   * Get default target port for a node (step or group)
   */
  function getDefaultTargetPort(nodeId: string): string | undefined {
    const step = steps.value.find(s => s.id === nodeId)
    if (step) {
      // For now, just return 'input' as default
      return 'input'
    }
    if (nodeId?.startsWith(GROUP_NODE_PREFIX)) {
      return 'in'
    }
    return undefined
  }

  // Convert edges to Vue Flow edges
  const flowEdges = computed<FlowEdge[]>(() => {
    const result: FlowEdge[] = []

    for (const edge of edges.value) {
      const source = edge.source_step_id || (edge.source_block_group_id ? `group_${edge.source_block_group_id}` : '')
      const target = edge.target_step_id || (edge.target_block_group_id ? `group_${edge.target_block_group_id}` : '')

      if (!source || !target) continue

      const baseColor = getEdgeColor(edge.source_port)
      const flowStatus = edge.source_step_id && edge.target_step_id
        ? getEdgeFlowStatus(edge.source_step_id, edge.target_step_id)
        : { animated: false, status: 'idle' }

      let color = baseColor
      let strokeWidth = 2
      if (flowStatus.status === 'flowing') {
        color = '#3b82f6'
        strokeWidth = 3
      } else if (flowStatus.status === 'completed') {
        color = '#22c55e'
      } else if (flowStatus.status === 'error') {
        color = '#ef4444'
      }

      const isGroupEdge = edge.source_block_group_id || edge.target_block_group_id
      if (isGroupEdge) {
        color = '#8b5cf6'
      }

      const isSelected = selectedEdgeId.value === edge.id
      if (isSelected) {
        color = '#3b82f6'
        strokeWidth = 3
      }

      const edgeLabel = isGroupEdge ? undefined : getEdgeLabel(edge.source_port, edge.condition)
      const edgePreviewClass = getEdgePreviewClass(edge.id, source, target)

      result.push({
        id: edge.id,
        source,
        target,
        sourceHandle: edge.source_port || undefined,
        targetHandle: edge.target_port || undefined,
        type: 'smoothstep',
        animated: flowStatus.animated || isSelected,
        label: edgeLabel,
        labelBgStyle: { fill: 'white', fillOpacity: 0.9 },
        labelStyle: { fill: color, fontWeight: 500, fontSize: 11 },
        labelShowBg: true,
        style: { stroke: color, strokeWidth },
        markerEnd: { type: MarkerType.ArrowClosed, color },
        interactionWidth: 20,
        class: edgePreviewClass || undefined,
        data: { isSelected, edgeId: edge.id },
      })
    }

    return result
  })

  return {
    flowEdges,
    stepRunMap,
    getOutputPorts,
    getEdgeColor,
    getEdgeLabel,
    getEdgeFlowStatus,
    getEdgePreviewClass,
    getOutgoingEdgesFromStep,
    getOutgoingEdgesFromGroup,
    getDefaultSourcePort,
    getDefaultTargetPort,
  }
}
