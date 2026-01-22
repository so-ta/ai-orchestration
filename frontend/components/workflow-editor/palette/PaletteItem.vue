<script setup lang="ts">
import type { BlockDefinition } from '~/types/api'
import { getBlockColor } from '~/composables/useBlocks'
import BlockTooltip from './BlockTooltip.vue'

const props = defineProps<{
  block: BlockDefinition
  dragging?: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  'drag-start': [event: DragEvent]
  'drag-end': []
}>()

// Tooltip state
const showTooltip = ref(false)
const tooltipPosition = ref({ x: 0, y: 0 })
let tooltipTimeout: ReturnType<typeof setTimeout> | null = null

function handleMouseEnter(event: MouseEvent) {
  if (props.dragging || props.disabled) return

  // Capture the element reference immediately (event.currentTarget becomes null after event)
  const element = event.currentTarget as HTMLElement

  // Delay showing tooltip to avoid flicker (200ms for responsive feel)
  tooltipTimeout = setTimeout(() => {
    // Ensure element is still in the DOM
    if (!element.isConnected) return

    const rect = element.getBoundingClientRect()
    tooltipPosition.value = {
      x: rect.right + 10,
      y: rect.top,
    }
    showTooltip.value = true
  }, 200)
}

function handleMouseLeave() {
  if (tooltipTimeout) {
    clearTimeout(tooltipTimeout)
    tooltipTimeout = null
  }
  showTooltip.value = false
}

function handleDragStart(event: DragEvent) {
  if (!event.dataTransfer || props.disabled) return

  // Hide tooltip when dragging
  showTooltip.value = false
  if (tooltipTimeout) {
    clearTimeout(tooltipTimeout)
    tooltipTimeout = null
  }

  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData('step-type', props.block.slug)
  event.dataTransfer.setData('step-name', props.block.name)

  emit('drag-start', event)
}

onUnmounted(() => {
  if (tooltipTimeout) {
    clearTimeout(tooltipTimeout)
  }
})
</script>

<template>
  <div
    :class="[
      'palette-item',
      {
        dragging,
        disabled
      }
    ]"
    :draggable="!disabled"
    @dragstart="handleDragStart"
    @dragend="emit('drag-end')"
    @mouseenter="handleMouseEnter"
    @mouseleave="handleMouseLeave"
  >
    <div
      class="item-color"
      :style="{ backgroundColor: getBlockColor(block.slug) }"
    />
    <div class="item-content">
      <div class="item-name">{{ block.name }}</div>
      <div class="item-desc">{{ block.description || '' }}</div>
    </div>

    <!-- Block preview tooltip -->
    <BlockTooltip
      :block="block"
      :visible="showTooltip"
      :position="tooltipPosition"
    />
  </div>
</template>

<style scoped>
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
</style>
