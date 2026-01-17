import type { Step, Project } from '~/types/api'

const STORAGE_KEY = 'project-editor-panel-widths'
const STORAGE_KEY_COLLAPSED = 'project-editor-panel-collapsed'
const STORAGE_KEY_LAST_PROJECT = 'project-editor-last-project'

// Clipboard data structure
interface StepClipboard {
  name: string
  type: string
  config: Record<string, unknown>
}

// SlideOut panel types
export type SlideOutPanel = 'runs' | 'schedules' | 'variables' | null

// Global state (singleton pattern)
const selectedStepId = ref<string | null>(null)
const clipboardStep = ref<StepClipboard | null>(null)
const leftPanelWidth = ref(280)
const rightPanelWidth = ref(360)
const leftCollapsed = ref(false)
const rightCollapsed = ref(false)
const activeSlideOut = ref<SlideOutPanel>(null)
const currentProjectId = ref<string | null>(null)
const lastProjectId = ref<string | null>(null)

// Initialize from localStorage (client-side only)
if (typeof window !== 'undefined') {
  try {
    // Load panel widths
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

    // Load collapsed states
    const collapsedStored = localStorage.getItem(STORAGE_KEY_COLLAPSED)
    if (collapsedStored) {
      const { left, right } = JSON.parse(collapsedStored)
      leftCollapsed.value = !!left
      rightCollapsed.value = !!right
    }

    // Load last project ID
    const lastProject = localStorage.getItem(STORAGE_KEY_LAST_PROJECT)
    if (lastProject) {
      lastProjectId.value = lastProject
    }
  } catch (e) {
    console.warn('Failed to load editor state from localStorage:', e)
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

// Watch and persist collapsed states
watch([leftCollapsed, rightCollapsed], () => {
  if (typeof window !== 'undefined') {
    try {
      localStorage.setItem(STORAGE_KEY_COLLAPSED, JSON.stringify({
        left: leftCollapsed.value,
        right: rightCollapsed.value
      }))
    } catch (e) {
      console.warn('Failed to save collapsed states to localStorage:', e)
    }
  }
}, { deep: true })

// Watch and persist last project ID
watch(currentProjectId, (newId) => {
  if (typeof window !== 'undefined' && newId) {
    try {
      localStorage.setItem(STORAGE_KEY_LAST_PROJECT, newId)
      lastProjectId.value = newId
    } catch (e) {
      console.warn('Failed to save last project to localStorage:', e)
    }
  }
})

/**
 * Editor state management composable
 * Manages selection, clipboard, and panel widths for the project editor
 */
export function useEditorState(project?: Ref<Project | null>) {
  // Computed: Get selected step from project
  const selectedStep = computed<Step | null>(() => {
    if (!selectedStepId.value || !project?.value?.steps) return null
    return project.value.steps.find(s => s.id === selectedStepId.value) || null
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

  // Collapse controls
  function setLeftCollapsed(collapsed: boolean) {
    leftCollapsed.value = collapsed
  }

  function setRightCollapsed(collapsed: boolean) {
    rightCollapsed.value = collapsed
  }

  function toggleLeftCollapsed() {
    leftCollapsed.value = !leftCollapsed.value
  }

  function toggleRightCollapsed() {
    rightCollapsed.value = !rightCollapsed.value
  }

  // SlideOut panel controls
  function openSlideOut(panel: SlideOutPanel) {
    activeSlideOut.value = panel
  }

  function closeSlideOut() {
    activeSlideOut.value = null
  }

  function toggleSlideOut(panel: Exclude<SlideOutPanel, null>) {
    if (activeSlideOut.value === panel) {
      activeSlideOut.value = null
    } else {
      activeSlideOut.value = panel
    }
  }

  // Project ID controls
  function setCurrentProjectId(projectId: string | null) {
    currentProjectId.value = projectId
  }

  function getLastProjectId(): string | null {
    return lastProjectId.value
  }

  return {
    // State (readonly where appropriate)
    selectedStepId: readonly(selectedStepId),
    selectedStep,
    leftPanelWidth,
    rightPanelWidth,
    leftCollapsed,
    rightCollapsed,
    activeSlideOut: readonly(activeSlideOut),
    currentProjectId: readonly(currentProjectId),

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
    setLeftCollapsed,
    setRightCollapsed,
    toggleLeftCollapsed,
    toggleRightCollapsed,
    openSlideOut,
    closeSlideOut,
    toggleSlideOut,
    setCurrentProjectId,
    getLastProjectId,
  }
}
