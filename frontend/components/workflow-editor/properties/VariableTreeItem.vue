<script setup lang="ts">
import type { AvailableVariable } from '../composables/useAvailableVariables'

const props = defineProps<{
  variable: AvailableVariable
  expanded?: boolean
  children?: AvailableVariable[]
  draggable?: boolean
}>()

const emit = defineEmits<{
  (e: 'click', variable: AvailableVariable): void
  (e: 'toggle'): void
  (e: 'drag-start', event: DragEvent, variable: AvailableVariable): void
}>()

const hasChildren = computed(() => props.children && props.children.length > 0)

// Type icon based on variable type
const typeIcon = computed(() => {
  switch (props.variable.type) {
    case 'string':
      return { icon: 'T', color: '#22c55e', bg: '#dcfce7' }
    case 'number':
      return { icon: '#', color: '#3b82f6', bg: '#dbeafe' }
    case 'boolean':
      return { icon: '?', color: '#f59e0b', bg: '#fef3c7' }
    case 'array':
      return { icon: '[]', color: '#8b5cf6', bg: '#ede9fe' }
    case 'object':
      return { icon: '{}', color: '#ec4899', bg: '#fce7f3' }
    default:
      return { icon: '•', color: '#6b7280', bg: '#f3f4f6' }
  }
})

function formatTemplateVariable(variable: string): string {
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)
  return openBrace + variable + closeBrace
}

function handleClick() {
  emit('click', props.variable)
}

function handleToggle(e: Event) {
  e.stopPropagation()
  emit('toggle')
}

function handleDragStart(event: DragEvent) {
  if (!props.draggable) return
  const template = formatTemplateVariable(props.variable.path)
  event.dataTransfer?.setData('text/plain', template)
  event.dataTransfer!.effectAllowed = 'copy'
  emit('drag-start', event, props.variable)
}
</script>

<template>
  <div class="variable-tree-item">
    <div
      class="variable-item-content"
      :class="{ draggable: draggable }"
      :draggable="draggable"
      @click="handleClick"
      @dragstart="handleDragStart"
    >
      <button
        v-if="hasChildren"
        class="expand-button"
        @click="handleToggle"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="10"
          height="10"
          viewBox="0 0 24 24"
          fill="currentColor"
          :class="{ expanded }"
        >
          <path d="M8 5v14l11-7z"/>
        </svg>
      </button>
      <span v-else class="expand-placeholder" />

      <span
        class="type-badge"
        :style="{ color: typeIcon.color, backgroundColor: typeIcon.bg }"
      >
        {{ typeIcon.icon }}
      </span>

      <code class="variable-path">{{ formatTemplateVariable(variable.path) }}</code>

      <span class="variable-type">{{ variable.type }}</span>
    </div>

    <div v-if="hasChildren && expanded" class="variable-children">
      <VariableTreeItem
        v-for="child in children"
        :key="child.path"
        :variable="child"
        :draggable="draggable"
        @click="(v) => emit('click', v)"
        @drag-start="(e, v) => emit('drag-start', e, v)"
      />
    </div>
  </div>
</template>

<style scoped>
.variable-tree-item {
  display: flex;
  flex-direction: column;
}

.variable-item-content {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.1s;
}

.variable-item-content:hover {
  background: rgba(34, 197, 94, 0.1);
}

.variable-item-content.draggable {
  cursor: grab;
}

.variable-item-content.draggable:active {
  cursor: grabbing;
}

.expand-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-secondary);
  transition: transform 0.15s;
}

.expand-button svg {
  transition: transform 0.15s;
}

.expand-button svg.expanded {
  transform: rotate(90deg);
}

.expand-placeholder {
  width: 16px;
  flex-shrink: 0;
}

.type-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 9px;
  font-weight: 600;
  border-radius: 4px;
  flex-shrink: 0;
}

.variable-path {
  flex: 1;
  font-size: 11px;
  font-family: 'SF Mono', Monaco, monospace;
  color: #166534;
  background: #dcfce7;
  padding: 2px 6px;
  border-radius: 4px;
  border: 1px solid #86efac;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.variable-type {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 9px;
  color: var(--color-text-secondary);
  background: var(--color-background);
  padding: 2px 4px;
  border-radius: 3px;
  flex-shrink: 0;
}

.variable-children {
  margin-left: 16px;
  border-left: 1px dashed var(--color-border);
  padding-left: 8px;
}
</style>
