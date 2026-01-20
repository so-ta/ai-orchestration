<script setup lang="ts">
/**
 * UndoRedoControls.vue
 * Undo/Redo buttons for the workflow editor
 *
 * Provides:
 * - Undo button (Cmd/Ctrl+Z)
 * - Redo button (Cmd/Ctrl+Shift+Z)
 * - Visual feedback for available actions
 * - Optional history count display
 */

import { useCommandHistory } from '~/composables/useCommandHistory'

const { t } = useI18n()
const { canUndo, canRedo, undo, redo, history } = useCommandHistory()

// Detect platform for shortcut labels
const isMac = ref(false)
onMounted(() => {
  isMac.value = navigator.platform.toUpperCase().includes('MAC')
})

// Shortcut labels
const undoShortcut = computed(() => isMac.value ? 'Cmd+Z' : 'Ctrl+Z')
const redoShortcut = computed(() => isMac.value ? 'Cmd+Shift+Z' : 'Ctrl+Shift+Z')

// Handle undo action
async function handleUndo() {
  if (canUndo.value) {
    await undo()
  }
}

// Handle redo action
async function handleRedo() {
  if (canRedo.value) {
    await redo()
  }
}
</script>

<template>
  <div class="undo-redo-controls">
    <button
      class="control-btn"
      :class="{ disabled: !canUndo }"
      :disabled="!canUndo"
      :title="`${t('editor.undo')} (${undoShortcut})`"
      @click="handleUndo"
    >
      <!-- Undo icon -->
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M3 7v6h6" />
        <path d="M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13" />
      </svg>
    </button>

    <button
      class="control-btn"
      :class="{ disabled: !canRedo }"
      :disabled="!canRedo"
      :title="`${t('editor.redo')} (${redoShortcut})`"
      @click="handleRedo"
    >
      <!-- Redo icon -->
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M21 7v6h-6" />
        <path d="M3 17a9 9 0 0 1 9-9 9 9 0 0 1 6 2.3L21 13" />
      </svg>
    </button>

    <!-- History count badge (optional) -->
    <span
      v-if="history.length > 0"
      class="history-badge"
      :title="t('editor.history')"
    >
      {{ history.length }}
    </span>
  </div>
</template>

<style scoped>
.undo-redo-controls {
  display: flex;
  align-items: center;
  gap: 4px;
}

.control-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  background: var(--color-surface, white);
  border: 1px solid var(--color-border, #e5e5e5);
  border-radius: 6px;
  color: var(--color-text, #171717);
  cursor: pointer;
  transition: all 0.15s ease;
}

.control-btn:hover:not(:disabled) {
  background: var(--color-background, #f5f5f5);
  border-color: var(--color-border-hover, #d4d4d4);
}

.control-btn:active:not(:disabled) {
  transform: scale(0.95);
}

.control-btn.disabled,
.control-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.history-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  background: var(--color-primary, #3b82f6);
  color: white;
  font-size: 0.6875rem;
  font-weight: 600;
  border-radius: 10px;
  margin-left: 4px;
}
</style>
