import { describe, it, expect } from 'vitest'
import { calculateLayout, calculateLayoutWithGroups } from '../graph-layout'
import type { Step, Edge, OutputPort, StepType } from '~/types/api'

// Helper to create a minimal step
function createStep(id: string, type: string, x = 0, y = 0): Step {
  return {
    id,
    workflow_id: 'test-workflow',
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
    workflow_id: 'test-workflow',
    source_step_id: sourceId,
    target_step_id: targetId,
    source_port: sourcePort,
    created_at: new Date().toISOString(),
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
      const getOutputPorts = (stepType: StepType): OutputPort[] => {
        if (stepType === 'condition') {
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

    it('should handle missing source_port by defaulting to "out"', () => {
      const steps: Step[] = [
        createStep('source', 'condition'),
        createStep('target1', 'tool'),
        createStep('target2', 'tool'),
      ]

      // One edge has no source_port specified
      const edges: Edge[] = [
        createEdge('source', 'target1'),           // No source_port, defaults to 'out'
        createEdge('source', 'target2', 'error'),  // Explicit error port
      ]

      const getOutputPorts = (stepType: StepType): OutputPort[] => {
        if (stepType === 'condition') {
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

      // target1 (defaulting to 'out') should be above target2 (error)
      expect(target1Result.y).toBeLessThan(target2Result.y)
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

      const getOutputPorts = (stepType: StepType): OutputPort[] => {
        if (stepType === 'condition') {
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

      const getOutputPorts = (stepType: StepType): OutputPort[] => {
        if (stepType === 'condition') {
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
  })
})
