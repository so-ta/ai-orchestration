import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import BlockCard from '../BlockCard.vue'
import type { BlockDefinition } from '~/types/api'

const messages = {
  blockEditor: {
    systemBlock: 'System',
    inherited: 'Inherited',
    inheritedFrom: 'Inherited from',
    versionHistory: 'Version History',
  },
  common: {
    edit: 'Edit',
    duplicate: 'Duplicate',
    delete: 'Delete',
  },
  categories: {
    ai: 'AI',
    flow: 'Flow',
    apps: 'Apps',
    custom: 'Custom',
  },
}

// Create i18n instance for tests
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {
    en: messages,
    ja: messages,
  },
})

// Mock useBlocks composable
vi.mock('~/composables/useBlocks', () => ({
  categoryConfig: {
    ai: { color: '#8B5CF6', nameKey: 'categories.ai' },
    flow: { color: '#3B82F6', nameKey: 'categories.flow' },
    apps: { color: '#10B981', nameKey: 'categories.apps' },
    custom: { color: '#F59E0B', nameKey: 'categories.custom' },
  },
  getBlockColor: () => '#8B5CF6',
}))

describe('BlockCard', () => {
  const mockBlock: BlockDefinition = {
    id: 'test-id',
    slug: 'test-block',
    name: 'Test Block',
    description: 'A test block description',
    category: 'custom',
    icon: '🧪',
    version: 2,
    enabled: true,
    is_system: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    config_schema: {},
    input_ports: [],
    output_ports: [],
    error_codes: [],
  }

  const createWrapper = (props = {}) => {
    return mount(BlockCard, {
      props: { block: mockBlock, ...props },
      global: {
        plugins: [i18n],
      },
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders block name and slug', () => {
    const wrapper = createWrapper()

    expect(wrapper.text()).toContain('Test Block')
    expect(wrapper.text()).toContain('test-block')
  })

  it('renders block description', () => {
    const wrapper = createWrapper()

    expect(wrapper.text()).toContain('A test block description')
  })

  it('renders block icon', () => {
    const wrapper = createWrapper()

    expect(wrapper.text()).toContain('🧪')
  })

  it('renders version badge', () => {
    const wrapper = createWrapper()

    expect(wrapper.text()).toContain('v2')
  })

  it('shows system badge for system blocks', () => {
    const systemBlock = { ...mockBlock, is_system: true }
    const wrapper = createWrapper({ block: systemBlock })

    expect(wrapper.text()).toContain('System')
  })

  it('shows inherited badge for blocks with parent', () => {
    const inheritedBlock = { ...mockBlock, parent_block_id: 'parent-id' }
    const wrapper = createWrapper({ block: inheritedBlock })

    expect(wrapper.text()).toContain('Inherited')
  })

  it('uses fallback icon when no icon provided', () => {
    const blockWithoutIcon = { ...mockBlock, icon: undefined }
    const wrapper = createWrapper({ block: blockWithoutIcon })

    expect(wrapper.text()).toContain('■')
  })

  it('emits edit event when edit button is clicked', async () => {
    const wrapper = createWrapper()

    const editButton = wrapper.findAll('.action-btn')[0]
    await editButton.trigger('click')

    expect(wrapper.emitted('edit')).toBeTruthy()
    expect(wrapper.emitted('edit')![0]).toEqual([mockBlock])
  })

  it('emits duplicate event when duplicate button is clicked', async () => {
    const wrapper = createWrapper()

    const duplicateButton = wrapper.findAll('.action-btn')[1]
    await duplicateButton.trigger('click')

    expect(wrapper.emitted('duplicate')).toBeTruthy()
    expect(wrapper.emitted('duplicate')![0]).toEqual([mockBlock])
  })

  it('emits viewVersions event when version history button is clicked', async () => {
    const wrapper = createWrapper()

    const versionButton = wrapper.findAll('.action-btn')[2]
    await versionButton.trigger('click')

    expect(wrapper.emitted('viewVersions')).toBeTruthy()
    expect(wrapper.emitted('viewVersions')![0]).toEqual([mockBlock])
  })

  it('emits delete event when delete button is clicked (non-system block)', async () => {
    const wrapper = createWrapper()

    const deleteButton = wrapper.findAll('.action-btn')[3]
    await deleteButton.trigger('click')

    expect(wrapper.emitted('delete')).toBeTruthy()
    expect(wrapper.emitted('delete')![0]).toEqual([mockBlock])
  })

  it('does not show delete button for system blocks', () => {
    const systemBlock = { ...mockBlock, is_system: true }
    const wrapper = createWrapper({ block: systemBlock })

    const deleteButton = wrapper.find('.action-btn.action-danger')
    expect(deleteButton.exists()).toBe(false)
  })

  it('applies is-system class for system blocks', () => {
    const systemBlock = { ...mockBlock, is_system: true }
    const wrapper = createWrapper({ block: systemBlock })

    expect(wrapper.find('.block-card').classes()).toContain('is-system')
  })

  it('renders with default version 1 when not specified', () => {
    const blockNoVersion = { ...mockBlock, version: undefined }
    const wrapper = createWrapper({ block: blockNoVersion })

    expect(wrapper.text()).toContain('v1')
  })
})
