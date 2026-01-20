/**
 * Copilot Draft composable for preview/draft mode
 *
 * Manages Copilot changes before they are applied to the project.
 * - Accumulates changes during AI tool calls
 * - Determines if changes are "small" (auto-apply) or "large" (require preview)
 * - Provides preview state for visual highlighting in DagEditor
 * - Converts draft to batch command for Undo/Redo integration
 */

import type { Step, StepType } from '~/types/api'
import type { Command } from './useCommandHistory'

export type DraftStatus = 'idle' | 'collecting' | 'previewing' | 'applying' | 'applied' | 'discarded'

export type DraftChangeType =
  | 'step:create'
  | 'step:update'
  | 'step:delete'
  | 'edge:create'
  | 'edge:delete'

export interface DraftStepCreate {
  type: 'step:create'
  tempId: string
  stepType: StepType
  name: string
  config: object
  position: { x: number; y: number }
}

export interface DraftStepUpdate {
  type: 'step:update'
  stepId: string
  patch: Partial<Step>
}

export interface DraftStepDelete {
  type: 'step:delete'
  stepId: string
}

export interface DraftEdgeCreate {
  type: 'edge:create'
  sourceId: string
  targetId: string
  sourcePort?: string
  targetPort?: string
}

export interface DraftEdgeDelete {
  type: 'edge:delete'
  edgeId: string
}

export type DraftChange =
  | DraftStepCreate
  | DraftStepUpdate
  | DraftStepDelete
  | DraftEdgeCreate
  | DraftEdgeDelete

export interface CopilotDraft {
  id: string
  status: DraftStatus
  description: string
  changes: DraftChange[]
  createdAt: number
}

// Readonly version for external consumers
export type ReadonlyCopilotDraft = Readonly<Omit<CopilotDraft, 'changes'>> & {
  readonly changes: readonly DraftChange[]
}

export interface PreviewState {
  addedStepIds: Set<string>
  modifiedStepIds: Set<string>
  deletedStepIds: Set<string>
  addedEdgeIds: Set<string>
  deletedEdgeIds: Set<string>
}

// Thresholds for auto-apply decision
const SMALL_CHANGE_THRESHOLDS = {
  maxTotalChanges: 2,
  maxStepCreates: 0,
  maxStepDeletes: 0,
}

// Global state for draft management
const currentDraft = ref<CopilotDraft | null>(null)

function generateDraftId(): string {
  return `draft-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

/**
 * Copilot Draft composable
 */
export function useCopilotDraft() {
  // Computed: preview state for DagEditor highlighting
  const previewState = computed<PreviewState | null>(() => {
    if (!currentDraft.value || currentDraft.value.status === 'idle') {
      return null
    }

    const state: PreviewState = {
      addedStepIds: new Set(),
      modifiedStepIds: new Set(),
      deletedStepIds: new Set(),
      addedEdgeIds: new Set(),
      deletedEdgeIds: new Set(),
    }

    for (const change of currentDraft.value.changes) {
      switch (change.type) {
        case 'step:create':
          state.addedStepIds.add(change.tempId)
          break
        case 'step:update':
          state.modifiedStepIds.add(change.stepId)
          break
        case 'step:delete':
          state.deletedStepIds.add(change.stepId)
          break
        case 'edge:create':
          // Use a composite key for edge identification
          state.addedEdgeIds.add(`${change.sourceId}->${change.targetId}`)
          break
        case 'edge:delete':
          state.deletedEdgeIds.add(change.edgeId)
          break
      }
    }

    return state
  })

  // Computed: whether we're in preview mode
  const isPreviewing = computed(() => currentDraft.value?.status === 'previewing')

  // Computed: whether we're collecting changes
  const isCollecting = computed(() => currentDraft.value?.status === 'collecting')

  // Computed: change summary
  const changeSummary = computed(() => {
    if (!currentDraft.value) {
      return { additions: 0, modifications: 0, deletions: 0, total: 0 }
    }

    let additions = 0
    let modifications = 0
    let deletions = 0

    for (const change of currentDraft.value.changes) {
      switch (change.type) {
        case 'step:create':
        case 'edge:create':
          additions++
          break
        case 'step:update':
          modifications++
          break
        case 'step:delete':
        case 'edge:delete':
          deletions++
          break
      }
    }

    return {
      additions,
      modifications,
      deletions,
      total: additions + modifications + deletions,
    }
  })

  /**
   * Start a new draft for collecting Copilot changes
   */
  function startDraft(description: string): string {
    const id = generateDraftId()
    currentDraft.value = {
      id,
      status: 'collecting',
      description,
      changes: [],
      createdAt: Date.now(),
    }
    return id
  }

  /**
   * Add a change to the current draft
   */
  function addToDraft(change: DraftChange): void {
    if (!currentDraft.value || currentDraft.value.status !== 'collecting') {
      console.warn('[CopilotDraft] Cannot add change: no active draft or not in collecting state')
      return
    }

    // Deduplicate: if same step is updated multiple times, merge the patches
    if (change.type === 'step:update') {
      const existingUpdate = currentDraft.value.changes.find(
        c => c.type === 'step:update' && (c as DraftStepUpdate).stepId === change.stepId
      ) as DraftStepUpdate | undefined

      if (existingUpdate) {
        existingUpdate.patch = { ...existingUpdate.patch, ...change.patch }
        return
      }
    }

    currentDraft.value.changes.push(change)
  }

  /**
   * Finalize draft collection and determine if it needs preview
   */
  function finalizeDraft(): { needsPreview: boolean } {
    if (!currentDraft.value || currentDraft.value.status !== 'collecting') {
      console.warn('[CopilotDraft] Cannot finalize: no active draft or not in collecting state')
      return { needsPreview: false }
    }

    if (currentDraft.value.changes.length === 0) {
      // No changes, discard the draft
      currentDraft.value = null
      return { needsPreview: false }
    }

    const needsPreview = !isSmallChange()

    if (needsPreview) {
      currentDraft.value.status = 'previewing'
    }

    return { needsPreview }
  }

  /**
   * Determine if current draft represents a "small" change that can be auto-applied
   */
  function isSmallChange(): boolean {
    if (!currentDraft.value) return true

    const changes = currentDraft.value.changes
    const total = changes.length

    if (total > SMALL_CHANGE_THRESHOLDS.maxTotalChanges) return false

    const stepCreates = changes.filter(c => c.type === 'step:create').length
    const stepDeletes = changes.filter(c => c.type === 'step:delete').length

    if (stepCreates > SMALL_CHANGE_THRESHOLDS.maxStepCreates) return false
    if (stepDeletes > SMALL_CHANGE_THRESHOLDS.maxStepDeletes) return false

    return true
  }

  /**
   * Convert draft changes to commands and apply them via command history
   */
  async function applyDraft(
    commandFactory: (change: DraftChange) => Promise<Command | null>
  ): Promise<boolean> {
    if (!currentDraft.value) {
      console.warn('[CopilotDraft] No draft to apply')
      return false
    }

    currentDraft.value.status = 'applying'

    try {
      const commands: Command[] = []

      for (const change of currentDraft.value.changes) {
        const cmd = await commandFactory(change)
        if (cmd) {
          commands.push(cmd)
        }
      }

      if (commands.length > 0) {
        const { executeBatch } = useCommandHistory()
        await executeBatch(commands, currentDraft.value.description)
      }

      currentDraft.value.status = 'applied'
      // Clear draft after successful apply
      currentDraft.value = null
      return true
    } catch (error) {
      console.error('[CopilotDraft] Failed to apply draft:', error)
      if (currentDraft.value) {
        currentDraft.value.status = 'previewing' // Revert to previewing state
      }
      throw error
    }
  }

  /**
   * Discard the current draft
   */
  function discardDraft(): void {
    if (currentDraft.value) {
      currentDraft.value.status = 'discarded'
      currentDraft.value = null
    }
  }

  /**
   * Get the current draft (for inspection)
   */
  function getDraft(): CopilotDraft | null {
    return currentDraft.value
  }

  return {
    // State
    currentDraft: readonly(currentDraft),
    previewState,
    isPreviewing,
    isCollecting,
    changeSummary,

    // Actions
    startDraft,
    addToDraft,
    finalizeDraft,
    applyDraft,
    discardDraft,
    isSmallChange,
    getDraft,
  }
}
