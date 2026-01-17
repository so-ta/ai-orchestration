<script setup lang="ts">
/**
 * ReleaseModal.vue
 * リリース（バージョンスナップショット）作成モーダル
 */

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  projectName?: string
}>()

const emit = defineEmits<{
  close: []
  create: [name: string, description: string]
}>()

const releaseName = ref('')
const releaseDescription = ref('')
const creating = ref(false)

// Reset form when modal opens
watch(() => props.show, (show) => {
  if (show) {
    releaseName.value = ''
    releaseDescription.value = ''
  }
})

async function handleCreate() {
  if (!releaseName.value.trim()) return

  creating.value = true
  try {
    emit('create', releaseName.value.trim(), releaseDescription.value.trim())
  } finally {
    creating.value = false
  }
}

function handleClose() {
  if (!creating.value) {
    emit('close')
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="modal-overlay" @click.self="handleClose">
        <div class="modal-content">
          <div class="modal-header">
            <h3>{{ t('editor.createReleaseTitle') }}</h3>
            <button class="close-btn" @click="handleClose">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>

          <div class="modal-body">
            <p class="modal-description">
              {{ t('editor.createReleaseDescription') }}
            </p>

            <div class="form-group">
              <label for="releaseName">{{ t('editor.releaseName') }}</label>
              <input
                id="releaseName"
                v-model="releaseName"
                type="text"
                class="form-input"
                :placeholder="t('editor.releaseNamePlaceholder')"
                autofocus
                @keyup.enter="handleCreate"
              >
            </div>

            <div class="form-group">
              <label for="releaseDescription">{{ t('editor.releaseDescription') }}</label>
              <textarea
                id="releaseDescription"
                v-model="releaseDescription"
                class="form-textarea"
                :placeholder="t('editor.releaseDescriptionPlaceholder')"
                rows="3"
              />
            </div>
          </div>

          <div class="modal-footer">
            <button class="btn-cancel" @click="handleClose">
              {{ t('common.cancel') }}
            </button>
            <button
              class="btn-create"
              :disabled="!releaseName.trim() || creating"
              @click="handleCreate"
            >
              <svg v-if="creating" class="spinning" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12a9 9 0 1 1-6.219-8.56" />
              </svg>
              <span>{{ t('editor.createReleaseButton') }}</span>
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
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(4px);
}

.modal-content {
  width: 100%;
  max-width: 420px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid #e5e7eb;
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #111827;
}

.close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 6px;
  color: #6b7280;
  cursor: pointer;
  transition: all 0.15s;
}

.close-btn:hover {
  background: #f3f4f6;
  color: #111827;
}

.modal-body {
  padding: 20px;
}

.modal-description {
  margin: 0 0 16px;
  font-size: 13px;
  color: #6b7280;
  line-height: 1.5;
}

.form-group {
  margin-bottom: 16px;
}

.form-group:last-child {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: #374151;
}

.form-input,
.form-textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  font-size: 14px;
  color: #111827;
  background: white;
  transition: border-color 0.15s;
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input::placeholder,
.form-textarea::placeholder {
  color: #9ca3af;
}

.form-textarea {
  resize: vertical;
  min-height: 80px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 20px;
  background: #f9fafb;
  border-top: 1px solid #e5e7eb;
}

.btn-cancel,
.btn-create {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-cancel {
  background: white;
  border: 1px solid #e5e7eb;
  color: #374151;
}

.btn-cancel:hover {
  background: #f9fafb;
  border-color: #d1d5db;
}

.btn-create {
  background: #3b82f6;
  border: none;
  color: white;
}

.btn-create:hover:not(:disabled) {
  background: #2563eb;
}

.btn-create:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Modal Transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .modal-content,
.modal-leave-active .modal-content {
  transition: transform 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.95) translateY(-10px);
}
</style>
