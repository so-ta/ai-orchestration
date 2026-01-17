// Block Registry API composable
import type { BlockDefinition, BlockListResponse, BlockCategory, BlockSubcategory, ApiResponse } from '~/types/api'

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
    code?: string
    ui_config?: object
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
    code?: string
    ui_config?: object
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

// Category display configuration (simplified to 4 categories)
export const categoryConfig: Record<BlockCategory, {
  nameKey: string
  icon: string
  order: number
  color: string
}> = {
  ai: { nameKey: 'editor.categories.ai', icon: 'sparkles', order: 1, color: '#8B5CF6' },
  flow: { nameKey: 'editor.categories.flow', icon: 'git-branch', order: 2, color: '#3B82F6' },
  apps: { nameKey: 'editor.categories.apps', icon: 'plug', order: 3, color: '#10B981' },
  custom: { nameKey: 'editor.categories.custom', icon: 'code', order: 4, color: '#F59E0B' },
}

// Subcategory display configuration
export const subcategoryConfig: Record<BlockSubcategory, {
  nameKey: string
  icon: string
  order: number
}> = {
  // AI subcategories
  chat: { nameKey: 'editor.subcategories.chat', icon: 'message-square', order: 1 },
  rag: { nameKey: 'editor.subcategories.rag', icon: 'book-open', order: 2 },
  routing: { nameKey: 'editor.subcategories.routing', icon: 'route', order: 3 },
  // Flow subcategories
  branching: { nameKey: 'editor.subcategories.branching', icon: 'git-branch', order: 1 },
  data: { nameKey: 'editor.subcategories.data', icon: 'database', order: 2 },
  control: { nameKey: 'editor.subcategories.control', icon: 'settings', order: 3 },
  utility: { nameKey: 'editor.subcategories.utility', icon: 'tool', order: 4 },
  // Apps subcategories
  slack: { nameKey: 'editor.subcategories.slack', icon: 'message-circle', order: 1 },
  discord: { nameKey: 'editor.subcategories.discord', icon: 'message-circle', order: 2 },
  notion: { nameKey: 'editor.subcategories.notion', icon: 'file-text', order: 3 },
  github: { nameKey: 'editor.subcategories.github', icon: 'github', order: 4 },
  google: { nameKey: 'editor.subcategories.google', icon: 'table', order: 5 },
  linear: { nameKey: 'editor.subcategories.linear', icon: 'check-square', order: 6 },
  email: { nameKey: 'editor.subcategories.email', icon: 'mail', order: 7 },
  web: { nameKey: 'editor.subcategories.web', icon: 'globe', order: 8 },
}

// Mapping of subcategories to their parent categories
export const subcategoryToCategory: Record<BlockSubcategory, BlockCategory> = {
  chat: 'ai',
  rag: 'ai',
  routing: 'ai',
  branching: 'flow',
  data: 'flow',
  control: 'flow',
  utility: 'flow',
  slack: 'apps',
  discord: 'apps',
  notion: 'apps',
  github: 'apps',
  google: 'apps',
  linear: 'apps',
  email: 'apps',
  web: 'apps',
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
  schedule_trigger: '#22c55e',
  webhook_trigger: '#3b82f6',
  wait: '#64748b',
  human_in_loop: '#ef4444',
  error: '#dc2626',
  // Utility blocks
  note: '#9ca3af',
  log: '#10b981',
}

// Default color for unknown blocks
export const defaultBlockColor = '#6b7280'

// Get color for a block
export function getBlockColor(slug: string): string {
  return blockColors[slug] || defaultBlockColor
}

// ============================================================================
// Block Grouping Utilities
// ============================================================================

// Group blocks by category
export function groupBlocksByCategory(blocks: BlockDefinition[]): Record<BlockCategory, BlockDefinition[]> {
  const result: Record<BlockCategory, BlockDefinition[]> = {
    ai: [],
    flow: [],
    apps: [],
    custom: [],
  }

  for (const block of blocks) {
    if (result[block.category]) {
      result[block.category].push(block)
    }
  }

  // Sort each category by subcategory order, then name
  for (const category of Object.keys(result) as BlockCategory[]) {
    result[category].sort((a, b) => {
      const aSubOrder = a.subcategory ? subcategoryConfig[a.subcategory]?.order || 999 : 999
      const bSubOrder = b.subcategory ? subcategoryConfig[b.subcategory]?.order || 999 : 999
      if (aSubOrder !== bSubOrder) return aSubOrder - bSubOrder
      return a.name.localeCompare(b.name)
    })
  }

  return result
}

// Group blocks by subcategory within a category (or all blocks if category is null)
export function groupBlocksBySubcategory(
  blocks: BlockDefinition[],
  category: BlockCategory | null
): Record<string, BlockDefinition[]> {
  const categoryBlocks = category ? blocks.filter(b => b.category === category) : blocks
  const result: Record<string, BlockDefinition[]> = {}

  for (const block of categoryBlocks) {
    const key = block.subcategory || 'other'
    if (!result[key]) {
      result[key] = []
    }
    result[key].push(block)
  }

  // Sort blocks within each subcategory by name
  for (const key of Object.keys(result)) {
    result[key].sort((a, b) => a.name.localeCompare(b.name))
  }

  return result
}

// Get sorted subcategories for a category
export function getSubcategoriesForCategory(category: BlockCategory): BlockSubcategory[] {
  const subcategories = Object.entries(subcategoryToCategory)
    .filter(([_, cat]) => cat === category)
    .map(([sub]) => sub as BlockSubcategory)
    .sort((a, b) => (subcategoryConfig[a]?.order || 999) - (subcategoryConfig[b]?.order || 999))

  return subcategories
}

// Search blocks by query (name, description, slug)
export function searchBlocks(blocks: BlockDefinition[], query: string): BlockDefinition[] {
  const lowerQuery = query.toLowerCase().trim()
  if (!lowerQuery) return blocks

  return blocks.filter(block => {
    return (
      block.name.toLowerCase().includes(lowerQuery) ||
      block.slug.toLowerCase().includes(lowerQuery) ||
      block.description?.toLowerCase().includes(lowerQuery)
    )
  })
}

// ============================================================================
// Admin API for System Block Management
// ============================================================================

// Block Version type
export interface BlockVersion {
  id: string
  block_id: string
  version: number
  code: string
  config_schema: object
  input_schema?: object
  output_schema?: object
  ui_config: object
  change_summary?: string
  changed_by?: string
  created_at: string
}

// System block list response
interface SystemBlockListResponse {
  blocks: BlockDefinition[]
}

// Block versions list response
interface BlockVersionsResponse {
  versions: BlockVersion[]
}

export function useAdminBlocks() {
  const api = useApi()

  // List all system blocks
  async function listSystemBlocks(): Promise<SystemBlockListResponse> {
    const response = await api.get<{ data: SystemBlockListResponse }>('/admin/blocks')
    return response.data
  }

  // Get a specific system block by ID
  async function getSystemBlock(id: string) {
    return api.get<{ data: BlockDefinition }>(`/admin/blocks/${id}`)
  }

  // Update a system block (code, schema, etc.)
  async function updateSystemBlock(id: string, data: {
    name?: string
    description?: string
    code?: string
    config_schema?: object
    input_schema?: object
    output_schema?: object
    ui_config?: object
    change_summary?: string
  }) {
    return api.put<{ data: BlockDefinition }>(`/admin/blocks/${id}`, data)
  }

  // List versions of a block
  async function listVersions(blockId: string): Promise<BlockVersionsResponse> {
    const response = await api.get<{ data: BlockVersionsResponse }>(`/admin/blocks/${blockId}/versions`)
    return response.data
  }

  // Get a specific version
  async function getVersion(blockId: string, version: number) {
    return api.get<{ data: BlockVersion }>(`/admin/blocks/${blockId}/versions/${version}`)
  }

  // Rollback to a previous version
  async function rollback(blockId: string, version: number) {
    return api.post<{ data: BlockDefinition }>(`/admin/blocks/${blockId}/rollback`, { version })
  }

  return {
    listSystemBlocks,
    getSystemBlock,
    updateSystemBlock,
    listVersions,
    getVersion,
    rollback,
  }
}
