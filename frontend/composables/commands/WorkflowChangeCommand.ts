/**
 * WorkflowChangeCommand - Generic diff-based command for Undo/Redo
 *
 * This command handles complex workflow changes that may involve multiple
 * steps, groups, and edges in a single operation. It uses a diff-based
 * approach where only the changed properties are recorded.
 */

import type { Command, CommandType } from '../useCommandHistory'
import type { Edge, BlockGroup, Project, GroupRole } from '~/types/api'

type ProjectsApi = ReturnType<typeof useProjects>
type BlockGroupsApi = ReturnType<typeof useBlockGroups>
type ProjectGetter = () => Project | null
type BlockGroupsGetter = () => BlockGroup[]

function generateCommandId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

/**
 * Change record for a step
 */
export interface StepChange {
  id: string
  before: {
    name?: string
    config?: object
    position_x?: number
    position_y?: number
    block_group_id?: string | null
    group_role?: GroupRole | null
  }
  after: {
    name?: string
    config?: object
    position_x?: number
    position_y?: number
    block_group_id?: string | null
    group_role?: GroupRole | null
  }
}

/**
 * Change record for a block group
 */
export interface GroupChange {
  id: string
  before: {
    name?: string
    config?: object
    position_x?: number
    position_y?: number
    width?: number
    height?: number
  }
  after: {
    name?: string
    config?: object
    position_x?: number
    position_y?: number
    width?: number
    height?: number
  }
}

/**
 * Change record for an edge (create/delete only)
 */
export interface EdgeChange {
  id: string
  action: 'create' | 'delete'
  data: Partial<Edge>
}

/**
 * Aggregated workflow changes
 */
export interface WorkflowChanges {
  steps?: StepChange[]
  groups?: GroupChange[]
  edges?: EdgeChange[]
}

/**
 * WorkflowChangeCommand implements the Command pattern for generic workflow changes.
 * It records the before/after state of steps, groups, and edges, allowing for
 * complex multi-entity operations to be undone/redone as a single unit.
 */
export class WorkflowChangeCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'workflow:change'
  readonly timestamp: number
  readonly description: string

  constructor(
    private projectId: string,
    private changes: WorkflowChanges,
    description: string,
    private projectsApi: ProjectsApi,
    private blockGroupsApi: BlockGroupsApi,
    private getProject: ProjectGetter,
    private getBlockGroups: BlockGroupsGetter
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = description
  }

  async execute(): Promise<void> {
    await this.applyChanges('after')
  }

  async undo(): Promise<void> {
    await this.applyChanges('before')
  }

  private async applyChanges(direction: 'before' | 'after'): Promise<void> {
    const promises: Promise<unknown>[] = []

    // Apply step changes
    for (const stepChange of this.changes.steps || []) {
      const state = direction === 'after' ? stepChange.after : stepChange.before
      promises.push(this.applyStepChange(stepChange.id, state))
    }

    // Apply group changes
    for (const groupChange of this.changes.groups || []) {
      const state = direction === 'after' ? groupChange.after : groupChange.before
      promises.push(this.applyGroupChange(groupChange.id, state))
    }

    // Apply edge changes (reverse for undo)
    for (const edgeChange of this.changes.edges || []) {
      if (direction === 'after') {
        if (edgeChange.action === 'create') {
          promises.push(this.createEdge(edgeChange.data))
        } else {
          promises.push(this.deleteEdge(edgeChange.id))
        }
      } else {
        // Undo: reverse the operation
        if (edgeChange.action === 'create') {
          promises.push(this.deleteEdge(edgeChange.id))
        } else {
          promises.push(this.createEdge(edgeChange.data))
        }
      }
    }

    await Promise.all(promises)

    // Trigger reactivity by creating new array references
    // This ensures Vue Flow and other components detect the changes
    const project = this.getProject()
    const blockGroups = this.getBlockGroups()
    if (project?.steps && this.changes.steps?.length) {
      project.steps = [...project.steps]
    }
    if (blockGroups.length && this.changes.groups?.length) {
      // Note: We can't reassign the array directly since it's from a getter
      // The individual mutations above should trigger reactivity
    }
  }

  private async applyStepChange(stepId: string, state: StepChange['before']): Promise<void> {
    const project = this.getProject()
    if (!project) return

    // Update local state
    const step = project.steps?.find(s => s.id === stepId)
    if (step) {
      if (state.name !== undefined) step.name = state.name
      if (state.config !== undefined) step.config = state.config
      if (state.position_x !== undefined) step.position_x = state.position_x
      if (state.position_y !== undefined) step.position_y = state.position_y
      if ('block_group_id' in state) step.block_group_id = state.block_group_id ?? undefined
      if ('group_role' in state) step.group_role = state.group_role ?? undefined
    }

    // Build API update data
    const updateData: Parameters<typeof this.projectsApi.updateStep>[2] = {}
    if (state.name !== undefined) updateData.name = state.name
    if (state.config !== undefined) updateData.config = state.config
    if (state.position_x !== undefined || state.position_y !== undefined) {
      updateData.position = {
        x: state.position_x ?? step?.position_x ?? 0,
        y: state.position_y ?? step?.position_y ?? 0,
      }
    }

    // Call API
    await this.projectsApi.updateStep(this.projectId, stepId, updateData)

    // Handle group membership changes
    if ('block_group_id' in state) {
      const currentStep = project.steps?.find(s => s.id === stepId)
      const currentGroupId = currentStep?.block_group_id
      const newGroupId = state.block_group_id

      // Remove from current group if needed
      if (currentGroupId && currentGroupId !== newGroupId) {
        try {
          await this.blockGroupsApi.removeStep(this.projectId, currentGroupId, stepId)
        } catch {
          // Ignore if already removed
        }
      }

      // Add to new group if specified
      if (newGroupId && newGroupId !== currentGroupId) {
        await this.blockGroupsApi.addStep(this.projectId, newGroupId, {
          step_id: stepId,
          group_role: state.group_role ?? 'body',
        })
      }
    }
  }

  private async applyGroupChange(groupId: string, state: GroupChange['before']): Promise<void> {
    // Update local state
    const blockGroups = this.getBlockGroups()
    const group = blockGroups.find(g => g.id === groupId)
    if (group) {
      if (state.name !== undefined) group.name = state.name
      if (state.config !== undefined) group.config = state.config
      if (state.position_x !== undefined) group.position_x = state.position_x
      if (state.position_y !== undefined) group.position_y = state.position_y
      if (state.width !== undefined) group.width = state.width
      if (state.height !== undefined) group.height = state.height
    }

    // Build API update data
    const updateData: Parameters<typeof this.blockGroupsApi.update>[2] = {}
    if (state.name !== undefined) updateData.name = state.name
    if (state.config !== undefined) updateData.config = state.config
    if (state.position_x !== undefined || state.position_y !== undefined) {
      updateData.position = {
        x: state.position_x ?? group?.position_x ?? 0,
        y: state.position_y ?? group?.position_y ?? 0,
      }
    }
    if (state.width !== undefined || state.height !== undefined) {
      updateData.size = {
        width: state.width ?? group?.width ?? 400,
        height: state.height ?? group?.height ?? 300,
      }
    }

    // Call API
    await this.blockGroupsApi.update(this.projectId, groupId, updateData)
  }

  private async createEdge(data: Partial<Edge>): Promise<void> {
    const project = this.getProject()
    if (!project) return

    const response = await this.projectsApi.createEdge(this.projectId, {
      source_step_id: data.source_step_id ?? undefined,
      target_step_id: data.target_step_id ?? undefined,
      source_block_group_id: data.source_block_group_id ?? undefined,
      target_block_group_id: data.target_block_group_id ?? undefined,
      source_port: data.source_port,
      condition: data.condition,
    })

    // Update local state
    if (project.edges) {
      project.edges = [...project.edges, response.data]
    } else {
      project.edges = [response.data]
    }
  }

  private async deleteEdge(edgeId: string): Promise<void> {
    const project = this.getProject()
    if (!project) return

    try {
      await this.projectsApi.deleteEdge(this.projectId, edgeId)
    } catch {
      // Ignore if already deleted
    }

    // Update local state
    if (project.edges) {
      project.edges = project.edges.filter(e => e.id !== edgeId)
    }
  }
}

/**
 * Helper function to create step changes for position updates
 */
export function createStepPositionChanges(
  steps: Array<{ id: string; name: string; before: { x: number; y: number }; after: { x: number; y: number } }>
): StepChange[] {
  return steps.map(s => ({
    id: s.id,
    before: { position_x: s.before.x, position_y: s.before.y },
    after: { position_x: s.after.x, position_y: s.after.y },
  }))
}

/**
 * Helper function to create group changes for position/size updates
 */
export function createGroupPositionChanges(
  groups: Array<{
    id: string
    before: { x: number; y: number; width?: number; height?: number }
    after: { x: number; y: number; width?: number; height?: number }
  }>
): GroupChange[] {
  return groups.map(g => ({
    id: g.id,
    before: {
      position_x: g.before.x,
      position_y: g.before.y,
      width: g.before.width,
      height: g.before.height,
    },
    after: {
      position_x: g.after.x,
      position_y: g.after.y,
      width: g.after.width,
      height: g.after.height,
    },
  }))
}
