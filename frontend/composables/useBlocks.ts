// Block Registry API composable
import type { BlockDefinition, BlockListResponse, BlockCategory, ApiResponse } from '~/types/api'

// Response wrapper for block list (API wraps responses in data field)
interface BlockListApiResponse {
  data: BlockListResponse
}

export function useBlocks() {
  const api = useApi()

  // List blocks with optional filtering
  async function list(params?: {
    category?: BlockCategory
    enabled?: boolean
  }): Promise<BlockListResponse> {
    const query = new URLSearchParams()
    if (params?.category) query.set('category', params.category)
    if (params?.enabled !== undefined) query.set('enabled', params.enabled.toString())

    const queryString = query.toString()
    const endpoint = `/blocks${queryString ? `?${queryString}` : ''}`

    const response = await api.get<BlockListApiResponse>(endpoint)
    return response.data
  }

  // Get block by slug
  async function get(slug: string) {
    return api.get<ApiResponse<BlockDefinition>>(`/blocks/${slug}`)
  }

  // Create custom block
  async function create(data: {
    slug: string
    name: string
    description?: string
    category: BlockCategory
    icon?: string
    config_schema?: object
    input_schema?: object
    output_schema?: object
    executor_type: string
    executor_config?: object
  }) {
    return api.post<ApiResponse<BlockDefinition>>('/blocks', data)
  }

  // Update custom block
  async function update(slug: string, data: {
    name?: string
    description?: string
    icon?: string
    config_schema?: object
    input_schema?: object
    output_schema?: object
    executor_config?: object
    enabled?: boolean
  }) {
    return api.put<ApiResponse<BlockDefinition>>(`/blocks/${slug}`, data)
  }

  // Delete custom block
  async function remove(slug: string) {
    return api.delete(`/blocks/${slug}`)
  }

  return {
    list,
    get,
    create,
    update,
    remove,
  }
}

// Category display configuration
export const categoryConfig: Record<BlockCategory, {
  nameKey: string
  icon: string
  order: number
}> = {
  ai: { nameKey: 'editor.categories.ai', icon: 'sparkles', order: 1 },
  logic: { nameKey: 'editor.categories.logic', icon: 'git-branch', order: 2 },
  data: { nameKey: 'editor.categories.data', icon: 'database', order: 3 },
  integration: { nameKey: 'editor.categories.integration', icon: 'plug', order: 4 },
  control: { nameKey: 'editor.categories.control', icon: 'clock', order: 5 },
  utility: { nameKey: 'editor.categories.utility', icon: 'info', order: 6 },
}

// Block color mapping by slug (for visual consistency)
export const blockColors: Record<string, string> = {
  // AI blocks
  llm: '#3b82f6',
  router: '#a855f7',
  // Logic blocks
  condition: '#f59e0b',
  switch: '#eab308',
  loop: '#14b8a6',
  map: '#8b5cf6',
  join: '#6366f1',
  // Data blocks
  filter: '#06b6d4',
  split: '#0ea5e9',
  aggregate: '#0284c7',
  transform: '#0891b2',
  // Integration blocks
  tool: '#22c55e',
  http: '#10b981',
  function: '#f97316',
  subflow: '#ec4899',
  // Control blocks
  start: '#10b981',
  wait: '#64748b',
  human_in_loop: '#ef4444',
  error: '#dc2626',
  // Utility blocks
  note: '#9ca3af',
}

// Default color for unknown blocks
export const defaultBlockColor = '#6b7280'

// Get color for a block
export function getBlockColor(slug: string): string {
  return blockColors[slug] || defaultBlockColor
}
