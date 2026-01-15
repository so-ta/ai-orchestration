<script setup lang="ts">
import type { StepType, BlockDefinition, BlockCategory, BlockSubcategory, BlockGroupType } from '~/types/api'
import {
  categoryConfig,
  subcategoryConfig,
  getBlockColor,
  groupBlocksBySubcategory,
  getSubcategoriesForCategory,
} from '~/composables/useBlocks'

const { t } = useI18n()
const blocksApi = useBlocks()

defineProps<{
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'drag-start', type: StepType): void
  (e: 'drag-end'): void
  (e: 'open-search'): void
}>()

// Block Group definitions for control flow constructs
const blockGroupTypes: Array<{
  type: BlockGroupType
  name: string
  description: string
  icon: string
  color: string
}> = [
  { type: 'parallel', name: 'Parallel', description: 'Execute steps in parallel', icon: '⫲', color: '#8b5cf6' },
  { type: 'try_catch', name: 'Try-Catch', description: 'Error handling block', icon: '⚡', color: '#ef4444' },
  { type: 'foreach', name: 'ForEach', description: 'Loop over items', icon: '∀', color: '#22c55e' },
  { type: 'while', name: 'While', description: 'Repeat while condition', icon: '↻', color: '#14b8a6' },
]

// Fetch blocks from API
const blocks = ref<BlockDefinition[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Active category tab
const activeCategory = ref<BlockCategory>('ai')

// Expanded subcategories
const expandedSubcategories = ref<Set<string>>(new Set())

onMounted(async () => {
  try {
    const response = await blocksApi.list({ enabled: true })
    blocks.value = response.blocks
    // Expand all subcategories by default (including block groups)
    expandedSubcategories.value.add('__groups__')
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

// Categories for tabs
const categories: BlockCategory[] = ['ai', 'flow', 'apps', 'custom']

// Get blocks grouped by subcategory for the active category
const blocksBySubcategory = computed(() => {
  return groupBlocksBySubcategory(
    blocks.value.filter(b => b.slug !== 'start'),
    activeCategory.value
  )
})

// Get sorted subcategories for the active category
const subcategoriesForActiveCategory = computed(() => {
  return getSubcategoriesForCategory(activeCategory.value)
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

const draggingType = ref<StepType | null>(null)

function handleDragStart(event: DragEvent, block: BlockDefinition) {
  if (!event.dataTransfer) return

  draggingType.value = block.slug as StepType
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData('step-type', block.slug)
  event.dataTransfer.setData('step-name', block.name)

  emit('drag-start', block.slug as StepType)
}

function handleDragEnd() {
  draggingType.value = null
  emit('drag-end')
}

// Drag handlers for block groups
const draggingGroupType = ref<BlockGroupType | null>(null)

function handleGroupDragStart(event: DragEvent, groupType: { type: BlockGroupType; name: string }) {
  if (!event.dataTransfer) return

  draggingGroupType.value = groupType.type
  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData('group-type', groupType.type)
  event.dataTransfer.setData('group-name', groupType.name)
}

function handleGroupDragEnd() {
  draggingGroupType.value = null
}

// Keyboard shortcut handler
function handleKeydown(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
    event.preventDefault()
    emit('open-search')
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
    <!-- Search Bar -->
    <div class="palette-header">
      <button
        class="search-trigger"
        @click="emit('open-search')"
      >
        <svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.35-4.35" />
        </svg>
        <span class="search-text">{{ $t('editor.searchBlocks') }}</span>
        <kbd class="search-shortcut">⌘K</kbd>
      </button>
    </div>

    <!-- Category Tabs -->
    <div class="category-tabs">
      <button
        v-for="category in categories"
        :key="category"
        :class="['category-tab', { active: activeCategory === category }]"
        :style="activeCategory === category ? { borderColor: getCategoryInfo(category).color } : {}"
        @click="activeCategory = category"
      >
        <span
          class="tab-indicator"
          :style="{ backgroundColor: getCategoryInfo(category).color }"
        />
        <span class="tab-label">{{ getCategoryInfo(category).name }}</span>
      </button>
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
      <!-- Control Flow Groups (show only in Flow tab) -->
      <div v-if="activeCategory === 'flow'" class="subcategory-section">
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
          <span class="subcategory-count">{{ blockGroupTypes.length }}</span>
        </button>

        <div v-show="expandedSubcategories.has('__groups__')" class="subcategory-items">
          <div
            v-for="groupType in blockGroupTypes"
            :key="groupType.type"
            :class="[
              'palette-item',
              'palette-item-group',
              {
                dragging: draggingGroupType === groupType.type,
                disabled: readonly
              }
            ]"
            :draggable="!readonly"
            @dragstart="handleGroupDragStart($event, groupType)"
            @dragend="handleGroupDragEnd"
          >
            <div
              class="item-icon"
              :style="{ backgroundColor: groupType.color }"
            >
              {{ groupType.icon }}
            </div>
            <div class="item-content">
              <div class="item-name">{{ groupType.name }}</div>
              <div class="item-desc">{{ groupType.description }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Subcategory Sections -->
      <template v-for="subcategory in subcategoriesForActiveCategory" :key="subcategory">
        <div
          v-if="blocksBySubcategory[subcategory]?.length"
          class="subcategory-section"
        >
          <button
            class="subcategory-header"
            @click="toggleSubcategory(subcategory)"
          >
            <svg
              :class="['chevron', { expanded: expandedSubcategories.has(subcategory) }]"
              width="12"
              height="12"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <path d="m9 18 6-6-6-6" />
            </svg>
            <span class="subcategory-name">{{ getSubcategoryLabel(subcategory) }}</span>
            <span class="subcategory-count">{{ blocksBySubcategory[subcategory]?.length || 0 }}</span>
          </button>

          <div v-show="expandedSubcategories.has(subcategory)" class="subcategory-items">
            <div
              v-for="block in blocksBySubcategory[subcategory]"
              :key="block.slug"
              :class="[
                'palette-item',
                {
                  dragging: draggingType === block.slug,
                  disabled: readonly
                }
              ]"
              :draggable="!readonly"
              @dragstart="handleDragStart($event, block)"
              @dragend="handleDragEnd"
            >
              <div
                class="item-color"
                :style="{ backgroundColor: getBlockColor(block.slug) }"
              />
              <div class="item-content">
                <div class="item-name">{{ block.name }}</div>
                <div class="item-desc">{{ block.description || '' }}</div>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- Uncategorized blocks (if any) -->
      <div
        v-if="blocksBySubcategory['other']?.length"
        class="subcategory-section"
      >
        <button
          class="subcategory-header"
          @click="toggleSubcategory('other')"
        >
          <svg
            :class="['chevron', { expanded: expandedSubcategories.has('other') }]"
            width="12"
            height="12"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="m9 18 6-6-6-6" />
          </svg>
          <span class="subcategory-name">Other</span>
          <span class="subcategory-count">{{ blocksBySubcategory['other']?.length || 0 }}</span>
        </button>

        <div v-show="expandedSubcategories.has('other')" class="subcategory-items">
          <div
            v-for="block in blocksBySubcategory['other']"
            :key="block.slug"
            :class="[
              'palette-item',
              {
                dragging: draggingType === block.slug,
                disabled: readonly
              }
            ]"
            :draggable="!readonly"
            @dragstart="handleDragStart($event, block)"
            @dragend="handleDragEnd"
          >
            <div
              class="item-color"
              :style="{ backgroundColor: getBlockColor(block.slug) }"
            />
            <div class="item-content">
              <div class="item-name">{{ block.name }}</div>
              <div class="item-desc">{{ block.description || '' }}</div>
            </div>
          </div>
        </div>
      </div>

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

.palette-header {
  padding: 12px;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.search-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  background: var(--color-background-secondary, #f9fafb);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.search-trigger:hover {
  border-color: var(--color-primary);
  background: white;
}

.search-icon {
  color: var(--color-text-secondary);
  flex-shrink: 0;
}

.search-text {
  flex: 1;
  text-align: left;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

.search-shortcut {
  font-size: 0.6875rem;
  padding: 2px 6px;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  color: var(--color-text-secondary);
}

/* Category Tabs */
.category-tabs {
  display: flex;
  padding: 0 12px;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  overflow-x: auto;
}

.category-tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 12px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  transition: all 0.15s;
  white-space: nowrap;
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
}

.tab-label {
  text-transform: uppercase;
  letter-spacing: 0.05em;
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

/* Subcategory Section */
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

/* Palette Item */
.palette-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  cursor: grab;
  transition: all 0.15s;
}

.palette-item:hover:not(.disabled) {
  border-color: var(--color-primary);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  transform: translateY(-1px);
}

.palette-item:active:not(.disabled) {
  cursor: grabbing;
  transform: translateY(0);
}

.palette-item.dragging {
  opacity: 0.5;
  cursor: grabbing;
}

.palette-item.disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.item-color {
  width: 4px;
  height: 28px;
  border-radius: 2px;
  flex-shrink: 0;
}

.item-icon {
  width: 26px;
  height: 26px;
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.8rem;
  color: white;
  flex-shrink: 0;
}

.palette-item-group {
  border-style: dashed;
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-name {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text);
  line-height: 1.2;
}

.item-desc {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  line-height: 1.3;
  margin-top: 1px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
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
