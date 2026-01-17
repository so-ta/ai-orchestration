import { describe, it, expect } from 'vitest'
import { calculateLayout, calculateLayoutWithGroups } from '../graph-layout'
import type { Step, Edge, OutputPort, StepType, BlockGroup, BlockGroupType } from '~/types/api'

// Helper to create a minimal step
function createStep(id: string, type: string, x = 0, y = 0): Step {
  return {
    id,
    project_id: 'test-project',
    name: `Step ${id}`,
    type: type as Step['type'],
    config: {},
    position_x: x,
    position_y: y,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }
}

// Helper to create an edge
function createEdge(sourceId: string, targetId: string, sourcePort?: string): Edge {
  return {
    id: `edge-${sourceId}-${targetId}`,
    project_id: 'test-project',
    source_step_id: sourceId,
    target_step_id: targetId,
    source_port: sourcePort,
    created_at: new Date().toISOString(),
  }
}

// Helper to create an edge from a block group to a step
function createGroupEdge(sourceGroupId: string, targetId: string, sourcePort: string): Edge {
  return {
    id: `edge-group-${sourceGroupId}-${targetId}`,
    project_id: 'test-project',
    source_block_group_id: sourceGroupId,
    target_step_id: targetId,
    source_port: sourcePort,
    created_at: new Date().toISOString(),
  }
}

// Helper to create an edge from a block group to another block group
function createGroupToGroupEdge(sourceGroupId: string, targetGroupId: string, sourcePort: string): Edge {
  return {
    id: `edge-group-${sourceGroupId}-group-${targetGroupId}`,
    project_id: 'test-project',
    source_block_group_id: sourceGroupId,
    target_block_group_id: targetGroupId,
    source_port: sourcePort,
    created_at: new Date().toISOString(),
  }
}

// Helper to create a block group
function createBlockGroup(id: string, type: BlockGroupType): BlockGroup {
  return {
    id,
    project_id: 'test-project',
    name: `Group ${id}`,
    type,
    config: {},
    position_x: 0,
    position_y: 0,
    width: 200,
    height: 150,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }
}

describe('graph-layout', () => {
  describe('calculateLayout', () => {
    it('should calculate layout for simple linear graph', () => {
      const steps: Step[] = [
        createStep('1', 'start'),
        createStep('2', 'tool'),
        createStep('3', 'tool'),
      ]
      const edges: Edge[] = [
        createEdge('1', '2'),
        createEdge('2', '3'),
      ]

      const results = calculateLayout(steps, edges)

      expect(results).toHaveLength(3)
      // All steps should have positions
      results.forEach(result => {
        expect(result.x).toBeGreaterThanOrEqual(0)
        expect(result.y).toBeGreaterThanOrEqual(0)
      })
    })

    it('should snap positions to grid', () => {
      const steps: Step[] = [
        createStep('1', 'start'),
        createStep('2', 'tool'),
      ]
      const edges: Edge[] = [
        createEdge('1', '2'),
      ]

      const results = calculateLayout(steps, edges)

      // All positions should be divisible by 20 (grid size)
      results.forEach(result => {
        expect(result.x % 20).toBe(0)
        expect(result.y % 20).toBe(0)
      })
    })
  })

  describe('calculateLayout with port ordering', () => {
    it('should order target nodes by source port order', () => {
      // Create a source step with multiple outputs
      const steps: Step[] = [
        createStep('source', 'condition'),
        createStep('target1', 'tool'),
        createStep('target2', 'tool'),
        createStep('target3', 'tool'),
      ]

      // Edges from different ports (out of order intentionally)
      const edges: Edge[] = [
        createEdge('source', 'target3', 'error'),  // Should be last (bottom)
        createEdge('source', 'target1', 'out'),    // Should be first (top)
        createEdge('source', 'target2', 'out2'),   // Should be middle
      ]

      // Mock getOutputPorts that returns ports in order
      const getOutputPorts = (_stepType: StepType): OutputPort[] => {
        if (_stepType === 'condition') {
          return [
            { name: 'out', label: 'Output', is_default: false },
            { name: 'out2', label: 'Output 2', is_default: false },
            { name: 'error', label: 'Error', is_default: false },
          ]
        }
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      const results = calculateLayout(steps, edges, { getOutputPorts })

      // Find results for each target
      const target1Result = results.find(r => r.stepId === 'target1')!
      const target2Result = results.find(r => r.stepId === 'target2')!
      const target3Result = results.find(r => r.stepId === 'target3')!

      // target1 (out) should be above target2 (out2) which should be above target3 (error)
      expect(target1Result.y).toBeLessThan(target2Result.y)
      expect(target2Result.y).toBeLessThan(target3Result.y)
    })

    it('should skip edges without source_port', () => {
      const steps: Step[] = [
        createStep('source', 'condition'),
        createStep('target1', 'tool'),
        createStep('target2', 'tool'),
      ]

      // One edge has no source_port specified - should be skipped
      const edges: Edge[] = [
        createEdge('source', 'target1'),           // No source_port - will be skipped
        createEdge('source', 'target2', 'error'),  // Explicit error port
      ]

      const getOutputPorts = (_stepType: StepType): OutputPort[] => {
        if (_stepType === 'condition') {
          return [
            { name: 'out', label: 'Output', is_default: false },
            { name: 'error', label: 'Error', is_default: false },
          ]
        }
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      const results = calculateLayout(steps, edges, { getOutputPorts })

      // Both targets should have valid positions (layout still works)
      const target1Result = results.find(r => r.stepId === 'target1')!
      const target2Result = results.find(r => r.stepId === 'target2')!

      expect(target1Result).toBeDefined()
      expect(target2Result).toBeDefined()
    })

    it('should not reorder when source has only one output port', () => {
      const steps: Step[] = [
        createStep('source', 'tool'),
        createStep('target1', 'tool'),
        createStep('target2', 'tool'),
      ]

      const edges: Edge[] = [
        createEdge('source', 'target1', 'out'),
        createEdge('source', 'target2', 'out'),
      ]

      const getOutputPorts = (): OutputPort[] => {
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      // Should not throw and should return valid results
      const results = calculateLayout(steps, edges, { getOutputPorts })

      expect(results).toHaveLength(3)
      results.forEach(result => {
        expect(result.x).toBeGreaterThanOrEqual(0)
        expect(result.y).toBeGreaterThanOrEqual(0)
      })
    })

    it('should handle unknown port names by placing them last', () => {
      const steps: Step[] = [
        createStep('source', 'condition'),
        createStep('target1', 'tool'),
        createStep('target2', 'tool'),
        createStep('target3', 'tool'),
      ]

      const edges: Edge[] = [
        createEdge('source', 'target1', 'out'),
        createEdge('source', 'target2', 'unknown_port'),  // Unknown port
        createEdge('source', 'target3', 'error'),
      ]

      const getOutputPorts = (_stepType: StepType): OutputPort[] => {
        if (_stepType === 'condition') {
          return [
            { name: 'out', label: 'Output', is_default: false },
            { name: 'error', label: 'Error', is_default: false },
          ]
        }
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      const results = calculateLayout(steps, edges, { getOutputPorts })

      const target1Result = results.find(r => r.stepId === 'target1')!
      const target2Result = results.find(r => r.stepId === 'target2')!
      const target3Result = results.find(r => r.stepId === 'target3')!

      // out should be first, error second, unknown_port should be last
      expect(target1Result.y).toBeLessThan(target3Result.y)
      expect(target3Result.y).toBeLessThan(target2Result.y)
    })
  })

  describe('calculateLayoutWithGroups', () => {
    it('should return empty groups array when no block groups', () => {
      const steps: Step[] = [
        createStep('1', 'start'),
        createStep('2', 'tool'),
      ]
      const edges: Edge[] = [
        createEdge('1', '2'),
      ]

      const results = calculateLayoutWithGroups(steps, edges, [])

      expect(results.groups).toHaveLength(0)
      expect(results.steps).toHaveLength(2)
    })

    it('should apply port ordering for ungrouped steps', () => {
      const steps: Step[] = [
        createStep('source', 'condition'),
        createStep('target1', 'tool'),
        createStep('target2', 'tool'),
      ]

      const edges: Edge[] = [
        createEdge('source', 'target2', 'error'),
        createEdge('source', 'target1', 'out'),
      ]

      const getOutputPorts = (_stepType: StepType): OutputPort[] => {
        if (_stepType === 'condition') {
          return [
            { name: 'out', label: 'Output', is_default: false },
            { name: 'error', label: 'Error', is_default: false },
          ]
        }
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      const results = calculateLayoutWithGroups(steps, edges, [], { getOutputPorts })

      const target1Result = results.steps.find(r => r.stepId === 'target1')!
      const target2Result = results.steps.find(r => r.stepId === 'target2')!

      // target1 (out) should be above target2 (error)
      expect(target1Result.y).toBeLessThan(target2Result.y)
    })

    it('should apply port ordering for edges from block groups', () => {
      // Create a step inside the group and steps outside
      const innerStep = { ...createStep('inner', 'tool'), block_group_id: 'group1' }
      const target1 = createStep('target1', 'tool')
      const target2 = createStep('target2', 'tool')
      const steps: Step[] = [innerStep, target1, target2]

      // Block group with two output ports (out, error)
      const group = createBlockGroup('group1', 'parallel')
      const blockGroups: BlockGroup[] = [group]

      // Edges from group to ungrouped steps (out of order intentionally)
      const edges: Edge[] = [
        createGroupEdge('group1', 'target2', 'error'),  // Should be last (bottom)
        createGroupEdge('group1', 'target1', 'out'),    // Should be first (top)
      ]

      const getOutputPorts = (_stepType: StepType): OutputPort[] => {
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      const getGroupOutputPorts = (groupType: BlockGroupType): OutputPort[] => {
        if (groupType === 'parallel') {
          return [
            { name: 'out', label: 'Output', is_default: true },
            { name: 'error', label: 'Error', is_default: false },
          ]
        }
        return [{ name: 'out', label: 'Output', is_default: true }]
      }

      const results = calculateLayoutWithGroups(steps, edges, blockGroups, {
        getOutputPorts,
        getGroupOutputPorts,
      })

      const target1Result = results.steps.find(r => r.stepId === 'target1')!
      const target2Result = results.steps.find(r => r.stepId === 'target2')!

      // target1 (out) should be above target2 (error)
      expect(target1Result.y).toBeLessThan(target2Result.y)
    })

    it('should handle try_catch group with out and error ports', () => {
      const innerStep = { ...createStep('inner', 'tool'), block_group_id: 'group1' }
      const successHandler = createStep('success', 'tool')
      const errorHandler = createStep('error-handler', 'tool')
      const steps: Step[] = [innerStep, successHandler, errorHandler]

      const group = createBlockGroup('group1', 'try_catch')
      const blockGroups: BlockGroup[] = [group]

      // Error port comes first in edges (intentionally wrong order)
      const edges: Edge[] = [
        createGroupEdge('group1', 'error-handler', 'error'),
        createGroupEdge('group1', 'success', 'out'),
      ]

      const getOutputPorts = (): OutputPort[] => {
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      const getGroupOutputPorts = (groupType: BlockGroupType): OutputPort[] => {
        if (groupType === 'try_catch') {
          return [
            { name: 'out', label: 'Output', is_default: true },
            { name: 'error', label: 'Error', is_default: false },
          ]
        }
        return [{ name: 'out', label: 'Output', is_default: true }]
      }

      const results = calculateLayoutWithGroups(steps, edges, blockGroups, {
        getOutputPorts,
        getGroupOutputPorts,
      })

      const successResult = results.steps.find(r => r.stepId === 'success')!
      const errorResult = results.steps.find(r => r.stepId === 'error-handler')!

      // success (out) should be above error-handler (error)
      expect(successResult.y).toBeLessThan(errorResult.y)
    })

    it('should apply port ordering for edges from block groups to other groups', () => {
      // Create steps inside each group
      const innerStep1 = { ...createStep('inner1', 'tool'), block_group_id: 'group1' }
      const innerStep2 = { ...createStep('inner2', 'tool'), block_group_id: 'group2' }
      const innerStep3 = { ...createStep('inner3', 'tool'), block_group_id: 'group3' }
      const steps: Step[] = [innerStep1, innerStep2, innerStep3]

      // Create three groups
      const group1 = createBlockGroup('group1', 'parallel')
      const group2 = createBlockGroup('group2', 'parallel')
      const group3 = createBlockGroup('group3', 'parallel')
      const blockGroups: BlockGroup[] = [group1, group2, group3]

      // Edges from group1 to group2 (error port) and group3 (out port)
      // Intentionally in wrong order - error first, out second
      const edges: Edge[] = [
        createGroupToGroupEdge('group1', 'group2', 'error'),  // Should be last (bottom)
        createGroupToGroupEdge('group1', 'group3', 'out'),    // Should be first (top)
      ]

      const getOutputPorts = (): OutputPort[] => {
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      const getGroupOutputPorts = (groupType: BlockGroupType): OutputPort[] => {
        if (groupType === 'parallel') {
          return [
            { name: 'out', label: 'Output', is_default: true },
            { name: 'error', label: 'Error', is_default: false },
          ]
        }
        return [{ name: 'out', label: 'Output', is_default: true }]
      }

      const results = calculateLayoutWithGroups(steps, edges, blockGroups, {
        getOutputPorts,
        getGroupOutputPorts,
      })

      const group2Result = results.groups.find(r => r.groupId === 'group2')!
      const group3Result = results.groups.find(r => r.groupId === 'group3')!

      // group3 (out) should be above group2 (error)
      expect(group3Result.y).toBeLessThan(group2Result.y)
    })

    it('should apply port ordering for mixed targets (steps and groups) from same source group', () => {
      // Create source group with internal step
      const innerStep = { ...createStep('inner', 'tool'), block_group_id: 'source-group' }
      // Create target group with internal step
      const innerStep2 = { ...createStep('inner2', 'tool'), block_group_id: 'target-group' }
      // Create ungrouped target step
      const targetStep = createStep('target-step', 'tool')
      const steps: Step[] = [innerStep, innerStep2, targetStep]

      // Create source group (try_catch type with out and error ports)
      const sourceGroup = createBlockGroup('source-group', 'try_catch')
      // Create target group
      const targetGroup = createBlockGroup('target-group', 'foreach')
      const blockGroups: BlockGroup[] = [sourceGroup, targetGroup]

      // Edges from source-group:
      // - out port -> target-group (should be top)
      // - error port -> target-step (should be bottom)
      // Intentionally in wrong order - error first
      const edges: Edge[] = [
        createGroupEdge('source-group', 'target-step', 'error'),  // Should be last (bottom)
        createGroupToGroupEdge('source-group', 'target-group', 'out'),  // Should be first (top)
      ]

      const getOutputPorts = (): OutputPort[] => {
        return [{ name: 'out', label: 'Output', is_default: false }]
      }

      const getGroupOutputPorts = (groupType: BlockGroupType): OutputPort[] => {
        if (groupType === 'try_catch') {
          return [
            { name: 'out', label: 'Output', is_default: true },
            { name: 'error', label: 'Error', is_default: false },
          ]
        }
        return [{ name: 'out', label: 'Output', is_default: true }]
      }

      const results = calculateLayoutWithGroups(steps, edges, blockGroups, {
        getOutputPorts,
        getGroupOutputPorts,
      })

      const targetGroupResult = results.groups.find(r => r.groupId === 'target-group')!
      const targetStepResult = results.steps.find(r => r.stepId === 'target-step')!

      // target-group (out) should be above target-step (error)
      expect(targetGroupResult.y).toBeLessThan(targetStepResult.y)
    })
  })
})
