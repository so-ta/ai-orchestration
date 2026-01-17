<script setup lang="ts">
/**
 * EditorProjectSelector.vue
 * プロジェクト選択ドロップダウン
 *
 * 機能:
 * - 現在のプロジェクト名を表示（クリックでドロップダウン）
 * - ステータスバッジ（draft/published）+ バージョン表示
 * - ドロップダウン内:
 *   - 検索入力
 *   - 最近のプロジェクト一覧
 *   - 「新規プロジェクト」ボタン
 */

import type { Project } from '~/types/api'

const { t } = useI18n()
const projects = useProjects()

defineProps<{
  project: Project | null
}>()

const emit = defineEmits<{
  (e: 'select', projectId: string): void
  (e: 'create'): void
}>()

const isOpen = ref(false)
const searchQuery = ref('')
const recentProjects = ref<Project[]>([])
const loading = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)

// Fetch recent projects when dropdown opens
async function fetchRecentProjects() {
  if (recentProjects.value.length > 0) return

  loading.value = true
  try {
    const response = await projects.list({
      limit: 20,
    })
    recentProjects.value = response.data || []
  } catch {
    recentProjects.value = []
  } finally {
    loading.value = false
  }
}

// Filtered projects based on search
const filteredProjects = computed(() => {
  if (!searchQuery.value) return recentProjects.value
  const query = searchQuery.value.toLowerCase()
  return recentProjects.value.filter(p =>
    p.name.toLowerCase().includes(query) ||
    p.description?.toLowerCase().includes(query)
  )
})

function toggleDropdown() {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    fetchRecentProjects()
  }
}

function selectProject(projectId: string) {
  emit('select', projectId)
  isOpen.value = false
  searchQuery.value = ''
}

function createNewProject() {
  emit('create')
  isOpen.value = false
  searchQuery.value = ''
}

// Close dropdown when clicking outside
function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    isOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<template>
  <div ref="dropdownRef" class="project-selector">
    <!-- Trigger button -->
    <button
      class="selector-trigger"
      :class="{ open: isOpen }"
      @click="toggleDropdown"
    >
      <span class="project-name">{{ project?.name || t('editor.noProjectSelected') }}</span>
      <svg class="chevron" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="6 9 12 15 18 9" />
      </svg>
    </button>

    <!-- Dropdown menu -->
    <Transition name="dropdown">
      <div v-if="isOpen" class="selector-dropdown">
        <!-- Search input -->
        <div class="search-container">
          <svg class="search-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            class="search-input"
            :placeholder="t('projects.searchPlaceholder')"
          >
        </div>

        <!-- Loading state -->
        <div v-if="loading" class="loading-state">
          <div class="loading-spinner" />
        </div>

        <!-- Project list -->
        <div v-else class="project-list">
          <button
            v-for="p in filteredProjects"
            :key="p.id"
            :class="['project-item', { active: p.id === project?.id }]"
            @click="selectProject(p.id)"
          >
            <span class="item-name">{{ p.name }}</span>
            <svg v-if="p.id === project?.id" class="check-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="20 6 9 17 4 12" />
            </svg>
          </button>

          <div v-if="filteredProjects.length === 0 && !loading" class="empty-state">
            {{ t('projects.noMatchingProjects') }}
          </div>
        </div>

        <!-- Create new button -->
        <div class="dropdown-footer">
          <button class="create-btn" @click="createNewProject">
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
            {{ t('projects.newProject') }}
          </button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.project-selector {
  position: relative;
}

.selector-trigger {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.625rem;
  background: transparent;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.15s;
  max-width: 300px;
}

.selector-trigger:hover,
.selector-trigger.open {
  background: var(--color-surface);
  border-color: var(--color-border-dark, #d1d5db);
}

.project-info {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
  text-align: left;
}

.project-name {
  font-weight: 500;
  font-size: 0.875rem;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.project-meta {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.status-badge {
  padding: 0.125rem 0.375rem;
  font-size: 0.625rem;
  font-weight: 500;
  border-radius: 3px;
  text-transform: uppercase;
}

.status-badge.published {
  background: #dcfce7;
  color: #16a34a;
}

.status-badge.draft {
  background: #fef3c7;
  color: #d97706;
}

.version-badge {
  font-size: 0.625rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  font-family: 'SF Mono', Monaco, monospace;
}

.draft-indicator {
  font-size: 0.625rem;
  color: #f59e0b;
}

.chevron {
  color: var(--color-text-secondary);
  transition: transform 0.15s;
  flex-shrink: 0;
}

.selector-trigger.open .chevron {
  transform: rotate(180deg);
}

/* Dropdown */
.selector-dropdown {
  position: absolute;
  top: calc(100% + 0.375rem);
  left: 0;
  min-width: 280px;
  max-width: 320px;
  background: white;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg, 0.5rem);
  box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
  z-index: 100;
  overflow: hidden;
}

/* Search */
.search-container {
  position: relative;
  padding: 0.5rem;
  border-bottom: 1px solid var(--color-border);
}

.search-icon {
  position: absolute;
  left: 1rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--color-text-secondary);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 0.5rem 0.75rem 0.5rem 2.25rem;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 0.875rem;
  background: var(--color-surface);
}

.search-input:focus {
  outline: none;
  border-color: var(--color-primary);
}

/* Loading */
.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Project list */
.project-list {
  max-height: 280px;
  overflow-y: auto;
  padding: 0.375rem;
}

.project-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 0.5rem 0.625rem;
  background: transparent;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  transition: background 0.15s;
  text-align: left;
}

.project-item:hover {
  background: var(--color-surface);
}

.project-item.active {
  background: var(--color-primary-light, #eff6ff);
}

.item-info {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  min-width: 0;
}

.item-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-meta {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.status-dot.published {
  background: #22c55e;
}

.status-dot.draft {
  background: #f59e0b;
}

.item-version {
  font-size: 0.625rem;
  color: var(--color-text-secondary);
  font-family: 'SF Mono', Monaco, monospace;
}

.check-icon {
  color: var(--color-primary);
  flex-shrink: 0;
}

.empty-state {
  padding: 1rem;
  text-align: center;
  font-size: 0.8125rem;
  color: var(--color-text-secondary);
}

/* Footer */
.dropdown-footer {
  padding: 0.5rem;
  border-top: 1px solid var(--color-border);
}

.create-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.5rem;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}

.create-btn:hover {
  opacity: 0.9;
}

/* Transition */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
