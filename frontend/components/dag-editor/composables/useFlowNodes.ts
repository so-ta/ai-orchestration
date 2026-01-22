import type { Node } from '@vue-flow/core'
import type { Step, StepRun, BlockDefinition, BlockGroup, InputPort, OutputPort } from '~/types/api'
import {
  getGroupColor,
  getGroupIcon,
  getGroupTypeLabel,
  getGroupOutputPorts,
  getGroupZones,
  hasMultipleZones,
} from '../utils/dagHelpers'
import { getBlockIcon } from '~/composables/useBlockIcons'

// Preview state for Copilot changes
export interface PreviewState {
  addedStepIds: Set<string>
  modifiedStepIds: Set<string>
  deletedStepIds: Set<string>
  addedEdgeIds: Set<string>
  deletedEdgeIds: Set<string>
}

interface UseFlowNodesOptions {
  steps: Ref<Step[]>
  blockGroups: Ref<BlockGroup[] | undefined>
  blockDefinitions: Ref<BlockDefinition[] | undefined>
  stepRuns: Ref<StepRun[] | undefined>
  selectedStepId: Ref<string | null | undefined>
  selectedGroupId: Ref<string | null | undefined>
  previewState: Ref<PreviewState | null | undefined>
}

export function useFlowNodes(options: UseFlowNodesOptions) {
  const { steps, blockGroups, blockDefinitions, stepRuns, selectedStepId, selectedGroupId, previewState } = options

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

  // Create a map of block slug to input ports
  const inputPortsMap = computed(() => {
    const map = new Map<string, InputPort[]>()
    if (blockDefinitions.value) {
      for (const block of blockDefinitions.value) {
        map.set(block.slug, block.input_ports || [])
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
   * Get input ports for a step type
   */
  function getInputPorts(stepType: string): InputPort[] {
    const ports = inputPortsMap.value.get(stepType)
    if (ports && ports.length > 0) {
      return ports
    }
    return [{ name: 'input', label: 'Input', required: true }]
  }

  /**
   * Get output ports for a step type (or step for dynamic ports like switch)
   */
  function getOutputPorts(stepType: string, step?: Step): OutputPort[] {
    const config = step?.config as Record<string, unknown> | undefined

    // Special handling for switch blocks
    if (stepType === 'switch' && config?.cases) {
      const casesConfig = config.cases
      const dynamicPorts: OutputPort[] = []

      if (Array.isArray(casesConfig)) {
        for (const caseItem of casesConfig as Array<{ name: string; expression?: string; is_default?: boolean }>) {
          if (caseItem.is_default) {
            dynamicPorts.push({ name: 'default', label: 'Default', is_default: true })
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
          dynamicPorts.push({ name: caseName, label: caseName, is_default: false })
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
      const dynamicPorts: OutputPort[] = [{ name: 'default', label: 'Default', is_default: true }]

      for (const route of routes) {
        dynamicPorts.push({ name: route.name, label: route.name, is_default: false })
      }

      return dynamicPorts
    }

    let basePorts = outputPortsMap.value.get(stepType) || []
    if (basePorts.length === 0) {
      basePorts = [{ name: 'output', label: 'Output', is_default: true }]
    }

    const customOutputPorts = config?.custom_output_ports as string[] | undefined
    if (customOutputPorts && customOutputPorts.length > 0) {
      basePorts = customOutputPorts.map((name, index) => ({
        name,
        label: name,
        is_default: index === 0,
      }))
    }

    const enableErrorPort = config?.enable_error_port as boolean | undefined
    if (enableErrorPort) {
      const hasErrorPort = basePorts.some(p => p.name === 'error')
      if (!hasErrorPort) {
        basePorts = [...basePorts, { name: 'error', label: 'Error', is_default: false }]
      }
    }

    return basePorts
  }

  /**
   * Get preview class for a step based on Copilot preview state
   */
  function getPreviewClass(stepId: string): string | undefined {
    if (!previewState.value) return undefined
    if (previewState.value.addedStepIds?.has(stepId)) return 'preview-added'
    if (previewState.value.modifiedStepIds?.has(stepId)) return 'preview-modified'
    if (previewState.value.deletedStepIds?.has(stepId)) return 'preview-deleted'
    return undefined
  }

  /**
   * Get step type color
   */
  function getStepColor(type: string) {
    const colors: Record<string, string> = {
      start: '#10b981',
      llm: '#3b82f6',
      tool: '#22c55e',
      condition: '#f59e0b',
      switch: '#eab308',
      map: '#8b5cf6',
      join: '#6366f1',
      subflow: '#ec4899',
      loop: '#14b8a6',
      wait: '#64748b',
      function: '#f97316',
      router: '#a855f7',
      human_in_loop: '#ef4444',
      filter: '#06b6d4',
      split: '#0ea5e9',
      aggregate: '#0284c7',
      error: '#dc2626',
      note: '#9ca3af',
    }
    return colors[type] || '#64748b'
  }

  /**
   * Get step icon based on type
   */
  function getStepIcon(type: string): string {
    if (blockDefinitions.value) {
      const blockDef = blockDefinitions.value.find(b => b.slug === type)
      if (blockDef?.icon) {
        return blockDef.icon
      }
    }
    return getBlockIcon(type)
  }

  // Convert block groups to Vue Flow group nodes
  const groupNodes = computed<Node[]>(() => {
    if (!blockGroups.value) return []

    return blockGroups.value.map(group => ({
      id: `group_${group.id}`,
      type: 'group',
      position: { x: group.position_x, y: group.position_y },
      style: {
        width: `${group.width}px`,
        height: `${group.height}px`,
        backgroundColor: `${getGroupColor(group.type)}10`,
        borderColor: getGroupColor(group.type),
        borderWidth: '2px',
        borderStyle: 'dashed',
        borderRadius: '12px',
      },
      data: {
        label: group.name,
        type: group.type,
        group,
        isSelected: group.id === selectedGroupId.value,
        color: getGroupColor(group.type),
        icon: getGroupIcon(group.type),
        typeLabel: getGroupTypeLabel(group.type),
        outputPorts: getGroupOutputPorts(group.type),
        height: group.height,
        width: group.width,
        zones: getGroupZones(group.type),
        hasMultipleZones: hasMultipleZones(group.type),
      },
      zIndex: -1,
    }))
  })

  // Convert steps to Vue Flow nodes
  const stepNodes = computed<Node[]>(() => {
    return steps.value.map(step => {
      const stepRun = stepRunMap.value.get(step.id)
      const inputPorts = getInputPorts(step.type)
      const outputPorts = getOutputPorts(step.type, step)

      const parentGroupId = step.block_group_id || undefined
      let position = { x: step.position_x, y: step.position_y }

      if (parentGroupId && blockGroups.value) {
        const parentGroup = blockGroups.value.find(g => g.id === parentGroupId)
        if (parentGroup) {
          position = {
            x: step.position_x - parentGroup.position_x,
            y: step.position_y - parentGroup.position_y,
          }
        }
      }

      const previewClass = getPreviewClass(step.id)

      return {
        id: step.id,
        type: 'custom',
        position,
        parentNode: parentGroupId ? `group_${parentGroupId}` : undefined,
        class: previewClass || undefined,
        data: {
          label: step.name,
          type: step.type,
          step,
          isSelected: step.id === selectedStepId.value,
          stepRun,
          inputPorts,
          outputPorts,
          icon: getStepIcon(step.type),
          previewClass,
        },
      }
    })
  })

  // Combine group nodes and step nodes
  const nodes = computed<Node[]>(() => {
    return [...groupNodes.value, ...stepNodes.value]
  })

  return {
    nodes,
    groupNodes,
    stepNodes,
    stepRunMap,
    getInputPorts,
    getOutputPorts,
    getPreviewClass,
    getStepColor,
    getStepIcon,
  }
}
