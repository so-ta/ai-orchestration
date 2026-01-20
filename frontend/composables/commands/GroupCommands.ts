/**
 * Group Commands for Undo/Redo functionality
 *
 * Commands for creating, updating, deleting, and moving block groups.
 */

import type { Command, CommandType } from '../useCommandHistory'
import type { BlockGroup, BlockGroupType, BlockGroupConfig, Project } from '~/types/api'
import type { Ref } from 'vue'

type BlockGroupsApi = ReturnType<typeof useBlockGroups>

function generateCommandId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

/**
 * Command to create a new block group
 */
export class CreateGroupCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'group:create'
  readonly timestamp: number
  readonly description: string

  private createdGroupId: string | null = null
  private createdGroup: BlockGroup | null = null

  constructor(
    private projectId: string,
    private groupData: {
      name: string
      type: BlockGroupType
      config?: BlockGroupConfig
      position: { x: number; y: number }
      size: { width: number; height: number }
    },
    private blockGroupsApi: BlockGroupsApi,
    private blockGroups: Ref<BlockGroup[]>
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = `Create group: ${groupData.name}`
  }

  async execute(): Promise<void> {
    const response = await this.blockGroupsApi.create(this.projectId, this.groupData)
    if (response?.data) {
      this.createdGroupId = response.data.id
      this.createdGroup = response.data

      // Update local state
      this.blockGroups.value = [...this.blockGroups.value, response.data]
    }
  }

  async undo(): Promise<void> {
    if (!this.createdGroupId) {
      throw new Error('Cannot undo: group was not created')
    }

    await this.blockGroupsApi.remove(this.projectId, this.createdGroupId)

    // Update local state
    this.blockGroups.value = this.blockGroups.value.filter(g => g.id !== this.createdGroupId)
  }

  /** Get the created group ID (available after execute) */
  getCreatedGroupId(): string | null {
    return this.createdGroupId
  }

  /** Get the created group (available after execute) */
  getCreatedGroup(): BlockGroup | null {
    return this.createdGroup
  }
}

/**
 * Command to update a group (name, config, etc.)
 */
export class UpdateGroupCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'group:update'
  readonly timestamp: number
  readonly description: string

  constructor(
    private projectId: string,
    private groupId: string,
    private groupName: string,
    private beforeState: Partial<BlockGroup>,
    private afterState: Partial<BlockGroup>,
    private blockGroupsApi: BlockGroupsApi,
    private blockGroups: Ref<BlockGroup[]>
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = `Update group: ${groupName}`
  }

  async execute(): Promise<void> {
    const updateData: Parameters<typeof this.blockGroupsApi.update>[2] = {}

    if (this.afterState.name !== undefined) updateData.name = this.afterState.name
    if (this.afterState.config !== undefined) updateData.config = this.afterState.config
    if (this.afterState.position_x !== undefined || this.afterState.position_y !== undefined) {
      updateData.position = {
        x: this.afterState.position_x ?? this.beforeState.position_x ?? 0,
        y: this.afterState.position_y ?? this.beforeState.position_y ?? 0,
      }
    }
    if (this.afterState.width !== undefined || this.afterState.height !== undefined) {
      updateData.size = {
        width: this.afterState.width ?? this.beforeState.width ?? 400,
        height: this.afterState.height ?? this.beforeState.height ?? 300,
      }
    }

    await this.blockGroupsApi.update(this.projectId, this.groupId, updateData)

    // Update local state
    const groupIndex = this.blockGroups.value.findIndex(g => g.id === this.groupId)
    if (groupIndex >= 0) {
      this.blockGroups.value[groupIndex] = {
        ...this.blockGroups.value[groupIndex],
        ...this.afterState,
      }
    }
  }

  async undo(): Promise<void> {
    const updateData: Parameters<typeof this.blockGroupsApi.update>[2] = {}

    if (this.beforeState.name !== undefined) updateData.name = this.beforeState.name
    if (this.beforeState.config !== undefined) updateData.config = this.beforeState.config
    if (this.beforeState.position_x !== undefined || this.beforeState.position_y !== undefined) {
      updateData.position = {
        x: this.beforeState.position_x ?? 0,
        y: this.beforeState.position_y ?? 0,
      }
    }
    if (this.beforeState.width !== undefined || this.beforeState.height !== undefined) {
      updateData.size = {
        width: this.beforeState.width ?? 400,
        height: this.beforeState.height ?? 300,
      }
    }

    await this.blockGroupsApi.update(this.projectId, this.groupId, updateData)

    // Update local state
    const groupIndex = this.blockGroups.value.findIndex(g => g.id === this.groupId)
    if (groupIndex >= 0) {
      this.blockGroups.value[groupIndex] = {
        ...this.blockGroups.value[groupIndex],
        ...this.beforeState,
      }
    }
  }
}

/**
 * Command to move a group (update position only)
 */
export class MoveGroupCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'group:move'
  readonly timestamp: number
  readonly description: string

  constructor(
    private projectId: string,
    private groupId: string,
    private groupName: string,
    private beforePosition: { x: number; y: number },
    private afterPosition: { x: number; y: number },
    private blockGroupsApi: BlockGroupsApi,
    private blockGroups: Ref<BlockGroup[]>
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = `Move group: ${groupName}`
  }

  async execute(): Promise<void> {
    await this.blockGroupsApi.update(this.projectId, this.groupId, {
      position: this.afterPosition,
    })

    // Update local state
    const group = this.blockGroups.value.find(g => g.id === this.groupId)
    if (group) {
      group.position_x = this.afterPosition.x
      group.position_y = this.afterPosition.y
    }
  }

  async undo(): Promise<void> {
    await this.blockGroupsApi.update(this.projectId, this.groupId, {
      position: this.beforePosition,
    })

    // Update local state
    const group = this.blockGroups.value.find(g => g.id === this.groupId)
    if (group) {
      group.position_x = this.beforePosition.x
      group.position_y = this.beforePosition.y
    }
  }
}

/**
 * Command to delete a block group
 */
export class DeleteGroupCommand implements Command {
  readonly id: string
  readonly type: CommandType = 'group:delete'
  readonly timestamp: number
  readonly description: string

  private recreatedGroupId: string | null = null

  constructor(
    private projectId: string,
    private deletedGroup: BlockGroup,
    private blockGroupsApi: BlockGroupsApi,
    private blockGroups: Ref<BlockGroup[]>,
    private project: Ref<Project | null>
  ) {
    this.id = generateCommandId()
    this.timestamp = Date.now()
    this.description = `Delete group: ${deletedGroup.name}`
  }

  async execute(): Promise<void> {
    // Clear block_group_id from steps that were in this group
    if (this.project.value?.steps) {
      for (const step of this.project.value.steps) {
        if (step.block_group_id === this.deletedGroup.id) {
          step.block_group_id = undefined
          step.group_role = undefined
        }
      }
    }

    await this.blockGroupsApi.remove(this.projectId, this.deletedGroup.id)

    // Update local state
    this.blockGroups.value = this.blockGroups.value.filter(g => g.id !== this.deletedGroup.id)
  }

  async undo(): Promise<void> {
    // Recreate the group
    const response = await this.blockGroupsApi.create(this.projectId, {
      name: this.deletedGroup.name,
      type: this.deletedGroup.type,
      config: this.deletedGroup.config,
      position: { x: this.deletedGroup.position_x, y: this.deletedGroup.position_y },
      size: { width: this.deletedGroup.width, height: this.deletedGroup.height },
    })

    if (response?.data) {
      this.recreatedGroupId = response.data.id
      this.blockGroups.value = [...this.blockGroups.value, response.data]
    }

    // Note: Steps that were in the group won't be automatically re-added
    // This is a known limitation - the user would need to manually re-add steps
  }

  /** Get the recreated group ID (available after undo, will be different from original) */
  getRecreatedGroupId(): string | null {
    return this.recreatedGroupId
  }
}
