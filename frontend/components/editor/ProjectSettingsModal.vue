<script setup lang="ts">
/**
 * ProjectSettingsModal.vue
 * プロジェクト設定モーダル
 *
 * 機能:
 * - プロジェクト名の編集
 * - プロジェクト説明の編集
 */

import type { Project } from '~/types/api'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  project: Project | null
}>()

const emit = defineEmits<{
  close: []
  save: [data: { name: string; description: string }]
}>()

const name = ref('')
const description = ref('')
const saving = ref(false)

// Initialize form when project changes or modal opens
watch([() => props.project, () => props.show], ([project, show]) => {
  if (show && project) {
    name.value = project.name
    description.value = project.description || ''
  }
}, { immediate: true })

function handleSave() {
  if (!name.value.trim()) return
  saving.value = true
  emit('save', {
    name: name.value.trim(),
    description: description.value.trim()
  })
}

function handleClose() {
  saving.value = false
  emit('close')
}

function handleOverlayClick(event: MouseEvent) {
  if (event.target === event.currentTarget) {
    handleClose()
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="modal-overlay" @click="handleOverlayClick">
        <div class="settings-modal">
          <!-- Header -->
          <div class="modal-header">
            <h2>{{ t('editor.projectSettings') }}</h2>
            <button class="close-btn" @click="handleClose">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          <!-- Form -->
          <div class="form-content">
            <!-- Project Name -->
            <div class="form-group">
              <label for="project-name">{{ t('editor.projectName') }}</label>
              <input
                id="project-name"
                v-model="name"
                type="text"
                class="form-input"
                :placeholder="t('editor.untitledProject')"
                autofocus
              >
            </div>

            <!-- Project Description -->
            <div class="form-group">
              <label for="project-description">{{ t('editor.projectDescription') }}</label>
              <textarea
                id="project-description"
                v-model="description"
                class="form-textarea"
                :placeholder="t('editor.projectDescriptionPlaceholder')"
                rows="4"
              />
            </div>
          </div>

          <!-- Footer -->
          <div class="modal-footer">
            <button class="cancel-btn" @click="handleClose">
              {{ t('common.cancel') }}
            </button>
            <button
              class="save-btn"
              :disabled="!name.trim() || saving"
              @click="handleSave"
            >
              <span v-if="saving" class="loading-spinner" />
              {{ saving ? t('editor.saving') : t('common.save') }}
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

.settings-modal {
  width: 100%;
  max-width: 480px;
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

/* Form */
.form-content {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  font-size: 15px;
  color: #111827;
  background: #f9fafb;
  transition: all 0.15s;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: #3b82f6;
  background: white;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input::placeholder,
.form-textarea::placeholder {
  color: #9ca3af;
}

.form-textarea {
  resize: vertical;
  min-height: 100px;
  font-family: inherit;
}

/* Footer */
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid #e5e7eb;
  background: #f9fafb;
}

.cancel-btn,
.save-btn {
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.cancel-btn {
  background: white;
  border: 1px solid #e5e7eb;
  color: #374151;
}

.cancel-btn:hover {
  background: #f9fafb;
  border-color: #d1d5db;
}

.save-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  background: #3b82f6;
  border: 1px solid #3b82f6;
  color: white;
}

.save-btn:hover:not(:disabled) {
  background: #2563eb;
  border-color: #2563eb;
}

.save-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.loading-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Modal Transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .settings-modal,
.modal-leave-active .settings-modal {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .settings-modal,
.modal-leave-to .settings-modal {
  opacity: 0;
  transform: scale(0.95) translateY(-20px);
}
</style>
