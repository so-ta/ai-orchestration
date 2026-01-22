import type { Step, Project, Run, StepRun } from '~/types/api'

const STORAGE_KEY = 'project-editor-panel-widths'
const STORAGE_KEY_COLLAPSED = 'project-editor-panel-collapsed'
const STORAGE_KEY_LAST_PROJECT = 'project-editor-last-project'
const STORAGE_KEY_BOTTOM_PANEL = 'project-editor-bottom-panel'
const STORAGE_KEY_COPILOT_SIDEBAR = 'project-editor-copilot-sidebar'

// Copilot Sidebar constants
export const COPILOT_SIDEBAR_DEFAULT_WIDTH = 380
export const COPILOT_SIDEBAR_MIN_WIDTH = 280
export const COPILOT_SIDEBAR_MAX_WIDTH = 1200
export const COPILOT_SIDEBAR_COLLAPSED_WIDTH = 48

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

// Bottom panel state
const bottomPanelCollapsed = ref(false)
const bottomPanelHeight = ref(200)
const bottomPanelResizing = ref(false)
const selectedRun = ref<Run | null>(null)
const selectedStepRun = ref<StepRun | null>(null)
const clipboardStep = ref<StepClipboard | null>(null)
const leftPanelWidth = ref(280)
const rightPanelWidth = ref(360)
const leftCollapsed = ref(false)
const rightCollapsed = ref(false)
const activeSlideOut = ref<SlideOutPanel>(null)
const currentProjectId = ref<string | null>(null)
const lastProjectId = ref<string | null>(null)

// Copilot Sidebar state
const copilotSidebarOpen = ref(false)
const copilotSidebarWidth = ref(COPILOT_SIDEBAR_DEFAULT_WIDTH)
const copilotSidebarResizing = ref(false)

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

    // Load bottom panel state
    const bottomPanelStored = localStorage.getItem(STORAGE_KEY_BOTTOM_PANEL)
    if (bottomPanelStored) {
      const { collapsed, height } = JSON.parse(bottomPanelStored)
      bottomPanelCollapsed.value = !!collapsed
      if (typeof height === 'number' && height >= 100 && height <= 400) {
        bottomPanelHeight.value = height
      }
    }

    // Load copilot sidebar state
    const copilotSidebarStored = localStorage.getItem(STORAGE_KEY_COPILOT_SIDEBAR)
    if (copilotSidebarStored) {
      const { open, width } = JSON.parse(copilotSidebarStored)
      copilotSidebarOpen.value = !!open
      if (typeof width === 'number' && width >= COPILOT_SIDEBAR_MIN_WIDTH && width <= COPILOT_SIDEBAR_MAX_WIDTH) {
        copilotSidebarWidth.value = width
      }
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

// Watch and persist bottom panel state
watch([bottomPanelCollapsed, bottomPanelHeight], () => {
  if (typeof window !== 'undefined') {
    try {
      localStorage.setItem(STORAGE_KEY_BOTTOM_PANEL, JSON.stringify({
        collapsed: bottomPanelCollapsed.value,
        height: bottomPanelHeight.value
      }))
    } catch (e) {
      console.warn('Failed to save bottom panel state to localStorage:', e)
    }
  }
}, { deep: true })

// Watch and persist copilot sidebar state
watch([copilotSidebarOpen, copilotSidebarWidth], () => {
  if (typeof window !== 'undefined') {
    try {
      localStorage.setItem(STORAGE_KEY_COPILOT_SIDEBAR, JSON.stringify({
        open: copilotSidebarOpen.value,
        width: copilotSidebarWidth.value,
      }))
    } catch (e) {
      console.warn('Failed to save copilot sidebar state to localStorage:', e)
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

  // Bottom panel controls
  function toggleBottomPanel() {
    bottomPanelCollapsed.value = !bottomPanelCollapsed.value
  }

  function setBottomPanelCollapsed(collapsed: boolean) {
    bottomPanelCollapsed.value = collapsed
  }

  function setBottomPanelHeight(height: number) {
    bottomPanelHeight.value = Math.max(100, Math.min(400, height))
  }

  function setBottomPanelResizing(resizing: boolean) {
    bottomPanelResizing.value = resizing
  }

  function setSelectedRun(run: Run | null) {
    selectedRun.value = run
  }

  function setSelectedStepRun(stepRun: StepRun | null) {
    selectedStepRun.value = stepRun
  }

  function clearRunSelection() {
    selectedRun.value = null
    selectedStepRun.value = null
  }

  // Copilot Sidebar controls
  function openCopilotSidebar() {
    copilotSidebarOpen.value = true
  }

  function closeCopilotSidebar() {
    copilotSidebarOpen.value = false
  }

  function toggleCopilotSidebar() {
    copilotSidebarOpen.value = !copilotSidebarOpen.value
  }

  function setCopilotSidebarWidth(width: number) {
    copilotSidebarWidth.value = Math.max(COPILOT_SIDEBAR_MIN_WIDTH, Math.min(COPILOT_SIDEBAR_MAX_WIDTH, width))
  }

  function setCopilotSidebarResizing(resizing: boolean) {
    copilotSidebarResizing.value = resizing
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

    // Bottom panel state
    bottomPanelCollapsed,
    bottomPanelHeight,
    bottomPanelResizing,
    selectedRun,
    selectedStepRun,

    // Copilot Sidebar state
    copilotSidebarOpen: readonly(copilotSidebarOpen),
    copilotSidebarWidth,
    copilotSidebarResizing,

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

    // Bottom panel actions
    toggleBottomPanel,
    setBottomPanelCollapsed,
    setBottomPanelHeight,
    setBottomPanelResizing,
    setSelectedRun,
    setSelectedStepRun,
    clearRunSelection,

    // Copilot Sidebar actions
    openCopilotSidebar,
    closeCopilotSidebar,
    toggleCopilotSidebar,
    setCopilotSidebarWidth,
    setCopilotSidebarResizing,
  }
}
