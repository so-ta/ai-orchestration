<script setup lang="ts">
import type { StepType, BlockDefinition, BlockCategory, BlockSubcategory, BlockGroupType } from '~/types/api'
import {
  categoryConfig,
  subcategoryConfig,
} from '~/composables/useBlocks'
import { useBlockSearchWithCategory } from '~/composables/useBlockSearch'
import SubcategorySection from './palette/SubcategorySection.vue'
import GroupItem, { type GroupItemData } from './palette/GroupItem.vue'

const { t } = useI18n()
const blocksApi = useBlocks()

defineProps<{
  readonly?: boolean
}>()

const emit = defineEmits<{
  'drag-start': [type: StepType]
  'drag-end': []
}>()

// Block Group definitions for control flow constructs
const blockGroupTypes: GroupItemData[] = [
  { type: 'parallel', name: 'Parallel', description: 'Execute steps in parallel', icon: '⫲', color: '#8b5cf6' },
  { type: 'try_catch', name: 'Try-Catch', description: 'Error handling block', icon: '⚡', color: '#ef4444' },
  { type: 'foreach', name: 'ForEach', description: 'Loop over items', icon: '∀', color: '#22c55e' },
  { type: 'while', name: 'While', description: 'Repeat while condition', icon: '↻', color: '#14b8a6' },
  { type: 'agent', name: 'Agent', description: 'AI Agent with tool calling', icon: '🤖', color: '#10b981' },
]

// Fetch blocks from API
const blocks = ref<BlockDefinition[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Block search composable
const {
  searchQuery,
  isSearchActive,
  clearSearch,
  activeCategory,
  blocksBySubcategory,
  activeSubcategories: subcategoriesForActiveCategory,
} = useBlockSearchWithCategory(blocks)

// Expanded subcategories
const expandedSubcategories = ref<Set<string>>(new Set())

onMounted(async () => {
  try {
    const response = await blocksApi.list({ enabled: true })
    blocks.value = response.blocks
    // Expand all subcategories by default (including block groups and trigger)
    expandedSubcategories.value.add('__groups__')
    expandedSubcategories.value.add('trigger')
    for (const block of response.blocks) {
      if (block.subcategory) {
        expandedSubcategories.value.add(block.subcategory)
      }
    }
  } catch (e) {
    console.error('Failed to load blocks:', e)
    error.value = 'Failed to load blocks'
  } finally {
    loading.value = false
  }
})

// Categories for tabs (order: フロー / AI / アプリ / カスタム)
const categories: BlockCategory[] = ['flow', 'ai', 'apps', 'custom']

// Get filtered block group types
const filteredBlockGroupTypes = computed(() => {
  if (!isSearchActive.value) return blockGroupTypes
  const query = searchQuery.value.toLowerCase().trim()
  return blockGroupTypes.filter(g =>
    g.name.toLowerCase().includes(query) ||
    g.description.toLowerCase().includes(query)
  )
})

// Toggle subcategory expansion
function toggleSubcategory(subcategory: string) {
  if (expandedSubcategories.value.has(subcategory)) {
    expandedSubcategories.value.delete(subcategory)
  } else {
    expandedSubcategories.value.add(subcategory)
  }
}

// Get category info
function getCategoryInfo(category: BlockCategory) {
  const config = categoryConfig[category]
  return {
    name: t(config?.nameKey || `editor.categories.${category}`),
    icon: config?.icon || 'folder',
    color: config?.color || '#6b7280',
  }
}

// Get subcategory label
function getSubcategoryLabel(subcategory: BlockSubcategory): string {
  return t(subcategoryConfig[subcategory]?.nameKey || `editor.subcategories.${subcategory}`)
}

// Drag state
const draggingType = ref<StepType | null>(null)
const draggingGroupType = ref<BlockGroupType | null>(null)

function handleDragStart(_event: DragEvent, block: BlockDefinition) {
  draggingType.value = block.slug as StepType
  emit('drag-start', block.slug as StepType)
}

function handleDragEnd() {
  draggingType.value = null
  emit('drag-end')
}

function handleGroupDragStart(_event: DragEvent, groupType: GroupItemData) {
  draggingGroupType.value = groupType.type
}

function handleGroupDragEnd() {
  draggingGroupType.value = null
}

// Reference to search input
const searchInput = ref<HTMLInputElement | null>(null)

// Focus search on Cmd/Ctrl+K
function handleKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
    event.preventDefault()
    searchInput.value?.focus()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div class="step-palette">
    <!-- Category Tabs -->
    <div class="category-tabs">
      <button
        v-for="category in categories"
        :key="category"
        :class="['category-tab', { active: activeCategory === category && !isSearchActive }]"
        :style="activeCategory === category && !isSearchActive ? { borderColor: getCategoryInfo(category).color } : {}"
        @click="activeCategory = category; clearSearch()"
      >
        <span
          class="tab-indicator"
          :style="{ backgroundColor: getCategoryInfo(category).color }"
        />
        <span class="tab-label">{{ getCategoryInfo(category).name }}</span>
      </button>
    </div>

    <!-- Inline Search Bar -->
    <div class="palette-search">
      <div class="search-wrapper">
        <svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.35-4.35" />
        </svg>
        <input
          ref="searchInput"
          v-model="searchQuery"
          type="text"
          class="search-input"
          :placeholder="$t('editor.searchBlocks')"
        >
        <button
          v-if="searchQuery"
          class="search-clear"
          @click="clearSearch"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6 6 18M6 6l12 12" />
          </svg>
        </button>
        <kbd v-else class="search-shortcut">⌘K</kbd>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="palette-loading">
      <span class="loading-spinner" />
      <span>{{ $t('common.loading') }}</span>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="palette-error">
      <span>{{ error }}</span>
    </div>

    <!-- Blocks list -->
    <div v-else class="palette-content">
      <!-- Start/Trigger section (always first when in Flow tab or searching) -->
      <SubcategorySection
        v-if="(activeCategory === 'flow' || isSearchActive) && blocksBySubcategory['trigger']?.length"
        :name="getSubcategoryLabel('trigger')"
        :blocks="blocksBySubcategory['trigger']"
        :expanded="expandedSubcategories.has('trigger')"
        :readonly="readonly"
        :dragging-type="draggingType"
        @toggle="toggleSubcategory('trigger')"
        @drag-start="handleDragStart"
        @drag-end="handleDragEnd"
      />

      <!-- Control Flow Groups (show in Flow tab or when searching) -->
      <div v-if="(activeCategory === 'flow' || isSearchActive) && filteredBlockGroupTypes.length > 0" class="subcategory-section">
        <button
          class="subcategory-header"
          @click="toggleSubcategory('__groups__')"
        >
          <svg
            :class="['chevron', { expanded: expandedSubcategories.has('__groups__') }]"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="m9 18 6-6-6-6" />
          </svg>
          <span class="subcategory-name">Control Flow Groups</span>
          <span class="subcategory-count">{{ filteredBlockGroupTypes.length }}</span>
        </button>

        <div v-show="expandedSubcategories.has('__groups__')" class="subcategory-items">
          <GroupItem
            v-for="groupType in filteredBlockGroupTypes"
            :key="groupType.type"
            :group="groupType"
            :dragging="draggingGroupType === groupType.type"
            :disabled="readonly"
            @drag-start="handleGroupDragStart($event, groupType)"
            @drag-end="handleGroupDragEnd"
          />
        </div>
      </div>

      <!-- Subcategory Sections (excluding 'trigger' which is shown first in Flow tab) -->
      <template v-for="subcategory in subcategoriesForActiveCategory" :key="subcategory">
        <SubcategorySection
          v-if="blocksBySubcategory[subcategory]?.length && !(subcategory === 'trigger' && (activeCategory === 'flow' || isSearchActive))"
          :name="getSubcategoryLabel(subcategory)"
          :blocks="blocksBySubcategory[subcategory]"
          :expanded="expandedSubcategories.has(subcategory)"
          :readonly="readonly"
          :dragging-type="draggingType"
          @toggle="toggleSubcategory(subcategory)"
          @drag-start="handleDragStart"
          @drag-end="handleDragEnd"
        />
      </template>

      <!-- Uncategorized blocks (if any) -->
      <SubcategorySection
        v-if="blocksBySubcategory['other']?.length"
        name="Other"
        :blocks="blocksBySubcategory['other']"
        :expanded="expandedSubcategories.has('other')"
        :readonly="readonly"
        :dragging-type="draggingType"
        @toggle="toggleSubcategory('other')"
        @drag-start="handleDragStart"
        @drag-end="handleDragEnd"
      />

      <!-- Empty state -->
      <div v-if="Object.keys(blocksBySubcategory).length === 0" class="empty-state">
        <p>No blocks in this category</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.step-palette {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* Category Tabs */
.category-tabs {
  display: flex;
  flex-shrink: 0;
  border-bottom: 1px solid var(--color-border);
}

.category-tab {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 10px 4px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  font-size: 0.6875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  transition: all 0.15s;
  min-width: 0;
}

.category-tab:hover {
  color: var(--color-text);
}

.category-tab.active {
  color: var(--color-text);
}

.tab-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.tab-label {
  text-transform: uppercase;
  letter-spacing: 0.02em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Inline Search */
.palette-search {
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.search-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--color-background-secondary, #f9fafb);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  transition: all 0.15s;
}

.search-wrapper:focus-within {
  border-color: var(--color-primary);
  background: white;
}

.search-icon {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 0.8125rem;
  color: var(--color-text);
  outline: none;
  min-width: 0;
}

.search-input::placeholder {
  color: var(--color-text-secondary);
}

.search-clear {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  background: var(--color-border);
  border: none;
  border-radius: 50%;
  cursor: pointer;
  color: var(--color-text-secondary);
  flex-shrink: 0;
  transition: all 0.15s;
}

.search-clear:hover {
  background: var(--color-text-secondary);
  color: white;
}

.search-shortcut {
  font-size: 0.625rem;
  padding: 2px 5px;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 3px;
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

/* Loading & Error */
.palette-loading,
.palette-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2rem 1rem;
  gap: 0.5rem;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.palette-error {
  color: var(--color-error, #ef4444);
}

/* Content */
.palette-content {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

/* Subcategory Section (for groups only - blocks use SubcategorySection component) */
.subcategory-section {
  margin-bottom: 4px;
}

.subcategory-header {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 8px 8px;
  background: none;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  text-align: left;
  transition: background 0.1s;
}

.subcategory-header:hover {
  background: var(--color-background-secondary);
}

.chevron {
  color: var(--color-text-secondary);
  transition: transform 0.15s;
  flex-shrink: 0;
}

.chevron.expanded {
  transform: rotate(90deg);
}

.subcategory-name {
  flex: 1;
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.subcategory-count {
  font-size: 0.625rem;
  color: var(--color-text-tertiary);
  background: var(--color-background-secondary);
  padding: 1px 6px;
  border-radius: 10px;
}

.subcategory-items {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 4px 0 8px 18px;
}

/* Empty state */
.empty-state {
  padding: 2rem;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 0.875rem;
}

/* Scrollbar */
.palette-content::-webkit-scrollbar {
  width: 6px;
}

.palette-content::-webkit-scrollbar-track {
  background: transparent;
}

.palette-content::-webkit-scrollbar-thumb {
  background: var(--color-border);
  border-radius: 3px;
}

.palette-content::-webkit-scrollbar-thumb:hover {
  background: var(--color-text-secondary);
}
</style>
