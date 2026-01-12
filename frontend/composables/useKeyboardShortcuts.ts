import type { Ref } from 'vue'
import type { Step, StepType } from '~/types/api'

interface KeyboardShortcutsOptions {
  selectedStep: Ref<Step | null>
  selectedGroupId?: Ref<string | null>
  isReadonly: Ref<boolean>
  onDelete: () => void
  onDeleteGroup?: () => void
  onCopy: () => void
  onPaste: (data: { type: StepType; name: string; config: Record<string, any> }) => void
  onClearSelection: () => void
}

interface ClipboardData {
  type: StepType
  name: string
  config: Record<string, any>
}

// Global clipboard for steps (persists across composable instances)
const stepClipboard = ref<ClipboardData | null>(null)

/**
 * Keyboard shortcuts composable for workflow editor
 * Handles: Delete, Cmd/Ctrl+C (copy), Cmd/Ctrl+V (paste), Escape (deselect)
 */
export function useKeyboardShortcuts(options: KeyboardShortcutsOptions) {
  const {
    selectedStep,
    selectedGroupId,
    isReadonly,
    onDelete,
    onDeleteGroup,
    onCopy,
    onPaste,
    onClearSelection,
  } = options

  // Check if user is typing in an input field
  function isTypingInInput(event: KeyboardEvent): boolean {
    const target = event.target as HTMLElement
    const tagName = target.tagName.toLowerCase()
    return (
      tagName === 'input' ||
      tagName === 'textarea' ||
      tagName === 'select' ||
      target.isContentEditable
    )
  }

  // Handle keydown events
  function handleKeyDown(event: KeyboardEvent) {
    // Skip if typing in an input field
    if (isTypingInInput(event)) return

    const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0
    const modKey = isMac ? event.metaKey : event.ctrlKey

    // Delete or Backspace - delete selected step or group
    if ((event.key === 'Delete' || event.key === 'Backspace') && !isReadonly.value) {
      // Delete group if a group is selected
      if (selectedGroupId?.value && onDeleteGroup) {
        event.preventDefault()
        onDeleteGroup()
        return
      }
      // Delete step if a step is selected
      if (selectedStep.value) {
        event.preventDefault()
        onDelete()
        return
      }
    }

    // Cmd/Ctrl + C - copy selected step
    if (modKey && event.key === 'c' && selectedStep.value) {
      // Don't prevent default - allow native copy for text selection
      // But still copy the step to our clipboard
      stepClipboard.value = {
        type: selectedStep.value.type,
        name: selectedStep.value.name + ' (Copy)',
        config: JSON.parse(JSON.stringify(selectedStep.value.config)),
      }
      onCopy()
      return
    }

    // Cmd/Ctrl + V - paste step
    if (modKey && event.key === 'v' && stepClipboard.value && !isReadonly.value) {
      event.preventDefault()
      onPaste(stepClipboard.value)
      return
    }

    // Escape - clear selection
    if (event.key === 'Escape' && (selectedStep.value || selectedGroupId?.value)) {
      event.preventDefault()
      onClearSelection()
      return
    }
  }

  // Register event listeners on mount
  onMounted(() => {
    document.addEventListener('keydown', handleKeyDown)
  })

  // Cleanup on unmount
  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeyDown)
  })

  return {
    hasClipboard: computed(() => stepClipboard.value !== null),
    clipboardData: readonly(stepClipboard),
  }
}
