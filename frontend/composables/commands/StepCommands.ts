/**
 * Step Commands for Undo/Redo functionality
 *
 * Commands for creating, updating, deleting, and moving steps in the workflow.
 */

import type { Command, CommandType } from '../useCommandHistory'
import type { Step, StepType, Edge, Project } from '~/types/api'

type ProjectsApi = ReturnType<typeof useProjects>
type ProjectGetter = () => Project | null

function generateCommandId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

/**
 * Command to create a new step
 */
export class CreateStepCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'step:create'
  readonly timestamp: number
  readonly description: string

  private createdStepId: string | null = null
  private createdStep: Step | null = null

  constructor(
    private projectId: string,
    private stepData: {
      name: string
      type: StepType
      config: object
      position: { x: number; y: number }
      trigger_type?: 'manual' | 'webhook' | 'schedule' | 'slack' | 'email'
      trigger_config?: object
    },
    private projectsApi: ProjectsApi,
    private getProject: ProjectGetter
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = `Create step: ${stepData.name}`
  }

  async execute(): Promise<void> {
    const response = await this.projectsApi.createStep(this.projectId, this.stepData)
    this.createdStepId = response.data.id
    this.createdStep = response.data

    // Update local state
    const project = this.getProject()
    if (project) {
      project.steps = [...(project.steps || []), response.data]
    }
  }

  async undo(): Promise<void> {
    if (!this.createdStepId) {
      throw new Error('Cannot undo: step was not created')
    }

    await this.projectsApi.deleteStep(this.projectId, this.createdStepId)

    // Update local state
    const project = this.getProject()
    if (project?.steps) {
      project.steps = project.steps.filter(s => s.id !== this.createdStepId)
    }
  }

  /** Get the created step ID (available after execute) */
  getCreatedStepId(): string | null {
    return this.createdStepId
  }

  /** Get the created step (available after execute) */
  getCreatedStep(): Step | null {
    return this.createdStep
  }
}

/**
 * Command to update a step (name, config, etc.)
 */
export class UpdateStepCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'step:update'
  readonly timestamp: number
  readonly description: string

  constructor(
    private projectId: string,
    private stepId: string,
    private beforeState: Partial<Step>,
    private afterState: Partial<Step>,
    private projectsApi: ProjectsApi,
    private getProject: ProjectGetter
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = `Update step: ${beforeState.name || stepId}`
  }

  async execute(): Promise<void> {
    await this.projectsApi.updateStep(this.projectId, this.stepId, this.afterState)

    // Update local state
    const project = this.getProject()
    if (project?.steps) {
      const stepIndex = project.steps.findIndex(s => s.id === this.stepId)
      if (stepIndex >= 0) {
        project.steps[stepIndex] = {
          ...project.steps[stepIndex],
          ...this.afterState,
        }
      }
    }
  }

  async undo(): Promise<void> {
    await this.projectsApi.updateStep(this.projectId, this.stepId, this.beforeState)

    // Update local state
    const project = this.getProject()
    if (project?.steps) {
      const stepIndex = project.steps.findIndex(s => s.id === this.stepId)
      if (stepIndex >= 0) {
        project.steps[stepIndex] = {
          ...project.steps[stepIndex],
          ...this.beforeState,
        }
      }
    }
  }
}

/**
 * Command to move a step (update position only)
 */
export class MoveStepCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'step:move'
  readonly timestamp: number
  readonly description: string

  constructor(
    private projectId: string,
    private stepId: string,
    private stepName: string,
    private beforePosition: { x: number; y: number },
    private afterPosition: { x: number; y: number },
    private projectsApi: ProjectsApi,
    private getProject: ProjectGetter
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = `Move step: ${stepName}`
  }

  async execute(): Promise<void> {
    await this.projectsApi.updateStep(this.projectId, this.stepId, {
      position: this.afterPosition,
    })

    // Update local state
    const project = this.getProject()
    if (project?.steps) {
      const step = project.steps.find(s => s.id === this.stepId)
      if (step) {
        step.position_x = this.afterPosition.x
        step.position_y = this.afterPosition.y
      }
    }
  }

  async undo(): Promise<void> {
    await this.projectsApi.updateStep(this.projectId, this.stepId, {
      position: this.beforePosition,
    })

    // Update local state
    const project = this.getProject()
    if (project?.steps) {
      const step = project.steps.find(s => s.id === this.stepId)
      if (step) {
        step.position_x = this.beforePosition.x
        step.position_y = this.beforePosition.y
      }
    }
  }
}

/**
 * Command to delete a step
 * Stores the deleted step and connected edges for restoration
 */
export class DeleteStepCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'step:delete'
  readonly timestamp: number
  readonly description: string

  private connectedEdges: Edge[] = []
  private recreatedStepId: string | null = null

  constructor(
    private projectId: string,
    private deletedStep: Step,
    private projectsApi: ProjectsApi,
    private getProject: ProjectGetter
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = `Delete step: ${deletedStep.name}`

    // Store connected edges for restoration (capture at construction time)
    const project = this.getProject()
    if (project?.edges) {
      this.connectedEdges = project.edges.filter(
        e => e.source_step_id === deletedStep.id || e.target_step_id === deletedStep.id
      )
    }
  }

  async execute(): Promise<void> {
    await this.projectsApi.deleteStep(this.projectId, this.deletedStep.id)

    // Update local state
    const project = this.getProject()
    if (project) {
      project.steps = (project.steps || []).filter(
        s => s.id !== this.deletedStep.id
      )
      project.edges = (project.edges || []).filter(
        e => e.source_step_id !== this.deletedStep.id && e.target_step_id !== this.deletedStep.id
      )
    }
  }

  async undo(): Promise<void> {
    // Recreate the step
    const response = await this.projectsApi.createStep(this.projectId, {
      name: this.deletedStep.name,
      type: this.deletedStep.type,
      config: this.deletedStep.config,
      position: { x: this.deletedStep.position_x, y: this.deletedStep.position_y },
    })

    this.recreatedStepId = response.data.id

    // Update local state with the new step
    const project = this.getProject()
    if (project) {
      project.steps = [...(project.steps || []), response.data]
    }

    // Note: Edges cannot be fully restored because the step has a new ID
    // This is a known limitation mentioned in the plan
    // For full edge restoration, we would need the backend to support step recreation with same ID
  }

  /** Get the recreated step ID (available after undo, will be different from original) */
  getRecreatedStepId(): string | null {
    return this.recreatedStepId
  }
}
