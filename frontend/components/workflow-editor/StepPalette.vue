<script setup lang="ts">
import type { StepType, BlockDefinition, BlockCategory, BlockGroupType } from '~/types/api'
import { categoryConfig, getBlockColor } from '~/composables/useBlocks'

const { t } = useI18n()
const blocksApi = useBlocks()

defineProps<{
  readonly?: boolean
}>()

const emit = defineEmits<{
  (e: 'drag-start', type: StepType): void
  (e: 'drag-end'): void
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

onMounted(async () => {
  try {
    const response = await blocksApi.list({ enabled: true })
    blocks.value = response.blocks
  } catch (e) {
    console.error('Failed to load blocks:', e)
    error.value = 'Failed to load blocks'
  } finally {
    loading.value = false
  }
})

// Group blocks by category
const blocksByCategory = computed(() => {
  const grouped = new Map<BlockCategory, BlockDefinition[]>()

  for (const block of blocks.value) {
    // Skip 'start' block as it's auto-created
    if (block.slug === 'start') continue

    const category = block.category
    if (!grouped.has(category)) {
      grouped.set(category, [])
    }
    grouped.get(category)!.push(block)
  }

  return grouped
})

// Sorted categories for display
const sortedCategories = computed(() => {
  const categories = Array.from(blocksByCategory.value.keys())
  return categories.sort((a, b) => {
    const orderA = categoryConfig[a]?.order ?? 99
    const orderB = categoryConfig[b]?.order ?? 99
    return orderA - orderB
  })
})

// Get category display info
function getCategoryInfo(category: BlockCategory) {
  const config = categoryConfig[category]
  return {
    name: t(config?.nameKey || `editor.categories.${category}`),
    icon: config?.icon || 'folder',
  }
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
</script>

<template>
  <div class="step-palette">
    <div class="palette-header">
      <h3 class="palette-title">{{ $t('editor.blocks') }}</h3>
      <p class="palette-hint">{{ $t('editor.dragToCanvas') }}</p>
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
      <!-- Control Flow Groups Section -->
      <div class="palette-category">
        <div class="category-header">
          <span class="category-name">Control Flow</span>
        </div>

        <div class="category-items">
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

      <!-- Block Categories -->
      <div
        v-for="category in sortedCategories"
        :key="category"
        class="palette-category"
      >
        <div class="category-header">
          <span class="category-name">{{ getCategoryInfo(category).name }}</span>
        </div>

        <div class="category-items">
          <div
            v-for="block in blocksByCategory.get(category)"
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
    </div>
  </div>
</template>

<style scoped>
.step-palette {
  display: flex;
  flex-direction: column;
  height: 100%;
}

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

.palette-header {
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.palette-title {
  font-size: 0.875rem;
  font-weight: 600;
  margin: 0;
  color: var(--color-text);
}

.palette-hint {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  margin: 0.25rem 0 0 0;
}

.palette-content {
  flex: 1;
  overflow-y: auto;
  padding: 0.75rem;
}

.palette-category {
  margin-bottom: 1.25rem;
}

.palette-category:last-child {
  margin-bottom: 0;
}

.category-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
  padding: 0 0.25rem;
}

.category-name {
  font-size: 0.6875rem;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.category-items {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.palette-item {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  padding: 0.5rem 0.625rem;
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
  height: 32px;
  border-radius: 2px;
  flex-shrink: 0;
}

/* Block group item icon */
.item-icon {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
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
  margin-top: 0.125rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Scrollbar styling */
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
