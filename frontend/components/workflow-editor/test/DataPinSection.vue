<script setup lang="ts">
import type { PinnedOutput } from '~/composables/test'

const props = defineProps<{
  pinnedOutput: PinnedOutput
}>()

const emit = defineEmits<{
  'unpin': []
  'edit': [data: unknown]
}>()

const { t } = useI18n()

// Edit modal state
const showEditModal = ref(false)
const editJson = ref('')
const editError = ref<string | null>(null)

// Format timestamp
const timestampDisplay = computed(() => {
  const date = new Date(props.pinnedOutput.pinnedAt)
  return date.toLocaleString('ja-JP', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
})

// Format data preview
const dataPreview = computed(() => {
  const data = props.pinnedOutput.data
  const json = JSON.stringify(data, null, 2)
  if (json.length > 200) {
    return json.substring(0, 200) + '...'
  }
  return json
})

// Open edit modal
function openEdit() {
  editJson.value = JSON.stringify(props.pinnedOutput.data, null, 2)
  editError.value = null
  showEditModal.value = true
}

// Save edit
function saveEdit() {
  try {
    const parsed = JSON.parse(editJson.value)
    emit('edit', parsed)
    showEditModal.value = false
  } catch {
    editError.value = t('execution.errors.invalidJson')
  }
}

// Cancel edit
function cancelEdit() {
  showEditModal.value = false
  editError.value = null
}
</script>

<template>
  <div class="pin-section">
    <div class="pin-header">
      <div class="pin-title">
        <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="m12 17 2 5 2-5"/>
          <path d="m6 12 1.09 4.36a2 2 0 0 0 1.94 1.64h5.94a2 2 0 0 0 1.94-1.64L18 12"/>
          <path d="M6 8a6 6 0 1 1 12 0c0 5-6 6-6 12h0c0-6-6-7-6-12Z"/>
        </svg>
        <span>{{ t('test.pinData.title') }}</span>
        <span class="pin-timestamp">{{ timestampDisplay }}</span>
      </div>
      <div class="pin-actions">
        <button class="btn-icon-sm" :title="t('test.pinData.edit')" @click="openEdit">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
          </svg>
        </button>
        <button class="btn-icon-sm btn-danger" :title="t('test.pinData.unpin')" @click="emit('unpin')">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 6h18"/>
            <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/>
          </svg>
        </button>
      </div>
    </div>
    <div class="pin-content">
      <pre class="pin-preview">{{ dataPreview }}</pre>
    </div>

    <!-- Edit Modal -->
    <Teleport to="body">
      <div v-if="showEditModal" class="modal-overlay" @click.self="cancelEdit">
        <div class="modal-content">
          <div class="modal-header">
            <h3>{{ t('test.pinData.editTitle') }}</h3>
            <button class="modal-close" @click="cancelEdit">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
          <div class="modal-body">
            <textarea
              v-model="editJson"
              class="edit-textarea"
              rows="12"
            />
            <p v-if="editError" class="edit-error">{{ editError }}</p>
          </div>
          <div class="modal-footer">
            <button class="btn btn-outline" @click="cancelEdit">
              {{ t('common.cancel') }}
            </button>
            <button class="btn btn-primary" @click="saveEdit">
              {{ t('common.save') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.pin-section {
  background: #fefce8;
  border: 1px solid #fef08a;
  border-radius: 8px;
  overflow: hidden;
}

.pin-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 0.75rem;
  background: #fef9c3;
  border-bottom: 1px solid #fef08a;
}

.pin-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.6875rem;
  font-weight: 600;
  color: #ca8a04;
}

.pin-timestamp {
  font-weight: 400;
  opacity: 0.8;
}

.pin-actions {
  display: flex;
  gap: 0.25rem;
}

.btn-icon-sm {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 1px solid #fef08a;
  border-radius: 4px;
  background: white;
  color: #ca8a04;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-icon-sm:hover {
  background: #fef9c3;
  border-color: #facc15;
}

.btn-icon-sm.btn-danger:hover {
  background: #fef2f2;
  border-color: #fecaca;
  color: #dc2626;
}

.pin-content {
  padding: 0.5rem 0.75rem;
}

.pin-preview {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 0.6875rem;
  line-height: 1.5;
  margin: 0;
  color: #713f12;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 100px;
  overflow-y: auto;
}

/* Modal styles */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-bottom: 1px solid var(--color-border);
}

.modal-header h3 {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
}

.modal-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}

.modal-close:hover {
  background: var(--color-surface);
  color: var(--color-text);
}

.modal-body {
  padding: 1rem;
  overflow-y: auto;
  flex: 1;
}

.edit-textarea {
  width: 100%;
  padding: 0.75rem;
  font-size: 0.75rem;
  font-family: 'SF Mono', Monaco, monospace;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  resize: vertical;
  min-height: 200px;
}

.edit-textarea:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.edit-error {
  font-size: 0.75rem;
  color: var(--color-error);
  margin: 0.5rem 0 0;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding: 1rem;
  border-top: 1px solid var(--color-border);
}

.btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
  border: none;
}

.btn-primary:hover {
  background: #2563eb;
}

.btn-outline {
  background: white;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.btn-outline:hover {
  background: var(--color-surface);
}
</style>
