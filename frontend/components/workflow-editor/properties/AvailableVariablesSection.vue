<script setup lang="ts">
import { ref } from 'vue'
import type { AvailableVariable } from '../composables/useAvailableVariables'
import VariableTreeItem from './VariableTreeItem.vue'

const { t } = useI18n()

const props = defineProps<{
  variables: AvailableVariable[]
}>()

const emit = defineEmits<{
  (e: 'insert', variable: AvailableVariable): void
}>()

// Expanded state for nested variables
const expandedSources = ref<Set<string>>(new Set())

// Group variables by source
const groupedVariables = computed(() => {
  const groups = new Map<string, AvailableVariable[]>()

  for (const v of props.variables) {
    const existing = groups.get(v.source) || []
    existing.push(v)
    groups.set(v.source, existing)
  }

  return groups
})

function formatTemplateVariable(variable: string): string {
  const openBrace = String.fromCharCode(123, 123)
  const closeBrace = String.fromCharCode(125, 125)
  return openBrace + variable + closeBrace
}

function toggleSource(source: string) {
  if (expandedSources.value.has(source)) {
    expandedSources.value.delete(source)
  } else {
    expandedSources.value.add(source)
  }
  expandedSources.value = new Set(expandedSources.value)
}

function handleVariableClick(variable: AvailableVariable) {
  emit('insert', variable)
}

function handleDragStart(event: DragEvent, variable: AvailableVariable) {
  const template = formatTemplateVariable(variable.path)
  event.dataTransfer?.setData('text/plain', template)
  event.dataTransfer!.effectAllowed = 'copy'

  // Add drag image
  const dragEl = document.createElement('div')
  dragEl.textContent = template
  dragEl.style.cssText = `
    position: absolute;
    top: -1000px;
    padding: 4px 8px;
    background: #dcfce7;
    border: 1px solid #86efac;
    border-radius: 4px;
    font-family: monospace;
    font-size: 11px;
    color: #166534;
  `
  document.body.appendChild(dragEl)
  event.dataTransfer?.setDragImage(dragEl, 0, 0)

  // Clean up drag element after a short delay
  setTimeout(() => dragEl.remove(), 0)
}
</script>

<template>
  <div class="form-section available-variables-section">
    <h4 class="section-title">
      <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
        <polyline points="7 10 12 15 17 10"/>
        <line x1="12" y1="15" x2="12" y2="3"/>
      </svg>
      {{ t('stepConfig.availableVariables.title') }}
    </h4>
    <p class="section-description">
      {{ t('stepConfig.availableVariables.description') }}
      <span class="hint">クリックまたはドラッグで挿入</span>
    </p>
    <div class="available-variables-list">
      <template v-for="[source, vars] in groupedVariables" :key="source">
        <div class="source-group">
          <button
            class="source-header"
            @click="toggleSource(source)"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="10"
              height="10"
              viewBox="0 0 24 24"
              fill="currentColor"
              :class="{ expanded: expandedSources.has(source) }"
            >
              <path d="M8 5v14l11-7z"/>
            </svg>
            <span class="source-name">{{ source === 'input' ? 'Workflow Input' : source }}</span>
            <span class="source-count">{{ vars.length }}</span>
          </button>
          <div v-if="expandedSources.has(source)" class="source-variables">
            <VariableTreeItem
              v-for="variable in vars"
              :key="variable.path"
              :variable="variable"
              :draggable="true"
              @click="handleVariableClick"
              @drag-start="handleDragStart"
            />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.form-section {
  margin-bottom: 1.5rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--color-border);
}

.form-section:last-child {
  margin-bottom: 0;
  padding-bottom: 0;
  border-bottom: none;
}

.section-title {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 0.75rem 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.section-description {
  font-size: 0.6875rem;
  color: var(--color-text-secondary);
  margin: 0 0 0.75rem 0;
  line-height: 1.4;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-description .hint {
  font-size: 0.5625rem;
  color: #15803d;
  background: rgba(255, 255, 255, 0.5);
  padding: 2px 6px;
  border-radius: 3px;
}

.available-variables-section {
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
  border: 1px solid #86efac;
  border-radius: 8px;
  padding: 0.875rem !important;
  margin-top: 0.5rem;
}

.available-variables-section .section-title {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  color: #166534;
  margin-bottom: 0.5rem;
}

.available-variables-section .section-title svg {
  color: #22c55e;
}

.available-variables-section .section-description {
  color: #15803d;
}

.available-variables-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  background: rgba(255, 255, 255, 0.7);
  border-radius: 6px;
  padding: 0.5rem;
  max-height: 250px;
  overflow-y: auto;
}

.source-group {
  display: flex;
  flex-direction: column;
}

.source-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid #bbf7d0;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  font-weight: 500;
  color: #166534;
  transition: background-color 0.1s;
}

.source-header:hover {
  background: white;
}

.source-header svg {
  color: #22c55e;
  transition: transform 0.15s;
}

.source-header svg.expanded {
  transform: rotate(90deg);
}

.source-name {
  flex: 1;
}

.source-count {
  font-size: 10px;
  color: #15803d;
  background: #dcfce7;
  padding: 2px 6px;
  border-radius: 10px;
}

.source-variables {
  margin-left: 16px;
  margin-top: 4px;
  padding-left: 8px;
  border-left: 1px dashed #86efac;
}

.available-variables-list::-webkit-scrollbar {
  width: 6px;
}

.available-variables-list::-webkit-scrollbar-track {
  background: transparent;
}

.available-variables-list::-webkit-scrollbar-thumb {
  background: #86efac;
  border-radius: 3px;
}
</style>
