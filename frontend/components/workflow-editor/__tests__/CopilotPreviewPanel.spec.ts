import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import CopilotPreviewPanel from '../CopilotPreviewPanel.vue'
import type { ReadonlyCopilotDraft, DraftStepCreate, DraftStepUpdate, DraftStepDelete, DraftEdgeCreate, DraftEdgeDelete } from '~/composables/useCopilotDraft'

const messages = {
  copilot: {
    preview: {
      title: 'Preview Changes',
      apply: 'Apply',
      discard: 'Discard',
      modify: 'Request Changes',
      modifyPlaceholder: 'Describe the changes you want...',
      sendModification: 'Send',
      connection: 'Connection',
      stepDeleted: 'Step deleted',
    },
  },
  common: {
    cancel: 'Cancel',
  },
  blockTypes: {
    llm: 'LLM',
    http: 'HTTP',
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

describe('CopilotPreviewPanel', () => {
  const createMockDraft = (changes: (DraftStepCreate | DraftStepUpdate | DraftStepDelete | DraftEdgeCreate | DraftEdgeDelete)[] = []): ReadonlyCopilotDraft => ({
    id: 'draft-123',
    status: 'previewing',
    description: 'Add LLM block and connect to existing workflow',
    changes,
    createdAt: Date.now(),
  })

  const createWrapper = (draft: ReadonlyCopilotDraft) => {
    return mount(CopilotPreviewPanel, {
      props: { draft },
      global: {
        plugins: [i18n],
      },
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders draft description', () => {
    const draft = createMockDraft()
    const wrapper = createWrapper(draft)

    expect(wrapper.text()).toContain('Add LLM block and connect to existing workflow')
  })

  it('renders preview title', () => {
    const draft = createMockDraft()
    const wrapper = createWrapper(draft)

    expect(wrapper.text()).toContain('Preview Changes')
  })

  it('shows additions badge for step:create changes', () => {
    const draft = createMockDraft([
      { type: 'step:create', tempId: 'temp-1', stepType: 'llm', name: 'New LLM', config: {}, position: { x: 0, y: 0 } },
    ])
    const wrapper = createWrapper(draft)

    const addedBadge = wrapper.find('.badge.added')
    expect(addedBadge.exists()).toBe(true)
    expect(addedBadge.text()).toBe('+1')
  })

  it('shows modifications badge for step:update changes', () => {
    const draft = createMockDraft([
      { type: 'step:update', stepId: 'step-123', patch: { name: 'Updated Name' } },
    ])
    const wrapper = createWrapper(draft)

    const modifiedBadge = wrapper.find('.badge.modified')
    expect(modifiedBadge.exists()).toBe(true)
    expect(modifiedBadge.text()).toBe('~1')
  })

  it('shows deletions badge for step:delete changes', () => {
    const draft = createMockDraft([
      { type: 'step:delete', stepId: 'step-456' },
    ])
    const wrapper = createWrapper(draft)

    const deletedBadge = wrapper.find('.badge.deleted')
    expect(deletedBadge.exists()).toBe(true)
    expect(deletedBadge.text()).toBe('-1')
  })

  it('shows multiple badges for mixed changes', () => {
    const draft = createMockDraft([
      { type: 'step:create', tempId: 'temp-1', stepType: 'llm', name: 'New LLM', config: {}, position: { x: 0, y: 0 } },
      { type: 'step:create', tempId: 'temp-2', stepType: 'tool', name: 'New HTTP', config: {}, position: { x: 100, y: 0 } },
      { type: 'step:update', stepId: 'step-123', patch: { name: 'Updated' } },
      { type: 'step:delete', stepId: 'step-456' },
      { type: 'edge:delete', edgeId: 'edge-789' },
    ])
    const wrapper = createWrapper(draft)

    expect(wrapper.find('.badge.added').text()).toBe('+2')
    expect(wrapper.find('.badge.modified').text()).toBe('~1')
    expect(wrapper.find('.badge.deleted').text()).toBe('-2')
  })

  it('renders change list items', () => {
    const draft = createMockDraft([
      { type: 'step:create', tempId: 'temp-1', stepType: 'llm', name: 'My LLM Block', config: {}, position: { x: 0, y: 0 } },
    ])
    const wrapper = createWrapper(draft)

    const changeItems = wrapper.findAll('.change-item')
    expect(changeItems).toHaveLength(1)
    expect(changeItems[0].classes()).toContain('added')
    expect(changeItems[0].text()).toContain('My LLM Block')
  })

  it('emits apply event when apply button is clicked', async () => {
    const draft = createMockDraft()
    const wrapper = createWrapper(draft)

    const applyButton = wrapper.find('.btn-primary')
    await applyButton.trigger('click')

    expect(wrapper.emitted('apply')).toHaveLength(1)
  })

  it('emits discard event when discard button is clicked', async () => {
    const draft = createMockDraft()
    const wrapper = createWrapper(draft)

    const discardButton = wrapper.find('.btn-ghost')
    await discardButton.trigger('click')

    expect(wrapper.emitted('discard')).toHaveLength(1)
  })

  it('shows modify input when modify button is clicked', async () => {
    const draft = createMockDraft()
    const wrapper = createWrapper(draft)

    // Verify modify input is hidden initially
    expect(wrapper.find('.modify-input-section').exists()).toBe(false)

    // Click modify button
    const modifyButton = wrapper.find('.btn-secondary')
    await modifyButton.trigger('click')

    // Verify modify input is shown
    expect(wrapper.find('.modify-input-section').exists()).toBe(true)
    expect(wrapper.find('textarea.modify-input').exists()).toBe(true)
  })

  it('emits modify event with feedback when send button is clicked', async () => {
    const draft = createMockDraft()
    const wrapper = createWrapper(draft)

    // Show modify input
    await wrapper.find('.btn-secondary').trigger('click')

    // Enter feedback
    const textarea = wrapper.find('textarea.modify-input')
    await textarea.setValue('Please add error handling')

    // Click send button
    const sendButton = wrapper.findAll('.modify-actions .btn-primary')[0]
    await sendButton.trigger('click')

    expect(wrapper.emitted('modify')).toHaveLength(1)
    expect(wrapper.emitted('modify')![0]).toEqual(['Please add error handling'])
  })

  it('disables send button when feedback is empty', async () => {
    const draft = createMockDraft()
    const wrapper = createWrapper(draft)

    // Show modify input
    await wrapper.find('.btn-secondary').trigger('click')

    // Verify send button is disabled
    const sendButton = wrapper.findAll('.modify-actions .btn-primary')[0]
    expect(sendButton.attributes('disabled')).toBeDefined()
  })

  it('hides modify input when cancel is clicked', async () => {
    const draft = createMockDraft()
    const wrapper = createWrapper(draft)

    // Show modify input
    await wrapper.find('.btn-secondary').trigger('click')
    expect(wrapper.find('.modify-input-section').exists()).toBe(true)

    // Click cancel
    await wrapper.find('.modify-actions .btn-secondary').trigger('click')

    // Verify modify input is hidden
    expect(wrapper.find('.modify-input-section').exists()).toBe(false)
  })

  it('handles edge:create changes correctly', () => {
    const draft = createMockDraft([
      { type: 'edge:create', sourceId: 'source-12345678', targetId: 'target-87654321' },
    ])
    const wrapper = createWrapper(draft)

    const changeItem = wrapper.find('.change-item')
    expect(changeItem.classes()).toContain('added')
    expect(changeItem.text()).toContain('Connection')
    expect(changeItem.text()).toContain('source-1')
    expect(changeItem.text()).toContain('target-8')
  })
})
