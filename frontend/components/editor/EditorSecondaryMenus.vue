<script setup lang="ts">
/**
 * EditorSecondaryMenus.vue
 * セカンダリメニュー（Runs/Schedules/Variables トグルボタン）
 */

import type { SlideOutPanel } from '~/composables/useEditorState'

const { t } = useI18n()

const props = defineProps<{
  activePanel: SlideOutPanel
}>()

const emit = defineEmits<{
  (e: 'toggle', panel: Exclude<SlideOutPanel, null>): void
}>()

function isActive(panel: Exclude<SlideOutPanel, null>): boolean {
  return props.activePanel === panel
}
</script>

<template>
  <div class="secondary-menus">
    <!-- Runs button -->
    <button
      :class="['menu-btn', { active: isActive('runs') }]"
      :title="t('editor.runs')"
      @click="emit('toggle', 'runs')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10" />
        <polyline points="12 6 12 12 16 14" />
      </svg>
      <span class="menu-label">{{ t('editor.runs') }}</span>
    </button>

    <!-- Schedules button -->
    <button
      :class="['menu-btn', { active: isActive('schedules') }]"
      :title="t('editor.schedules')"
      @click="emit('toggle', 'schedules')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
        <line x1="16" y1="2" x2="16" y2="6" />
        <line x1="8" y1="2" x2="8" y2="6" />
        <line x1="3" y1="10" x2="21" y2="10" />
      </svg>
      <span class="menu-label">{{ t('editor.schedules') }}</span>
    </button>

    <!-- Variables button -->
    <button
      :class="['menu-btn', { active: isActive('variables') }]"
      :title="t('editor.variables')"
      @click="emit('toggle', 'variables')"
    >
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="16 18 22 12 16 6" />
        <polyline points="8 6 2 12 8 18" />
      </svg>
      <span class="menu-label">{{ t('editor.variables') }}</span>
    </button>
  </div>
</template>

<style scoped>
.secondary-menus {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.menu-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.625rem;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius);
  color: var(--color-text-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.menu-btn:hover {
  background: var(--color-surface);
  color: var(--color-text);
}

.menu-btn.active {
  background: var(--color-primary-light, #eff6ff);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.menu-label {
  display: none;
}

@media (min-width: 1024px) {
  .menu-label {
    display: inline;
  }
}
</style>
