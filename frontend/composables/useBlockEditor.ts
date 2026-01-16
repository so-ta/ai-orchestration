/**
 * useBlockEditor - ブロック編集ロジック用コンポーザブル
 *
 * ブロックの作成・編集・削除・バージョン管理などのロジックを提供。
 */
import type { BlockDefinition, BlockCategory } from '~/types/api'

// Block form data interface
export interface BlockFormData {
  slug: string
  name: string
  description: string
  category: BlockCategory
  icon: string
  code: string
  config_schema: string
  ui_config: string
  change_summary: string
  parent_block_id?: string
  config_defaults?: string
  pre_process?: string
  post_process?: string
}

// Block editor state
interface BlockEditorState {
  blocks: Ref<BlockDefinition[]>
  loading: Ref<boolean>
  error: Ref<string | null>
  selectedBlock: Ref<BlockDefinition | null>
  showCreateWizard: Ref<boolean>
  showEditModal: Ref<boolean>
  showDeleteModal: Ref<boolean>
  showVersionModal: Ref<boolean>
}

export function useBlockEditor() {
  const blocksApi = useBlocks()
  const adminBlocks = useAdminBlocks()
  const { t } = useI18n()

  // State
  const blocks = ref<BlockDefinition[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const selectedBlock = ref<BlockDefinition | null>(null)
  const showCreateWizard = ref(false)
  const showEditModal = ref(false)
  const showDeleteModal = ref(false)
  const showVersionModal = ref(false)

  // Filters
  const searchQuery = ref('')
  const categoryFilter = ref<BlockCategory | ''>('')
  const showSystemOnly = ref(false)
  const showCustomOnly = ref(false)

  // Computed: filtered blocks
  const filteredBlocks = computed(() => {
    let result = blocks.value

    // Category filter
    if (categoryFilter.value) {
      result = result.filter(b => b.category === categoryFilter.value)
    }

    // System/Custom filter
    if (showSystemOnly.value) {
      result = result.filter(b => b.is_system)
    } else if (showCustomOnly.value) {
      result = result.filter(b => !b.is_system)
    }

    // Search filter
    if (searchQuery.value.trim()) {
      const query = searchQuery.value.toLowerCase()
      result = result.filter(b =>
        b.name.toLowerCase().includes(query) ||
        b.slug.toLowerCase().includes(query) ||
        b.description?.toLowerCase().includes(query)
      )
    }

    // Sort by category, then by name
    return [...result].sort((a, b) => {
      if (a.category !== b.category) {
        return a.category.localeCompare(b.category)
      }
      return a.name.localeCompare(b.name)
    })
  })

  // Computed: system blocks
  const systemBlocks = computed(() => blocks.value.filter(b => b.is_system))

  // Computed: custom blocks
  const customBlocks = computed(() => blocks.value.filter(b => !b.is_system))

  // Fetch blocks
  async function fetchBlocks() {
    try {
      loading.value = true
      error.value = null

      // Try admin API first for full block data
      try {
        const response = await adminBlocks.listSystemBlocks()
        blocks.value = response.blocks || []
      } catch {
        // Fall back to regular API
        const response = await blocksApi.list()
        blocks.value = response.blocks || []
      }
    } catch (err) {
      error.value = t('errors.loadFailed')
      console.error('Failed to fetch blocks:', err)
    } finally {
      loading.value = false
    }
  }

  // Create block
  async function createBlock(formData: BlockFormData): Promise<BlockDefinition | null> {
    try {
      loading.value = true
      error.value = null

      // Parse JSON fields
      let configSchema, uiConfig
      try {
        configSchema = JSON.parse(formData.config_schema || '{}')
        uiConfig = JSON.parse(formData.ui_config || '{}')
      } catch {
        error.value = t('blockEditor.errors.invalidJson')
        return null
      }

      const response = await blocksApi.create({
        slug: formData.slug,
        name: formData.name,
        description: formData.description || undefined,
        category: formData.category,
        icon: formData.icon || undefined,
        code: formData.code || undefined,
        config_schema: configSchema,
        ui_config: uiConfig,
      })

      // Refresh list
      await fetchBlocks()

      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : t('errors.generic')
      console.error('Failed to create block:', err)
      return null
    } finally {
      loading.value = false
    }
  }

  // Update block
  async function updateBlock(id: string, formData: BlockFormData): Promise<BlockDefinition | null> {
    try {
      loading.value = true
      error.value = null

      // Parse JSON fields
      let configSchema, uiConfig
      try {
        configSchema = JSON.parse(formData.config_schema || '{}')
        uiConfig = JSON.parse(formData.ui_config || '{}')
      } catch {
        error.value = t('blockEditor.errors.invalidJson')
        return null
      }

      const response = await adminBlocks.updateSystemBlock(id, {
        name: formData.name,
        description: formData.description || undefined,
        code: formData.code,
        config_schema: configSchema,
        ui_config: uiConfig,
        change_summary: formData.change_summary,
      })

      // Refresh list
      await fetchBlocks()

      return response.data
    } catch (err) {
      error.value = err instanceof Error ? err.message : t('errors.generic')
      console.error('Failed to update block:', err)
      return null
    } finally {
      loading.value = false
    }
  }

  // Delete block
  async function deleteBlock(slug: string): Promise<boolean> {
    try {
      loading.value = true
      error.value = null

      await blocksApi.remove(slug)

      // Refresh list
      await fetchBlocks()

      return true
    } catch (err) {
      error.value = err instanceof Error ? err.message : t('errors.generic')
      console.error('Failed to delete block:', err)
      return false
    } finally {
      loading.value = false
    }
  }

  // Duplicate block
  async function duplicateBlock(block: BlockDefinition): Promise<BlockDefinition | null> {
    const formData: BlockFormData = {
      slug: block.slug + '_copy',
      name: block.name + ' (Copy)',
      description: block.description || '',
      category: block.category,
      icon: block.icon || '',
      code: block.code || '',
      config_schema: JSON.stringify(block.config_schema || {}, null, 2),
      ui_config: JSON.stringify(block.ui_config || {}, null, 2),
      change_summary: 'Duplicated from ' + block.name,
    }

    return createBlock(formData)
  }

  // Open modals
  function openCreateWizard() {
    showCreateWizard.value = true
  }

  function openEditModal(block: BlockDefinition) {
    selectedBlock.value = block
    showEditModal.value = true
  }

  function openDeleteModal(block: BlockDefinition) {
    selectedBlock.value = block
    showDeleteModal.value = true
  }

  function openVersionModal(block: BlockDefinition) {
    selectedBlock.value = block
    showVersionModal.value = true
  }

  // Close modals
  function closeModals() {
    showCreateWizard.value = false
    showEditModal.value = false
    showDeleteModal.value = false
    showVersionModal.value = false
    selectedBlock.value = null
  }

  // Convert block to form data
  function blockToFormData(block: BlockDefinition): BlockFormData {
    return {
      slug: block.slug,
      name: block.name,
      description: block.description || '',
      category: block.category,
      icon: block.icon || '',
      code: block.code || '',
      config_schema: JSON.stringify(block.config_schema || {}, null, 2),
      ui_config: JSON.stringify(block.ui_config || {}, null, 2),
      change_summary: '',
      parent_block_id: block.parent_block_id,
      config_defaults: JSON.stringify(block.config_defaults || {}, null, 2),
      pre_process: block.pre_process || '',
      post_process: block.post_process || '',
    }
  }

  return {
    // State
    blocks,
    loading,
    error,
    selectedBlock,
    showCreateWizard,
    showEditModal,
    showDeleteModal,
    showVersionModal,

    // Filters
    searchQuery,
    categoryFilter,
    showSystemOnly,
    showCustomOnly,

    // Computed
    filteredBlocks,
    systemBlocks,
    customBlocks,

    // Actions
    fetchBlocks,
    createBlock,
    updateBlock,
    deleteBlock,
    duplicateBlock,

    // Modal controls
    openCreateWizard,
    openEditModal,
    openDeleteModal,
    openVersionModal,
    closeModals,

    // Utilities
    blockToFormData,
  }
}

