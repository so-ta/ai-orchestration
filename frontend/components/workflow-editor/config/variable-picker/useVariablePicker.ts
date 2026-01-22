import { ref, computed, watch, type Ref } from 'vue'
import type { AvailableVariable } from '../../composables/useAvailableVariables'

export interface VariableTreeNode extends AvailableVariable {
  children?: VariableTreeNode[]
  expanded?: boolean
  level: number
  parentPath?: string
}

export interface UseVariablePickerOptions {
  variables: Ref<AvailableVariable[]>
  onSelect?: (variable: AvailableVariable) => void
}

export function useVariablePicker(options: UseVariablePickerOptions) {
  const { variables, onSelect } = options

  // State
  const isOpen = ref(false)
  const searchQuery = ref('')
  const selectedIndex = ref(0)
  const expandedPaths = ref<Set<string>>(new Set())

  // Build tree structure from flat variables
  const variableTree = computed<VariableTreeNode[]>(() => {
    const tree: VariableTreeNode[] = []
    const nodeMap = new Map<string, VariableTreeNode>()

    // Group variables by source
    const bySource = new Map<string, AvailableVariable[]>()
    for (const v of variables.value) {
      const existing = bySource.get(v.source) || []
      existing.push(v)
      bySource.set(v.source, existing)
    }

    // Create tree nodes
    for (const [source, vars] of bySource) {
      const sourceNode: VariableTreeNode = {
        path: source,
        type: 'group',
        title: source === 'input' ? 'Workflow Input' : source,
        source,
        level: 0,
        children: [],
        expanded: expandedPaths.value.has(source)
      }

      for (const v of vars) {
        const childNode: VariableTreeNode = {
          ...v,
          level: 1,
          parentPath: source
        }
        sourceNode.children!.push(childNode)
        nodeMap.set(v.path, childNode)
      }

      tree.push(sourceNode)
      nodeMap.set(source, sourceNode)
    }

    return tree
  })

  // Flatten tree for keyboard navigation (only visible items)
  // Groups are always shown, and their children are shown when expanded
  const flattenedVariables = computed<VariableTreeNode[]>(() => {
    const result: VariableTreeNode[] = []

    function flatten(nodes: VariableTreeNode[]) {
      for (const node of nodes) {
        // Always include the node
        result.push(node)
        // If it's an expanded group, also include its children
        if (node.type === 'group' && node.expanded && node.children) {
          flatten(node.children)
        }
      }
    }

    flatten(variableTree.value)
    return result
  })

  // Filtered variables based on search query
  const filteredVariables = computed<VariableTreeNode[]>(() => {
    const query = searchQuery.value.toLowerCase().trim()
    if (!query) return flattenedVariables.value

    return flattenedVariables.value.filter(v => {
      if (v.type === 'group') return false
      return (
        v.path.toLowerCase().includes(query) ||
        v.title?.toLowerCase().includes(query) ||
        v.description?.toLowerCase().includes(query)
      )
    })
  })

  // Reset selection when filtered list changes
  watch(filteredVariables, () => {
    selectedIndex.value = 0
  })

  // Open/close methods
  function open() {
    isOpen.value = true
    selectedIndex.value = 0
    searchQuery.value = ''
  }

  function close() {
    isOpen.value = false
    searchQuery.value = ''
  }

  // Expand group and move focus to first child
  function expandGroup(path: string) {
    if (!expandedPaths.value.has(path)) {
      expandedPaths.value.add(path)
      expandedPaths.value = new Set(expandedPaths.value)

      // Move focus to first child after the list updates
      const currentIndex = selectedIndex.value
      // After expansion, the first child will be at currentIndex + 1
      setTimeout(() => {
        if (filteredVariables.value.length > currentIndex + 1) {
          selectedIndex.value = currentIndex + 1
        }
      }, 0)
    }
  }

  // Collapse group
  function collapseGroup(path: string) {
    if (expandedPaths.value.has(path)) {
      expandedPaths.value.delete(path)
      expandedPaths.value = new Set(expandedPaths.value)
    }
  }

  // Toggle group expansion
  function toggleExpand(path: string) {
    if (expandedPaths.value.has(path)) {
      collapseGroup(path)
    } else {
      expandGroup(path)
    }
  }

  // Find index of a variable by path
  function findIndexByPath(path: string): number {
    return filteredVariables.value.findIndex(v => v.path === path)
  }

  // Keyboard navigation
  function handleKeydown(event: KeyboardEvent): boolean {
    switch (event.key) {
      case 'ArrowDown':
        event.preventDefault()
        selectedIndex.value = Math.min(
          selectedIndex.value + 1,
          filteredVariables.value.length - 1
        )
        return true

      case 'ArrowUp':
        event.preventDefault()
        selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
        return true

      case 'Enter':
        event.preventDefault()
        selectCurrent()
        return true

      case 'Tab':
      case 'ArrowRight': {
        event.preventDefault()
        const current = filteredVariables.value[selectedIndex.value]
        if (current?.type === 'group') {
          // Expand group and move to first child
          if (!current.expanded) {
            expandGroup(current.path)
          } else {
            // Already expanded, move to first child
            if (selectedIndex.value + 1 < filteredVariables.value.length) {
              selectedIndex.value = selectedIndex.value + 1
            }
          }
        } else {
          // Not a group - select the variable
          selectVariable(current)
        }
        return true
      }

      case 'ArrowLeft': {
        event.preventDefault()
        const current = filteredVariables.value[selectedIndex.value]
        if (current?.type === 'group' && current.expanded) {
          // Collapse the group
          collapseGroup(current.path)
        } else if (current?.parentPath) {
          // Move to parent group
          const parentIndex = findIndexByPath(current.parentPath)
          if (parentIndex >= 0) {
            selectedIndex.value = parentIndex
          }
        }
        return true
      }

      case 'Escape':
        event.preventDefault()
        close()
        return true

      default:
        return false
    }
  }

  // Select current item
  function selectCurrent() {
    const current = filteredVariables.value[selectedIndex.value]
    if (current) {
      if (current.type === 'group') {
        toggleExpand(current.path)
      } else {
        selectVariable(current)
      }
    }
  }

  // Select a specific variable
  function selectVariable(variable: AvailableVariable) {
    onSelect?.(variable)
    close()
  }

  // Set selected index
  function setSelectedIndex(index: number) {
    selectedIndex.value = index
  }

  return {
    // State
    isOpen,
    searchQuery,
    selectedIndex,
    expandedPaths,

    // Computed
    variableTree,
    filteredVariables,

    // Methods
    open,
    close,
    toggleExpand,
    handleKeydown,
    selectCurrent,
    selectVariable,
    setSelectedIndex
  }
}

// Detect {{ in input and return cursor position info
export function detectTemplateStart(
  value: string,
  cursorPosition: number
): { detected: boolean; startIndex: number; query: string } {
  // Find the last {{ before cursor
  const beforeCursor = value.substring(0, cursorPosition)
  const lastOpenBrace = beforeCursor.lastIndexOf('{{')

  if (lastOpenBrace === -1) {
    return { detected: false, startIndex: -1, query: '' }
  }

  // Check if there's a closing }} between {{ and cursor
  const afterOpen = beforeCursor.substring(lastOpenBrace + 2)
  if (afterOpen.includes('}}')) {
    return { detected: false, startIndex: -1, query: '' }
  }

  // Extract the query (text between {{ and cursor)
  const query = afterOpen.trim()

  return {
    detected: true,
    startIndex: lastOpenBrace,
    query
  }
}

// Format variable path for template insertion
export function formatVariableForTemplate(path: string): string {
  return `{{${path}}}`
}
