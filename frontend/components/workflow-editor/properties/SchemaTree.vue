<script setup lang="ts">
import { ref, computed } from 'vue'

export interface SchemaNode {
  name: string
  type: string
  description?: string
  children?: SchemaNode[]
  path: string
  required?: boolean
  example?: unknown
}

const props = defineProps<{
  schema: SchemaNode
  level?: number
  draggable?: boolean
}>()

const emit = defineEmits<{
  (e: 'select', path: string): void
  (e: 'drag-start', event: DragEvent, path: string): void
}>()

const level = computed(() => props.level ?? 0)
const expanded = ref(level.value < 2) // Auto-expand first 2 levels

const hasChildren = computed(() => props.schema.children && props.schema.children.length > 0)

// Type styling
const typeStyle = computed(() => {
  switch (props.schema.type) {
    case 'string':
      return { color: '#22c55e', bg: '#dcfce7' }
    case 'number':
    case 'integer':
      return { color: '#3b82f6', bg: '#dbeafe' }
    case 'boolean':
      return { color: '#f59e0b', bg: '#fef3c7' }
    case 'array':
      return { color: '#8b5cf6', bg: '#ede9fe' }
    case 'object':
      return { color: '#ec4899', bg: '#fce7f3' }
    default:
      return { color: '#6b7280', bg: '#f3f4f6' }
  }
})

function formatTemplateVariable(path: string): string {
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)
  return openBrace + path + closeBrace
}

function handleClick() {
  emit('select', props.schema.path)
}

function toggleExpand() {
  expanded.value = !expanded.value
}

function handleDragStart(event: DragEvent) {
  if (!props.draggable) return
  const template = formatTemplateVariable(props.schema.path)
  event.dataTransfer?.setData('text/plain', template)
  event.dataTransfer!.effectAllowed = 'copy'
  emit('drag-start', event, props.schema.path)
}
</script>

<template>
  <div class="schema-tree-node" :style="{ '--level': level }">
    <div
      class="node-content"
      :class="{ draggable, 'has-children': hasChildren }"
      :draggable="draggable"
      @click="handleClick"
      @dragstart="handleDragStart"
    >
      <button
        v-if="hasChildren"
        class="expand-btn"
        @click.stop="toggleExpand"
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

      <span class="node-name">
        {{ schema.name }}
        <span v-if="schema.required" class="required-mark">*</span>
      </span>

      <span
        class="node-type"
        :style="{ color: typeStyle.color, backgroundColor: typeStyle.bg }"
      >
        {{ schema.type }}
      </span>

      <span v-if="schema.example !== undefined" class="node-example">
        {{ JSON.stringify(schema.example) }}
      </span>
    </div>

    <p v-if="schema.description && expanded" class="node-description">
      {{ schema.description }}
    </p>

    <div v-if="hasChildren && expanded" class="node-children">
      <SchemaTree
        v-for="child in schema.children"
        :key="child.path"
        :schema="child"
        :level="level + 1"
        :draggable="draggable"
        @select="(p) => emit('select', p)"
        @drag-start="(e, p) => emit('drag-start', e, p)"
      />
    </div>
  </div>
</template>

<style scoped>
.schema-tree-node {
  display: flex;
  flex-direction: column;
}

.node-content {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 6px;
  border-radius: 4px;
  cursor: pointer;
  margin-left: calc(var(--level) * 12px);
  transition: background-color 0.1s;
}

.node-content:hover {
  background: var(--color-primary-alpha, rgba(59, 130, 246, 0.1));
}

.node-content.draggable {
  cursor: grab;
}

.node-content.draggable:active {
  cursor: grabbing;
}

.expand-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  padding: 0;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-secondary);
}

.expand-btn svg {
  transition: transform 0.15s;
}

.expand-btn svg.expanded {
  transform: rotate(90deg);
}

.expand-placeholder {
  width: 14px;
  flex-shrink: 0;
}

.node-name {
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text);
}

.required-mark {
  color: var(--color-error);
  margin-left: 1px;
}

.node-type {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 9px;
  font-weight: 500;
  padding: 2px 5px;
  border-radius: 3px;
}

.node-example {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 9px;
  color: var(--color-text-secondary);
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-description {
  font-size: 10px;
  color: var(--color-text-secondary);
  margin: 2px 0 4px calc(var(--level) * 12px + 20px);
  line-height: 1.3;
}

.node-children {
  border-left: 1px dashed var(--color-border);
  margin-left: calc(var(--level) * 12px + 7px);
  padding-left: 5px;
}
</style>
