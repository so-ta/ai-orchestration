<script setup lang="ts">
/**
 * FloatingHeader.vue
 * フローティングヘッダー（エディタ上部）
 *
 * 機能:
 * - プロジェクト名表示（クリックで他プロジェクトを開くモーダル表示）
 * - 設定メニュー: リリースとして保存 / 実行履歴 / 環境変数
 * - 保存ステータス表示（灰色テキスト）
 */

import type { Project } from '~/types/api'
import { onClickOutside } from '@vueuse/core'
import UndoRedoControls from './UndoRedoControls.vue'

const { t } = useI18n()

const props = defineProps<{
  project: Project | null
  saving?: boolean
  saveStatus?: 'saved' | 'saving' | 'unsaved' | 'error'
}>()

const emit = defineEmits<{
  save: []
  createRelease: []
  openHistory: []
  openVariables: []
  selectProject: [id: string]
  createProject: []
}>()

// Tools menu state
const showToolsMenu = ref(false)
const toolsMenuRef = ref<HTMLElement | null>(null)

// Project picker modal state
const showProjectPicker = ref(false)

// Run history modal state
const showRunHistory = ref(false)

// Close menu when clicking outside
onClickOutside(toolsMenuRef, () => {
  showToolsMenu.value = false
})

function handleSelectProject(projectId: string) {
  emit('selectProject', projectId)
}

function handleCreateProject() {
  emit('createProject')
}

// Tools menu handlers
function handleCreateRelease() {
  emit('createRelease')
  showToolsMenu.value = false
}

function handleOpenHistory() {
  showRunHistory.value = true
  showToolsMenu.value = false
}

function handleOpenVariables() {
  emit('openVariables')
  showToolsMenu.value = false
}

// Save status display
const saveStatusText = computed(() => {
  switch (props.saveStatus) {
    case 'saving': return t('editor.saving')
    case 'unsaved': return t('editor.unsavedChanges')
    case 'error': return t('editor.saveError')
    default: return t('editor.saved')
  }
})
</script>

<template>
  <header class="floating-header">
    <!-- Project Name (click to open project picker) -->
    <button
      class="project-trigger"
      @click="showProjectPicker = true"
    >
      <span class="project-name">{{ project?.name || t('editor.noProjectSelected') }}</span>
      <svg class="chevron" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="m6 9 6 6 6-6" />
      </svg>
    </button>

    <!-- Tools Menu -->
    <div ref="toolsMenuRef" class="tools-menu-container">
      <button
        class="tools-trigger"
        :title="t('editor.tools')"
        @click="showToolsMenu = !showToolsMenu"
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="3" />
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
        </svg>
        <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      <!-- Tools Menu Dropdown -->
      <Transition name="dropdown">
        <div v-if="showToolsMenu" class="tools-menu">
          <button class="menu-item" @click="handleCreateRelease">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83" />
            </svg>
            {{ t('editor.createRelease') }}
          </button>
          <button class="menu-item" @click="handleOpenHistory">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <polyline points="12 6 12 12 16 14" />
            </svg>
            {{ t('editor.history') }}
          </button>
          <button class="menu-item" @click="handleOpenVariables">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="16 18 22 12 16 6" />
              <polyline points="8 6 2 12 8 18" />
            </svg>
            {{ t('editor.variables') }}
          </button>
        </div>
      </Transition>
    </div>

    <!-- Separator -->
    <div class="header-separator" />

    <!-- Undo/Redo Controls -->
    <UndoRedoControls />

    <!-- Save Status (gray text) -->
    <span class="save-status">{{ saveStatusText }}</span>

    <!-- Project Picker Modal -->
    <ProjectPickerModal
      :show="showProjectPicker"
      :current-project-id="project?.id"
      @close="showProjectPicker = false"
      @select="handleSelectProject"
      @create="handleCreateProject"
    />

    <!-- Run History Modal -->
    <RunHistoryModal
      :show="showRunHistory"
      :project-id="project?.id"
      @close="showRunHistory = false"
    />
  </header>
</template>

<style scoped>
.floating-header {
  position: fixed;
  top: 12px;
  left: 12px;
  z-index: 100;

  display: flex;
  align-items: center;
  gap: 12px;

  padding: 6px 12px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 10px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

/* Project Trigger */
.project-trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: transparent;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.project-trigger:hover {
  background: rgba(0, 0, 0, 0.05);
}

.project-name {
  font-size: 14px;
  font-weight: 600;
  color: #111827;
  max-width: 200px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.project-trigger .chevron {
  color: #6b7280;
  transition: transform 0.15s;
}

.project-trigger:hover .chevron {
  color: #374151;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  background: none;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  color: #374151;
  cursor: pointer;
  text-align: left;
}

.menu-item:hover {
  background: #f3f4f6;
}

.menu-item svg {
  color: #6b7280;
}

.menu-divider {
  height: 1px;
  margin: 4px 0;
  background: #e5e7eb;
}

/* Header Separator */
.header-separator {
  width: 1px;
  height: 20px;
  background: #e5e7eb;
}

/* Save Status */
.save-status {
  font-size: 12px;
  color: #9ca3af;
}

/* Tools Menu Container */
.tools-menu-container {
  position: relative;
}

/* Tools Trigger */
.tools-trigger {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.15s;
}

.tools-trigger:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #374151;
}

.tools-trigger .chevron {
  transition: transform 0.15s;
}

/* Tools Menu */
.tools-menu {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 10;

  min-width: 200px;
  padding: 4px;
  background: white;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

/* Dropdown Transition */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.15s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
