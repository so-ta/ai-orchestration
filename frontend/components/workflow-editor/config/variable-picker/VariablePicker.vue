<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import type { AvailableVariable } from '../../composables/useAvailableVariables'
import type { VariableTreeNode } from './useVariablePicker'
import { useVariablePicker } from './useVariablePicker'
import VariablePickerItem from './VariablePickerItem.vue'

const props = defineProps<{
  variables: AvailableVariable[]
  position?: { top: number; left: number }
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'select', variable: AvailableVariable): void
}>()

const variablesRef = computed(() => props.variables)
const searchInputRef = ref<HTMLInputElement | null>(null)
const listRef = ref<HTMLDivElement | null>(null)

const {
  searchQuery,
  selectedIndex,
  filteredVariables,
  toggleExpand,
  handleKeydown,
  selectVariable,
  setSelectedIndex
} = useVariablePicker({
  variables: variablesRef,
  onSelect: (variable) => {
    emit('select', variable)
    emit('update:modelValue', false)
  }
})

// Focus search input when opened
watch(() => props.modelValue, async (isOpen) => {
  if (isOpen) {
    searchQuery.value = ''
    selectedIndex.value = 0
    await nextTick()
    searchInputRef.value?.focus()
  }
})

// Scroll selected item into view
watch(selectedIndex, async () => {
  await nextTick()
  const list = listRef.value
  if (!list) return

  const selectedEl = list.querySelector('.variable-picker-item.selected')
  if (selectedEl) {
    selectedEl.scrollIntoView({ block: 'nearest' })
  }
})

function handleClose() {
  emit('update:modelValue', false)
}

function handleInputKeydown(event: KeyboardEvent) {
  // Handle Escape to close picker
  if (event.key === 'Escape') {
    event.preventDefault()
    handleClose()
    return
  }

  if (handleKeydown(event)) {
    return
  }
}

function handleItemSelect(variable: VariableTreeNode) {
  if (variable.type === 'group') {
    toggleExpand(variable.path)
  } else {
    selectVariable(variable)
  }
}

function handleItemHover(variable: VariableTreeNode, index: number) {
  setSelectedIndex(index)
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="variable-picker-overlay"
      @click="handleClose"
    />
    <div
      v-if="modelValue"
      class="variable-picker"
      :style="{
        top: position ? `${position.top}px` : '50%',
        left: position ? `${position.left}px` : '50%',
        transform: position ? 'none' : 'translate(-50%, -50%)'
      }"
    >
      <div class="picker-header">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <path d="m21 21-4.3-4.3"/>
        </svg>
        <input
          ref="searchInputRef"
          v-model="searchQuery"
          type="text"
          class="search-input"
          placeholder="変数を検索..."
          @keydown="handleInputKeydown"
        >
      </div>

      <div ref="listRef" class="picker-list">
        <template v-if="filteredVariables.length > 0">
          <VariablePickerItem
            v-for="(variable, index) in filteredVariables"
            :key="variable.path"
            :variable="variable"
            :selected="index === selectedIndex"
            :search-query="searchQuery"
            @select="handleItemSelect"
            @mouseenter="handleItemHover(variable, index)"
          />
        </template>
        <div v-else class="picker-empty">
          <span>変数が見つかりません</span>
        </div>
      </div>

      <div class="picker-footer">
        <span class="shortcut"><kbd>↑</kbd><kbd>↓</kbd> 移動</span>
        <span class="shortcut"><kbd>→</kbd> 展開/選択</span>
        <span class="shortcut"><kbd>←</kbd> 折りたたみ</span>
        <span class="shortcut"><kbd>Esc</kbd> 閉じる</span>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.variable-picker-overlay {
  position: fixed;
  inset: 0;
  z-index: 9998;
}

.variable-picker {
  position: fixed;
  z-index: 9999;
  width: 320px;
  max-height: 400px;
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 8px;
  box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.picker-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--color-border, #e5e7eb);
  background: var(--color-background, #f9fafb);
}

.picker-header svg {
  color: var(--color-text-secondary, #6b7280);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 13px;
  color: var(--color-text, #111827);
  outline: none;
}

.search-input::placeholder {
  color: var(--color-text-secondary, #6b7280);
}

.picker-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
  max-height: 280px;
}

.picker-empty {
  padding: 24px;
  text-align: center;
  color: var(--color-text-secondary, #6b7280);
  font-size: 12px;
}

.picker-footer {
  display: flex;
  gap: 12px;
  padding: 8px 12px;
  border-top: 1px solid var(--color-border, #e5e7eb);
  background: var(--color-background, #f9fafb);
}

.shortcut {
  display: flex;
  align-items: center;
  gap: 2px;
  font-size: 10px;
  color: var(--color-text-secondary, #6b7280);
}

.shortcut kbd {
  display: inline-block;
  padding: 2px 4px;
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 9px;
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border, #e5e7eb);
  border-radius: 3px;
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.05);
}

.picker-list::-webkit-scrollbar {
  width: 6px;
}

.picker-list::-webkit-scrollbar-track {
  background: transparent;
}

.picker-list::-webkit-scrollbar-thumb {
  background: var(--color-border, #e5e7eb);
  border-radius: 3px;
}
</style>
