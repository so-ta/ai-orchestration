import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useToast } from '../useToast'

describe('useToast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    // Clear all toasts before each test
    const { toasts, removeToast } = useToast()
    const ids = toasts.value.map((t) => t.id)
    ids.forEach((id) => removeToast(id))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('should add a toast with success type', () => {
    const { success, toasts } = useToast()

    const id = success('Success Title', 'Success message')

    expect(id).toBeDefined()
    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0]).toMatchObject({
      type: 'success',
      title: 'Success Title',
      message: 'Success message',
    })
  })

  it('should add a toast with error type', () => {
    const { error, toasts } = useToast()

    error('Error Title', 'Error message')

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0]).toMatchObject({
      type: 'error',
      title: 'Error Title',
      message: 'Error message',
      duration: 8000,
    })
  })

  it('should add a toast with warning type', () => {
    const { warning, toasts } = useToast()

    warning('Warning Title')

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0]).toMatchObject({
      type: 'warning',
      title: 'Warning Title',
    })
  })

  it('should add a toast with info type', () => {
    const { info, toasts } = useToast()

    info('Info Title', 'Info message')

    expect(toasts.value).toHaveLength(1)
    expect(toasts.value[0]).toMatchObject({
      type: 'info',
      title: 'Info Title',
      message: 'Info message',
    })
  })

  it('should remove toast manually', () => {
    const { success, removeToast, toasts } = useToast()

    const id = success('Test')
    expect(toasts.value).toHaveLength(1)

    removeToast(id)
    expect(toasts.value).toHaveLength(0)
  })

  it('should auto-remove toast after duration', () => {
    const { success, toasts } = useToast()

    success('Auto Remove Test')
    expect(toasts.value).toHaveLength(1)

    // Default duration is 5000ms
    vi.advanceTimersByTime(5000)
    expect(toasts.value).toHaveLength(0)
  })

  it('should handle multiple toasts', () => {
    const { success, error, toasts } = useToast()

    success('First')
    error('Second')

    expect(toasts.value).toHaveLength(2)
    expect(toasts.value[0].title).toBe('First')
    expect(toasts.value[1].title).toBe('Second')
  })

  it('should generate unique IDs for each toast', () => {
    const { success, toasts } = useToast()

    success('First')
    success('Second')

    expect(toasts.value[0].id).not.toBe(toasts.value[1].id)
  })

  it('should not remove toast if ID not found', () => {
    const { success, removeToast, toasts } = useToast()

    success('Test')
    expect(toasts.value).toHaveLength(1)

    removeToast('non-existent-id')
    expect(toasts.value).toHaveLength(1)
  })
})
