<script setup lang="ts">
/**
 * CopilotOptionCard.vue
 *
 * Selectable option card for inline select actions.
 * Shows icon, label, description with hover/select states.
 */
import type { SelectOption } from './types'

const props = defineProps<{
  option: SelectOption
  selected?: boolean
  disabled?: boolean
}>()

const emit = defineEmits<{
  select: []
}>()

function handleClick() {
  if (!props.disabled) {
    emit('select')
  }
}
</script>

<template>
  <button
    class="option-card"
    :class="{ selected, disabled, recommended: option.recommended }"
    :disabled="disabled"
    @click="handleClick"
  >
    <span v-if="option.icon" class="option-icon">{{ option.icon }}</span>
    <div class="option-content">
      <span class="option-label">
        {{ option.label }}
        <span v-if="option.recommended" class="recommended-badge">
          推奨
        </span>
      </span>
      <span v-if="option.description" class="option-description">
        {{ option.description }}
      </span>
    </div>
    <span class="select-indicator">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
      >
        <polyline points="9 18 15 12 9 6" />
      </svg>
    </span>
  </button>
</template>

<style scoped>
.option-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.75rem 1rem;
  text-align: left;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.option-card:hover:not(:disabled) {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.05);
}

.option-card.selected {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.1);
}

.option-card.recommended {
  border-color: var(--color-primary);
}

.option-card.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.option-icon {
  font-size: 1.25rem;
  flex-shrink: 0;
}

.option-content {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  flex: 1;
  min-width: 0;
}

.option-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
}

.recommended-badge {
  padding: 0.125rem 0.375rem;
  font-size: 0.625rem;
  font-weight: 600;
  color: var(--color-primary);
  background: rgba(99, 102, 241, 0.15);
  border-radius: 4px;
  text-transform: uppercase;
}

.option-description {
  font-size: 0.75rem;
  color: var(--color-text-secondary);
  line-height: 1.4;
}

.select-indicator {
  display: flex;
  color: var(--color-text-tertiary);
  flex-shrink: 0;
  transition: transform 0.15s, color 0.15s;
}

.option-card:hover:not(:disabled) .select-indicator {
  color: var(--color-primary);
  transform: translateX(2px);
}
</style>
