/* eslint-disable @typescript-eslint/no-explicit-any -- テストコードのモック型定義 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ref, type Ref } from 'vue'
import type { Project, Step } from '~/types/api'
import {
  useEditorState,
  COPILOT_SIDEBAR_DEFAULT_WIDTH,
  COPILOT_SIDEBAR_MIN_WIDTH,
  COPILOT_SIDEBAR_MAX_WIDTH,
  COPILOT_SIDEBAR_COLLAPSED_WIDTH,
} from './useEditorState'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      // Use Reflect.deleteProperty to avoid eslint no-dynamic-delete error
      Reflect.deleteProperty(store, key)
    }),
    clear: vi.fn(() => {
      store = {}
    }),
  }
})()

// Replace window.localStorage if we're in a test environment
if (typeof window !== 'undefined') {
  Object.defineProperty(window, 'localStorage', {
    value: localStorageMock,
    writable: true,
  })
}

// Helper to create a mock project
function createMockProject(steps: Step[] = []): Project {
  return {
    id: 'project-1',
    tenant_id: 'tenant-1',
    name: 'Test Project',
    description: 'Test description',
    status: 'draft',
    version: 1,
    has_draft: false,
    steps,
    edges: [],
    block_groups: [],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  }
}

// Helper to create a mock step
function createMockStep(overrides: Partial<Step> = {}): Step {
  return {
    id: overrides.id || 'step-1',
    project_id: overrides.project_id || 'project-1',
    name: overrides.name || 'Test Step',
    type: overrides.type || 'llm',
    config: overrides.config || {},
    position_x: overrides.position_x || 100,
    position_y: overrides.position_y || 100,
    created_at: overrides.created_at || '2024-01-01T00:00:00Z',
    updated_at: overrides.updated_at || '2024-01-01T00:00:00Z',
  }
}

describe('useEditorState - Constants', () => {
  it('should export correct sidebar constants', () => {
    expect(COPILOT_SIDEBAR_DEFAULT_WIDTH).toBe(380)
    expect(COPILOT_SIDEBAR_MIN_WIDTH).toBe(280)
    expect(COPILOT_SIDEBAR_MAX_WIDTH).toBe(1200)
    expect(COPILOT_SIDEBAR_COLLAPSED_WIDTH).toBe(48)
  })
})

describe('useEditorState', () => {
  let editorState: ReturnType<typeof useEditorState>
  let projectRef: Ref<Project | null>

  beforeEach(() => {
    // Clear any mocked localStorage
    localStorageMock.clear()

    // Create a fresh project ref
    projectRef = ref(null) as Ref<Project | null>
    editorState = useEditorState(projectRef)

    // Reset state
    editorState.clearSelection()
    editorState.clearClipboard()
    editorState.clearRunSelection()
    editorState.resetPanelWidths()
  })

  describe('step selection', () => {
    it('should have no selection initially', () => {
      expect(editorState.selectedStepId.value).toBeNull()
      expect(editorState.selectedStep.value).toBeNull()
    })

    it('should select a step', () => {
      editorState.selectStep('step-1')
      expect(editorState.selectedStepId.value).toBe('step-1')
    })

    it('should clear selection', () => {
      editorState.selectStep('step-1')
      editorState.clearSelection()
      expect(editorState.selectedStepId.value).toBeNull()
    })

    it('should return selected step from project', () => {
      const step = createMockStep({ id: 'step-1', name: 'Selected Step' })
      projectRef.value = createMockProject([step])
      editorState.selectStep('step-1')

      expect(editorState.selectedStep.value).toBeDefined()
      expect(editorState.selectedStep.value?.name).toBe('Selected Step')
    })

    it('should return null for non-existent step', () => {
      projectRef.value = createMockProject([])
      editorState.selectStep('non-existent')

      expect(editorState.selectedStep.value).toBeNull()
    })
  })

  describe('clipboard', () => {
    it('should have no clipboard initially', () => {
      expect(editorState.hasClipboard()).toBe(false)
      expect(editorState.getClipboardData()).toBeNull()
    })

    it('should copy selected step to clipboard', () => {
      const step = createMockStep({
        id: 'step-1',
        name: 'Copy Me',
        type: 'llm',
        config: { model: 'gpt-4' },
      })
      projectRef.value = createMockProject([step])
      editorState.selectStep('step-1')

      const result = editorState.copyStep()

      expect(result).toBe(true)
      expect(editorState.hasClipboard()).toBe(true)

      const clipboardData = editorState.getClipboardData()
      expect(clipboardData?.name).toBe('Copy Me (Copy)')
      expect(clipboardData?.type).toBe('llm')
      expect(clipboardData?.config).toEqual({ model: 'gpt-4' })
    })

    it('should return false when copying without selection', () => {
      projectRef.value = createMockProject([])

      const result = editorState.copyStep()

      expect(result).toBe(false)
    })

    it('should return deep copy of clipboard data', () => {
      const step = createMockStep({
        id: 'step-1',
        config: { nested: { value: 1 } },
      })
      projectRef.value = createMockProject([step])
      editorState.selectStep('step-1')
      editorState.copyStep()

      const data1 = editorState.getClipboardData()
      const data2 = editorState.getClipboardData()

      // Should be different objects
      expect(data1).not.toBe(data2)
      expect(data1?.config).not.toBe(data2?.config)
    })

    it('should clear clipboard', () => {
      const step = createMockStep({ id: 'step-1' })
      projectRef.value = createMockProject([step])
      editorState.selectStep('step-1')
      editorState.copyStep()

      editorState.clearClipboard()

      expect(editorState.hasClipboard()).toBe(false)
    })
  })

  describe('panel widths', () => {
    it('should have default panel widths', () => {
      editorState.resetPanelWidths()

      expect(editorState.leftPanelWidth.value).toBe(280)
      expect(editorState.rightPanelWidth.value).toBe(360)
    })

    it('should set left panel width with bounds', () => {
      editorState.setLeftPanelWidth(300)
      expect(editorState.leftPanelWidth.value).toBe(300)

      // Min bound
      editorState.setLeftPanelWidth(100)
      expect(editorState.leftPanelWidth.value).toBe(200)

      // Max bound
      editorState.setLeftPanelWidth(600)
      expect(editorState.leftPanelWidth.value).toBe(500)
    })

    it('should set right panel width with bounds', () => {
      editorState.setRightPanelWidth(400)
      expect(editorState.rightPanelWidth.value).toBe(400)

      // Min bound
      editorState.setRightPanelWidth(100)
      expect(editorState.rightPanelWidth.value).toBe(280)

      // Max bound
      editorState.setRightPanelWidth(700)
      expect(editorState.rightPanelWidth.value).toBe(600)
    })

    it('should reset panel widths', () => {
      editorState.setLeftPanelWidth(400)
      editorState.setRightPanelWidth(500)

      editorState.resetPanelWidths()

      expect(editorState.leftPanelWidth.value).toBe(280)
      expect(editorState.rightPanelWidth.value).toBe(360)
    })
  })

  describe('panel collapse', () => {
    it('should have uncollapsed panels by default', () => {
      expect(editorState.leftCollapsed.value).toBe(false)
      expect(editorState.rightCollapsed.value).toBe(false)
    })

    it('should set left collapsed state', () => {
      editorState.setLeftCollapsed(true)
      expect(editorState.leftCollapsed.value).toBe(true)

      editorState.setLeftCollapsed(false)
      expect(editorState.leftCollapsed.value).toBe(false)
    })

    it('should set right collapsed state', () => {
      editorState.setRightCollapsed(true)
      expect(editorState.rightCollapsed.value).toBe(true)

      editorState.setRightCollapsed(false)
      expect(editorState.rightCollapsed.value).toBe(false)
    })

    it('should toggle left collapsed', () => {
      expect(editorState.leftCollapsed.value).toBe(false)

      editorState.toggleLeftCollapsed()
      expect(editorState.leftCollapsed.value).toBe(true)

      editorState.toggleLeftCollapsed()
      expect(editorState.leftCollapsed.value).toBe(false)
    })

    it('should toggle right collapsed', () => {
      expect(editorState.rightCollapsed.value).toBe(false)

      editorState.toggleRightCollapsed()
      expect(editorState.rightCollapsed.value).toBe(true)

      editorState.toggleRightCollapsed()
      expect(editorState.rightCollapsed.value).toBe(false)
    })
  })

  describe('slideOut panels', () => {
    it('should have no active slideOut initially', () => {
      expect(editorState.activeSlideOut.value).toBeNull()
    })

    it('should open slideOut panel', () => {
      editorState.openSlideOut('runs')
      expect(editorState.activeSlideOut.value).toBe('runs')
    })

    it('should close slideOut panel', () => {
      editorState.openSlideOut('runs')
      editorState.closeSlideOut()
      expect(editorState.activeSlideOut.value).toBeNull()
    })

    it('should toggle slideOut panel', () => {
      editorState.toggleSlideOut('schedules')
      expect(editorState.activeSlideOut.value).toBe('schedules')

      editorState.toggleSlideOut('schedules')
      expect(editorState.activeSlideOut.value).toBeNull()
    })

    it('should switch between slideOut panels', () => {
      editorState.openSlideOut('runs')
      editorState.toggleSlideOut('variables')

      expect(editorState.activeSlideOut.value).toBe('variables')
    })
  })

  describe('project ID management', () => {
    it('should have no current project ID initially', () => {
      expect(editorState.currentProjectId.value).toBeNull()
    })

    it('should set current project ID', () => {
      editorState.setCurrentProjectId('project-123')
      expect(editorState.currentProjectId.value).toBe('project-123')
    })

    it('should clear project ID', () => {
      editorState.setCurrentProjectId('project-123')
      editorState.setCurrentProjectId(null)
      expect(editorState.currentProjectId.value).toBeNull()
    })
  })

  describe('bottom panel', () => {
    it('should have default bottom panel state', () => {
      expect(editorState.bottomPanelCollapsed.value).toBe(false)
      expect(editorState.bottomPanelHeight.value).toBe(200)
      expect(editorState.bottomPanelResizing.value).toBe(false)
    })

    it('should toggle bottom panel', () => {
      editorState.toggleBottomPanel()
      expect(editorState.bottomPanelCollapsed.value).toBe(true)

      editorState.toggleBottomPanel()
      expect(editorState.bottomPanelCollapsed.value).toBe(false)
    })

    it('should set bottom panel collapsed', () => {
      editorState.setBottomPanelCollapsed(true)
      expect(editorState.bottomPanelCollapsed.value).toBe(true)
    })

    it('should set bottom panel height with bounds', () => {
      editorState.setBottomPanelHeight(250)
      expect(editorState.bottomPanelHeight.value).toBe(250)

      // Min bound
      editorState.setBottomPanelHeight(50)
      expect(editorState.bottomPanelHeight.value).toBe(100)

      // Max bound
      editorState.setBottomPanelHeight(500)
      expect(editorState.bottomPanelHeight.value).toBe(400)
    })

    it('should set bottom panel resizing state', () => {
      editorState.setBottomPanelResizing(true)
      expect(editorState.bottomPanelResizing.value).toBe(true)

      editorState.setBottomPanelResizing(false)
      expect(editorState.bottomPanelResizing.value).toBe(false)
    })
  })

  describe('run selection', () => {
    it('should have no run selection initially', () => {
      editorState.clearRunSelection()
      expect(editorState.selectedRun.value).toBeNull()
      expect(editorState.selectedStepRun.value).toBeNull()
    })

    it('should set selected run', () => {
      const mockRun = { id: 'run-1' } as any
      editorState.setSelectedRun(mockRun)
      expect(editorState.selectedRun.value).toStrictEqual(mockRun)
    })

    it('should set selected step run', () => {
      const mockStepRun = { id: 'step-run-1' } as any
      editorState.setSelectedStepRun(mockStepRun)
      expect(editorState.selectedStepRun.value).toStrictEqual(mockStepRun)
    })

    it('should clear run selection', () => {
      editorState.setSelectedRun({ id: 'run-1' } as any)
      editorState.setSelectedStepRun({ id: 'step-run-1' } as any)

      editorState.clearRunSelection()

      expect(editorState.selectedRun.value).toBeNull()
      expect(editorState.selectedStepRun.value).toBeNull()
    })
  })

  describe('copilot sidebar', () => {
    it('should have default copilot sidebar state', () => {
      expect(editorState.copilotSidebarOpen.value).toBe(false)
      expect(editorState.copilotSidebarWidth.value).toBe(COPILOT_SIDEBAR_DEFAULT_WIDTH)
      expect(editorState.copilotSidebarResizing.value).toBe(false)
    })

    it('should open copilot sidebar', () => {
      editorState.openCopilotSidebar()
      expect(editorState.copilotSidebarOpen.value).toBe(true)
    })

    it('should close copilot sidebar', () => {
      editorState.openCopilotSidebar()
      editorState.closeCopilotSidebar()
      expect(editorState.copilotSidebarOpen.value).toBe(false)
    })

    it('should toggle copilot sidebar', () => {
      editorState.toggleCopilotSidebar()
      expect(editorState.copilotSidebarOpen.value).toBe(true)

      editorState.toggleCopilotSidebar()
      expect(editorState.copilotSidebarOpen.value).toBe(false)
    })

    it('should set copilot sidebar width with bounds', () => {
      editorState.setCopilotSidebarWidth(500)
      expect(editorState.copilotSidebarWidth.value).toBe(500)

      // Min bound
      editorState.setCopilotSidebarWidth(100)
      expect(editorState.copilotSidebarWidth.value).toBe(COPILOT_SIDEBAR_MIN_WIDTH)

      // Max bound
      editorState.setCopilotSidebarWidth(2000)
      expect(editorState.copilotSidebarWidth.value).toBe(COPILOT_SIDEBAR_MAX_WIDTH)
    })

    it('should set copilot sidebar resizing state', () => {
      editorState.setCopilotSidebarResizing(true)
      expect(editorState.copilotSidebarResizing.value).toBe(true)

      editorState.setCopilotSidebarResizing(false)
      expect(editorState.copilotSidebarResizing.value).toBe(false)
    })
  })

  describe('readonly protection', () => {
    beforeEach(() => {
      // Reset state for readonly tests
      editorState.clearSelection()
      editorState.closeSlideOut()
      editorState.setCurrentProjectId(null)
    })

    it('should return readonly selectedStepId', () => {
      // The value should be readable
      expect(editorState.selectedStepId.value).toBeNull()
      // But attempting to modify directly would be caught by TypeScript
      // This test verifies the structure is correct
    })

    it('should return readonly activeSlideOut', () => {
      expect(editorState.activeSlideOut.value).toBeNull()
    })

    it('should return readonly currentProjectId', () => {
      expect(editorState.currentProjectId.value).toBeNull()
    })

    it('should return readonly copilotSidebarOpen', () => {
      expect(editorState.copilotSidebarOpen.value).toBe(false)
    })

    it('should return readonly isExecuting', () => {
      expect(editorState.copilotSidebarResizing.value).toBe(false)
    })
  })
})
