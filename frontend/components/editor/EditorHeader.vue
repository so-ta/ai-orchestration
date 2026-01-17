<script setup lang="ts">
/**
 * EditorHeader.vue
 * フルスクリーンエディタのフローティングヘッダー
 *
 * 機能:
 * - ロゴ
 * - ProjectSelector: ドロップダウンでプロジェクト切替・検索・新規作成
 * - ProjectActions: Save/Save Draft/Discard Draft/Run ボタン
 * - SecondaryMenus: Runs/Schedules/Variables トグルボタン
 * - UserArea: Admin リンク + UserMenu
 */

import type { Project, Step, Edge, BlockDefinition } from '~/types/api'
import type { SlideOutPanel } from '~/composables/useEditorState'

const { t } = useI18n()
const { isAdmin } = useAuth()

defineProps<{
  project: Project | null
  saving: boolean
  activeSlideOut: SlideOutPanel
  steps?: Step[]
  edges?: Edge[]
  blockDefinitions?: BlockDefinition[]
}>()

const emit = defineEmits<{
  (e: 'save' | 'saveDraft' | 'discardDraft' | 'run' | 'createProject' | 'openSettings' | 'autoLayout'): void
  (e: 'toggleSlideOut', panel: Exclude<SlideOutPanel, null>): void
  (e: 'selectProject', projectId: string): void
}>()

const showSettingsModal = ref(false)

function handleOpenSettings() {
  showSettingsModal.value = true
}

function handleCloseSettings() {
  showSettingsModal.value = false
}
</script>

<template>
  <header class="editor-header">
    <!-- Left section: Logo + Project Selector -->
    <div class="header-left">
      <div class="logo">
        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
        </svg>
      </div>

      <EditorProjectSelector
        :project="project"
        @select="emit('selectProject', $event)"
        @create="emit('createProject')"
      />
    </div>

    <!-- Center section: Primary Actions -->
    <div class="header-center">
      <EditorProjectActions
        :project="project"
        :saving="saving"
        @save="emit('save')"
        @save-draft="emit('saveDraft')"
        @discard-draft="emit('discardDraft')"
        @run="emit('run')"
        @auto-layout="emit('autoLayout')"
      />
    </div>

    <!-- Right section: Secondary Menus + Admin + User -->
    <div class="header-right">
      <EditorSecondaryMenus
        :active-panel="activeSlideOut"
        @toggle="emit('toggleSlideOut', $event)"
      />

      <div class="divider" />

      <!-- Admin Link -->
      <NuxtLink v-if="isAdmin()" to="/admin" class="admin-link">
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        </svg>
        {{ t('nav.admin') }}
      </NuxtLink>

      <!-- User Menu -->
      <UserMenu @open-settings="handleOpenSettings" />
    </div>

    <!-- Settings Modal -->
    <SettingsModal :show="showSettingsModal" @close="handleCloseSettings" />
  </header>
</template>

<style scoped>
.editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1rem;
  height: 48px;
  background: white;
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
  gap: 1rem;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
  min-width: 0;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  flex-shrink: 0;
}

.header-center {
  display: flex;
  align-items: center;
  justify-content: center;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
  justify-content: flex-end;
}

.divider {
  width: 1px;
  height: 24px;
  background: var(--color-border);
}

.admin-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.625rem;
  text-decoration: none;
  color: var(--color-text-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  border-radius: var(--radius);
  transition: all 0.15s;
}

.admin-link:hover {
  background: var(--color-surface);
  color: var(--color-text);
}

@media (max-width: 900px) {
  .header-center {
    display: none;
  }
}

@media (max-width: 768px) {
  .admin-link span {
    display: none;
  }

  .divider {
    display: none;
  }
}
</style>
