<script setup lang="ts">
import type { BlockDefinition } from '~/types/api'
import PaletteItem from './PaletteItem.vue'

defineProps<{
  name: string
  blocks: BlockDefinition[]
  expanded: boolean
  readonly?: boolean
  draggingType?: string | null
}>()

const emit = defineEmits<{
  'toggle': []
  'drag-start': [event: DragEvent, block: BlockDefinition]
  'drag-end': []
}>()
</script>

<template>
  <div class="subcategory-section">
    <button
      class="subcategory-header"
      @click="emit('toggle')"
    >
      <svg
        :class="['chevron', { expanded }]"
        width="12"
        height="12"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <path d="m9 18 6-6-6-6" />
      </svg>
      <span class="subcategory-name">{{ name }}</span>
      <span class="subcategory-count">{{ blocks.length }}</span>
    </button>

    <div v-show="expanded" class="subcategory-items">
      <PaletteItem
        v-for="block in blocks"
        :key="block.slug"
        :block="block"
        :dragging="draggingType === block.slug"
        :disabled="readonly"
        @drag-start="emit('drag-start', $event, block)"
        @drag-end="emit('drag-end')"
      />
    </div>
  </div>
</template>

<style scoped>
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
</style>
