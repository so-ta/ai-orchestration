<script setup lang="ts">
/**
 * RunHistoryModal.vue
 * 実行履歴ポップアップモーダル
 *
 * WorkflowRunHistory をモーダル内に表示
 */

const { t } = useI18n()

defineProps<{
  show: boolean
  projectId?: string | null
}>()

const emit = defineEmits<{
  close: []
}>()

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
        <div class="history-modal">
          <!-- Header -->
          <div class="modal-header">
            <h2>{{ t('editor.history') }}</h2>
            <button class="close-btn" @click="emit('close')">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          <!-- Content -->
          <div class="modal-content">
            <WorkflowRunHistory v-if="projectId" :workflow-id="projectId" />
            <div v-else class="empty-state">
              <p>{{ t('editor.noProjectSelected') }}</p>
            </div>
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

.history-modal {
  width: 100%;
  max-width: 600px;
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
  flex-shrink: 0;
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

/* Content */
.modal-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
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

/* Override nested component styles */
.modal-content :deep(.run-history) {
  padding: 0;
}

.modal-content :deep(.run-history-title) {
  display: none;
}

/* Modal Transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .history-modal,
.modal-leave-active .history-modal {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .history-modal,
.modal-leave-to .history-modal {
  opacity: 0;
  transform: scale(0.95) translateY(-20px);
}
</style>
