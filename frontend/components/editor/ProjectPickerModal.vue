<script setup lang="ts">
/**
 * ProjectPickerModal.vue
 * プロジェクト選択ポップアップ（リスト型）
 *
 * 機能:
 * - 検索入力
 * - プロジェクト一覧
 * - 現在選択中のプロジェクトにチェックマーク
 * - 新規プロジェクト作成ボタン
 */

import type { Project } from '~/types/api'

const { t } = useI18n()
const projects = useProjects()

const props = defineProps<{
  show: boolean
  currentProjectId?: string | null
}>()

const emit = defineEmits<{
  close: []
  select: [projectId: string]
  create: []
}>()

const searchQuery = ref('')
const projectList = ref<Project[]>([])
const loading = ref(false)

// Fetch projects when modal opens
watch(() => props.show, async (show) => {
  if (show) {
    searchQuery.value = ''
    await fetchProjects()
  }
})

async function fetchProjects() {
  loading.value = true
  try {
    const response = await projects.list({ limit: 50 })
    projectList.value = response.data || []
  } catch {
    projectList.value = []
  } finally {
    loading.value = false
  }
}

// Filtered projects based on search
const filteredProjects = computed(() => {
  if (!searchQuery.value) return projectList.value
  const query = searchQuery.value.toLowerCase()
  return projectList.value.filter(p =>
    p.name.toLowerCase().includes(query) ||
    p.description?.toLowerCase().includes(query)
  )
})

function selectProject(projectId: string) {
  emit('select', projectId)
  emit('close')
}

function createProject() {
  emit('create')
  emit('close')
}

function handleOverlayClick(event: MouseEvent) {
  if (event.target === event.currentTarget) {
    emit('close')
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="modal-overlay" @click="handleOverlayClick">
        <div class="picker-modal">
          <!-- Header -->
          <div class="modal-header">
            <h2>{{ t('projectPicker.title') }}</h2>
            <button class="close-btn" @click="emit('close')">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          <!-- Search -->
          <div class="search-container">
            <svg class="search-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8" />
              <path d="m21 21-4.3-4.3" />
            </svg>
            <input
              v-model="searchQuery"
              type="text"
              class="search-input"
              :placeholder="t('projectPicker.searchPlaceholder')"
              autofocus
            >
          </div>

          <!-- Project List -->
          <div class="project-list">
            <!-- Loading -->
            <div v-if="loading" class="loading-state">
              <div class="loading-spinner" />
            </div>

            <!-- Empty state -->
            <div v-else-if="filteredProjects.length === 0" class="empty-state">
              <p>{{ searchQuery ? t('projectPicker.noResults') : t('projectPicker.noProjects') }}</p>
            </div>

            <!-- Projects -->
            <button
              v-for="project in filteredProjects"
              v-else
              :key="project.id"
              :class="['project-item', { active: project.id === currentProjectId }]"
              @click="selectProject(project.id)"
            >
              <span class="project-name">{{ project.name }}</span>
              <svg
                v-if="project.id === currentProjectId"
                class="check-icon"
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
            </button>
          </div>

          <!-- Footer -->
          <div class="modal-footer">
            <button class="create-btn" @click="createProject">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="12" y1="5" x2="12" y2="19" />
                <line x1="5" y1="12" x2="19" y2="12" />
              </svg>
              {{ t('projectPicker.createNew') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 10vh;
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(4px);
}

.picker-modal {
  width: 100%;
  max-width: 480px;
  max-height: 70vh;
  background: white;
  border-radius: 16px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Header */
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid #e5e7eb;
}

.modal-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 8px;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.15s;
}

.close-btn:hover {
  background: #f3f4f6;
  color: #111827;
}

/* Search */
.search-container {
  position: relative;
  padding: 16px 24px;
  border-bottom: 1px solid #e5e7eb;
}

.search-icon {
  position: absolute;
  left: 40px;
  top: 50%;
  transform: translateY(-50%);
  color: #9ca3af;
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 12px 16px 12px 44px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  font-size: 15px;
  color: #111827;
  background: #f9fafb;
  transition: all 0.15s;
}

.search-input:focus {
  outline: none;
  border-color: #3b82f6;
  background: white;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-input::placeholder {
  color: #9ca3af;
}

/* Project List */
.project-list {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #e5e7eb;
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
}

.empty-state p {
  margin: 0;
  font-size: 14px;
  color: #6b7280;
}

/* Project Item */
.project-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 12px 16px;
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;
  text-align: left;
}

.project-item:hover {
  background: #f3f4f6;
}

.project-item.active {
  background: #eff6ff;
}

.project-item.active:hover {
  background: #dbeafe;
}

.project-name {
  font-size: 14px;
  font-weight: 500;
  color: #111827;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.check-icon {
  color: #3b82f6;
  flex-shrink: 0;
}

/* Footer */
.modal-footer {
  padding: 16px 24px;
  border-top: 1px solid #e5e7eb;
  background: #f9fafb;
}

.create-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 12px;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  color: #374151;
  cursor: pointer;
  transition: all 0.15s;
}

.create-btn:hover {
  background: #f9fafb;
  border-color: #d1d5db;
}

/* Modal Transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .picker-modal,
.modal-leave-active .picker-modal {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .picker-modal,
.modal-leave-to .picker-modal {
  opacity: 0;
  transform: scale(0.95) translateY(-20px);
}
</style>
