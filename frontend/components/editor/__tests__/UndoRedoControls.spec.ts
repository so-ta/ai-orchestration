import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { ref } from 'vue'
import UndoRedoControls from '../UndoRedoControls.vue'

const messages = {
  editor: {
    undo: 'Undo',
    redo: 'Redo',
    history: 'History',
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

// Mock useCommandHistory
const mockUndo = vi.fn()
const mockRedo = vi.fn()
const mockCanUndo = ref(false)
const mockCanRedo = ref(false)
const mockHistory = ref<unknown[]>([])

vi.mock('~/composables/useCommandHistory', () => ({
  useCommandHistory: () => ({
    canUndo: mockCanUndo,
    canRedo: mockCanRedo,
    undo: mockUndo,
    redo: mockRedo,
    history: mockHistory,
  }),
}))

describe('UndoRedoControls', () => {
  const createWrapper = () => {
    return mount(UndoRedoControls, {
      global: {
        plugins: [i18n],
      },
    })
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockCanUndo.value = false
    mockCanRedo.value = false
    mockHistory.value = []
  })

  it('renders undo and redo buttons', () => {
    const wrapper = createWrapper()

    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(2)
  })

  it('disables undo button when canUndo is false', () => {
    mockCanUndo.value = false
    const wrapper = createWrapper()

    const undoButton = wrapper.findAll('button')[0]
    expect(undoButton.attributes('disabled')).toBeDefined()
    expect(undoButton.classes()).toContain('disabled')
  })

  it('enables undo button when canUndo is true', () => {
    mockCanUndo.value = true
    const wrapper = createWrapper()

    const undoButton = wrapper.findAll('button')[0]
    expect(undoButton.attributes('disabled')).toBeUndefined()
    expect(undoButton.classes()).not.toContain('disabled')
  })

  it('disables redo button when canRedo is false', () => {
    mockCanRedo.value = false
    const wrapper = createWrapper()

    const redoButton = wrapper.findAll('button')[1]
    expect(redoButton.attributes('disabled')).toBeDefined()
    expect(redoButton.classes()).toContain('disabled')
  })

  it('enables redo button when canRedo is true', () => {
    mockCanRedo.value = true
    const wrapper = createWrapper()

    const redoButton = wrapper.findAll('button')[1]
    expect(redoButton.attributes('disabled')).toBeUndefined()
    expect(redoButton.classes()).not.toContain('disabled')
  })

  it('calls undo when undo button is clicked', async () => {
    mockCanUndo.value = true
    const wrapper = createWrapper()

    const undoButton = wrapper.findAll('button')[0]
    await undoButton.trigger('click')

    expect(mockUndo).toHaveBeenCalledOnce()
  })

  it('calls redo when redo button is clicked', async () => {
    mockCanRedo.value = true
    const wrapper = createWrapper()

    const redoButton = wrapper.findAll('button')[1]
    await redoButton.trigger('click')

    expect(mockRedo).toHaveBeenCalledOnce()
  })

  it('does not call undo when button is disabled', async () => {
    mockCanUndo.value = false
    const wrapper = createWrapper()

    const undoButton = wrapper.findAll('button')[0]
    await undoButton.trigger('click')

    expect(mockUndo).not.toHaveBeenCalled()
  })

  it('does not call redo when button is disabled', async () => {
    mockCanRedo.value = false
    const wrapper = createWrapper()

    const redoButton = wrapper.findAll('button')[1]
    await redoButton.trigger('click')

    expect(mockRedo).not.toHaveBeenCalled()
  })

  it('shows history badge when history has items', () => {
    mockHistory.value = [{ id: '1' }, { id: '2' }]
    const wrapper = createWrapper()

    const badge = wrapper.find('.history-badge')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toBe('2')
  })

  it('hides history badge when history is empty', () => {
    mockHistory.value = []
    const wrapper = createWrapper()

    const badge = wrapper.find('.history-badge')
    expect(badge.exists()).toBe(false)
  })

  it('displays correct shortcut in title', () => {
    const wrapper = createWrapper()

    const undoButton = wrapper.findAll('button')[0]
    const redoButton = wrapper.findAll('button')[1]

    // Should contain either Mac or Windows shortcut
    expect(undoButton.attributes('title')).toMatch(/Undo \((Cmd|Ctrl)\+Z\)/)
    expect(redoButton.attributes('title')).toMatch(/Redo \((Cmd|Ctrl)\+Shift\+Z\)/)
  })
})
