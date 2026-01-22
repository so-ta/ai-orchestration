<script setup lang="ts">
import type { VariableTreeNode } from './useVariablePicker'

const props = defineProps<{
  variable: VariableTreeNode
  selected: boolean
  searchQuery?: string
}>()

const emit = defineEmits<{
  (e: 'select', variable: VariableTreeNode): void
  (e: 'expand', path: string): void
}>()

// Type icon based on variable type
const typeIcon = computed(() => {
  switch (props.variable.type) {
    case 'string':
      return { icon: 'T', color: '#22c55e' }
    case 'number':
      return { icon: '#', color: '#3b82f6' }
    case 'boolean':
      return { icon: '?', color: '#f59e0b' }
    case 'array':
      return { icon: '[]', color: '#8b5cf6' }
    case 'object':
      return { icon: '{}', color: '#ec4899' }
    case 'group':
      return { icon: props.variable.expanded ? '▼' : '▶', color: '#6b7280' }
    default:
      return { icon: '•', color: '#6b7280' }
  }
})

// Highlight matching text in path
function highlightMatch(text: string, query: string | undefined): string {
  if (!query) return escapeHtml(text)
  const regex = new RegExp(`(${escapeRegExp(query)})`, 'gi')
  return escapeHtml(text).replace(regex, '<mark>$1</mark>')
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function handleClick() {
  if (props.variable.type === 'group') {
    emit('expand', props.variable.path)
  } else {
    emit('select', props.variable)
  }
}
</script>

<template>
  <div
    class="variable-picker-item"
    :class="{
      selected: selected,
      'is-group': variable.type === 'group',
      expanded: variable.expanded
    }"
    :style="{ paddingLeft: `${(variable.level || 0) * 12 + 8}px` }"
    @click="handleClick"
  >
    <span
      class="type-icon"
      :style="{ color: typeIcon.color }"
    >
      {{ typeIcon.icon }}
    </span>

    <!-- eslint-disable vue/no-v-html -- 検索ハイライトの安全なレンダリング -->
    <span
      class="variable-path"
      v-html="highlightMatch(variable.path, searchQuery)"
    />
    <!-- eslint-enable vue/no-v-html -->

    <span v-if="variable.type !== 'group'" class="variable-type">
      {{ variable.type }}
    </span>
  </div>
</template>

<style scoped>
.variable-picker-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  cursor: pointer;
  border-radius: 4px;
  font-size: 12px;
  transition: background-color 0.1s;
}

.variable-picker-item:hover,
.variable-picker-item.selected {
  background: var(--color-primary-alpha, rgba(59, 130, 246, 0.1));
}

.variable-picker-item.is-group {
  font-weight: 500;
  color: var(--color-text-secondary);
  background: var(--color-background);
  margin-top: 4px;
}

.variable-picker-item.is-group:first-child {
  margin-top: 0;
}

.variable-picker-item.is-group:hover,
.variable-picker-item.is-group.selected {
  background: var(--color-border);
}

.type-icon {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 10px;
  font-weight: 600;
  width: 16px;
  text-align: center;
  flex-shrink: 0;
}

.variable-path {
  flex: 1;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 11px;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.variable-path :deep(mark) {
  background: #fef08a;
  color: inherit;
  border-radius: 2px;
  padding: 0 1px;
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
</style>
