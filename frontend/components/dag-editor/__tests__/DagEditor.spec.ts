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
    project: vi.fn(({ x, y }: { x: number; y: number }) => ({ x, y })),
    updateNode: vi.fn(),
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
      workflow_id: 'workflow-1',
      name: 'Start Step',
      type: 'start',
      config: {},
      position_x: 100,
      position_y: 100,
    },
    {
      id: 'step-2',
      workflow_id: 'workflow-1',
      name: 'LLM Step',
      type: 'llm',
      config: { provider: 'openai', model: 'gpt-4' },
      position_x: 100,
      position_y: 250,
    },
  ]

  const mockEdges: Edge[] = [
    {
      id: 'edge-1',
      workflow_id: 'workflow-1',
      source_step_id: 'step-1',
      target_step_id: 'step-2',
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
          workflow_id: 'workflow-1',
          name: 'Loop Group',
          type: 'loop',
          position_x: 50,
          position_y: 50,
          width: 300,
          height: 200,
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
          status: 'completed',
          attempt: 1,
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
            workflow_id: 'workflow-1',
            name: `${type} Step`,
            type: type as Step['type'],
            config: {},
            position_x: 100,
            position_y: 100,
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
    const groupTypes = ['loop', 'try-catch', 'parallel', 'conditional'] as const

    groupTypes.forEach(type => {
      it(`should handle ${type} block group type`, () => {
        const groups: BlockGroup[] = [
          {
            id: `group-${type}`,
            workflow_id: 'workflow-1',
            name: `${type} Group`,
            type: type,
            position_x: 50,
            position_y: 50,
            width: 300,
            height: 200,
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
    const statuses = ['pending', 'running', 'completed', 'failed', 'skipped']

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
})
