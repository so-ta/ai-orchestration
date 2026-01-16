/**
 * Utility composable for managing async operation state.
 * Provides consistent loading, error, and data state management.
 *
 * This is an internal utility - not intended for direct use by components.
 */

/**
 * Extracts error message from an unknown error type
 */
export function extractErrorMessage(error: unknown, fallbackMessage = 'An unknown error occurred'): string {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === 'string') {
    return error
  }
  return fallbackMessage
}

/**
 * Creates a reusable async state handler for API operations
 */
export function useAsyncState<T>() {
  const data = ref<T | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * Wraps an async function with loading and error handling
   * @param fn - The async function to execute
   * @param errorMessage - Optional custom error message prefix
   */
  async function execute<R>(
    fn: () => Promise<R>,
    errorMessage?: string
  ): Promise<R> {
    loading.value = true
    error.value = null

    try {
      const result = await fn()
      return result
    } catch (e) {
      const message = extractErrorMessage(e)
      error.value = errorMessage ? `${errorMessage}: ${message}` : message
      throw e
    } finally {
      loading.value = false
    }
  }

  /**
   * Wraps an async function with loading and error handling, storing result in data
   * @param fn - The async function to execute
   * @param errorMessage - Optional custom error message prefix
   */
  async function executeAndStore(
    fn: () => Promise<T>,
    errorMessage?: string
  ): Promise<T> {
    const result = await execute(fn, errorMessage)
    data.value = result
    return result
  }

  function reset() {
    data.value = null
    loading.value = false
    error.value = null
  }

  return {
    data: readonly(data),
    loading: readonly(loading),
    error: readonly(error),
    execute,
    executeAndStore,
    reset,
  }
}

/**
 * Creates a list state handler with pagination support
 *
 * Note: Unlike useAsyncState, the state refs are returned as-is (not readonly)
 * to preserve backward compatibility with existing components that mutate them.
 */
export function useListState<T>() {
  const items = ref<T[]>([]) as Ref<T[]>
  const loading = ref(false)
  const error = ref<string | null>(null)
  const pagination = ref({
    page: 1,
    limit: 20,
    total: 0,
  })

  async function execute<R>(
    fn: () => Promise<R>,
    errorMessage?: string
  ): Promise<R> {
    loading.value = true
    error.value = null

    try {
      return await fn()
    } catch (e) {
      const message = extractErrorMessage(e)
      error.value = errorMessage ? `${errorMessage}: ${message}` : message
      throw e
    } finally {
      loading.value = false
    }
  }

  function setItems(newItems: T[]) {
    items.value = newItems
  }

  function addItem(item: T) {
    items.value = [item, ...items.value]
    pagination.value.total += 1
  }

  function updateItem(predicate: (item: T) => boolean, newItem: T) {
    const index = items.value.findIndex(predicate)
    if (index !== -1) {
      items.value[index] = newItem
    }
  }

  function removeItem(predicate: (item: T) => boolean) {
    items.value = items.value.filter(item => !predicate(item))
    pagination.value.total -= 1
  }

  function setPagination(meta: { page: number; limit: number; total: number }) {
    pagination.value = meta
  }

  function reset() {
    items.value = []
    loading.value = false
    error.value = null
    pagination.value = { page: 1, limit: 20, total: 0 }
  }

  // Note: State refs are not wrapped in readonly() to preserve backward compatibility
  // with components that directly mutate pagination.value.page, etc.
  return {
    items,
    loading,
    error,
    pagination,
    execute,
    setItems,
    addItem,
    updateItem,
    removeItem,
    setPagination,
    reset,
  }
}
