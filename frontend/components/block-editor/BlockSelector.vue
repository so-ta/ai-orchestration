<script setup lang="ts">
/**
 * BlockSelector - 親ブロック選択コンポーネント
 *
 * 継承可能なブロック（codeを持つブロック）を選択するためのドロップダウン。
 */
import type { BlockDefinition } from '~/types/api'

const props = defineProps<{
  modelValue?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  select: [block: BlockDefinition]
}>()

const { t } = useI18n()
const blocksApi = useBlocks()

// State
const searchQuery = ref('')
const showDropdown = ref(false)
const blocks = ref<BlockDefinition[]>([])
const loading = ref(false)
const selectedBlock = ref<BlockDefinition | null>(null)

// Fetch blocks
async function fetchBlocks() {
  try {
    loading.value = true
    const response = await blocksApi.list()
    blocks.value = response.blocks || []

    // If we have a modelValue, find the selected block
    if (props.modelValue) {
      selectedBlock.value = blocks.value.find(b => b.id === props.modelValue) || null
    }
  } catch (error) {
    console.error('Failed to fetch blocks:', error)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchBlocks()
})

// Filter only blocks that can be inherited (have code)
const inheritableBlocks = computed(() => {
  return blocks.value.filter(b => b.code && b.code.trim() !== '')
})

// Filtered blocks based on search
const filteredBlocks = computed(() => {
  const query = searchQuery.value.toLowerCase().trim()
  if (!query) return inheritableBlocks.value

  return inheritableBlocks.value.filter(block => {
    return (
      block.name.toLowerCase().includes(query) ||
      block.slug.toLowerCase().includes(query) ||
      block.description?.toLowerCase().includes(query)
    )
  })
})

// Group blocks by category
const groupedBlocks = computed(() => {
  const groups: Record<string, BlockDefinition[]> = {}

  for (const block of filteredBlocks.value) {
    if (!groups[block.category]) {
      groups[block.category] = []
    }
    groups[block.category].push(block)
  }

  return groups
})

// Select a block
function selectBlock(block: BlockDefinition) {
  selectedBlock.value = block
  emit('update:modelValue', block.id)
  emit('select', block)
  showDropdown.value = false
  searchQuery.value = ''
}

// Clear selection
function clearSelection() {
  selectedBlock.value = null
  emit('update:modelValue', '')
}

// Handle click outside
const selectorRef = ref<HTMLElement | null>(null)

function handleClickOutside(event: MouseEvent) {
  if (selectorRef.value && !selectorRef.value.contains(event.target as Node)) {
    showDropdown.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div ref="selectorRef" class="block-selector">
    <!-- Selected Block Display / Search Input -->
    <div class="selector-input" @click="showDropdown = true">
      <template v-if="selectedBlock">
        <div class="selected-block">
          <span class="block-icon">{{ selectedBlock.icon || '&#9632;' }}</span>
          <span class="block-name">{{ selectedBlock.name }}</span>
          <span class="block-slug">{{ selectedBlock.slug }}</span>
          <button class="clear-btn" @click.stop="clearSelection">&#10005;</button>
        </div>
      </template>
      <template v-else>
        <input
          v-model="searchQuery"
          type="text"
          class="search-input"
          :placeholder="t('blockEditor.selectParentBlock')"
          @focus="showDropdown = true"
        />
      </template>
    </div>

    <!-- Dropdown -->
    <div v-if="showDropdown" class="dropdown">
      <!-- Search (when block is selected) -->
      <div v-if="selectedBlock" class="dropdown-search">
        <input
          v-model="searchQuery"
          type="text"
          class="search-input"
          :placeholder="t('blockEditor.searchBlocks')"
        />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="dropdown-loading">
        {{ t('common.loading') }}
      </div>

      <!-- No inheritable blocks -->
      <div v-else-if="inheritableBlocks.length === 0" class="dropdown-empty">
        {{ t('blockEditor.noInheritableBlocks') }}
      </div>

      <!-- No results -->
      <div v-else-if="filteredBlocks.length === 0" class="dropdown-empty">
        {{ t('blockEditor.noBlocksFound') }}
      </div>

      <!-- Block List -->
      <div v-else class="dropdown-list">
        <template v-for="(categoryBlocks, category) in groupedBlocks" :key="category">
          <div class="category-header">{{ category }}</div>
          <div
            v-for="block in categoryBlocks"
            :key="block.id"
            class="block-option"
            :class="{ selected: modelValue === block.id }"
            @click="selectBlock(block)"
          >
            <div class="block-info">
              <span class="block-icon">{{ block.icon || '&#9632;' }}</span>
              <div class="block-details">
                <span class="block-name">{{ block.name }}</span>
                <span class="block-slug">{{ block.slug }}</span>
              </div>
            </div>
            <div class="block-badges">
              <span v-if="block.is_system" class="badge badge-system">
                {{ t('blockEditor.systemBlock') }}
              </span>
              <span class="badge badge-inheritable">
                {{ t('blockEditor.inheritable') }}
              </span>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.block-selector {
  position: relative;
}

.selector-input {
  border: 1px solid var(--color-border);
  border-radius: 0.375rem;
  background: var(--color-background);
  cursor: pointer;
  min-height: 2.5rem;
}

.search-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: none;
  background: transparent;
  font-size: 0.875rem;
  color: var(--color-text);
}

.search-input:focus {
  outline: none;
}

.selected-block {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
}

.block-icon {
  font-size: 1rem;
}

.block-name {
  font-weight: 500;
}

.block-slug {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  font-family: 'Monaco', 'Menlo', monospace;
}

.clear-btn {
  margin-left: auto;
  background: none;
  border: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  font-size: 0.75rem;
}

.clear-btn:hover {
  color: var(--color-text);
}

.dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  margin-top: 0.25rem;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 100;
  max-height: 300px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.dropdown-search {
  padding: 0.5rem;
  border-bottom: 1px solid var(--color-border);
}

.dropdown-search .search-input {
  border: 1px solid var(--color-border);
  border-radius: 0.25rem;
  padding: 0.375rem 0.5rem;
}

.dropdown-loading,
.dropdown-empty {
  padding: 1.5rem;
  text-align: center;
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.dropdown-list {
  overflow-y: auto;
  flex: 1;
}

.category-header {
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  background: var(--color-background);
  border-bottom: 1px solid var(--color-border);
  position: sticky;
  top: 0;
}

.block-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem;
  cursor: pointer;
  transition: background 0.1s;
}

.block-option:hover {
  background: var(--color-bg-hover, rgba(0, 0, 0, 0.05));
}

.block-option.selected {
  background: rgba(99, 102, 241, 0.1);
}

.block-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.block-details {
  display: flex;
  flex-direction: column;
}

.block-details .block-name {
  font-size: 0.875rem;
}

.block-details .block-slug {
  font-size: 0.6875rem;
}

.block-badges {
  display: flex;
  gap: 0.375rem;
}

.badge {
  padding: 0.125rem 0.375rem;
  border-radius: 0.25rem;
  font-size: 0.6875rem;
  font-weight: 500;
}

.badge-system {
  background: rgba(99, 102, 241, 0.1);
  color: var(--color-primary);
}

.badge-inheritable {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
}
</style>
