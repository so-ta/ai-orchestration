<script setup lang="ts">
import type { BlockGroupType } from '~/types/api'

export interface GroupItemData {
  type: BlockGroupType
  name: string
  description: string
  icon: string
  color: string
}

const props = defineProps<{
  group: GroupItemData
  dragging?: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  'drag-start': [event: DragEvent]
  'drag-end': []
}>()

function handleDragStart(event: DragEvent) {
  if (!event.dataTransfer || props.disabled) return

  event.dataTransfer.effectAllowed = 'copy'
  event.dataTransfer.setData('group-type', props.group.type)
  event.dataTransfer.setData('group-name', props.group.name)

  emit('drag-start', event)
}
</script>

<template>
  <div
    :class="[
      'palette-item',
      'palette-item-group',
      {
        dragging,
        disabled
      }
    ]"
    :draggable="!disabled"
    @dragstart="handleDragStart"
    @dragend="emit('drag-end')"
  >
    <div
      class="item-icon"
      :style="{ backgroundColor: group.color }"
    >
      {{ group.icon }}
    </div>
    <div class="item-content">
      <div class="item-name">{{ group.name }}</div>
      <div class="item-desc">{{ group.description }}</div>
    </div>
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

.palette-item-group {
  border-style: dashed;
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
