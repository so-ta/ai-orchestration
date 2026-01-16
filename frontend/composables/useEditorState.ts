import type { Step, Workflow } from '~/types/api'

const STORAGE_KEY = 'workflow-editor-panel-widths'

// Clipboard data structure
interface StepClipboard {
  name: string
  type: string
  config: Record<string, unknown>
}

// Global state (singleton pattern)
const selectedStepId = ref<string | null>(null)
const clipboardStep = ref<StepClipboard | null>(null)
const leftPanelWidth = ref(280)
const rightPanelWidth = ref(360)

// Initialize from localStorage (client-side only)
if (typeof window !== 'undefined') {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      const { left, right } = JSON.parse(stored)
      if (typeof left === 'number' && left >= 200 && left <= 500) {
        leftPanelWidth.value = left
      }
      if (typeof right === 'number' && right >= 280 && right <= 600) {
        rightPanelWidth.value = right
      }
    }
  } catch (e) {
    console.warn('Failed to load panel widths from localStorage:', e)
  }
}

// Watch and persist panel widths
watch([leftPanelWidth, rightPanelWidth], () => {
  if (typeof window !== 'undefined') {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify({
        left: leftPanelWidth.value,
        right: rightPanelWidth.value
      }))
    } catch (e) {
      console.warn('Failed to save panel widths to localStorage:', e)
    }
  }
}, { deep: true })

/**
 * Editor state management composable
 * Manages selection, clipboard, and panel widths for the workflow editor
 */
export function useEditorState(workflow?: Ref<Workflow | null>) {
  // Computed: Get selected step from workflow
  const selectedStep = computed<Step | null>(() => {
    if (!selectedStepId.value || !workflow?.value?.steps) return null
    return workflow.value.steps.find(s => s.id === selectedStepId.value) || null
  })

  // Actions
  function selectStep(stepId: string | null) {
    selectedStepId.value = stepId
  }

  function clearSelection() {
    selectedStepId.value = null
  }

  function copyStep() {
    if (!selectedStep.value) return false

    clipboardStep.value = {
      name: selectedStep.value.name + ' (Copy)',
      type: selectedStep.value.type,
      config: JSON.parse(JSON.stringify(selectedStep.value.config))
    }
    return true
  }

  function hasClipboard(): boolean {
    return clipboardStep.value !== null
  }

  function getClipboardData(): StepClipboard | null {
    if (!clipboardStep.value) return null
    return {
      ...clipboardStep.value,
      config: JSON.parse(JSON.stringify(clipboardStep.value.config))
    }
  }

  function clearClipboard() {
    clipboardStep.value = null
  }

  // Panel width controls
  function setLeftPanelWidth(width: number) {
    leftPanelWidth.value = Math.max(200, Math.min(500, width))
  }

  function setRightPanelWidth(width: number) {
    rightPanelWidth.value = Math.max(280, Math.min(600, width))
  }

  function resetPanelWidths() {
    leftPanelWidth.value = 280
    rightPanelWidth.value = 360
  }

  return {
    // State (readonly where appropriate)
    selectedStepId: readonly(selectedStepId),
    selectedStep,
    leftPanelWidth,
    rightPanelWidth,

    // Actions
    selectStep,
    clearSelection,
    copyStep,
    hasClipboard,
    getClipboardData,
    clearClipboard,
    setLeftPanelWidth,
    setRightPanelWidth,
    resetPanelWidths,
  }
}
