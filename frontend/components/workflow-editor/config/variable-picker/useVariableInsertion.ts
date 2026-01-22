import { ref, computed, inject, onMounted, onUnmounted, type Ref, type ComputedRef, type InjectionKey } from 'vue'
import type { AvailableVariable } from '../../composables/useAvailableVariables'
import { detectTemplateStart, formatVariableForTemplate } from './useVariablePicker'

// Injection keys
export const AVAILABLE_VARIABLES_KEY = Symbol('availableVariables') as InjectionKey<ComputedRef<AvailableVariable[]>>
export const ACTIVE_FIELD_INSERTER_KEY = Symbol('activeFieldInserter') as InjectionKey<{
  register: (id: string, inserter: FieldInserter) => void
  unregister: (id: string) => void
  setActive: (id: string | null) => void
  activeId: Ref<string | null>
}>

export interface FieldInserter {
  insert: (text: string) => void
  focus: () => void
}

export interface UseVariableInsertionOptions {
  modelValue: Ref<string | undefined>
  emit: (value: string) => void
  inputRef: Ref<HTMLInputElement | HTMLTextAreaElement | null>
  fieldId?: string
}

export interface PickerPosition {
  top: number
  left: number
}

export function useVariableInsertion(options: UseVariableInsertionOptions) {
  const { modelValue, emit, inputRef, fieldId } = options

  // Inject available variables from parent
  const injectedVariables = inject(AVAILABLE_VARIABLES_KEY, computed(() => []))

  // State
  const pickerVisible = ref(false)
  const pickerPosition = ref<PickerPosition>({ top: 0, left: 0 })
  const templateStartIndex = ref(-1)
  const isDragOver = ref(false)

  // Get cursor position in input
  function getCursorPosition(): number {
    const el = inputRef.value
    if (!el) return 0
    return el.selectionStart ?? 0
  }

  // Set cursor position in input
  function setCursorPosition(pos: number) {
    const el = inputRef.value
    if (!el) return
    el.setSelectionRange(pos, pos)
  }

  // Calculate popup position based on cursor
  function calculatePickerPosition(): PickerPosition {
    const el = inputRef.value
    if (!el) return { top: 0, left: 0 }

    const rect = el.getBoundingClientRect()
    const cursorPos = getCursorPosition()

    // For text inputs, position below the input
    // For textareas, try to position near cursor
    const isTextarea = el.tagName.toLowerCase() === 'textarea'

    if (isTextarea) {
      // Approximate cursor position in textarea
      const text = modelValue.value || ''
      const beforeCursor = text.substring(0, cursorPos)
      const lines = beforeCursor.split('\n')
      const lineIndex = lines.length - 1
      const lineHeight = 18 // approximate

      return {
        top: rect.top + Math.min(lineIndex * lineHeight, rect.height - 100) + window.scrollY + 24,
        left: rect.left + 10 + window.scrollX
      }
    }

    return {
      top: rect.bottom + window.scrollY + 4,
      left: rect.left + window.scrollX
    }
  }

  // Handle input event - detect {{ trigger
  function handleInput(event?: Event) {
    // Get value from event target (more reliable) or fallback to modelValue
    const target = event?.target as HTMLInputElement | HTMLTextAreaElement | null
    const value = target?.value ?? modelValue.value ?? ''
    const cursorPos = target?.selectionStart ?? getCursorPosition()

    const detection = detectTemplateStart(value, cursorPos)

    if (detection.detected) {
      templateStartIndex.value = detection.startIndex
      pickerPosition.value = calculatePickerPosition()
      pickerVisible.value = true
    } else {
      pickerVisible.value = false
      templateStartIndex.value = -1
    }
  }

  // Handle keydown - manage picker state
  function handleKeydown(event: KeyboardEvent): boolean {
    // Close picker on Escape
    if (event.key === 'Escape' && pickerVisible.value) {
      pickerVisible.value = false
      return true
    }
    return false
  }

  // Insert variable at current position
  function insertVariable(variable: AvailableVariable) {
    const el = inputRef.value
    if (!el) return

    const value = modelValue.value || ''
    const formattedVar = formatVariableForTemplate(variable.path)

    if (templateStartIndex.value >= 0) {
      // Replace from {{ to cursor with the full template
      const cursorPos = getCursorPosition()
      const before = value.substring(0, templateStartIndex.value)
      const after = value.substring(cursorPos)
      const newValue = before + formattedVar + after

      emit(newValue)

      // Set cursor after inserted variable
      const newCursorPos = templateStartIndex.value + formattedVar.length
      requestAnimationFrame(() => {
        el.focus()
        setCursorPosition(newCursorPos)
      })
    } else {
      // Insert at cursor position
      const cursorPos = getCursorPosition()
      const before = value.substring(0, cursorPos)
      const after = value.substring(cursorPos)
      const newValue = before + formattedVar + after

      emit(newValue)

      // Set cursor after inserted variable
      const newCursorPos = cursorPos + formattedVar.length
      requestAnimationFrame(() => {
        el.focus()
        setCursorPosition(newCursorPos)
      })
    }

    pickerVisible.value = false
    templateStartIndex.value = -1
  }

  // Direct insertion (for click-to-insert from AvailableVariablesSection)
  function insertAtCursor(text: string) {
    const el = inputRef.value
    if (!el) return

    const value = modelValue.value || ''
    const cursorPos = getCursorPosition()
    const before = value.substring(0, cursorPos)
    const after = value.substring(cursorPos)
    const newValue = before + text + after

    emit(newValue)

    // Set cursor after inserted text
    const newCursorPos = cursorPos + text.length
    requestAnimationFrame(() => {
      el.focus()
      setCursorPosition(newCursorPos)
    })
  }

  // Drag & drop handlers
  function handleDragEnter(event: DragEvent) {
    event.preventDefault()
    const data = event.dataTransfer?.getData('text/plain')
    if (data?.startsWith('{{')) {
      isDragOver.value = true
    }
  }

  function handleDragOver(event: DragEvent) {
    event.preventDefault()
    const data = event.dataTransfer?.types.includes('text/plain')
    if (data) {
      event.dataTransfer!.dropEffect = 'copy'
    }
  }

  function handleDragLeave(_event: DragEvent) {
    isDragOver.value = false
  }

  function handleDrop(event: DragEvent) {
    event.preventDefault()
    isDragOver.value = false

    const data = event.dataTransfer?.getData('text/plain')
    if (data?.startsWith('{{')) {
      insertAtCursor(data)
    }
  }

  // Open picker manually (for toolbar button, etc.)
  function openPicker() {
    pickerPosition.value = calculatePickerPosition()
    pickerVisible.value = true
  }

  // Close picker
  function closePicker() {
    pickerVisible.value = false
    templateStartIndex.value = -1
  }

  // Focus the input element
  function focus() {
    inputRef.value?.focus()
  }

  // Register with parent field manager (for click-to-insert)
  const activeFieldManager = inject(ACTIVE_FIELD_INSERTER_KEY, null)

  if (activeFieldManager && fieldId) {
    const inserter: FieldInserter = {
      insert: insertAtCursor,
      focus
    }

    onMounted(() => {
      activeFieldManager.register(fieldId, inserter)
    })

    onUnmounted(() => {
      activeFieldManager.unregister(fieldId)
    })
  }

  return {
    // State
    pickerVisible,
    pickerPosition,
    isDragOver,
    availableVariables: injectedVariables,

    // Methods
    handleInput,
    handleKeydown,
    insertVariable,
    insertAtCursor,
    handleDragEnter,
    handleDragOver,
    handleDragLeave,
    handleDrop,
    openPicker,
    closePicker,
    focus
  }
}
