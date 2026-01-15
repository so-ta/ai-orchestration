import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { createI18n } from 'vue-i18n'

// Create i18n instance for tests
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: {
      workflows: {
        runDialog: {
          title: 'Run Workflow',
          inputTitle: 'Input',
          inputDescription: 'Input for {stepName}',
          noInputRequired: 'No input required',
          schemaDetails: 'Schema Details',
          run: 'Run',
        },
      },
      common: {
        cancel: 'Cancel',
      },
    },
    ja: {
      workflows: {
        runDialog: {
          title: 'ワークフロー実行',
          inputTitle: '入力',
          inputDescription: '{stepName}への入力',
          noInputRequired: '入力は不要です',
          schemaDetails: 'スキーマ詳細',
          run: '実行',
        },
      },
      common: {
        cancel: 'キャンセル',
      },
    },
  },
})

// Mock UiModal component
const MockUiModal = defineComponent({
  name: 'UiModal',
  props: ['show', 'title', 'size'],
  emits: ['close'],
  setup(props, { slots }) {
    return () => props.show ? h('div', { class: 'modal-mock' }, [
      h('div', { class: 'modal-header' }, props.title),
      h('div', { class: 'modal-body' }, slots.default?.()),
      h('div', { class: 'modal-footer' }, slots.footer?.()),
    ]) : null
  }
})

// Mock DynamicConfigForm component
const MockDynamicConfigForm = defineComponent({
  name: 'DynamicConfigForm',
  props: ['modelValue', 'schema', 'disabled'],
  emits: ['update:modelValue', 'validation-change'],
  setup() {
    return () => h('div', { class: 'dynamic-config-form-mock' })
  }
})

// Import after mocking
import RunDialog from '../RunDialog.vue'
import type { Step, BlockDefinition } from '~/types/api'

describe('RunDialog', () => {
  // Steps with input_schema defined in Start step's config
  const mockSteps: Step[] = [
    {
      id: 'step-start',
      workflow_id: 'workflow-1',
      name: 'Start',
      type: 'start',
      config: {
        input_schema: {
          type: 'object',
          properties: {
            message: { type: 'string', title: 'Message' },
          },
          required: ['message'],
        },
      },
      position_x: 100,
      position_y: 100,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 'step-llm',
      workflow_id: 'workflow-1',
      name: 'LLM Step',
      type: 'llm',
      config: { provider: 'openai', model: 'gpt-4' },
      position_x: 100,
      position_y: 250,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ]

  // Steps without input_schema in Start step's config
  const mockStepsWithoutSchema: Step[] = [
    {
      id: 'step-start',
      workflow_id: 'workflow-1',
      name: 'Start',
      type: 'start',
      config: {},
      position_x: 100,
      position_y: 100,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 'step-llm',
      workflow_id: 'workflow-1',
      name: 'LLM Step',
      type: 'llm',
      config: { provider: 'openai', model: 'gpt-4' },
      position_x: 100,
      position_y: 250,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ]

  const mockEdges = [
    {
      source_step_id: 'step-start',
      target_step_id: 'step-llm',
    },
  ]

  const mockBlocks: BlockDefinition[] = [
    {
      id: 'block-start',
      slug: 'start',
      name: 'Start',
      category: 'control',
      description: 'Start block',
      is_system: true,
      config_schema: {},
      input_ports: [],
      output_ports: [],
      error_codes: [],
      enabled: true,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
    {
      id: 'block-llm',
      slug: 'llm',
      name: 'LLM',
      category: 'ai',
      description: 'LLM block',
      is_system: true,
      config_schema: {},
      input_schema: {
        type: 'object',
        properties: {
          prompt: { type: 'string', title: 'Prompt' },
        },
        required: ['prompt'],
      },
      input_ports: [],
      output_ports: [],
      error_codes: [],
      enabled: true,
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ]

  const defaultProps = {
    show: true,
    workflowId: 'workflow-1',
    workflowName: 'Test Workflow',
    steps: mockSteps,
    edges: mockEdges,
    blocks: mockBlocks,
  }

  const createWrapper = (props = {}) => {
    return mount(RunDialog, {
      props: { ...defaultProps, ...props },
      global: {
        plugins: [i18n],
        stubs: {
          UiModal: MockUiModal,
          DynamicConfigForm: MockDynamicConfigForm,
        },
      },
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('mounting', () => {
    it('should mount with required props', () => {
      const wrapper = createWrapper()
      expect(wrapper.exists()).toBe(true)
    })

    it('should not render when show is false', () => {
      const wrapper = createWrapper({ show: false })
      expect(wrapper.find('.modal-mock').exists()).toBe(false)
    })

    it('should render when show is true', () => {
      const wrapper = createWrapper()
      expect(wrapper.find('.modal-mock').exists()).toBe(true)
    })
  })

  describe('props', () => {
    it('should display workflow name', () => {
      const wrapper = createWrapper()
      expect(wrapper.text()).toContain('Test Workflow')
    })

    it('should accept steps prop', () => {
      const wrapper = createWrapper()
      expect(wrapper.props('steps')).toEqual(mockSteps)
    })

    it('should accept edges prop', () => {
      const wrapper = createWrapper()
      expect(wrapper.props('edges')).toEqual(mockEdges)
    })

    it('should accept blocks prop', () => {
      const wrapper = createWrapper()
      expect(wrapper.props('blocks')).toEqual(mockBlocks)
    })
  })

  describe('events', () => {
    it('should emit close event when cancel button is clicked', async () => {
      const wrapper = createWrapper()
      const cancelButton = wrapper.find('.btn-secondary')
      await cancelButton.trigger('click')
      expect(wrapper.emitted('close')).toBeTruthy()
    })

    it('should emit run event when run button is clicked', async () => {
      const wrapper = createWrapper()
      const runButton = wrapper.find('.btn-primary')
      await runButton.trigger('click')
      expect(wrapper.emitted('run')).toBeTruthy()
      expect(wrapper.emitted('run')![0]).toEqual([{}])
    })
  })

  describe('input schema handling', () => {
    it('should show input form when Start step has input_schema in config', () => {
      // mockSteps has input_schema defined in Start step's config
      const wrapper = createWrapper()
      expect(wrapper.find('.input-section').exists()).toBe(true)
      expect(wrapper.find('.dynamic-config-form-mock').exists()).toBe(true)
    })

    it('should show no input message when Start step has no input_schema', () => {
      // Use steps without input_schema in Start step's config
      const wrapper = createWrapper({ steps: mockStepsWithoutSchema })
      expect(wrapper.find('.no-input-message').exists()).toBe(true)
    })
  })

  describe('workflow name display', () => {
    it('should display workflow name in input description', () => {
      const wrapper = createWrapper()
      // The input description should mention the workflow name
      expect(wrapper.text()).toContain('Test Workflow')
    })

    it('should handle missing start step', () => {
      const stepsWithoutStart: Step[] = [
        {
          id: 'step-llm',
          workflow_id: 'workflow-1',
          name: 'LLM Step',
          type: 'llm',
          config: {},
          position_x: 100,
          position_y: 100,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: '2024-01-01T00:00:00Z',
        },
      ]

      const wrapper = createWrapper({ steps: stepsWithoutStart })
      // Should still mount without error
      expect(wrapper.exists()).toBe(true)
      // Should show no input message when start step is missing
      expect(wrapper.find('.no-input-message').exists()).toBe(true)
    })

    it('should show no input message when Start step has no input_schema', () => {
      const wrapper = createWrapper({ steps: mockStepsWithoutSchema })
      // Should show no input message when Start step has no input_schema
      expect(wrapper.find('.no-input-message').exists()).toBe(true)
    })
  })

  describe('loading state', () => {
    it('should disable buttons when loading', async () => {
      const wrapper = createWrapper()

      // Trigger run to set loading state
      const runButton = wrapper.find('.btn-primary')
      await runButton.trigger('click')

      // Both buttons should be disabled during loading
      await wrapper.vm.$nextTick()
      expect(wrapper.find('.btn-secondary').attributes('disabled')).toBeDefined()
      expect(wrapper.find('.btn-primary').attributes('disabled')).toBeDefined()
    })
  })

  describe('form reset', () => {
    it('should reset form when dialog is reopened', async () => {
      const wrapper = createWrapper()

      // Trigger run
      await wrapper.find('.btn-primary').trigger('click')

      // Close and reopen
      await wrapper.setProps({ show: false })
      await wrapper.setProps({ show: true })

      // Loading should be reset
      expect(wrapper.find('.btn-primary').attributes('disabled')).toBeUndefined()
    })
  })
})
