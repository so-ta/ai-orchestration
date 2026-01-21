/**
 * Edge Commands for Undo/Redo functionality
 *
 * Commands for creating and deleting edges in the workflow.
 */

import type { Command, CommandType } from '../useCommandHistory'
import type { Edge, Project } from '~/types/api'

type ProjectsApi = ReturnType<typeof useProjects>
type ProjectGetter = () => Project | null

function generateCommandId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

/**
 * Command to create a new edge
 */
export class CreateEdgeCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'edge:create'
  readonly timestamp: number
  readonly description: string

  private createdEdgeId: string | null = null
  private createdEdge: Edge | null = null

  constructor(
    private projectId: string,
    private edgeData: {
      source_step_id?: string
      target_step_id?: string
      source_block_group_id?: string
      target_block_group_id?: string
      source_port?: string
      target_port?: string
      condition?: string
    },
    private projectsApi: ProjectsApi,
    private getProject: ProjectGetter
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()

    // Generate description based on edge type
    const sourceDesc = edgeData.source_step_id || edgeData.source_block_group_id || 'unknown'
    const targetDesc = edgeData.target_step_id || edgeData.target_block_group_id || 'unknown'
    this.description = `Create edge: ${sourceDesc.slice(0, 8)} → ${targetDesc.slice(0, 8)}`
  }

  async execute(): Promise<void> {
    const response = await this.projectsApi.createEdge(this.projectId, this.edgeData)
    this.createdEdgeId = response.data.id
    this.createdEdge = response.data

    // Update local state
    const project = this.getProject()
    if (project) {
      project.edges = [...(project.edges || []), response.data]
    }
  }

  async undo(): Promise<void> {
    if (!this.createdEdgeId) {
      throw new Error('Cannot undo: edge was not created')
    }

    await this.projectsApi.deleteEdge(this.projectId, this.createdEdgeId)

    // Update local state
    const project = this.getProject()
    if (project?.edges) {
      project.edges = project.edges.filter(e => e.id !== this.createdEdgeId)
    }
  }

  /** Get the created edge ID (available after execute) */
  getCreatedEdgeId(): string | null {
    return this.createdEdgeId
  }

  /** Get the created edge (available after execute) */
  getCreatedEdge(): Edge | null {
    return this.createdEdge
  }
}

/**
 * Command to delete an edge
 */
export class DeleteEdgeCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'edge:delete'
  readonly timestamp: number
  readonly description: string

  private recreatedEdgeId: string | null = null

  constructor(
    private projectId: string,
    private deletedEdge: Edge,
    private projectsApi: ProjectsApi,
    private getProject: ProjectGetter
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()

    const sourceDesc = deletedEdge.source_step_id || deletedEdge.source_block_group_id || 'unknown'
    const targetDesc = deletedEdge.target_step_id || deletedEdge.target_block_group_id || 'unknown'
    this.description = `Delete edge: ${sourceDesc.slice(0, 8)} → ${targetDesc.slice(0, 8)}`
  }

  async execute(): Promise<void> {
    await this.projectsApi.deleteEdge(this.projectId, this.deletedEdge.id)

    // Update local state
    const project = this.getProject()
    if (project?.edges) {
      project.edges = project.edges.filter(e => e.id !== this.deletedEdge.id)
    }
  }

  async undo(): Promise<void> {
    // Recreate the edge
    const edgeData: Parameters<typeof this.projectsApi.createEdge>[1] = {}

    if (this.deletedEdge.source_step_id) {
      edgeData.source_step_id = this.deletedEdge.source_step_id
    }
    if (this.deletedEdge.target_step_id) {
      edgeData.target_step_id = this.deletedEdge.target_step_id
    }
    if (this.deletedEdge.source_block_group_id) {
      edgeData.source_block_group_id = this.deletedEdge.source_block_group_id
    }
    if (this.deletedEdge.target_block_group_id) {
      edgeData.target_block_group_id = this.deletedEdge.target_block_group_id
    }
    if (this.deletedEdge.source_port) {
      edgeData.source_port = this.deletedEdge.source_port
    }
    if (this.deletedEdge.target_port) {
      edgeData.target_port = this.deletedEdge.target_port
    }
    if (this.deletedEdge.condition) {
      edgeData.condition = this.deletedEdge.condition
    }

    const response = await this.projectsApi.createEdge(this.projectId, edgeData)
    this.recreatedEdgeId = response.data.id

    // Update local state
    const project = this.getProject()
    if (project) {
      project.edges = [...(project.edges || []), response.data]
    }
  }

  /** Get the recreated edge ID (available after undo, will be different from original) */
  getRecreatedEdgeId(): string | null {
    return this.recreatedEdgeId
  }
}
