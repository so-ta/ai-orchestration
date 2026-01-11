// Block Groups API composable
import type {
  BlockGroup,
  Step,
  ApiResponse,
  CreateBlockGroupRequest,
  UpdateBlockGroupRequest,
  AddStepToGroupRequest,
  GroupRole
} from '~/types/api'

export function useBlockGroups() {
  const api = useApi()

  // List block groups for a workflow
  async function list(workflowId: string) {
    return api.get<ApiResponse<BlockGroup[]>>(`/workflows/${workflowId}/block-groups`)
  }

  // Get block group by ID
  async function get(workflowId: string, groupId: string) {
    return api.get<ApiResponse<BlockGroup>>(`/workflows/${workflowId}/block-groups/${groupId}`)
  }

  // Create block group
  async function create(workflowId: string, data: CreateBlockGroupRequest) {
    return api.post<ApiResponse<BlockGroup>>(`/workflows/${workflowId}/block-groups`, data)
  }

  // Update block group
  async function update(workflowId: string, groupId: string, data: UpdateBlockGroupRequest) {
    return api.put<ApiResponse<BlockGroup>>(`/workflows/${workflowId}/block-groups/${groupId}`, data)
  }

  // Delete block group
  async function remove(workflowId: string, groupId: string) {
    return api.delete(`/workflows/${workflowId}/block-groups/${groupId}`)
  }

  // Get steps in a block group
  async function getSteps(workflowId: string, groupId: string) {
    return api.get<ApiResponse<Step[]>>(`/workflows/${workflowId}/block-groups/${groupId}/steps`)
  }

  // Add step to block group
  async function addStep(workflowId: string, groupId: string, data: AddStepToGroupRequest) {
    return api.post<ApiResponse<Step>>(`/workflows/${workflowId}/block-groups/${groupId}/steps`, data)
  }

  // Remove step from block group
  async function removeStep(workflowId: string, groupId: string, stepId: string) {
    return api.delete<ApiResponse<Step>>(`/workflows/${workflowId}/block-groups/${groupId}/steps/${stepId}`)
  }

  // Helper: Create a parallel block group
  async function createParallel(
    workflowId: string,
    name: string,
    position: { x: number; y: number },
    options?: { maxConcurrent?: number; failFast?: boolean }
  ) {
    return create(workflowId, {
      name,
      type: 'parallel',
      config: {
        max_concurrent: options?.maxConcurrent,
        fail_fast: options?.failFast
      },
      position,
      size: { width: 400, height: 300 }
    })
  }

  // Helper: Create a try-catch block group
  async function createTryCatch(
    workflowId: string,
    name: string,
    position: { x: number; y: number },
    options?: { errorTypes?: string[]; retryCount?: number }
  ) {
    return create(workflowId, {
      name,
      type: 'try_catch',
      config: {
        error_types: options?.errorTypes || ['*'],
        retry_count: options?.retryCount || 0
      },
      position,
      size: { width: 400, height: 400 }
    })
  }

  // Helper: Create an if-else block group
  async function createIfElse(
    workflowId: string,
    name: string,
    position: { x: number; y: number },
    condition: string
  ) {
    return create(workflowId, {
      name,
      type: 'if_else',
      config: { condition },
      position,
      size: { width: 400, height: 300 }
    })
  }

  // Helper: Create a foreach block group
  async function createForeach(
    workflowId: string,
    name: string,
    position: { x: number; y: number },
    inputPath: string,
    options?: { parallel?: boolean; maxWorkers?: number }
  ) {
    return create(workflowId, {
      name,
      type: 'foreach',
      config: {
        input_path: inputPath,
        parallel: options?.parallel,
        max_workers: options?.maxWorkers
      },
      position,
      size: { width: 400, height: 300 }
    })
  }

  // Helper: Create a while block group
  async function createWhile(
    workflowId: string,
    name: string,
    position: { x: number; y: number },
    condition: string,
    options?: { maxIterations?: number; doWhile?: boolean }
  ) {
    return create(workflowId, {
      name,
      type: 'while',
      config: {
        condition,
        max_iterations: options?.maxIterations || 100,
        do_while: options?.doWhile
      },
      position,
      size: { width: 400, height: 300 }
    })
  }

  return {
    // CRUD operations
    list,
    get,
    create,
    update,
    remove,

    // Step management
    getSteps,
    addStep,
    removeStep,

    // Convenience helpers
    createParallel,
    createTryCatch,
    createIfElse,
    createForeach,
    createWhile
  }
}
