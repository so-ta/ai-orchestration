/**
 * Composable for variable entry management
 * Handles type detection, value conversion, and entry operations
 */

export interface VariableEntry {
  key: string
  value: string
  type: 'string' | 'number' | 'boolean' | 'json'
}

export interface UseVariableEditorOptions {
  initialVariables?: Record<string, unknown>
}

export function useVariableEditor(options: UseVariableEditorOptions = {}) {
  const entries = ref<VariableEntry[]>([])
  const jsonContent = ref('')
  const jsonError = ref<string | null>(null)

  /**
   * Detect value type from a value
   */
  function detectType(value: unknown): VariableEntry['type'] {
    if (typeof value === 'boolean') return 'boolean'
    if (typeof value === 'number') return 'number'
    if (typeof value === 'object') return 'json'
    return 'string'
  }

  /**
   * Convert entry to proper typed value
   */
  function convertValue(entry: VariableEntry): unknown {
    switch (entry.type) {
      case 'number':
        return parseFloat(entry.value) || 0
      case 'boolean':
        return entry.value === 'true'
      case 'json':
        try {
          return JSON.parse(entry.value)
        } catch {
          return entry.value
        }
      default:
        return entry.value
    }
  }

  /**
   * Build variables object from current entries
   */
  function buildVariablesFromEntries(): Record<string, unknown> {
    const result: Record<string, unknown> = {}
    for (const entry of entries.value) {
      if (entry.key.trim()) {
        result[entry.key.trim()] = convertValue(entry)
      }
    }
    return result
  }

  /**
   * Initialize entries from a variables object
   */
  function initFromVariables(variables: Record<string, unknown>) {
    entries.value = Object.entries(variables).map(([key, value]) => ({
      key,
      value: typeof value === 'object' ? JSON.stringify(value) : String(value),
      type: detectType(value),
    }))
    jsonContent.value = JSON.stringify(variables, null, 2)
    jsonError.value = null
  }

  /**
   * Add a new empty entry
   */
  function addEntry() {
    entries.value.push({
      key: '',
      value: '',
      type: 'string',
    })
  }

  /**
   * Remove an entry by index
   */
  function removeEntry(index: number) {
    entries.value.splice(index, 1)
  }

  /**
   * Update an entry field
   */
  function updateEntry(index: number, field: keyof VariableEntry, value: string) {
    entries.value[index][field] = value as never
  }

  /**
   * Parse JSON content and return parsed object or null on error
   */
  function parseJsonContent(): Record<string, unknown> | null {
    try {
      const parsed = JSON.parse(jsonContent.value)
      jsonError.value = null
      return parsed
    } catch (err) {
      jsonError.value = err instanceof Error ? err.message : 'Invalid JSON'
      return null
    }
  }

  /**
   * Switch from form mode to JSON mode - sync entries to JSON
   */
  function syncEntriesToJson() {
    jsonContent.value = JSON.stringify(buildVariablesFromEntries(), null, 2)
    jsonError.value = null
  }

  /**
   * Switch from JSON mode to form mode - sync JSON to entries
   * Returns true if successful, false if JSON is invalid
   */
  function syncJsonToEntries(): boolean {
    const parsed = parseJsonContent()
    if (parsed === null) return false

    entries.value = Object.entries(parsed).map(([key, value]) => ({
      key,
      value: typeof value === 'object' ? JSON.stringify(value) : String(value),
      type: detectType(value),
    }))
    return true
  }

  /**
   * Handle JSON input change
   */
  function handleJsonInput(value: string) {
    jsonContent.value = value
    parseJsonContent()
  }

  // Initialize if initial variables provided
  if (options.initialVariables) {
    initFromVariables(options.initialVariables)
  }

  return {
    entries,
    jsonContent,
    jsonError,
    detectType,
    convertValue,
    buildVariablesFromEntries,
    initFromVariables,
    addEntry,
    removeEntry,
    updateEntry,
    parseJsonContent,
    syncEntriesToJson,
    syncJsonToEntries,
    handleJsonInput,
  }
}
