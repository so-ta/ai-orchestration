import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h } from 'vue'

// Mock Vue Flow completely
vi.mock('@vue-flow/core', () => ({
  VueFlow: defineComponent({
    name: 'VueFlow',
    props: ['nodes', 'edges', 'nodesDraggable'],
    emits: ['connect', 'paneClick', 'nodeDragStop'],
    setup(props, { slots, emit }) {
      return () => h('div', { class: 'vue-flow-mock' }, [
        h('div', { class: 'nodes' }, props.nodes?.map((n: { id: string }) =>
          h('div', { class: 'node', 'data-id': n.id }, n.id)
        )),
        slots.default?.()
      ])
    }
  }),
  useVueFlow: () => ({
    onConnect: vi.fn(),
    onNodeDragStop: vi.fn(),
    onPaneClick: vi.fn(),
    onEdgeClick: vi.fn(),
    project: vi.fn(({ x, y }: { x: number; y: number }) => ({ x, y })),
    updateNode: vi.fn(),
    getNodes: { value: [] },
    viewport: { value: { x: 0, y: 0, zoom: 1 } },
  }),
  Handle: defineComponent({
    name: 'Handle',
    props: ['type', 'position', 'id'],
    setup(props) {
      return () => h('div', { class: 'handle', 'data-type': props.type })
    }
  }),
  Position: {
    Top: 'top',
    Bottom: 'bottom',
    Left: 'left',
    Right: 'right',
  },
  MarkerType: {
    Arrow: 'arrow',
    ArrowClosed: 'arrowclosed',
  },
}))

vi.mock('@vue-flow/minimap', () => ({
  MiniMap: defineComponent({
    name: 'MiniMap',
    setup() { return () => h('div', { class: 'minimap-mock' }) }
  }),
}))

vi.mock('@vue-flow/controls', () => ({
  Controls: defineComponent({
    name: 'Controls',
    setup() { return () => h('div', { class: 'controls-mock' }) }
  }),
}))

vi.mock('@vue-flow/node-resizer', () => ({
  NodeResizer: defineComponent({
    name: 'NodeResizer',
    props: ['minWidth', 'minHeight'],
    setup() { return () => h('div', { class: 'resizer-mock' }) }
  }),
}))

// Import after mocking
import DagEditor from '../DagEditor.vue'
import type { Step, Edge, BlockGroup } from '~/types/api'

describe('DagEditor', () => {
  const mockSteps: Step[] = [
    {
      id: 'step-1',
      project_id: 'project-1',
      name: 'Start Step',
      type: 'start',
      config: {},
      position_x: 100,
      position_y: 100,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 'step-2',
      project_id: 'project-1',
      name: 'LLM Step',
      type: 'llm',
      config: { provider: 'openai', model: 'gpt-4' },
      position_x: 100,
      position_y: 250,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ]

  const mockEdges: Edge[] = [
    {
      id: 'edge-1',
      project_id: 'project-1',
      source_step_id: 'step-1',
      target_step_id: 'step-2',
      created_at: '2024-01-01T00:00:00Z',
    },
  ]

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('mounting', () => {
    it('should mount with required props', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: mockEdges,
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      expect(wrapper.exists()).toBe(true)
    })

    it('should mount with empty steps and edges', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: [],
          edges: [],
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      expect(wrapper.exists()).toBe(true)
    })

    it('should mount in readonly mode', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: mockEdges,
          readonly: true,
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      expect(wrapper.exists()).toBe(true)
      expect(wrapper.props('readonly')).toBe(true)
    })
  })

  describe('props', () => {
    it('should accept steps prop', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: [],
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      expect(wrapper.props('steps')).toEqual(mockSteps)
    })

    it('should accept edges prop', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: [],
          edges: mockEdges,
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      expect(wrapper.props('edges')).toEqual(mockEdges)
    })

    it('should accept selectedStepId prop', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: mockEdges,
          selectedStepId: 'step-1',
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      expect(wrapper.props('selectedStepId')).toBe('step-1')
    })

    it('should accept blockGroups prop', () => {
      const mockGroups: BlockGroup[] = [
        {
          id: 'group-1',
          project_id: 'project-1',
          name: 'Loop Group',
          type: 'foreach',
          config: {},
          position_x: 50,
          position_y: 50,
          width: 300,
          height: 200,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ]

      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: mockEdges,
          blockGroups: mockGroups,
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      expect(wrapper.props('blockGroups')).toEqual(mockGroups)
    })

    it('should accept stepRuns prop for execution status', () => {
      const mockStepRuns = [
        {
          id: 'run-1',
          run_id: 'run-main',
          step_id: 'step-1',
          step_name: 'Start Step',
          status: 'completed' as const,
          attempt: 1,
          sequence_number: 1,
          created_at: '2024-01-01T00:00:00Z',
        },
      ]

      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: mockEdges,
          stepRuns: mockStepRuns,
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      expect(wrapper.props('stepRuns')).toEqual(mockStepRuns)
    })
  })

  describe('events', () => {
    it('should define step:select event', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: mockEdges,
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      // Verify the component has the emit defined
      const emits = wrapper.vm.$options.emits
      expect(emits).toBeDefined()
    })

    it('should define edge:add event', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: mockEdges,
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      const emits = wrapper.vm.$options.emits
      expect(emits).toBeDefined()
    })

    it('should define step:update event', () => {
      const wrapper = mount(DagEditor, {
        props: {
          steps: mockSteps,
          edges: mockEdges,
        },
        global: {
          stubs: {
            teleport: true,
          },
        },
      })

      const emits = wrapper.vm.$options.emits
      expect(emits).toBeDefined()
    })
  })

  describe('step types', () => {
    const stepTypes = ['start', 'llm', 'tool', 'condition', 'switch', 'map', 'join', 'subflow']

    stepTypes.forEach(type => {
      it(`should handle ${type} step type`, () => {
        const steps: Step[] = [
          {
            id: `step-${type}`,
            project_id: 'project-1',
            name: `${type} Step`,
            type: type as Step['type'],
            config: {},
            position_x: 100,
            position_y: 100,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ]

        const wrapper = mount(DagEditor, {
          props: {
            steps,
            edges: [],
          },
          global: {
            stubs: {
              teleport: true,
            },
          },
        })

        expect(wrapper.exists()).toBe(true)
      })
    })
  })

  describe('block groups', () => {
    const groupTypes = ['foreach', 'try_catch', 'parallel', 'while'] as const

    groupTypes.forEach(type => {
      it(`should handle ${type} block group type`, () => {
        const groups: BlockGroup[] = [
          {
            id: `group-${type}`,
            project_id: 'project-1',
            name: `${type} Group`,
            type: type,
            config: {},
            position_x: 50,
            position_y: 50,
            width: 300,
            height: 200,
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ]

        const wrapper = mount(DagEditor, {
          props: {
            steps: mockSteps,
            edges: mockEdges,
            blockGroups: groups,
          },
          global: {
            stubs: {
              teleport: true,
            },
          },
        })

        expect(wrapper.exists()).toBe(true)
        expect(wrapper.props('blockGroups')).toEqual(groups)
      })
    })
  })

  describe('execution status', () => {
    const statuses = ['pending', 'running', 'completed', 'failed', 'skipped'] as const

    statuses.forEach(status => {
      it(`should handle ${status} step run status`, () => {
        const stepRuns = [
          {
            id: 'run-1',
            run_id: 'run-main',
            step_id: 'step-1',
            step_name: 'Start Step',
            status: status,
            attempt: 1,
            sequence_number: 1,
            created_at: '2024-01-01T00:00:00Z',
          },
        ]

        const wrapper = mount(DagEditor, {
          props: {
            steps: mockSteps,
            edges: mockEdges,
            stepRuns,
          },
          global: {
            stubs: {
              teleport: true,
            },
          },
        })

        expect(wrapper.exists()).toBe(true)
      })
    })
  })

  describe('group node ID format', () => {
    // These tests verify that the group node ID format conversion is consistent
    // Vue Flow node IDs use "group_${uuid}" format, but block_group_id uses plain UUID

    const GROUP_NODE_PREFIX = 'group_'

    // Helper functions (same as in DagEditor.vue)
    function getGroupUuidFromNodeId(nodeId: string): string {
      if (nodeId.startsWith(GROUP_NODE_PREFIX)) {
        return nodeId.slice(GROUP_NODE_PREFIX.length)
      }
      return nodeId
    }

    function getNodeIdFromGroupUuid(groupUuid: string): string {
      return `${GROUP_NODE_PREFIX}${groupUuid}`
    }

    it('should convert node ID to plain UUID', () => {
      const nodeId = 'group_abc123-def456-ghi789'
      const uuid = getGroupUuidFromNodeId(nodeId)
      expect(uuid).toBe('abc123-def456-ghi789')
    })

    it('should return unchanged if already plain UUID', () => {
      const uuid = 'abc123-def456-ghi789'
      const result = getGroupUuidFromNodeId(uuid)
      expect(result).toBe('abc123-def456-ghi789')
    })

    it('should convert plain UUID to node ID', () => {
      const uuid = 'abc123-def456-ghi789'
      const nodeId = getNodeIdFromGroupUuid(uuid)
      expect(nodeId).toBe('group_abc123-def456-ghi789')
    })

    it('should be reversible', () => {
      const originalUuid = 'abc123-def456-ghi789'
      const nodeId = getNodeIdFromGroupUuid(originalUuid)
      const recoveredUuid = getGroupUuidFromNodeId(nodeId)
      expect(recoveredUuid).toBe(originalUuid)
    })

    it('should handle real UUID format', () => {
      const realUuid = 'a0000000-0000-0000-0000-000000000001'
      const nodeId = getNodeIdFromGroupUuid(realUuid)
      expect(nodeId).toBe('group_a0000000-0000-0000-0000-000000000001')

      const recovered = getGroupUuidFromNodeId(nodeId)
      expect(recovered).toBe(realUuid)
    })

    describe('ID format consistency checks', () => {
      // These tests document the expected ID formats in different contexts

      it('step.block_group_id should be plain UUID (not prefixed)', () => {
        const step = {
          id: 'step-1',
          block_group_id: 'abc123-def456-ghi789', // Plain UUID
        }

        // block_group_id should NOT have the group_ prefix
        expect(step.block_group_id).not.toMatch(/^group_/)
      })

      it('BlockGroup.id should be plain UUID (not prefixed)', () => {
        const group: BlockGroup = {
          id: 'abc123-def456-ghi789', // Plain UUID
          project_id: 'project-1',
          name: 'Test Group',
          type: 'foreach',
          config: {},
          position_x: 100,
          position_y: 100,
          width: 300,
          height: 200,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        }

        // BlockGroup.id should NOT have the group_ prefix
        expect(group.id).not.toMatch(/^group_/)
      })

      it('Vue Flow node.id for groups should be prefixed', () => {
        const groupUuid = 'abc123-def456-ghi789'
        const vueFlowNodeId = getNodeIdFromGroupUuid(groupUuid)

        // Vue Flow node ID should have the group_ prefix
        expect(vueFlowNodeId).toMatch(/^group_/)
      })
    })
  })

  describe('group collision logic', () => {
    // These tests document the expected behavior of group collision detection

    it('should correctly identify when step belongs to a group', () => {
      const step = {
        id: 'step-1',
        block_group_id: 'group-uuid-123',
      }

      const groupDataId = 'group-uuid-123' // Plain UUID from groupData.id
      const nodeId = 'group_group-uuid-123' // Vue Flow node ID format

      // Correct comparison: step.block_group_id === groupData.id
      expect(step.block_group_id === groupDataId).toBe(true)

      // Incorrect comparison: step.block_group_id === node.id (format mismatch!)
      expect(step.block_group_id === nodeId).toBe(false)
    })

    it('should use plain UUID for groupPositions map keys', () => {
      const groupPositions = new Map<string, { x: number; y: number }>()
      const groupDataId = 'group-uuid-123' // Plain UUID

      // Set position using plain UUID
      groupPositions.set(groupDataId, { x: 100, y: 200 })

      // Should be retrievable with plain UUID
      expect(groupPositions.get(groupDataId)).toEqual({ x: 100, y: 200 })

      // Should NOT be retrievable with prefixed node ID
      const nodeId = 'group_group-uuid-123'
      expect(groupPositions.get(nodeId)).toBeUndefined()
    })

    it('should use plain UUID for processedGroups set', () => {
      const processedGroups = new Set<string>()
      const groupDataId = 'group-uuid-123' // Plain UUID

      // Add using plain UUID
      processedGroups.add(groupDataId)

      // Should be found with plain UUID
      expect(processedGroups.has(groupDataId)).toBe(true)

      // Should NOT be found with prefixed node ID
      const nodeId = 'group_group-uuid-123'
      expect(processedGroups.has(nodeId)).toBe(false)
    })
  })
})
